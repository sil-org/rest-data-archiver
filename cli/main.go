package main

import (
	"os"

	rda "github.com/sil-org/rest-data-archiver"
)

func main() {
	configFile := ""
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}
	if err := rda.Run(configFile); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
