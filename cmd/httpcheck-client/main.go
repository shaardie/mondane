package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/shaardie/mondane/httpcheck/proto"
)

var (
	// Command line arguments
	server = kingpin.Flag("server", "server address").Default("127.0.0.1:8085").String()

	do    = kingpin.Command("do", "do a HTTP Check")
	doURL = do.Arg("url", "URL to check").Required().String()
)

func mainWithError() error {
	parse := kingpin.Parse()

	// Connect to user service
	d, err := grpc.Dial(*server, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("unable to connect to user server, %v", err)
	}
	c := proto.NewHTTPCheckServiceClient(d)

	// Switch to different modes
	switch parse {
	case "do":
		result, err := c.Do(context.Background(), &proto.Check{
			Url: *doURL,
		})
		if err != nil {
			return fmt.Errorf("Error during check: %v", err)
		}
		fmt.Printf("success=%v, status_code=%v, duration=%v, error=%v\n",
			result.Success, result.StatusCode, time.Duration(result.Duration), result.Error)
	}

	return nil
}

func main() {
	if err := mainWithError(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
