// Mail Service
package main

import (
	"log"

	"github.com/shaardie/mondane/mail"
)

func mainWithError() error {
	// run api server
	return mail.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalln(err)
	}
}
