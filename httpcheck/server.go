package httpcheck

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/joeshaw/envdecode"
	"google.golang.org/grpc"

	"github.com/shaardie/mondane/httpcheck/proto"
)

// Config read from environment
type config struct {
	Database string `env:"MONDANE_HTTPCHECK_DATABASE,required"`
	Listen   string `env:"MONDANE_HTTPCHECK_LISTEN,default=:8080"`
}

// grpc server with all resources
type server struct {
	config   *config
	db       repository
	m        manager
	initOnce sync.Once
}

func (s *server) init() {
	s.initOnce.Do(func() {
		// Connect to database
		if s.db == nil {
			db, err := newSQLRepository("mysql", s.config.Database)
			if err != nil {
				log.Fatalf("Unable to connect to database, %v", err)
			}
			s.db = db
			log.Printf("Connected to database %v", s.config.Database)
		}
		// start check http manager
		if s.m == nil {
			m, err := newCheckHTTPCheckManager(30*time.Second, s.db)
			if err != nil {
				log.Fatalf("Unable to start HTTP Check Manager, %v", err)
			}
			s.m = m
			log.Printf("HTTP Check Manager started")
		}
	})
}

// init the resources of the server on first grpc call
func (s *server) initInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	s.init()

	// Calls the next handler
	return handler(ctx, req)
}

// GetCheck gets a check by id
func (s *server) GetCheck(ctx context.Context, c *proto.Check) (*proto.Check, error) {
	res, err := s.db.getCheck(ctx, c.Id)
	return unmarshalCheck(res), err
}

// GetChecksByUGetChecksByUser gets a check by its user id
func (s *server) GetChecksByUser(ctx context.Context, u *proto.User) (*proto.Checks, error) {
	res := &proto.Checks{}
	r, err := s.db.getChecksByUser(ctx, u.Id)
	if err != nil {
		return res, err
	}
	return unmarshalCheckCollection(r), err
}

// CreateCheck creates a check and starts it in the manager
func (s *server) CreateCheck(ctx context.Context, c *proto.Check) (*proto.Check, error) {
	check, err := s.db.createCheck(ctx, marshalCheck(c))
	if err != nil {
		return &proto.Check{}, err
	}
	if err := s.m.start(check); err != nil {
		return &proto.Check{}, err
	}
	return unmarshalCheck(check), err
}

// GetResults get all results of a check id
func (s *server) GetResults(ctx context.Context, c *proto.Check) (*proto.Results, error) {
	rs, err := s.db.getResults(ctx, c.Id)
	if err != nil {
		return &proto.Results{}, err
	}
	return unmarshalResultCollection(rs)
}

// DeleteCheck deletes a check and stops it in the manager
func (s *server) DeleteCheck(ctx context.Context, c *proto.Check) (*proto.Response, error) {
	err := s.db.deleteCheck(ctx, c.Id)
	if err != nil {
		return &proto.Response{}, err
	}
	err = s.m.stop(marshalCheck(c))
	return &proto.Response{}, err
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
	proto.RegisterHTTPCheckServiceServer(grpcServer, s)

	// Init to start internal services
	go func() {
		time.Sleep(10 * time.Second)
		s.init()
	}()

	// Serve
	return grpcServer.Serve(l)
}
