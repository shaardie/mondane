package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/golang/protobuf/ptypes"
	"github.com/shaardie/mondane/alert/proto"
)

var (
	// Command line arguments
	server = kingpin.Flag("server", "server address").Default("127.0.0.1:8084").String()

	create           = kingpin.Command("create", "create an alert")
	createUserID     = create.Arg("user-id", "user id of the alert").Required().Int64()
	createCheckID    = create.Arg("check-id", "check id of the alert").Required().Int64()
	createCheckType  = create.Arg("check-type", "type of the alert").Required().String()
	createSendMail   = create.Arg("send-mail", "if mail is sent").Required().Bool()
	createSendPeriod = create.Arg("send-period", "period in second between sends").Required().Int64()

	firing     = kingpin.Command("firing", "firing an alert")
	firingID   = firing.Arg("id", "id of the check to fire").Required().Int64()
	firingType = firing.Arg("type", "type of the check to fire").Required().String()
)

func mainWithError() error {
	parse := kingpin.Parse()

	// Connect to user service
	d, err := grpc.Dial(*server, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("unable to connect to user server, %v", err)
	}
	c := proto.NewAlertServiceClient(d)
	defer d.Close()

	// Switch to different modes
	switch parse {
	case "create":
		_, err := c.CreateAlert(context.Background(), &proto.Alert{
			UserId:    *createUserID,
			CheckId:   *createCheckID,
			CheckType: *createCheckType,
			SendMail:  *createSendMail,
			SendPeriod: ptypes.DurationProto(
				time.Second * time.Duration(*createSendPeriod)),
		})
		if err != nil {
			return fmt.Errorf("Unable to trigger alert: %v", err)
		}
	case "firing":
		_, err := c.Firing(context.Background(), &proto.Check{
			Id: *firingID, Type: *firingType,
		})
		if err != nil {
			return fmt.Errorf("Unable to fire alert: %v", err)
		}
	}
	return nil
}

func main() {
	if err := mainWithError(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
