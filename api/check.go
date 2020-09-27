package api

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/shaardie/mondane/collector"
)

func (s *Service) createCheck(cs collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Context().Value(userKey{}).(uuid.UUID)

		response, err := cs.Create(r.Context(), id, r.Body)
		if err != nil {
			s.response(w, r, http.StatusBadRequest, err, jsonError)
			return
		}

		// Create response
		s.response(w, r, http.StatusCreated, nil, response)

	}
}

func (s *Service) readChecks(cs collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Context().Value(userKey{}).(uuid.UUID)

		response, err := cs.ReadByUser(r.Context(), id)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}

		// Create response
		s.response(w, r, http.StatusOK, nil, response)
	}
}

func (s *Service) readCheck(cs collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Context().Value(userKey{}).(uuid.UUID)
		checkID, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}

		response, err := cs.Read(r.Context(), id, uint(checkID))
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}

		s.response(w, r, http.StatusOK, nil, response)
	}
}

func (s *Service) readCheckResults(cs collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Context().Value(userKey{}).(uuid.UUID)
		checkID, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}

		response, err := cs.ReadResults(r.Context(), id, uint(checkID))
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}

		s.response(w, r, http.StatusOK, nil, response)
	}
}

func (s *Service) updateCheck(cs collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Context().Value(userKey{}).(uuid.UUID)

		checkID, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}
		response, err := cs.Update(r.Context(), id, uint(checkID), r.Body)
		if err != nil {
			s.response(w, r, http.StatusBadRequest, err, jsonError)
			return
		}

		// Create response
		s.response(w, r, http.StatusOK, nil, response)
	}
}

func (s *Service) deleteCheck(cs collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Context().Value(userKey{}).(uuid.UUID)
		checkID, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}

		err = cs.Delete(r.Context(), id, uint(checkID))
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}

		s.response(w, r, http.StatusOK, nil, nil)
	}
}
