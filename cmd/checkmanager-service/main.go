// Check Manager Service
package main

import (
	"log"

	"github.com/shaardie/mondane/checkmanager"
)

func mainWithError() error {
	// run server
	return checkmanager.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalln(err)
	}
}
