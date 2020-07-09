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
		s.logRequest(s.enforceJSON(s.CreateLogin())))

	// Register
	s.router.Path("/api/v1/register").Methods(http.MethodGet).
		Queries("token", "{token}").HandlerFunc(
		s.logRequest(s.ActivateUser()),
	)

	// Route user requests
	userRouter := s.router.Path("/api/v1/user").Subrouter()
	userRouter.Methods(http.MethodPost).HandlerFunc(
		s.logRequest(s.enforceJSON(s.CreateUser())),
	)
	userRouter.Methods(http.MethodGet).HandlerFunc(
		s.logRequest(s.AuthenticateUser(s.ReadUser())),
	)
	userRouter.Methods(http.MethodPatch).HandlerFunc(
		s.logRequest(s.AuthenticateUser(s.UpdateUser())),
	)
	userRouter.Methods(http.MethodDelete).HandlerFunc(
		s.logRequest(s.AuthenticateUser(s.DeleteUser())),
	)

	// Route alert requests
	alertRouter := s.router.PathPrefix("/api/v1/alert").Subrouter()
	alertRouter.Methods(http.MethodPost).HandlerFunc(
		s.logRequest(s.enforceJSON(s.AuthenticateUser(s.CreateAlert()))),
	)
	alertRouter.Path("").Methods(http.MethodGet).HandlerFunc(
		s.logRequest(s.AuthenticateUser(s.ReadAllAlerts())),
	)
	alertRouter.Path("/{id:[1-9][0-9]*}").Methods(http.MethodGet).HandlerFunc(
		s.logRequest(s.AuthenticateUser(s.ReadAlert())),
	)
	alertRouter.Path("/{id:[1-9][0-9]*}").Methods(http.MethodPut).HandlerFunc(
		s.logRequest(s.enforceJSON(s.AuthenticateUser(s.UpdateAlert()))),
	)
	alertRouter.Path("/{id:[1-9][0-9]*}").Methods(http.MethodDelete).HandlerFunc(
		s.logRequest(s.AuthenticateUser(s.DeleteAlert())),
	)

	// Use initHandler in all requests
	s.router.Use(s.initHandler)

	// Set router to srv handler
	s.srv.Handler = s.router
}
