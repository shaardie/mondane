package checkmanager

import (
	"context"
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

	alert "github.com/shaardie/mondane/alert/proto"
	"github.com/shaardie/mondane/checkmanager/proto"
	httpcheck "github.com/shaardie/mondane/httpcheck/proto"
)

// Config read from environment
type config struct {
	Database  string `env:"MONDANE_CHECKMANAGER_DATABASE,required"`
	Listen    string `env:"MONDANE_CHECKMANAGER_LISTEN,default=:8083"`
	Alert     string `env:"MONDANE_ALERT_SERVER,required"`
	HTTPCheck string `env:"MONDANE_HTTPCHECK_SERVER,required"`
}

// grpc server with all resources
type server struct {
	config    *config
	db        repository
	m         *memoryManager
	alert     alert.AlertServiceClient
	httpcheck httpcheck.HTTPCheckServiceClient
	logger    *zap.SugaredLogger
	initOnce  sync.Once
}

func (s *server) init() {
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

		// Start manager
		s.m = newMemoryManager(30*time.Second, s.logger)
		cs, err := s.db.GetHTTPChecks(context.Background())
		if err != nil {
			s.logger.Infow("Unable to get checks from database", "error", err)
		}

		// Connect to alert service
		d, err := grpc.Dial(s.config.Alert, grpc.WithInsecure())
		if err != nil {
			s.logger.Fatalw("Unable to connect to alert service", "error", err)
		}
		s.alert = alert.NewAlertServiceClient(d)
		s.logger.Info("Connected to alert service")

		// Connect to httpcheck service
		d, err = grpc.Dial(s.config.HTTPCheck, grpc.WithInsecure())
		if err != nil {
			s.logger.Fatalw("Unable to connect to httpcheck service", "error", err)
		}
		s.httpcheck = httpcheck.NewHTTPCheckServiceClient(d)
		s.logger.Info("Connected to httpcheck service")

		s.logger.Infow("Start all stored http checks")
		for _, c := range *cs {
			s.m.start(&httpRunnerCheck{
				httpCheck: c,
				alert:     s.alert,
				db:        s.db,
				httpcheck: s.httpcheck,
			})
		}
	})
}

// init the resources of the server on first grpc call
func (s *server) initInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	s.init()

	// Calls the next handler
	return handler(ctx, req)
}

func (s *server) GetHTTPCheck(ctx context.Context, id *proto.Id) (*proto.HTTPCheck, error) {
	c, err := s.db.GetHTTPCheck(ctx, id.Id)
	if err != nil {
		s.logger.Errorw("Unable to get http check by id", "error", err, "check_id", id.Id)
		return nil, err
	}

	return unmarshalHTTPCheck(c), nil
}

func (s *server) GetHTTPCheckByUser(ctx context.Context, id *proto.Id) (*proto.HTTPChecks, error) {
	cs, err := s.db.GetHTTPChecksByUser(ctx, id.Id)
	if err != nil {
		s.logger.Errorw("Unable to get http checks by user id", "error", err, "user_id", id.Id)
		return nil, err
	}
	return unmarshalCheckCollection(cs), nil
}

func (s *server) CreateHTTPCheck(ctx context.Context, c *proto.HTTPCheck) (*proto.Id, error) {
	check := marshalHTTPCheck(c)
	id, err := s.db.CreateHTTPCheck(ctx, check)
	if err != nil {
		s.logger.Errorw("Unable to create http check", "error", err, "check", c.String())
		return nil, err
	}
	check.ID = id

	s.m.start(&httpRunnerCheck{
		httpCheck: *check,
		alert:     s.alert,
		db:        s.db,
		httpcheck: s.httpcheck,
	})

	s.logger.Infow("Created http check", "check", c.String())
	return &proto.Id{Id: id}, nil
}

func (s *server) UpdateHTTPCheck(ctx context.Context, c *proto.HTTPCheck) (*proto.Response, error) {
	err := s.db.UpdateHTTPCheck(ctx, marshalHTTPCheck(c))
	if err != nil {
		s.logger.Errorw("Unable to update http check", "error", err, "check", c.String())
		return nil, err
	}

	s.logger.Infow("Updated http check", "check", c.String())
	return &proto.Response{}, nil
}

func (s *server) DeleteHTTPCheck(ctx context.Context, id *proto.Id) (*proto.Response, error) {
	err := s.db.DeleteHTTPCheck(ctx, id.Id)
	if err != nil {
		s.logger.Errorw("Unable to delete http check", "error", err, "check_id", id.Id)
		return nil, err
	}

	s.logger.Infow("Deleted http check", "id", id.String())
	return &proto.Response{}, nil
}

func (s *server) GetHTTPCheckResultsByCheck(ctx context.Context, id *proto.Id) (*proto.HTTPResults, error) {
	return nil, nil
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

	// Start sync directly
	go s.init()

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
	proto.RegisterCheckManagerServiceServer(grpcServer, s)

	// Serve
	if err := grpcServer.Serve(l); err != nil {
		logger.Errorw("Error while serving grpc server", "error", err)
		return err
	}
	return nil
}
