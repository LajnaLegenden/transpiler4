package main

import (
	"log"
	"os"

	"github.com/LajnaLegenden/transpiler4/cli"
)

func main() {
	// Assign our cli to the app variable
	app := cli.SetupCLI()

	err := app.Run(os.Args)

	// Exit program when we get an error and show error
	if err != nil {
		log.Fatal("Error: ", err)
	}
}
