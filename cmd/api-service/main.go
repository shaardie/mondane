// API Service
package main

import (
	"os"

	"github.com/shaardie/mondane/api"
)

func mainWithError() error {
	// run server
	return api.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		os.Exit(1)
	}
}
