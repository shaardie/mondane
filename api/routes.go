// Contains the routing for the api server.

package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *server) routes() {
	s.router = mux.NewRouter()

	// Route session requests
	loginRouter := s.router.PathPrefix("/api/v1/login").Subrouter()
	loginRouter.Path("").Methods(http.MethodPost).HandlerFunc(
		s.logRequest(s.enforceJSON(s.createLogin())))

	// Route user requests
	userRouter := s.router.PathPrefix("/api/v1/user").Subrouter()
	userRouter.Path("/").Methods(http.MethodPost).HandlerFunc(
		s.logRequest(s.enforceJSON(s.createUser())),
	)

	userRouter.Path("/register").Methods(http.MethodGet).
		Queries("token", "{token}").HandlerFunc(
		s.logRequest(s.registerUser()),
	)

	// Use initHandler in all requests
	s.router.Use(s.initHandler)

	// Set router to srv handler
	s.srv.Handler = s.router
}
