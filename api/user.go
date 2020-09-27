package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	uuid "github.com/satori/go.uuid"
	"github.com/shaardie/mondane/db"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieName = "mondane-login"
)

type userKey struct{}

func (s *Service) createUser() http.HandlerFunc {

	type userJSON struct {
		Email     string `json:"email"`
		Firstname string `json:"firstname"`
		Surname   string `json:"surname"`
		Password  string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		user := &userJSON{}

		err := json.NewDecoder(r.Body).Decode(user)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, jsonError)
			return
		}

		// Verify that email and password are set
		if user.Email == "" || user.Password == "" {
			s.response(w, r, http.StatusBadRequest,
				nil, responseError{"email and password are mandatory"})
		}

		// Generate hash from password
		password, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, fmt.Errorf("unable to generate password, %w", err), internalError)
			return
		}

		// Generate registration token
		token, err := generateToken(32)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, fmt.Errorf("unable to generate actication token, %w", err), internalError)
			return
		}

		userDB := &db.User{
			Email:           user.Email,
			Firstname:       user.Firstname,
			Surname:         user.Surname,
			ActivationToken: token,
			Password:        password,
		}

		result := s.db.WithContext(r.Context()).Create(userDB)
		if result.Error != nil {
			s.response(w, r, http.StatusBadRequest, fmt.Errorf("unable to create user in database, %w", result.Error), responseError{"user already exist"})
		}

		err = s.mail.SendRegistration(r.Context(), userDB, r.Host)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *Service) activateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("token")

		result := s.db.WithContext(r.Context()).Model(&db.User{}).Where("activation_token = ?", token).Update("activated", true)
		if result.Error != nil {
			s.response(
				w, r, http.StatusBadRequest,
				fmt.Errorf("unable to activate with token %s, %w", token, result.Error),
				responseError{"activation not successful"})
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *Service) readUser() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := r.Context().Value(userKey{}).(uuid.UUID)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}
		s.logger.Error(id)

		user := &db.User{}
		if err := s.db.WithContext(r.Context()).First(user, id).Error; err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}
		s.response(w, r, http.StatusOK, nil, user)
	}
}

func (s *Service) updateUser() http.HandlerFunc {

	type updateJSON struct {
		Firstname string `json:"firstname"`
		Surname   string `json:"surname"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(userKey{}).(uuid.UUID)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("no user id in context"), internalError)
			return
		}

		// Get Updates
		updates := &updateJSON{}
		err := json.NewDecoder(r.Body).Decode(updates)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, jsonError)
			return
		}

		user := &db.User{ID: userID}
		result := s.db.Model(user).Updates(
			&db.User{
				Surname:   updates.Surname,
				Firstname: updates.Firstname,
			},
		)
		if result.Error != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}

		s.response(w, r, http.StatusOK, nil, user)
	}
}

func (s *Service) deleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(userKey{}).(uuid.UUID)
		if !ok {
			s.response(w, r, http.StatusInternalServerError,
				errors.New("No user in context"), internalError)
			return
		}
		err := s.db.WithContext(r.Context()).Delete(&db.User{}, userID).Error
		if err != nil {
			s.response(w, r, http.StatusInternalServerError, err, internalError)
			return
		}
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

func (s *Service) authenticateUser(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			s.response(w, r, http.StatusUnauthorized,
				err, unauthError)
			return
		}

		claims, err := s.tokenService.decode(cookie.Value)
		if err != nil {
			s.response(w, r, http.StatusUnauthorized,
				err, unauthError)
			return
		}

		ctx := context.WithValue(r.Context(),
			userKey{}, claims.UserID)
		h(w, r.WithContext(ctx))
	}
}

func (s *Service) createUserAuthentication() http.HandlerFunc {

	type authJSON struct {
		Email    string
		Password string
	}

	authError := responseError{"authentication failed"}

	return func(w http.ResponseWriter, r *http.Request) {
		auth := &authJSON{}

		err := json.NewDecoder(r.Body).Decode(auth)
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, jsonError)
			return
		}

		user := &db.User{Email: auth.Email}
		result := s.db.WithContext(r.Context()).First(user)
		if result.Error != nil {
			s.response(
				w, r, http.StatusBadRequest,
				result.Error, authError,
			)
			return
		}

		if !user.Activated {
			s.response(
				w, r, http.StatusBadRequest,
				nil, authError,
			)
			return
		}

		err = bcrypt.CompareHashAndPassword(user.Password, []byte(auth.Password))
		if err != nil {
			s.response(
				w, r, http.StatusBadRequest,
				err, authError,
			)
			return
		}

		token, err := s.tokenService.encode(user.ID)
		if err != nil {
			s.response(w, r, http.StatusInternalServerError,
				err, internalError)
		}

		// Set cookie
		http.SetCookie(
			w,
			&http.Cookie{
				Name:     cookieName,
				HttpOnly: true,
				Value:    token,
				Path:     "/",
			})
		s.response(w, r, http.StatusOK, nil, nil)
	}
}

// generateToken generates a url friendly token secure token
func generateToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
