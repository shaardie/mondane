// Contains the routing for the api server.

package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *server) routes() {
	s.router = mux.NewRouter()

	// Route session requests
	s.router.Path("/api/session/").Methods(http.MethodPost).HandlerFunc(
		s.logAccess(s.enforceJSON(s.createSession())),
	)

	// Route user requests
	userRouter := s.router.PathPrefix("/api/user/").Subrouter()
	userRouter.Path("/").Methods(http.MethodPost).HandlerFunc(
		s.logAccess(s.enforceJSON(s.createUser())),
	)
	userRouter.Path("/").Methods(http.MethodGet).HandlerFunc(
		s.logAccess(s.withSession(s.getUser())),
	)
	userRouter.Path("/").Methods(http.MethodPatch).HandlerFunc(
		s.logAccess(s.withSession(s.updateUser())),
	)
	userRouter.Path("/").Methods(http.MethodDelete).HandlerFunc(
		s.logAccess(s.withSession(s.deleteUser())),
	)
	userRouter.Path("/register/").Methods(http.MethodGet).Queries("token", "{token}").HandlerFunc(
		s.logAccess(s.registerUser()),
	)

	// Route hosts requests
	hostsRouter := s.router.PathPrefix("/api/hosts/").Subrouter()
	hostsRouter.Path("/").Methods(http.MethodPost).HandlerFunc(
		s.logAccess(s.withSession(s.createHost())),
	)
	hostsRouter.Path("/").Methods(http.MethodGet).HandlerFunc(
		s.logAccess(s.withSession(s.getHosts())),
	)
	hostsRouter.Path("/{id:[0-9]+}/").Methods(http.MethodGet).HandlerFunc(
		s.logAccess(s.withSession(s.getHost())),
	)
	hostsRouter.Path("/{id:[0-9]+}/").Methods(http.MethodPatch).HandlerFunc(
		s.logAccess(s.withSession(s.updateHost())),
	)

	// Use initHandler in all requests
	s.router.Use(s.initHandler)

	// Set router to srv handler
	s.srv.Handler = s.router
}
