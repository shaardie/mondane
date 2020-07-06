package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"sync"

	empty "github.com/golang/protobuf/ptypes/empty"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/joeshaw/envdecode"
	"go.uber.org/zap"
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
	logger       *zap.SugaredLogger
}

// initInterceptor to call server inititialization before request
func (s *server) initInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	s.initOnce.Do(func() {
		// Connect to database
		if s.db == nil {
			db, err := newSQLRepository("mysql", s.config.Database)
			if err != nil {
				s.logger.Fatalw("Unable to connect to database", "error", err)
			}
			s.db = db
			s.logger.Info("Connected to database")
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

// Create a new user
func (s *server) Create(ctx context.Context, pCreateUser *proto.CreateUser) (*proto.ActivationToken, error) {

	// Check for mandatory keys
	if pCreateUser.Email == "" || pCreateUser.Password == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"mandatory keys missing",
		)
	}

	// New User
	user := &user{
		Email:     pCreateUser.Email,
		Firstname: pCreateUser.Firstname,
		Surname:   pCreateUser.Surname,
	}

	// Generate hash from password
	password, err := bcrypt.GenerateFromPassword([]byte(pCreateUser.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("unable to generate password, %w", err)
	}
	user.Password = password

	// Generate registration token
	token, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("unable to generate token, %v", err)
	}
	user.ActivationToken = token

	_, err = s.db.Create(ctx, user)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"unable to store new user, %w", err)
	}

	return &proto.ActivationToken{Token: user.ActivationToken}, nil
}

// Read a user from the service
func (s *server) Read(ctx context.Context, pID *proto.Id) (*proto.User, error) {
	// Read user from database
	u, err := s.db.Read(ctx, pID.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound,
			"unable to get user from database, %w", err)
	}
	return unmarshalUser(u), err
}

// Update updates an existing user
func (s *server) Update(ctx context.Context, pUser *proto.User) (*proto.User, error) {
	// Read user from database
	user, err := s.db.Read(ctx, pUser.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound,
			"unable to get user from database, %w", err)
	}

	// Update user
	if pUser.Email != "" {
		user.Email = pUser.Email
	}
	if pUser.Firstname != "" {
		user.Firstname = pUser.Firstname
	}
	if pUser.Surname != "" {
		user.Surname = pUser.Surname
	}
	if pUser.Password != "" {
		// Generate hash from password
		password, err := bcrypt.GenerateFromPassword([]byte(pUser.Password), bcrypt.DefaultCost)
		if err != nil {
			if err != nil {
				return nil, fmt.Errorf("unable to generate password, %w", err)
			}
		}
		user.Password = password
	}

	// Update user in database
	err = s.db.Update(ctx, user)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "database error, %w", err)
	}
	return unmarshalUser(user), nil
}

// Delete a user by id
func (s *server) Delete(ctx context.Context, pID *proto.Id) (*empty.Empty, error) {
	err := s.db.Delete(ctx, pID.Id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "database error, %w", err,
		)
	}
	return &empty.Empty{}, nil
}

// AcActivate a user with its activation token
func (s *server) Activate(ctx context.Context, req *proto.ActivationToken) (*empty.Empty, error) {
	return &empty.Empty{}, s.db.Activate(ctx, req.Token)
}

// Auth authenticats a user an returns a JWT
func (s *server) Auth(ctx context.Context, pAuthUser *proto.AuthUser) (*proto.Token, error) {
	// Get user by mail
	user, err := s.db.ReadByMail(ctx, pAuthUser.Email)
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
		user.Password, []byte(pAuthUser.Password)); err != nil {
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

// Run the user server
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
	proto.RegisterUserServiceServer(grpcServer, s)

	// Serve
	if err := grpcServer.Serve(l); err != nil {
		logger.Errorw("Error while serving grpc server", "error", err)
		return err
	}
	return nil
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
