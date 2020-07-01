package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/shaardie/mondane/checkmanager/proto"
)

var (
	// Command line arguments
	server = kingpin.Flag("server", "server address").Default("127.0.0.1:8083").String()

	httpCheck = kingpin.Command("httpcheck", "httpcheck related commands")

	httpCheckCreate       = httpCheck.Command("create", "create a check")
	httpCheckCreateUserID = httpCheckCreate.Arg("user-id", "id of the user").Required().Int64()
	httpCheckCreateURL    = httpCheckCreate.Arg("url", "url for the check").Required().String()

	httpCheckget   = httpCheck.Command("get", "get a check")
	httpCheckgetID = httpCheckget.Arg("id", "id of the check").Required().Int64()

	httpCheckgetByUser   = httpCheck.Command("get-by-user", "get checks by user id")
	httpCheckgetByUserID = httpCheckgetByUser.Arg("id", "id of the user").Required().Int64()

	httpCheckdelete   = kingpin.Command("delete", "delete a check")
	httpCheckdeleteID = httpCheckdelete.Arg("id", "id of the check").Required().Int64()
)

func printCheck(c *proto.HTTPCheck) {
	fmt.Printf("id=%v, user_id=%v, url=%v\n", c.Id, c.UserId, c.Url)
}

func printID(id *proto.Id) {
	fmt.Printf("id=%v\n", id.Id)
}

func mainWithError() error {
	parse := kingpin.Parse()

	// Connect to user service
	d, err := grpc.Dial(*server, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("unable to connect to user server, %v", err)
	}
	c := proto.NewCheckManagerServiceClient(d)

	// Switch to different modes
	switch parse {
	case "httpcheck create":
		id, err := c.CreateHTTPCheck(context.Background(), &proto.HTTPCheck{
			Url:    *httpCheckCreateURL,
			UserId: *httpCheckCreateUserID,
		})
		if err != nil {
			return fmt.Errorf("Unable to create new check: %v", err)
		}
		printID(id)
	case "httpcheck get":
		check, err := c.GetHTTPCheck(context.Background(), &proto.Id{Id: *httpCheckgetID})
		if err != nil {
			return fmt.Errorf("Unable to get check %v: %v", *httpCheckgetID, err)
		}
		printCheck(check)
	case "httpcheck get-by-user":
		checks, err := c.GetHTTPCheckByUser(context.Background(), &proto.Id{Id: *httpCheckgetByUserID})
		if err != nil {
			return fmt.Errorf("Unable to get check by user id %v: %v", *httpCheckgetByUserID, err)
		}
		for _, check := range checks.Checks {
			printCheck(check)
		}
	case "httpcheck delete":
		_, err := c.DeleteHTTPCheck(context.Background(), &proto.Id{Id: *httpCheckdeleteID})
		if err != nil {
			return fmt.Errorf("Unable to delete check %v: %v", *httpCheckdeleteID, err)
		}
		fmt.Println("Check deleted")
	}

	return nil
}

func main() {
	if err := mainWithError(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
