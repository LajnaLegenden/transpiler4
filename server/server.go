package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/LajnaLegenden/transpiler4/helpers"
)

// Serve handles serving the directory
func Serve() {
	// Generate our port number
	port := helpers.GeneratePortNumber()
	// Inform the user which port we are running
	log.Println("We are running on port: " + strconv.Itoa(port))
	// Exit to a live server
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), http.FileServer(http.Dir("."))))
}
