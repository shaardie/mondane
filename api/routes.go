// Contains the routing for the api server.

package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Service) routes() {
	s.router = mux.NewRouter()

	// Route session requests
	loginRouter := s.router.PathPrefix("/api/v1/login").Subrouter()
	loginRouter.Path("").Methods(http.MethodPost).HandlerFunc(
		s.logRequest(s.enforceJSON(s.createUserAuthentication())))

	// Register
	s.router.Path("/api/v1/register").Methods(http.MethodGet).
		Queries("token", "{token}").HandlerFunc(
		s.logRequest(s.activateUser()),
	)

	// Route user requests
	userRouter := s.router.Path("/api/v1/user").Subrouter()
	userRouter.Methods(http.MethodPost).HandlerFunc(
		s.logRequest(s.enforceJSON(s.createUser())),
	)
	userRouter.Methods(http.MethodGet).HandlerFunc(
		s.logRequest(s.authenticateUser(s.readUser())),
	)
	userRouter.Methods(http.MethodPatch).HandlerFunc(
		s.logRequest(s.authenticateUser(s.updateUser())),
	)
	userRouter.Methods(http.MethodDelete).HandlerFunc(
		s.logRequest(s.authenticateUser(s.deleteUser())),
	)

	for _, cs := range s.checkServices {
		csRouter := s.router.PathPrefix(
			fmt.Sprintf("/api/v1/check/%v", cs.Type()),
		).Subrouter()
		csRouter.Methods(http.MethodPost).Path("").HandlerFunc(
			s.logRequest(s.authenticateUser(s.enforceJSON(s.createCheck(cs)))),
		)
		csRouter.Methods(http.MethodGet).Path("").HandlerFunc(
			s.logRequest(s.authenticateUser(s.enforceJSON(s.readChecks(cs)))),
		)
		csRouter.Methods(http.MethodGet).Path("/{id:[0-9]+}").HandlerFunc(
			s.logRequest(s.authenticateUser(s.readCheck(cs))),
		)
		csRouter.Methods(http.MethodGet).Path("/{id:[0-9]+}/results").HandlerFunc(
			s.logRequest(s.authenticateUser(s.readCheckResults(cs))),
		)
		csRouter.Methods(http.MethodDelete).Path("/{id:[0-9]+}").HandlerFunc(
			s.logRequest(s.authenticateUser(s.deleteCheck(cs))),
		)

	}

	// Set router to srv handler
	s.srv.Handler = s.router
}
