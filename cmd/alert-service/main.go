// Alert Service
package main

import (
	"os"

	"github.com/shaardie/mondane/alert"
)

func mainWithError() error {
	// run server
	return alert.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		os.Exit(1)
	}
}
