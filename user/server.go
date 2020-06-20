package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/joeshaw/envdecode"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/shaardie/mondane/user/proto"
)

// Config read from environment
type config struct {
	TokenKey        string `env:"MONDANE_USER_TOKEN_KEY,required"`
	DatabaseDialect string `env:"MONDANE_USER_DATABASE_DIALECT,default=sqlite3"`
	Database        string `env:"MONDANE_USER_DATABASE,default=./mondane.db"`
	Listen          string `env:"MONDANE_USER_LISTEN,default=:8080"`
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
			db, err := newSQLRepository(s.config.DatabaseDialect, s.config.Database)
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

// checkAuth is a helper function checking the JWT
func (s *server) checkAuth(ctx context.Context) (*proto.ValidatedToken, error) {
	// Get meta from context
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no metadata")
	}

	// Get Auth header from meta
	authHeader, ok := meta["authorization"]
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "authorization token is not supplied")
	}
	if len(authHeader) != 1 {
		return nil, status.Error(codes.Unauthenticated, "")
	}

	// Validate Token
	vt, err := s.ValidateToken(ctx, &proto.Token{Token: authHeader[0]})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if !vt.Valid {
		return nil, status.Error(codes.Unauthenticated, "token not valid")
	}
	return vt, err
}

// Get a user from the service
func (s *server) Get(ctx context.Context, req *proto.User) (*proto.User, error) {
	res := &proto.User{}

	// Check authentication
	vc, err := s.checkAuth(ctx)
	if err != nil {
		return res, err
	}

	// Check if authentication id matches
	if vc.User.Id != req.Id {
		return res, status.Error(codes.Unauthenticated, "wrong user")
	}

	// Get user from database
	u, err := s.db.get(ctx, req.Id)
	if err == nil {
		res = unmarshalUser(u)
	}
	return res, err
}

// AcActivate a user with its activation token
func (s *server) Activate(ctx context.Context, req *proto.ActivationToken) (*proto.Response, error) {
	res := &proto.Response{}
	err := s.db.activate(ctx, req.Token)
	return res, err
}

// Get a user by mail from the service
func (s *server) GetByEmail(ctx context.Context, req *proto.User) (*proto.User, error) {
	res := &proto.User{}

	// Check Authentication
	vc, err := s.checkAuth(ctx)
	if err != nil {
		return res, err
	}

	// Check if authentication email matches
	if vc.User.Email != req.Email {
		return res, status.Error(codes.Unauthenticated, "wrong user")
	}

	// Get user from database
	u, err := s.db.getByMail(ctx, req.Email)
	if err == nil {
		res = unmarshalUser(u)
	}
	return res, err
}

// New user is created
func (s *server) New(ctx context.Context, req *proto.User) (*proto.ActivationToken, error) {
	res := &proto.ActivationToken{}

	// Create new user in database
	token, err := s.db.new(ctx, marshallUser(req))
	if err != nil {
		return res, err
	}
	res.Token = token
	return res, nil
}

// Update updates an existing user
func (s *server) Update(ctx context.Context, req *proto.User) (*proto.User, error) {
	res := &proto.User{}

	// Check Auth
	vc, err := s.checkAuth(ctx)
	if err != nil {
		return res, err
	}

	// Check if authentication id matches
	if vc.User.Id != req.Id {
		return res, status.Error(codes.Unauthenticated, "wrong user")
	}

	// Update user in database
	user, err := s.db.update(ctx, marshallUser(req))
	if err == nil {
		res = unmarshalUser(user)
	}
	return res, err
}

// Auth authenticats a user an returns a JWT
func (s *server) Auth(ctx context.Context, req *proto.User) (*proto.Token, error) {
	res := &proto.Token{}

	// Get user by mail
	user, err := s.db.getByMail(ctx, req.Email)
	if err != nil {
		return res, err
	}

	// Check if user is activated
	if !user.Activated {
		return res, fmt.Errorf("user %v not activated", user.ID)
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword(user.Password, req.Password); err != nil {
		return res, err
	}

	// Generate JWT
	token, err := s.tokenService.encode(unmarshalUser(user))
	if err == nil {
		res.Token = token
	}
	return res, err
}

// Validates the JWT and returns the decoded user
func (s *server) ValidateToken(ctx context.Context, req *proto.Token) (*proto.ValidatedToken, error) {
	res := &proto.ValidatedToken{}
	// decode JWT
	claims, err := s.tokenService.decode(req.Token)
	if err != nil {
		return res, err
	}

	// Check if id is valid
	if claims.User.Id == 0 {
		return res, errors.New("invalid user")
	}

	res.Valid = true
	res.User = claims.User
	return res, nil
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
