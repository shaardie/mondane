// User Service
package main

import (
	"log"

	"github.com/shaardie/mondane/user"
)

func mainWithError() error {
	// run api server
	return user.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalln(err)
	}
}
