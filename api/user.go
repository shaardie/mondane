package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/shaardie/mondane/mail/proto"
	user "github.com/shaardie/mondane/user/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const registrationMail = `
Hej {{.Firstname}},

please register to the Mondane Service by using the link below:

URL: {{.URL}}

Regards
`

func (s *server) createUser() http.HandlerFunc {
	type newUser struct {
		Email     string `json:"email"`
		Firstname string `json:"firstname"`
		Surname   string `json:"surname"`
		Password  string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Get new User
		nu := &newUser{}
		err := json.NewDecoder(r.Body).Decode(nu)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, responseError{"Improper JSON"})
			return
		}
		// Check mandatory keys
		if nu.Email == "" {
			s.response(
				w, r, http.StatusBadRequest,
				nil, responseError{"Email required"})
			return
		}
		if nu.Password == "" {
			s.response(
				w, r, http.StatusBadRequest,
				nil, responseError{"Password required"})
			return
		}

		activationToken, err := s.user.New(r.Context(), &user.User{
			Email:     nu.Email,
			Password:  []byte(nu.Password),
			Surname:   nu.Surname,
			Firstname: nu.Firstname,
		})
		if err != nil {
			e, ok := status.FromError(err)
			if !ok {
				s.response(w, r, http.StatusInternalServerError,
					err, internalError)
				return
			}
			switch e.Code() {
			case codes.AlreadyExists:
				s.response(w, r, http.StatusForbidden, err,
					responseError{"choose another email"})
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
			Firstname: nu.Firstname,
			URL: fmt.Sprintf(
				"http://%v/api/v1/register/?token=%v",
				r.Host, activationToken.Token),
		})
		if err != nil {
			s.response(w, r, http.StatusInternalServerError,
				err, internalError)
			return
		}

		_, err = s.mail.SendMail(r.Context(), &proto.Mail{
			Recipient: nu.Email,
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
		_, err := s.user.Activate(r.Context(), &user.ActivationToken{Token: token})
		if err != nil {
			s.response(w, r, http.StatusBadRequest, err,
				responseError{"Invalid Token"})
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}
