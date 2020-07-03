package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shaardie/mondane/mail/proto"
	userService "github.com/shaardie/mondane/user/proto"
)

const (
	cookieName       = "mondane-login"
	registrationMail = `
Hej {{.Firstname}},

please register to the Mondane Service by using the link below:

URL: {{.URL}}

Regards
`
)

var (
	unauthError = responseError{"unauthenticated"}
)

type userKey struct{}

type userRequest struct {
	ID        int64  `json:"-"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Surname   string `json:"surname"`
	Password  string `json:"password"`
}

type userResponse struct {
	ID        int64  `json:"-"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Surname   string `json:"surname"`
}

func unmarshalUserRequest(ur *userRequest) *userService.User {
	return &userService.User{
		Id:        ur.ID,
		Email:     ur.Email,
		Firstname: ur.Firstname,
		Surname:   ur.Surname,
		Password:  ur.Password,
	}
}

func marshalUserResponse(pUser *userService.User) *userResponse {
	return &userResponse{
		ID:        pUser.Id,
		Email:     pUser.Email,
		Firstname: pUser.Firstname,
		Surname:   pUser.Surname,
	}
}

func (s *server) createUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get new User
		newUser := &userRequest{}
		err := json.NewDecoder(r.Body).Decode(newUser)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, responseError{"Improper JSON"})
			return
		}
		activationToken, err := s.user.New(
			r.Context(), unmarshalUserRequest(newUser))
		if err != nil {
			e, ok := status.FromError(err)
			if !ok {
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
				return
			}
			switch e.Code() {
			case codes.InvalidArgument:
				s.response(w, r, http.StatusForbidden, err,
					responseError{"Invalid Argument"})
			default:
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
			}
			return
		}

		// Create a new template and parse the letter into it.
		t := template.Must(template.New("letter").Parse(registrationMail))
		var buf bytes.Buffer
		err = t.Execute(&buf, struct {
			Firstname string
			URL       string
		}{
			Firstname: newUser.Firstname,
			URL: fmt.Sprintf(
				"http://%v/api/v1/register?token=%v",
				r.Host, activationToken.Token),
		})
		if err != nil {
			s.response(w, r, http.StatusInternalServerError,
				err, internalError)
			return
		}

		_, err = s.mail.SendMail(r.Context(), &proto.Mail{
			Recipient: newUser.Email,
			Subject:   "Mondane registration",
			Message:   buf.String(),
		})
		if err != nil {
			s.response(w, r, http.StatusInternalServerError,
				err, internalError)
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *server) registerUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from url
		token := r.FormValue("token")

		// Get user from database via token
		_, err := s.user.Activate(r.Context(), &userService.ActivationToken{Token: token})
		if err != nil {
			s.response(w, r, http.StatusBadRequest, err,
				responseError{"Invalid Token"})
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *server) getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(userKey{}).(*userResponse)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}
		s.response(w, r, http.StatusOK, nil, u)
	}
}

func (s *server) updateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(userKey{}).(*userResponse)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}

		// Get User update
		updateUserRequest := &userRequest{}
		err := json.NewDecoder(r.Body).Decode(updateUserRequest)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, responseError{"Improper JSON"})
			return
		}

		updateUserRequest.ID = u.ID

		pUser, err := s.user.Update(
			r.Context(), unmarshalUserRequest(updateUserRequest))
		if err != nil {
			e, ok := status.FromError(err)
			if !ok {
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
				return
			}
			switch e.Code() {
			case codes.InvalidArgument:
				s.response(w, r, http.StatusForbidden, err,
					responseError{"Invalid Argument"})
			default:
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
			}
			return
		}
		s.response(w, r, http.StatusOK, nil, marshalUserResponse(pUser))
	}
}

func (s *server) deleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(userKey{}).(*userResponse)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}
		_, err := s.user.Delete(r.Context(), &userService.User{Id: u.ID})
		if err != nil {
			e, ok := status.FromError(err)
			if !ok {
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
				return
			}
			switch e.Code() {
			case codes.InvalidArgument:
				s.response(w, r, http.StatusForbidden, err,
					responseError{"Invalid Argument"})
			default:
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
			}
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *server) authentication(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			if err == http.ErrNoCookie {
				s.response(w, r, http.StatusUnauthorized,
					err, unauthError)
				return
			}
			s.response(w, r, http.StatusInternalServerError, err,
				internalError)
			return
		}

		validatedToken, err := s.user.ValidateToken(r.Context(), &userService.Token{Token: cookie.Value})
		if err != nil {
			e, ok := status.FromError(err)
			if !ok {
				s.response(w, r, http.StatusInternalServerError, err,
					internalError)
				return
			}
			switch e.Code() {
			case codes.InvalidArgument:
				s.response(w, r, http.StatusUnauthorized, err, unauthError)
			default:
				s.logger.Infow("Unknown GRPC Server error", "error", err)
				s.response(w, r, http.StatusInternalServerError, err,
					internalError)
			}
			return
		}
		if !validatedToken.Valid {
			s.response(w, r, http.StatusUnauthorized, nil, unauthError)
			return
		}

		ctx := context.WithValue(r.Context(),
			userKey{}, marshalUserResponse(validatedToken.User))
		h(w, r.WithContext(ctx))
	}
}

func (s *server) createLogin() http.HandlerFunc {
	type loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
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

		token, err := s.user.Auth(r.Context(), &userService.User{
			Email:    lr.Email,
			Password: lr.Password,
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
