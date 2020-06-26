// Alert Service
package main

import (
	"log"

	"github.com/shaardie/mondane/alert"
)

func mainWithError() error {
	// run server
	return alert.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalln(err)
	}
}
