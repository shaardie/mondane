// API Server Binary
package main

import (
	"log"

	"github.com/shaardie/mondane/api"
)

func mainWithError() error {
	// run api server
	return api.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalln(err)
	}
}
