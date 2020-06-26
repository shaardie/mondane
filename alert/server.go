package alert

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/joeshaw/envdecode"
	"google.golang.org/grpc"

	"github.com/shaardie/mondane/alert/proto"
	mail "github.com/shaardie/mondane/mail/proto"
	user "github.com/shaardie/mondane/user/proto"
)

// Config read from environment
type config struct {
	Database string `env:"MONDANE_ALERT_DATABASE,required"`
	Mail     string `env:"MONDANE_ALERT_MAIL_SERVER,required"`
	User     string `env:"MONDANE_ALERT_USER_SERVER,required"`
	Listen   string `env:"MONDANE_ALERT_LISTEN,default=:8084"`
}

// grpc server with all resources
type server struct {
	config   *config
	db       repository
	initOnce sync.Once
	mail     mail.MailServiceClient
	user     user.UserServiceClient
}

// init the server resources, just once
func (s *server) init() {
	s.initOnce.Do(func() {
		// Connect to database
		if s.db == nil {
			db, err := newSQLRepository("mysql", s.config.Database)
			if err != nil {
				log.Fatalf("Unable to connect to database, %v", err)
			}
			s.db = db
			log.Println("Connected to database")
		}

		d, err := grpc.Dial(s.config.Mail, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("unable to connect to mail server, %v", err)
		}
		s.mail = mail.NewMailServiceClient(d)

		d, err = grpc.Dial(s.config.User, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("unable to connect to user server, %v", err)
		}
		s.user = user.NewUserServiceClient(d)
	})
}

// initInterceptor to call server inititialization before request
func (s *server) initInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	s.init()

	// Calls the next handler
	return handler(ctx, req)
}

// GetAlert gets an alert by id
func (s *server) GetAlert(ctx context.Context, id *proto.Id) (*proto.Alert, error) {
	alert, err := s.db.Get(ctx, id.Id)
	if err != nil {
		return nil, err
	}
	return unmarshalAlert(alert)
}

// GetAlertsByUser gets an alert by user id
func (s *server) GetAlertsByUser(ctx context.Context, id *proto.Id) (*proto.Alerts, error) {
	alerts, err := s.db.GetByUser(ctx, id.Id)
	if err != nil {
		return nil, err
	}
	return unmarshalAlerts(alerts)
}

// CreateAlert creates new alert
func (s *server) CreateAlert(ctx context.Context, pAlert *proto.Alert) (*proto.Response, error) {
	alert, err := marshalAlert(pAlert)
	if err != nil {
		return nil, err
	}
	err = s.db.Create(ctx, alert)
	return &proto.Response{}, nil
}

// DeleteAlert delete an alert by id
func (s *server) DeleteAlert(ctx context.Context, id *proto.Id) (*proto.Response, error) {
	return &proto.Response{}, s.db.Delete(ctx, id.Id)
}

// Firing triggers the firing of all alerts of a check
func (s *server) Firing(ctx context.Context, check *proto.Check) (*proto.Response, error) {
	// Get all alerts matching the check
	alerts, err := s.db.GetByCheck(ctx, check.Id, check.Type)
	if err != nil {
		return nil, err
	}

	// Fire for all found alerts
	for _, alert := range *alerts {
		// Check if mail should be send
		if !alert.SendMail {
			log.Printf("Do not fire alert %v, since email sending is disabled", alert.ID)
			continue
		}

		// Check if alert was already been fired during send period
		if alert.LastSend.Add(alert.SendPeriod).After(time.Now()) {
			log.Printf("Do not fire alert %v, since send period is not over yet", alert.ID)
			continue
		}

		// Alert should be fired
		log.Printf("Attempt to fire alert %v", alert.ID)

		// Get user
		u, err := s.user.Get(ctx, &user.User{Id: alert.UserID})
		if err != nil {
			return nil, err
		}

		// Send mail to user emails
		_, err = s.mail.SendMail(ctx, &mail.Mail{
			Recipient: u.Email,
			Subject:   "[Mondane] Problem found",
			Message:   fmt.Sprintf("Check of type %v with id %v failed", check.Type, check.Id),
		})
		if err != nil {
			return nil, err
		}

		// Update last send
		err = s.db.UpdateLastSend(ctx, alert.ID)
		if err != nil {
			return nil, err
		}
	}
	return &proto.Response{}, nil
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

	// Create server
	s := &server{config: &c}

	// GRPC Server with init interceptor
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(s.initInterceptor))
	proto.RegisterAlertServiceServer(grpcServer, s)

	// Init to start internal services
	go func() {
		time.Sleep(10 * time.Second)
		s.init()
	}()

	// Serve
	return grpcServer.Serve(l)
}
