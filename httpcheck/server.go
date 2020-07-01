package httpcheck

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/joeshaw/envdecode"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/shaardie/mondane/httpcheck/proto"
)

// Config read from environment
type config struct {
	Listen string `env:"MONDANE_HTTPCHECK_LISTEN,default=:8085"`
}

// grpc server with all resources
type server struct {
	config   *config
	client   *http.Client
	initOnce sync.Once
	logger   *zap.SugaredLogger
}

// init the resources of the server on first grpc call
func (s *server) initInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	s.initOnce.Do(func() {
		s.client = &http.Client{
			Timeout: 10 * time.Second,
		}
	})

	// Calls the next handler
	return handler(ctx, req)
}

func (s *server) Do(ctx context.Context, c *proto.Check) (*proto.Result, error) {
	t := time.Now()
	resp, err := s.client.Get(c.Url)
	if err != nil {
		s.logger.Infow("HTTP Check failed", "error", err, "check", c)
		return &proto.Result{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer resp.Body.Close()
	return &proto.Result{
		Duration:   int64(time.Now().Sub(t)),
		StatusCode: int64(resp.StatusCode),
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
	}, nil
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
			s.initInterceptor))
	// GRPC Server with init interceptor
	proto.RegisterHTTPCheckServiceServer(grpcServer, s)

	// Serve
	if err := grpcServer.Serve(l); err != nil {
		logger.Errorw("Error while serving grpc server", "error", err)
		return err
	}
	return nil
}
