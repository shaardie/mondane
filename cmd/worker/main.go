// Worker Binary
package main

import (
	"log"

	"github.com/shaardie/mondane/worker"
)

func mainWithError() error {
	// run server
	return worker.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalln(err)
	}
}
