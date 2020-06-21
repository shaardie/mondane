// HTTP Check Service
package main

import (
	"log"

	"github.com/shaardie/mondane/httpcheck"
)

func mainWithError() error {
	// run server
	return httpcheck.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalln(err)
	}
}
