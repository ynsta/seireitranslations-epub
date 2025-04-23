package main

import (
	"os"

	"github.com/ynsta/seireitranslations-epub/cmd/seireitranslations-epub/app"
)

func main() {
	// Simply call the Execute function from the app package
	os.Exit(app.Execute())
}
