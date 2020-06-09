// Contains server definition and miscellaneous middleware and helper functions

package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	"github.com/joeshaw/envdecode"

	"github.com/shaardie/mondane/database"
)

const (
	notProperJSON = "not proper json" // Response to user for broken json
)

// Config read from environment
type config struct {
	CookieKey       string `env:"MONDANE_API_COOKIE_KEY,required"`
	DatabaseDialect string `env:"MONDANE_API_DATABASE_DIALECT,default=sqlite3"`
	Database        string `env:"MONDANE_API_DATABASE,default=./mondane.db"`
	Listen          string `env:"MONDANE_API_LISTEN,default=:8080"`
}

// Server from which all handler and handler functions are hanging and
// where global resources are saved.
// It is the core structure for the API.
type server struct {
	srv    *http.Server
	router *mux.Router
	config *config
	db     *gorm.DB
	ss     *sessions.CookieStore
	init   sync.Once
}

// initHandler initialize resources lazy on first request
func (s *server) initHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do once
		s.init.Do(func() {
			// Connect to database
			db, err := database.ConnectDB(s.config.DatabaseDialect, s.config.Database)
			if err != nil {
				log.Printf("Unable to connect to database, %v\n", err)
				s.srv.Close()
			}
			s.db = db
			log.Println("Connected to Database")

			// Connect to session cookie store
			s.ss = sessions.NewCookieStore([]byte(s.config.CookieKey))
			log.Println("Initialized Session Cookie Store")
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
			s.errorLog(w, r, nil, "Wrong content type. Only 'application/json' allowed", http.StatusBadRequest)
			return
		}
		h(w, r)
	}
}

// logAccess is a middleware which logs all requests
func (s *server) logAccess(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := time.Now().Format(time.RFC3339)
		h(w, r)
		log.Printf("%v: %v %v %v %v %v %v",
			t, r.RemoteAddr, r.Host, r.UserAgent(), r.Method,
			r.URL.RequestURI(), r.Proto)
	}
}

// errorLog is a helper function to handle all errors from hanler the same way.
// It logs the error and creates an http error.
func (s *server) errorLog(w http.ResponseWriter, r *http.Request, err error, reply string, statuscode int) {
	log.Printf("%v: %v %v %v %v %v %v %v '%v', replied: %v",
		time.Now().Format(time.RFC3339), r.RemoteAddr, r.Host, r.UserAgent(),
		r.Method, r.URL.RequestURI(), r.Proto, statuscode, err, reply)
	http.Error(w, reply, statuscode)
}

// writeJSON is a helper function to write arbitrary data as json to the response.
func (s *server) writeJSON(data interface{}, w http.ResponseWriter, r *http.Request) {
	// Marshal data
	js, err := json.Marshal(data)
	if err != nil {
		s.errorLog(w, r, fmt.Errorf("unable to marshal, %v", err), "", http.StatusInternalServerError)
		return
	}

	// Set Header
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Write JSON to response
	_, err = w.Write(js)
	if err != nil {
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("unable to write to body, %v", err), "", http.StatusInternalServerError)
			return
		}
	}
}

// Run runs the server
func Run() error {
	// Get Config
	var c config
	if err := envdecode.StrictDecode(&c); err != nil {
		return fmt.Errorf("unable to read config, %v", err)
	}

	// Create server
	s := server{
		config: &c,
		srv:    &http.Server{Addr: c.Listen},
	}

	// Setup routes
	s.routes()

	// Run Server
	return s.srv.ListenAndServe()
}
