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

	// Register
	s.router.Path("/api/v1/register").Methods(http.MethodGet).
		Queries("token", "{token}").HandlerFunc(
		s.logRequest(s.registerUser()),
	)

	// Route user requests
	userRouter := s.router.PathPrefix("/api/v1/user").Subrouter()
	userRouter.Path("/").Methods(http.MethodPost).HandlerFunc(
		s.logRequest(s.enforceJSON(s.createUser())),
	)
	userRouter.Path("/").Methods(http.MethodGet).HandlerFunc(
		s.logRequest(s.authentication(s.getUser())),
	)
	userRouter.Path("/").Methods(http.MethodPatch).HandlerFunc(
		s.logRequest(s.authentication(s.updateUser())),
	)
	userRouter.Path("/").Methods(http.MethodDelete).HandlerFunc(
		s.logRequest(s.authentication(s.deleteUser())),
	)

	// Use initHandler in all requests
	s.router.Use(s.initHandler)

	// Set router to srv handler
	s.srv.Handler = s.router
}
