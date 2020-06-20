package mail

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/joeshaw/envdecode"
	"google.golang.org/grpc"
	"gopkg.in/gomail.v2"

	pb "github.com/shaardie/mondane/mail/proto"
)

type config struct {
	Username string `env:"MONDANE_MAIL_USERNAME,required"`
	Password string `env:"MONDANE_MAIL_PASSWORD,required"`
	Server   string `env:"MONDANE_MAIL_SERVER,required"`
	Port     int    `env:"MONDANE_MAIL_HOST,default=25"`
	From     string `env:"MONDANE_MAIL_FROM,required"`
	Listen   string `env:"MONDANE_API_LISTEN,default=:8080"`
}

type server struct {
	dialer *gomail.Dialer
	config *config
}

func (s *server) SendMail(ctx context.Context, mail *pb.Mail) (*pb.Response, error) {
	r := &pb.Response{}
	if mail.Recipient == "" {
		return r, errors.New("recipient empty")
	}

	// New Message
	msg := gomail.NewMessage()
	msg.SetHeader("From", s.config.From)
	msg.SetHeader("Subject", mail.Subject)
	msg.SetHeader("To", mail.Recipient)
	msg.SetBody("text/plain", mail.Message)

	// Dial and Send
	err := s.dialer.DialAndSend(msg)
	if err != nil {
		log.Printf("Failure sending mail: %v", err)
		return r, fmt.Errorf("unable to send mail, %v", err)
	}

	log.Printf("Sent mail to %v", mail.Recipient)
	return r, nil
}

// Run the mail server
func Run() error {
	// Get Config
	var c config
	if err := envdecode.StrictDecode(&c); err != nil {
		return fmt.Errorf("unable to read config, %v", err)
	}

	// TCP Listener
	l, err := net.Listen("tcp", c.Listen)
	if err != nil {
		return err
	}

	// GRPC Server
	s := grpc.NewServer()
	pb.RegisterMailServiceServer(s, &server{
		dialer: gomail.NewDialer(c.Server, c.Port, c.Username, c.Password),
		config: &c,
	})

	// Serve
	return s.Serve(l)
}
