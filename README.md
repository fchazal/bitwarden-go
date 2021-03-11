# BITWARDEN-GO

*(Note: This is still a work in progress. This project is not associated with the [Bitwarden](https://bitwarden.com/) project nor 8bit Solutions LLC. Please use gitter or the issue tracker for this repo if you need support. If you need to use the official Bitwarden channels make it clear that you are using a 3rd party backend server)*

## ORIGINS

This project is originating from VictorNine source code who did most of the work. I'm just trying to update it to new Bitwarden standards and solve main issues that disallow me to use this neat code on my server to host my own copy of Bitwarden.


## TODO LIST
 
	[x] Unable to sync on mobile app
	[x] Unable to delete cyphers
	[ ] Unable to delete folders
	[ ] Cyphers bin not implemented
	[ ] HTTPS layer would be a nice feature

## Usage

### Fetching the code
Make sure you have the ```go``` package installed.
*Note: package name may vary based on distribution*

You can then run ```go get github.com/VictorNine/bitwarden-go/cmd/bitwarden-go``` to fetch the latest code.

### Build/Install
Run in your favorite terminal:
```
cd $GOPATH/src/github.com/VictorNine/bitwarden-go/cmd/bitwarden-go
```
followed by
```
go build
```
or
```
go install
```
The former will create a executable named ```bitwarden-go``` in the current directory, and ```go install``` will build and install the executable ```bitwarden-go``` as a system-wide application (located in ```$GOPATH/bin```).
*Note: From here on, this guide assumes you ran ```go install```*

#### Initalizing the Database
*Note: This step only has to be performed once*

Run the following to initalize the database:
```
bitwarden-go -init
```
This will create a database called ```db``` in the directory of the application. Use `-location` to set a different directory for the database.

### Running
To run [bitwarden-go](https://github.com/VictorNine/bitwarden-go), run the following in the terminal:
```
bitwarden-go
```

### Usage with Flags
To see all current flags and options with the application, run
```
bitwarden-go -h
```
