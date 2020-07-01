package alert

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"

	"github.com/joeshaw/envdecode"
	"go.uber.org/zap"
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
	logger   *zap.SugaredLogger
}

// init the server resources, just once
func (s *server) init() {
	s.initOnce.Do(func() {
		s.logger.Info("Initialize resources")
		// Connect to database
		if s.db == nil {
			db, err := newSQLRepository("mysql", s.config.Database)
			if err != nil {
				s.logger.Fatalw("Unable to connect to database", "error", err)
			}
			s.db = db
			s.logger.Info("Connected to database.")
		}

		d, err := grpc.Dial(s.config.Mail, grpc.WithInsecure())
		if err != nil {
			s.logger.Fatalw("Unable to connect to mail server", "error", err)
		}
		s.mail = mail.NewMailServiceClient(d)
		s.logger.Info("Connected to mail service")

		d, err = grpc.Dial(s.config.User, grpc.WithInsecure())
		if err != nil {
			s.logger.Fatalf("Unable to connect to user server", "error", err)
		}
		s.user = user.NewUserServiceClient(d)
		s.logger.Info("Connected to user service")
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
		s.logger.Warnw("Unable to get alert",
			"check_id", check.Id,
			"check_type", check.Type)
		return nil, err
	}

	if len(*alerts) == 0 {
		s.logger.Infow("No alerts found",
			"check_id", check.Id,
			"check_type", check.Type)
	}

	// Fire for all found alerts
	for _, alert := range *alerts {
		// Check if mail should be send
		if !alert.SendMail {
			s.logger.Infow("Do not fire alert, since email sending is disabled",
				"alert", alert)
			continue
		}

		// Check if alert was already been fired during send period
		if alert.LastSend.Add(alert.SendPeriod).After(time.Now()) {
			s.logger.Infow("Do not fire alert, since send period is not over yet",
				"alert", alert)
			continue
		}

		// Alert should be fired
		s.logger.Infow("Attempt to fire alert", "alert", alert)

		// Get user
		u, err := s.user.Get(ctx, &user.User{Id: alert.UserID})
		if err != nil {
			s.logger.Infow("Unable to get user from user service", "error", err)
			return nil, err
		}

		// Send mail to user emails
		_, err = s.mail.SendMail(ctx, &mail.Mail{
			Recipient: u.Email,
			Subject:   "[Mondane] Problem found",
			Message:   fmt.Sprintf("Check of type %v with id %v failed", check.Type, check.Id),
		})
		if err != nil {
			s.logger.Infow("Unable to send email with email service", "error", err)
			return nil, err
		}

		// Update last send
		err = s.db.UpdateLastSend(ctx, alert.ID)
		if err != nil {
			s.logger.Infow("Unable to update alert", "error", err, "alert", alert)
			return nil, err
		}
	}
	return &proto.Response{}, nil
}

// Run the server
func Run() error {
	baseLogger, err := zap.NewProduction()
	if err != nil {
		log.Printf("Unable to initialize logger, %v", err)
		return err
	}
	logger := baseLogger.Sugar()
	logger.Info("Initialized logger")

	// Get Config
	var c config
	if err := envdecode.StrictDecode(&c); err != nil {
		logger.Errorw("Unable to read config", "error", err)
		return err
	}

	// TCP Listener
	l, err := net.Listen("tcp", c.Listen)
	if err != nil {
		logger.Errorw("Unable to open tcp connection for grpc server", "error", err)
		return err
	}

	// Create server
	s := &server{
		config: &c,
		logger: logger,
	}

	// Make sure that log statements internal to gRPC library are logged using the zapLogger as well.
	grpc_zap.ReplaceGrpcLoggerV2(baseLogger)
	// Create a server, make sure we put the grpc_ctxtags context before everything else.
	grpcServer := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(baseLogger),
			s.initInterceptor,
		))
	// GRPC Server with init interceptor
	proto.RegisterAlertServiceServer(grpcServer, s)

	// Serve
	if err := grpcServer.Serve(l); err != nil {
		logger.Errorw("Error while serving grpc server", "error", err)
		return err
	}
	return nil
}
