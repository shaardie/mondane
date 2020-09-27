// Contains server definition and miscellaneous middleware and helper functions

package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joeshaw/envdecode"
	"github.com/shaardie/mondane/collector"
	"github.com/shaardie/mondane/mail"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	notProperJSON = "not proper json" // Response to user for broken json
)

type responseError struct {
	Error string `json:"error"`
}

// config read from environment
type config struct {
	Listen   string `env:"MONDANE_API_LISTEN,default=:8080"`
	TokenKey string `env:"MONDANE_API_TOKEN_KEY,required"`
	DBString string `env:"MONDANE_DATABASE,requireed"`
}

// Service from which all handler and handler functions are hanging and
// where global resources are saved.
// It is the core structure for the API.
type Service struct {
	srv           *http.Server
	router        *mux.Router
	config        *config
	logger        *zap.SugaredLogger
	tokenService  tokenService
	db            *gorm.DB
	checkServices []collector.Collector
	mail          *mail.Service
}

func New(logger *zap.SugaredLogger, db *gorm.DB, checkServices []collector.Collector, mail *mail.Service) (*Service, error) {
	// Get Config
	var c config
	if err := envdecode.StrictDecode(&c); err != nil {
		logger.Infow("Unable to read config",
			"error", err)
		return nil, fmt.Errorf("unable to read config, %w", err)
	}
	// Create server
	s := &Service{
		config:        &c,
		srv:           &http.Server{Addr: c.Listen},
		logger:        logger,
		tokenService:  tokenService{key: []byte(c.TokenKey)},
		db:            db,
		checkServices: checkServices,
		mail:          mail,
	}
	s.routes()
	return s, nil
}

func (s *Service) Run() error {
	return s.srv.ListenAndServe()
}
