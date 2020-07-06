// API Service
package main

import (
	"fmt"
	"os"

	"github.com/shaardie/mondane/api"
)

func mainWithError() error {
	// run server
	return api.Run()
}

func main() {
	if err := mainWithError(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
