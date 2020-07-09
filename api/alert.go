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
		// Override user id
		alert.UserId = u.Id

		newAlert, err := s.alert.Create(r.Context(), alert)
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusCreated, nil, newAlert)
	}
}

func (s *server) ReadAlert() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		u, ok := r.Context().Value(userKey{}).(*userService.User)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}
		id, err := getID(r)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError,
				err, internalError)
			return
		}
		alert, err := s.alert.Read(r.Context(),
			&alertService.Ids{Id: id, UserId: u.Id})
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusOK, nil, alert)
	}
}

func (s *server) ReadAllAlerts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		u, ok := r.Context().Value(userKey{}).(*userService.User)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}

		alerts, err := s.alert.ReadAll(r.Context(),
			&alertService.UserId{UserId: u.Id})
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusOK, nil, alerts)
	}
}

func (s *server) UpdateAlert() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		u, ok := r.Context().Value(userKey{}).(*userService.User)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}

		// Update user id
		alert := &alertService.UpdateAlert{}
		err := readJSON(r, alert)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, responseError{"Improper JSON"})
			return
		}
		alert.UserId = u.Id

		// Update alert id
		id, err := getID(r)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError,
				err, internalError)
			return
		}
		alert.Id = id

		updatedAlert, err := s.alert.Update(r.Context(), alert)
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusOK, nil, updatedAlert)
	}
}

func (s *server) DeleteAlert() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		u, ok := r.Context().Value(userKey{}).(*userService.User)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}

		id, err := getID(r)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError,
				err, internalError)
			return
		}

		_, err = s.alert.Delete(r.Context(),
			&alertService.Ids{Id: id, UserId: u.Id})
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}
