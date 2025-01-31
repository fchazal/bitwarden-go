package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fchazal/bitwarden-go/auth"
	bw "github.com/fchazal/bitwarden-go/common"
)

type APIHandler struct {
	db database
}

func New(db database) APIHandler {
	h := APIHandler{
		db: db,
	}

	return h
}

// Interface to make testing easier
type database interface {
	GetAccount(username string, refreshtoken string) (bw.Account, error)
	UpdateAccountInfo(acc bw.Account) error
	GetCipher(owner string, ciphID string) (bw.Cipher, error)
	GetCiphers(owner string) ([]bw.Cipher, error)
	NewCipher(ciph bw.Cipher, owner string) (bw.Cipher, error)
	UpdateCipher(newData bw.Cipher, owner string, ciphID string) error
	DeleteCipher(owner string, ciphID string) error
	AddFolder(name string, owner string) (bw.Folder, error)
	UpdateFolder(newFolder bw.Folder, owner string) error
	GetFolders(owner string) ([]bw.Folder, error)
}

func (h *APIHandler) HandleKeysUpdate(w http.ResponseWriter, req *http.Request) {
	email := auth.GetEmail(req)

	acc, err := h.db.GetAccount(email, "")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Adding key pair")

	decoder := json.NewDecoder(req.Body)
	var kp bw.KeyPair
	err = decoder.Decode(&kp)
	if err != nil {
		log.Fatal(err)
	}
	defer req.Body.Close()

	acc.KeyPair = kp

	h.db.UpdateAccountInfo(acc)
}

