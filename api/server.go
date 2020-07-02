// Contains server definition and miscellaneous middleware and helper functions

package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/joeshaw/envdecode"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	alert "github.com/shaardie/mondane/alert/proto"
	checkmanager "github.com/shaardie/mondane/checkmanager/proto"
	mail "github.com/shaardie/mondane/mail/proto"
	user "github.com/shaardie/mondane/user/proto"
)

const (
	notProperJSON = "not proper json" // Response to user for broken json
)

var (
	internalError = responseError{Error: "unexpected server error"}
)

type responseError struct {
	Error string `json:"error"`
}

// Config read from environment
type config struct {
	Listen       string `env:"MONDANE_API_LISTEN,default=:8080"`
	User         string `env:"MONDANE_API_USER_SERVER,required"`
	Mail         string `env:"MONDANE_API_MAIL_SERVER,required"`
	Alert        string `env:"MONDANE_API_ALERT_SERVER,required"`
	CheckManager string `env:"MONDANE_API_CHECKMANAGER_SERVER,required"`
}

// Server from which all handler and handler functions are hanging and
// where global resources are saved.
// It is the core structure for the API.
type server struct {
	srv          *http.Server
	router       *mux.Router
	config       *config
	logger       *zap.SugaredLogger
	initOnce     sync.Once
	user         user.UserServiceClient
	mail         mail.MailServiceClient
	checkmanager checkmanager.CheckManagerServiceClient
	alert        alert.AlertServiceClient
}

// initHandler initialize resources lazy on first request
func (s *server) initHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do once
		s.initOnce.Do(func() {
			// Connect to mail grpc
			if s.mail == nil {
				d, err := grpc.Dial(s.config.Mail, grpc.WithInsecure())
				if err != nil {
					s.logger.Errorf("unable to connect ot mail service",
						"error", err)
					s.srv.Close()
					return
				}
				s.mail = mail.NewMailServiceClient(d)
				s.logger.Info("Connected to mail server")
			}

			// Connect to user grpc
			if s.user == nil {
				d, err := grpc.Dial(s.config.User, grpc.WithInsecure())
				if err != nil {
					s.logger.Errorf("unable to connect ot user service",
						"error", err)
					s.srv.Close()
					return
				}
				s.user = user.NewUserServiceClient(d)
				s.logger.Info("Connected to user server")
			}

			// Connect to alert grpc
			if s.alert == nil {
				d, err := grpc.Dial(s.config.Alert, grpc.WithInsecure())
				if err != nil {
					s.logger.Errorf("unable to connect ot alert service",
						"error", err)
					s.srv.Close()
					return
				}
				s.alert = alert.NewAlertServiceClient(d)
				s.logger.Info("Connected to alert server")
			}

			// Connect to checkmanager grpc
			if s.mail == nil {
				d, err := grpc.Dial(s.config.CheckManager, grpc.WithInsecure())
				if err != nil {
					s.logger.Errorf("unable to connect ot checkmanager service",
						"error", err)
					s.srv.Close()
					return
				}
				s.checkmanager = checkmanager.NewCheckManagerServiceClient(d)
				s.logger.Info("Connected to checkmanager server")
			}
		})

		// Call next handler function
		h.ServeHTTP(w, r)
	})
}

// enforceJSON is a middleware which ensure the content type `application/json`
func (s *server) enforceJSON(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			s.response(w, r, http.StatusBadRequest, nil,
				responseError{"Only content type application/json supported"})
			return
		}
		h(w, r)
	}
}

func (s *server) logRequest(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Infow("Request",
			"Remote Address", r.RemoteAddr,
			"host", r.Host,
			"user agent", r.UserAgent(),
			"method", r.Method,
			"uri", r.URL.RequestURI(),
			"proto", r.Proto,
		)
		h(w, r)
	}
}

func (s *server) response(w http.ResponseWriter, r *http.Request, statuscode int, requestErr error, response interface{}) {
	s.logger.Infow("Response",
		"remote address", r.RemoteAddr,
		"host", r.Host,
		"user agent", r.UserAgent(),
		"method", r.Method,
		"uri", r.URL.RequestURI(),
		"proto", r.Proto,
		"status code", statuscode,
		"request error", requestErr,
	)
	if response != nil {
		if err := s.writeJSON(w, r, response); err != nil {
			s.logger.Errorw("Failure while writing response",
				"error", err,
			)
		}
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// Writer status code
	w.WriteHeader(statuscode)
	return
}

// writeJSON is a helper function to write arbitrary data as json to the response.
func (s *server) writeJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	// Marshal data
	js, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("unable to marshal %v, %w", data, err)
	}

	// Set Header
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Write JSON to response
	_, err = w.Write(js)
	if err != nil {
		if err != nil {
			return fmt.Errorf("unable to write to body, %v", err)
		}
	}
	return nil
}

// Run runs the server
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
		return fmt.Errorf("unable to read config, %v", err)
	}

	// Create server
	s := server{
		config: &c,
		srv:    &http.Server{Addr: c.Listen},
		logger: logger,
	}

	// Setup routes
	s.routes()

	// Run Server
	return s.srv.ListenAndServe()
}
