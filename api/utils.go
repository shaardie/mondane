package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	jsonError      = responseError{"Improper JSON"}
	forbiddenError = responseError{"forbidden"}
	unauthError    = responseError{"unauthenticated"}
	internalError  = responseError{"unexpected server error"}
	invalidError   = responseError{"Invalid arguments"}
	notFoundError  = responseError{"Not found"}
)

// enforceJSON is a middleware which ensure the content type `application/json`
func (s *Service) enforceJSON(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			s.response(w, r, http.StatusBadRequest, nil,
				responseError{"Only content type application/json supported"})
			return
		}
		h(w, r)
	}
}

func (s *Service) logRequest(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Infow("Request",
			"Remote Address", r.RemoteAddr,
			"host", r.Host,
			"user agent", r.UserAgent(),
			"method", r.Method,
			"uri", r.URL.RequestURI(),
			"proto", r.Proto,
		)
		h(w, r)
	}
}

func (s *Service) response(w http.ResponseWriter, r *http.Request, statuscode int, requestErr error, response interface{}) {
	s.logger.Infow("Response",
		"remote address", r.RemoteAddr,
		"host", r.Host,
		"user agent", r.UserAgent(),
		"method", r.Method,
		"uri", r.URL.RequestURI(),
		"proto", r.Proto,
		"status code", statuscode,
		"request error", requestErr,
		"response", response,
	)

	// Writer status code
	w.WriteHeader(statuscode)

	if response != nil {
		if err := s.writeJSON(w, r, response); err != nil {
			s.logger.Errorw("Failure while writing response",
				"error", err,
			)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	}
	return
}

// writeJSON is a helper function to write arbitrary data as json to the response.
func (s *Service) writeJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {

	// Marshal data
	js, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("unable to marshal %v, %w", data, err)
	}

	// Set Header
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Write JSON to response
	_, err = w.Write(js)
	if err != nil {
		if err != nil {
			return fmt.Errorf("unable to write to body, %v", err)
		}
	}
	return nil
}
