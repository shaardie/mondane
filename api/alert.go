package api

import (
	"errors"
	"net/http"

	alertService "github.com/shaardie/mondane/alert/proto"
	userService "github.com/shaardie/mondane/user/proto"
)

func (s *server) CreateAlert() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		u, ok := r.Context().Value(userKey{}).(*userService.User)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}

		alert := &alertService.CreateAlert{}
		err := readJSON(r, alert)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, responseError{"Improper JSON"})
			return
		}

		if u.Id != alert.UserId {
			s.response(w, r, http.StatusForbidden, nil, forbiddenError)
			return
		}

		NewAlert, err := s.alert.Create(r.Context(), alert)
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusCreated, nil, NewAlert)

	}
}
