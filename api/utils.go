package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	jsonError      = responseError{"Improper JSON"}
	forbiddenError = responseError{"forbidden"}
	unauthError    = responseError{"unauthenticated"}
	internalError  = responseError{"unexpected server error"}
	invalidError   = responseError{"Invalid arguments"}
	notFoundError  = responseError{"Not found"}
)

func readJSON(r *http.Request, v proto.Message) error {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body, %w", err)
	}
	err = protojson.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("unable to unmarshal request body, %w", err)
	}
	return nil
}

func (s *server) handleGRPCError(w http.ResponseWriter, r *http.Request, err error) {
	e, ok := status.FromError(err)
	if !ok {
		s.response(w, r, http.StatusInternalServerError,
			err, internalError)
		return
	}
	switch e.Code() {
	case codes.InvalidArgument:
		s.response(w, r, http.StatusBadGateway, err, invalidError)
	case codes.NotFound:
		s.response(w, r, http.StatusNotFound, err, notFoundError)
	case codes.Unauthenticated:
		s.response(w, r, http.StatusUnauthorized, err, unauthError)
	case codes.PermissionDenied:
		s.response(w, r, http.StatusForbidden, err, forbiddenError)
	case codes.Unavailable:
		s.response(w, r, http.StatusServiceUnavailable, err, nil)
	default:
		s.response(w, r, http.StatusInternalServerError,
			err, internalError)
	}
}

// enforceJSON is a middleware which ensure the content type `application/json`
func (s *server) enforceJSON(h http.HandlerFunc) http.HandlerFunc {
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

func (s *server) logRequest(h http.HandlerFunc) http.HandlerFunc {
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

func (s *server) response(w http.ResponseWriter, r *http.Request, statuscode int, requestErr error, response interface{}) {
	s.logger.Infow("Response",
		"remote address", r.RemoteAddr,
		"host", r.Host,
		"user agent", r.UserAgent(),
		"method", r.Method,
		"uri", r.URL.RequestURI(),
		"proto", r.Proto,
		"status code", statuscode,
		"request error", requestErr,
	)
	if response != nil {
		if err := s.writeJSON(w, r, response); err != nil {
			s.logger.Errorw("Failure while writing response",
				"error", err,
			)
		}
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// Writer status code
	w.WriteHeader(statuscode)
	return
}

// writeJSON is a helper function to write arbitrary data as json to the response.
func (s *server) writeJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {

	// Marshal data
	js := []byte{}
	var err error
	switch v := data.(type) {
	case proto.Message:
		js, err = protojson.Marshal(v)
	default:
		js, err = json.Marshal(v)
	}
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
