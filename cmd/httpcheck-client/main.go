package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/shaardie/mondane/httpcheck/proto"
)

var (
	// Command line arguments
	server = kingpin.Flag("server", "server address").Default("127.0.0.1:8083").String()

	create       = kingpin.Command("create", "create a check")
	createUserID = create.Arg("user-id", "id of the user").Required().Uint64()
	createURL    = create.Arg("url", "url for the check").Required().String()

	get   = kingpin.Command("get", "get a check")
	getID = get.Arg("id", "id of the check").Required().Uint64()

	getByUser   = kingpin.Command("get-by-user", "get checks by user id")
	getByUserID = getByUser.Arg("id", "id of the user").Required().Uint64()

	delete   = kingpin.Command("delete", "delete a check")
	deleteID = delete.Arg("id", "id of the check").Required().Uint64()

	results   = kingpin.Command("results", "get results of a check")
	resultsID = results.Arg("id", "id of the check").Required().Uint64()
)

func printCheck(c *proto.Check) {
	fmt.Printf("id=%v, user_id=%v, url=%v\n", c.Id, c.UserId, c.Url)
}

func printResult(r *proto.Result) {
	fmt.Printf("id=%v, check_id=%v, timestamp=%v, success=%v\n",
		r.Id, r.CheckId, r.Timestamp, r.Success)
}

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
	case "create":
		check, err := c.CreateCheck(context.Background(), &proto.Check{
			Url:    *createURL,
			UserId: *createUserID,
		})
		if err != nil {
			return fmt.Errorf("Unable to create new check: %v", err)
		}
		fmt.Println("Check created")
		printCheck(check)
	case "get":
		check, err := c.GetCheck(context.Background(), &proto.Check{Id: *getID})
		if err != nil {
			return fmt.Errorf("Unable to get check %v: %v", *getID, err)
		}
		printCheck(check)
	case "get-by-user":
		checks, err := c.GetChecksByUser(context.Background(), &proto.User{Id: *getByUserID})
		if err != nil {
			return fmt.Errorf("Unable to get check by user id %v: %v", *getByUserID, err)
		}
		for _, check := range checks.Checks {
			printCheck(check)
		}
	case "delete":
		_, err := c.DeleteCheck(context.Background(), &proto.Check{Id: *deleteID})
		if err != nil {
			return fmt.Errorf("Unable to delete check %v: %v", *deleteID, err)
		}
		fmt.Println("Check deleted")
	case "results":
		results, err := c.GetResults(context.Background(), &proto.Check{Id: *resultsID})
		if err != nil {
			return fmt.Errorf("Unable to get results from check %v: %v", *resultsID, err)
		}
		for _, r := range results.Results {
			printResult(r)
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
