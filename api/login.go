package api

import (
	"encoding/json"
	"net/http"

	user "github.com/shaardie/mondane/user/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const cookieName = "mondane-login"

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
}

func (s *server) createLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get authentication
		var lr loginRequest
		err := json.NewDecoder(r.Body).Decode(&lr)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, responseError{"Improper JSON"})
			return
		}

		if lr.Email == "" {
			s.response(
				w, r, http.StatusBadRequest,
				nil, responseError{"Email missing"})
			return
		}

		if lr.Password == "" {
			s.response(
				w, r, http.StatusBadRequest,
				nil, responseError{"Password missing"})
			return
		}

		token, err := s.user.Auth(r.Context(), &user.User{
			Email:    lr.Email,
			Password: []byte(lr.Password),
		})
		if err != nil {
			e, ok := status.FromError(err)
			if !ok {
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
				return
			}
			switch e.Code() {
			case codes.PermissionDenied:
				s.response(w, r, http.StatusForbidden, err,
					responseError{"wrong authentication"})
			case codes.NotFound:
				s.response(w, r, http.StatusForbidden, err,
					responseError{"wrong authentication"})
			default:
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
			}
			return
		}

		http.SetCookie(
			w,
			&http.Cookie{
				Name:     cookieName,
				HttpOnly: true,
				Value:    token.Token,
			})
		s.response(w, r, http.StatusOK, nil, nil)
	}
}
