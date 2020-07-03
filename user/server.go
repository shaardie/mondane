package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/joeshaw/envdecode"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shaardie/mondane/user/proto"
)

// Config read from environment
type config struct {
	TokenKey string `env:"MONDANE_USER_TOKEN_KEY,required"`
	Database string `env:"MONDANE_USER_DATABASE,required"`
	Listen   string `env:"MONDANE_USER_LISTEN,default=:8082"`
}

// grpc server with all resources
type server struct {
	config       *config
	db           repository
	tokenService authable
	initOnce     sync.Once
}

// init the resources of the server on first grpc call
func (s *server) init(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
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
		// Create token service
		if s.tokenService == nil {
			s.tokenService = &tokenService{
				db:  s.db,
				key: []byte(s.config.TokenKey),
			}
		}
	})

	// Calls the next handler
	return handler(ctx, req)
}

// Get a user from the service
func (s *server) Get(ctx context.Context, req *proto.User) (*proto.User, error) {
	res := &proto.User{}

	// Get user from database
	u, err := s.db.get(ctx, req.Id)
	if err == nil {
		res = unmarshalUser(u)
	}
	return res, err
}

// AcActivate a user with its activation token
func (s *server) Activate(ctx context.Context, req *proto.ActivationToken) (*proto.Response, error) {
	err := s.db.activate(ctx, req.Token)
	return &proto.Response{}, err
}

// Get a user by mail from the service
func (s *server) GetByEmail(ctx context.Context, req *proto.User) (*proto.User, error) {
	res := &proto.User{}

	// Get user from database
	u, err := s.db.getByMail(ctx, req.Email)
	if err == nil {
		res = unmarshalUser(u)
	}
	return res, err
}

// New user is created
func (s *server) New(ctx context.Context, req *proto.User) (*proto.ActivationToken, error) {
	// Create new user in database
	token, err := s.db.new(ctx, marshalUser(req))
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "database error, %w", err)
	}
	return &proto.ActivationToken{Token: token}, nil
}

// Update updates an existing user
func (s *server) Update(ctx context.Context, req *proto.User) (*proto.User, error) {
	// Update user in database
	user, err := s.db.update(ctx, marshalUser(req))
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "database error, %w", err)
	}
	return unmarshalUser(user), nil
}

// Delete a user by id
func (s *server) Delete(ctx context.Context, req *proto.User) (*proto.Response, error) {
	err := s.db.DeleteUser(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "database error, %w", err,
		)
	}
	return &proto.Response{}, nil
}

// Auth authenticats a user an returns a JWT
func (s *server) Auth(ctx context.Context, req *proto.User) (*proto.Token, error) {
	// Get user by mail
	user, err := s.db.getByMail(ctx, req.Email)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound, "database error %w", err)
	}

	// Check if user is activated
	if !user.Activated {
		return nil, status.Errorf(
			codes.PermissionDenied,
			"user %v not activated", user.ID,
		)
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword(
		user.Password, []byte(req.Password)); err != nil {
		return nil, status.Errorf(
			codes.PermissionDenied,
			"password wrong, %w", err)
	}

	// Generate JWT
	token, err := s.tokenService.encode(unmarshalUser(user))
	if err != nil {
		return nil, status.Errorf(
			codes.Unknown, "unable to generate token, %w", err,
		)
	}
	return &proto.Token{Token: token}, err
}

// Validates the JWT and returns the decoded user
func (s *server) ValidateToken(ctx context.Context, req *proto.Token) (*proto.ValidatedToken, error) {
	// decode JWT
	claims, err := s.tokenService.decode(req.Token)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"unable to get token", err)
	}

	// Check if id is valid
	if claims.User.Id == 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"invalid user")
	}

	return &proto.ValidatedToken{
		User:  claims.User,
		Valid: true,
	}, nil
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
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(s.init))
	proto.RegisterUserServiceServer(grpcServer, s)

	// Serve
	return grpcServer.Serve(l)
}

// generateToken generates a url friendly token secure token
func generateToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
