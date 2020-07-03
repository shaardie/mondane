package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/shaardie/mondane/user/proto"
)

var (
	// Command line arguments
	server            = kingpin.Flag("server", "server address").Default("127.0.0.1:8082").String()
	authenticateToken = kingpin.Flag("token", "authentication token").String()
	register          = kingpin.Command("register", "Register a new user.")
	registerEmail     = register.Arg("email", "email for user").Required().String()
	registerFirstname = register.Arg("firstname", "firstname for user").Required().String()
	registerSurname   = register.Arg("surname", "surname for user").Required().String()
	registerPassword  = register.Arg("password", "password for user").Required().String()

	activate      = kingpin.Command("activate", "activate a new user.")
	activateToken = activate.Arg("token", "Activation token").Required().String()

	get      = kingpin.Command("get", "get a user.")
	getID    = get.Flag("id", "id of user").Int64()
	getEmail = get.Flag("email", "email of user").String()

	update          = kingpin.Command("update", "update a user.")
	updateID        = update.Flag("id", "id of user").Int64()
	updateEmail     = update.Flag("email", "email of user").String()
	updateFirstname = update.Flag("firstname", "firstname of user").String()
	updateSurname   = update.Flag("surname", "firstname of user").String()
	updatePassword  = update.Flag("password", "password of user").String()

	auth         = kingpin.Command("auth", "authenticate a user.")
	authEmail    = auth.Arg("email", "email of user").Required().String()
	authPassword = auth.Arg("password", "password of user").Required().String()

	validate      = kingpin.Command("validate", "validate a JWT.")
	validateToken = validate.Arg("token", "JWT Token of user").Required().String()
)

func printUser(u *proto.User) {
	fmt.Printf("id: %v\nemail: %v\nfirstname: %v\nsurname: %v\n",
		u.Id, u.Email, u.Firstname, u.Surname)
}

func printActivationToken(t *proto.ActivationToken) {
	fmt.Printf("token: %v\n", t.Token)
}

func printJWT(t *proto.Token) {
	fmt.Printf("JTW: %v\n", t.Token)
}

func printValidatedToken(v *proto.ValidatedToken) {
	printUser(v.User)
	fmt.Printf("valid: %v\n", v.Valid)
}

func mainWithError() error {
	parse := kingpin.Parse()

	// Connect to user service
	d, err := grpc.Dial(*server, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("unable to connect to user server, %v", err)
	}
	c := proto.NewUserServiceClient(d)

	// Context with JWT
	ctx := context.Background()
	if *authenticateToken != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", *authenticateToken)
	}

	// Switch to different modes
	switch parse {
	case "register":
		token, err := c.New(context.Background(), &proto.User{
			Email:     *registerEmail,
			Firstname: *registerFirstname,
			Surname:   *registerSurname,
			Password:  *registerPassword,
		})
		if err != nil {
			return fmt.Errorf("Unable to register user %v", err)
		}
		fmt.Println("Registered user.")
		printActivationToken(token)
	case "activate":
		_, err := c.Activate(ctx, &proto.ActivationToken{
			Token: *activateToken,
		})
		if err != nil {
			return fmt.Errorf("Unable to activate user %v", err)
		}
		log.Println("Activated user.")
	case "get":
		var u *proto.User
		var err error
		if *getID != 0 {
			u, err = c.Get(ctx, &proto.User{Id: *getID})
			if err != nil {
				return fmt.Errorf("unable to get user with id %v, %v", *getID, err)
			}
		} else if *getEmail != "" {
			u, err = c.GetByEmail(ctx, &proto.User{Email: *getEmail})
			if err != nil {
				return fmt.Errorf("unable to get user with email %v, %v", *getEmail, err)
			}
		} else {
			return errors.New("id or email has to be set")
		}
		printUser(u)
	case "update":
		u, err := c.Update(ctx, &proto.User{
			Id:        *updateID,
			Email:     *updateEmail,
			Firstname: *updateFirstname,
			Surname:   *updateSurname,
			Password:  *registerPassword,
		})
		if err != nil {
			return fmt.Errorf("unable to update user %v, %v", *updateEmail, err)
		}
		printUser(u)
	case "auth":
		token, err := c.Auth(ctx, &proto.User{
			Email:    *authEmail,
			Password: *authPassword})
		if err != nil {
			return fmt.Errorf("unable to authenticate user %v, %v", *authEmail, err)
		}
		printJWT(token)
	case "validate":
		t, err := c.ValidateToken(ctx, &proto.Token{Token: *validateToken})
		if err != nil {
			return fmt.Errorf("unable to validate JWT Token %v, %v", *validateToken, err)
		}
		printValidatedToken(t)
	}

	return nil
}

func main() {
	if err := mainWithError(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