func (h *APIHandler) HandleProfile(w http.ResponseWriter, req *http.Request) {
	email := auth.GetEmail(req)
	log.Println("Profile requested")

	acc, err := h.db.GetAccount(email, "")
	if err != nil {
		log.Fatal(err)
	}

	prof := acc.GetProfile()

	data, err := json.Marshal(&prof)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *APIHandler) HandleCollections(w http.ResponseWriter, req *http.Request) {

	collections := bw.Data{Object: "list", Data: []string{}}
	data, err := json.Marshal(collections)
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *APIHandler) HandleCipher(w http.ResponseWriter, req *http.Request) {
	email := auth.GetEmail(req)

	log.Println(email + " is trying to add data")

	acc, err := h.db.GetAccount(email, "")
	if err != nil {
		log.Fatal("Account lookup " + err.Error())
	}

	var data []byte

	if req.Method == "POST" {
		rCiph, err := unmarshalCipher(req.Body)
		if err != nil {
			log.Fatal("Cipher decode error" + err.Error())
		}

		// Store the new cipher object in db
		newCiph, err := h.db.NewCipher(rCiph, acc.Id)
		if err != nil {
			log.Fatal("newCipher error" + err.Error())
		}
		data, err = json.Marshal(&newCiph)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		ciphs, err := h.db.GetCiphers(acc.Id)
		if err != nil {
			log.Println(err)
		}
		for i, _ := range ciphs {
			ciphs[i].CollectionIds = make([]string, 0)
			ciphs[i].Object = "cipherDetails"
		}
		list := bw.Data{Object: "list", Data: ciphs}
		data, err = json.Marshal(&list)
		if err != nil {
			log.Fatal(err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// This function handles updates and deleteing
func (h *APIHandler) HandleCipherUpdate(w http.ResponseWriter, req *http.Request) {
	email := auth.GetEmail(req)
	log.Println(email + " is trying to edit his data")

	// Get the cipher id
	id := strings.TrimPrefix(req.URL.Path, "/api/ciphers/")

	acc, err := h.db.GetAccount(email, "")
	if err != nil {
		log.Fatal("Account lookup " + err.Error())
	}

	method := req.Method
	if method == "PUT" && strings.HasSuffix(id, "/delete") {
		method = "DELETE" // strange use of API verbs
		id = strings.TrimSuffix(id, "/delete")
	}

	log.Println("Method : " + method)

	switch method {
	case "GET":
		log.Println("GET Ciphers for " + acc.Id)
		var data []byte
		ciph, err := h.db.GetCipher(acc.Id, id)
		if err != nil {
			log.Fatal(err)
		}
		data, err = json.Marshal(&ciph)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	case "POST":
		fallthrough // Do same as PUT. Web Vault want's to post
	case "PUT":
		log.Println(req.Body)
		rCiph, err := unmarshalCipher(req.Body)
		if err != nil {
			log.Fatal("Cipher decode error" + err.Error())
		}

		// Set correct ID
		rCiph.Id = id

		err = h.db.UpdateCipher(rCiph, acc.Id, id)
		if err != nil {
			w.Write([]byte("0"))
			log.Println(err)
			return
		}

		// Send response
		data, err := json.Marshal(&rCiph)
		if err != nil {
			log.Fatal(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
		log.Println("Cipher " + id + " updated")
		return

	case "DELETE":
		err := h.db.DeleteCipher(acc.Id, id)
		if err != nil {
			w.Write([]byte("0"))
			log.Println(err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(""))
		log.Println("Cipher " + id + " deleted")
		return
	default:
		w.Write([]byte("0"))
		return
	}

}

func (h *APIHandler) HandleSync(w http.ResponseWriter, req *http.Request) {
	email := auth.GetEmail(req)

	log.Println(email + " is trying to sync")

	acc, err := h.db.GetAccount(email, "")

	prof := bw.Profile{
		Id:               acc.Id,
		Name:             &acc.Name,
		Email:            acc.Email,
		EmailVerified:    false,
		Premium:          false,
		Culture:          "en-US",
		TwoFactorEnabled: false,
		Key:              acc.Key,
		SecurityStamp:    &acc.Id,
		Organizations:    []string{},
		Object:           "profile",
	}

	ciphs, err := h.db.GetCiphers(acc.Id)
	if err != nil {
		log.Println(err)
	}

	folders, err := h.db.GetFolders(acc.Id)
	if err != nil {
		log.Println(err)
	}

	Domains := bw.Domains{
		Object:            "domains",
		EquivalentDomains: nil,
		GlobalEquivalentDomains: []bw.GlobalEquivalentDomains{
			bw.GlobalEquivalentDomains{Type: 1, Domains: []string{"youtube.com", "google.com", "gmail.com"}, Excluded: false},
		},
	}

	data := bw.SyncData{
		Profile:     prof,
		Folders:     folders,
		Domains:     Domains,
		Object:      "sync",
		Ciphers:     ciphs,
		Collections: []string{},
		Policies:    []string{},
		Sends:       []string{},
	}

	jdata, err := json.Marshal(&data)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jdata)
}

// Only handles ciphers
// TODO: handle folders and folderRelationships
func (h *APIHandler) HandleImport(w http.ResponseWriter, req *http.Request) {
	email := auth.GetEmail(req)

	log.Println(email + " is trying to import data")

	acc, err := h.db.GetAccount(email, "")
	if err != nil {
		log.Fatal("Account lookup " + err.Error())
	}

	decoder := json.NewDecoder(req.Body)
	data := struct {
		Ciphers             []newCipher `json:"ciphers"`
		Foders              []string    `json:"folders"`
		FolderRelationships []string    `json:"folderRelationships"`
	}{}

	err = decoder.Decode(&data)
	if err != nil {
		log.Fatal(err)
	}
	defer req.Body.Close()

	for _, nc := range data.Ciphers {
		c, err := nc.toCipher()
		if err != nil {
			log.Fatal(err.Error())
		}

		_, err = h.db.NewCipher(c, acc.Id)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	w.Write([]byte{0x00})
}

func (h *APIHandler) HandleFolder(w http.ResponseWriter, req *http.Request) {
	email := auth.GetEmail(req)

	log.Println(email + " is trying to add a new folder")

	acc, err := h.db.GetAccount(email, "")
	if err != nil {
		log.Fatal("Account lookup " + err.Error())
	}

	var data []byte
	if req.Method == "POST" {
		decoder := json.NewDecoder(req.Body)

		var folderData struct {
			Name string `json:"name"`
		}

		err = decoder.Decode(&folderData)
		if err != nil {
			log.Fatal(err)
		}
		defer req.Body.Close()

		folder, err := h.db.AddFolder(folderData.Name, acc.Id)
		if err != nil {
			log.Fatal("newFolder error" + err.Error())
		}

		data, err = json.Marshal(&folder)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		folders, err := h.db.GetFolders(acc.Id)
		if err != nil {
			log.Println(err)
		}
		list := bw.Data{Object: "list", Data: folders}
		data, err = json.Marshal(list)
		if err != nil {
			log.Fatal(err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *APIHandler) HandleFolderUpdate(w http.ResponseWriter, req *http.Request) {
	email := auth.GetEmail(req)

	log.Println(email + " is trying to update a folder")

	acc, err := h.db.GetAccount(email, "")
	if err != nil {
		log.Fatal("Account lookup " + err.Error())
	}

	switch req.Method {
	case "POST":
		fallthrough // Do same as PUT. Web Vault wants to post
	case "PUT":
		// Get the folder id
		folderID := strings.TrimPrefix(req.URL.Path, "/api/folders/")

		decoder := json.NewDecoder(req.Body)

		var folderData struct {
			Name string `json:"name"`
		}

		err := decoder.Decode(&folderData)
		if err != nil {
			log.Fatal(err)
		}
		defer req.Body.Close()

		newFolder := bw.Folder{
			Id:           folderID,
			Name:         folderData.Name,
			RevisionDate: time.Now().UTC(),
			Object:       "folder",
		}

		err = h.db.UpdateFolder(newFolder, acc.Id)
		if err != nil {
			w.Write([]byte("0"))
			log.Println(err)
			return
		}

		// Send response
		data, err := json.Marshal(&newFolder)
		if err != nil {
			log.Fatal(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
		log.Println("Folder " + folderID + " updated")
		return

	case "DELETE":
		// Get the folder id
		folderID := strings.TrimPrefix(req.URL.Path, "/api/folders/")

		w.Header().Set("Content-Type", "application/json")
		log.Println("Folder " + folderID + " deleted")
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

// The data we get from the client. Only used to parse data
type newCipher struct {
	Type           int       `json:"type"`
	FolderId       string    `json:"folderId"`
	OrganizationId string    `json:"organizationId"`
	Name           string    `json:"name"`
	Notes          string    `json:"notes"`
	Favorite       bool      `json:"favorite"`
	Login          loginData `json:"login"`
	Fields         string    `json:"fields"`
}

type loginData struct {
	URI      string   `json:"uri"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	ToTp     string   `json:"totp"`
	Uris     []bw.Uri `json:"uris"`
}

func (nciph *newCipher) toCipher() (bw.Cipher, error) {
	// Create new
	cdata := bw.CipherData{
		Uri:      &nciph.Login.URI,
		Username: &nciph.Login.Username,
		Password: &nciph.Login.Password,
		Totp:     nil,
		Name:     &nciph.Name,
		Notes:    new(string),
		Fields:   nil,
		Uris:     nciph.Login.Uris,
	}

	(*cdata.Notes) = nciph.Notes

	if *cdata.Notes == "" {
		cdata.Notes = nil
	}

	if *cdata.Uri == "" {
		cdata.Uri = nil
	}

	if cdata.Uri == nil {
		if len(nciph.Login.Uris) > 0 { // TODO: Also add to Uris
			cdata.Uri = nciph.Login.Uris[0].Uri
		}
	}

	if *cdata.Username == "" {
		cdata.Username = nil
	}

	if *cdata.Password == "" {
		cdata.Password = nil
	}

	if *cdata.Name == "" {
		cdata.Name = nil
	}

	ciph := bw.Cipher{ // Only including the data we use when we store it
		Type:     nciph.Type,
		Data:     cdata,
		Favorite: nciph.Favorite,
	}

	if nciph.FolderId != "" {
		ciph.FolderId = &nciph.FolderId
	}

	bw.FakeNewAPI(&ciph)

	return ciph, nil
}

// unmarshalCipher Take the recived bytes and make it a Cipher struct
func unmarshalCipher(data io.ReadCloser) (bw.Cipher, error) {
	decoder := json.NewDecoder(data)
	var nciph newCipher
	err := decoder.Decode(&nciph)
	if err != nil {
		return bw.Cipher{}, nil
	}

	defer data.Close()

	return nciph.toCipher()
}
