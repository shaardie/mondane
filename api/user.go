package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"

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

type userKey struct{}

func (s *server) CreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get new User
		newUser := &userService.CreateUser{}
		err := readJSON(r, newUser)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, jsonError)
			return
		}

		activationToken, err := s.user.Create(r.Context(), newUser)
		if err != nil {
			s.handleGRPCError(w, r, err)
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
			Subject:   "Mondane Registration",
			Message:   buf.String(),
		})
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *server) ActivateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from url
		token := r.FormValue("token")

		// Get user from database via token
		_, err := s.user.Activate(r.Context(), &userService.ActivationToken{Token: token})
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *server) ReadUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(userKey{}).(*userService.User)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}
		s.response(w, r, http.StatusOK, nil, u)
	}
}

func (s *server) UpdateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(userKey{}).(*userService.User)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}

		// Get Updates
		updates := &userService.User{}
		err := readJSON(r, updates)
		if err != nil {
			s.response(w, r, http.StatusBadRequest, err, jsonError)
			return
		}

		// Set id of the user
		updates.Id = u.Id

		pUser, err := s.user.Update(r.Context(), updates)
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusOK, nil, pUser)
	}
}

func (s *server) DeleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := r.Context().Value(userKey{}).(*userService.User)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}
		_, err := s.user.Delete(r.Context(), &userService.Id{Id: u.Id})
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *server) AuthenticateUser(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			s.response(w, r, http.StatusUnauthorized,
				err, unauthError)
			return
		}

		validatedToken, err := s.user.ValidateToken(
			r.Context(), &userService.Token{Token: cookie.Value})
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}

		if !validatedToken.Valid {
			s.response(w, r, http.StatusUnauthorized, nil, unauthError)
			return
		}

		ctx := context.WithValue(r.Context(),
			userKey{}, validatedToken.User)
		h(w, r.WithContext(ctx))
	}
}

func (s *server) CreateLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get authentication
		authUser := &userService.AuthUser{}
		err := readJSON(r, authUser)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, responseError{"Improper JSON"})
			return
		}

		// Get token
		token, err := s.user.Auth(r.Context(), authUser)
		if err != nil {
			s.handleGRPCError(w, r, err)
			return
		}

		// Set cookie
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
