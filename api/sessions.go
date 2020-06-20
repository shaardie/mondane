// Contains the session handling.

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/shaardie/mondane/database"
	"golang.org/x/crypto/bcrypt"
)

// Name of the session cookie
const sessionCookie = "mondane-login"

// Key in http contetxt representing a user
type userKey struct{}

// withSession is a middleware which checks the session cookie and
// writes the matching user to the http context.
func (s *server) withSession(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session
		session, err := s.ss.Get(r, sessionCookie)
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("unable to get session, %v", err), "", http.StatusBadRequest)
			session.Options.MaxAge = -1
			session.Save(r, w)
			return
		}

		// Get user id from session
		id, ok := session.Values["user_id"]
		if !ok {
			session.Options.MaxAge = -1
			session.Save(r, w)
			s.errorLog(w, r, errors.New("login attempt without proper session cookie"), "", http.StatusForbidden)
			return
		}

		// Get user from database
		user := database.User{}
		if s.db.First(&user, id).RecordNotFound() {
			session.Options.MaxAge = -1
			session.Save(r, w)
			s.errorLog(w, r, fmt.Errorf("user not in database, %v", err), "user not found", http.StatusForbidden)
			return
		}

		// Write user to context
		r = r.WithContext(context.WithValue(r.Context(), userKey{}, &user))

		h(w, r)
	}
}

// deleteSession returns a handler function which delete the current session
func (s *server) deleteSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session
		session, err := s.ss.Get(r, sessionCookie)
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("unable to get session, %v", err), "", http.StatusInternalServerError)
			return
		}

		// Set session max age to expired
		session.Options.MaxAge = -1
		session.Save(r, w)
		w.WriteHeader(http.StatusOK)
	}
}

// createSession returns a handler function which creates checks the user authentication
// and creates a new session.
func (s *server) createSession() http.HandlerFunc {
	type auth struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Get authentication
		var a auth
		err := json.NewDecoder(r.Body).Decode(&a)
		if err != nil {
			s.errorLog(w, r, nil, notProperJSON, http.StatusBadRequest)
			return
		}

		// Check for mandatory keys
		if a.Email == "" || a.Password == "" {
			s.errorLog(w, r, nil, "mandatory keys email and password", http.StatusBadRequest)
			return
		}

		// Get user from database
		user := database.User{}
		if s.db.Where("email = ?", a.Email).First(&user).RecordNotFound() {
			s.errorLog(w, r, fmt.Errorf("failed login with unregistered user %v", a.Email), "", http.StatusForbidden)
			return
		}

		// Compare password with stored hashed password
		err = bcrypt.CompareHashAndPassword(user.Password, []byte(a.Password))
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("failed login with wrong password for user %v", user.Email), "", http.StatusForbidden)
			return
		}

		// Check if user is activated
		if !user.Activated {
			s.errorLog(w, r, fmt.Errorf("failed login with unactivated user %v", user.Email), "user not activated", http.StatusBadRequest)
			return
		}

		// Create new session
		session := sessions.NewSession(s.ss, "mondane-login")
		session.Values["user_id"] = user.ID
		session.Options.MaxAge = 10 * 365 * 24 * 60 * 60
		session.Options.Path = "/"
		session.Save(r, w)

		// Write response
		w.WriteHeader(http.StatusOK)
	}
}
