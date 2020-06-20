package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/shaardie/mondane/mail/proto"
)

var (
	// Command line arguments
	server    = kingpin.Flag("server", "server address").Default("127.0.0.1:8080").String()
	recipient = kingpin.Arg("recipient", "recipient of the mail").Required().String()
	subject   = kingpin.Arg("subject", "subject of the mail").Required().String()
	message   = kingpin.Arg("message", "message of the mail").Required().String()
)

func mainWithError() error {
	kingpin.Parse()

	// Connect to user service
	d, err := grpc.Dial(*server, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("unable to connect to user server, %v", err)
	}
	c := proto.NewMailServiceClient(d)

	_, err = c.SendMail(context.Background(), &proto.Mail{
		Recipient: *recipient,
		Subject:   *subject,
		Message:   *message,
	})
	return err
}

func main() {
	if err := mainWithError(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
