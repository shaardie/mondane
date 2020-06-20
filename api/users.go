// Contains the handler functions to get, create, update or delete hosts.

package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/shaardie/mondane/database"
	pb "github.com/shaardie/mondane/mail/api"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Response if email address is already assigned
	emailAssigned = "email address already assigned"
)

// userRequest is for unmarshal json requests.
// It is a gatekeeper to limit the amount of keys,
// which can be set from incoming requests.
type userRequest struct {
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Surname   string `json:"surname"`
	Password  string `json:"password"`
}

// userResponse is for marshal json responses.
// It is a gatekeeper to limit the information from the database
// to the response.
type userResponse struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Surname   string `json:"surname"`
}

// userToResponse creates a userResponse from the user from the database.
func userToResponse(user database.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Firstname: user.Firstname,
		Surname:   user.Surname,
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

// createUser returns a handler function creating a new user.
// This user is not activated and has to register itself,
// before it can be used.
func (s *server) createUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from request
		var user userRequest
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			s.errorLog(w, r, nil, notProperJSON, http.StatusBadRequest)
			return
		}

		// Check for mandatory keys
		if user.Email == "" || user.Password == "" {
			s.errorLog(w, r, nil, "mandatory keys email and password", http.StatusBadRequest)
			return
		}

		// Generate hash from password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("unable to hash password, %v", err), "", http.StatusInternalServerError)
			return
		}

		// Check if there is already a user with the email address in the database
		if !s.db.Where("email = ?", user.Email).First(&database.User{}).RecordNotFound() {
			s.errorLog(w, r, nil, emailAssigned, http.StatusBadRequest)
			return
		}

		// Generate registration token
		token, err := generateToken(32)
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("unable to generate token, %v", err), "", http.StatusInternalServerError)
			return
		}

		// Send mail
		_, err = s.mail.SendMail(context.Background(), &pb.SendMailRequest{
			Recipient: user.Email,
			Subject:   "Login Token",
			Message:   token,
		})
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("unable to send mail, %v", err), "", http.StatusInternalServerError)
			return
		}

		// Create user in database
		s.db.Create(&database.User{
			Email:           user.Email,
			Password:        hashedPassword,
			Firstname:       user.Firstname,
			Surname:         user.Surname,
			ActivationToken: token,
			Activated:       false,
		})

		w.WriteHeader(http.StatusCreated)
	}
}

// registerUser returns a handler functions which registers and unactive user from the database
func (s *server) registerUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from url
		token := r.FormValue("token")

		// Get user from database via token
		user := database.User{}
		if s.db.Where("activation_token = ?", token).First(&user).RecordNotFound() {
			s.errorLog(w, r, fmt.Errorf("use wrong token %v", token), "", http.StatusBadRequest)
			return
		}

		// Save activated user in the database
		user.Activated = true
		s.db.Save(&user)

		// Write user into response
		s.writeJSON(userToResponse(user), w, r)
	}
}

// getUser returns a handler function getting a user from the database.
func (s *server) getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		user := r.Context().Value(userKey{}).(*database.User)

		// Write user into response
		s.writeJSON(userToResponse(*user), w, r)
	}
}

// deleteUser returns a handler function which deletes a user from the database.
func (s *server) deleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		user := r.Context().Value(userKey{}).(*database.User)

		// Delete user
		if err := s.db.Delete(user).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				s.errorLog(w, r, nil, "", http.StatusNotFound)
				return
			}
			s.errorLog(w, r, err, "", http.StatusInternalServerError)
			return
		}
	}
}

// updateUser returns a handler function updating a user.
func (s *server) updateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		user := r.Context().Value(userKey{}).(*database.User)

		// Get new values from request
		var updateUser userRequest
		err := json.NewDecoder(r.Body).Decode(&updateUser)
		if err != nil {
			s.errorLog(w, r, nil, notProperJSON, http.StatusBadRequest)
			return
		}

		// Update values
		if updateUser.Email != "" {
			if !s.db.Where("email = ?", updateUser.Email).RecordNotFound() {
				s.errorLog(w, r,
					fmt.Errorf("user update email clash for %v", updateUser.Email),
					"email address already assigned", 400)
			}
			user.Email = updateUser.Email
		}
		if updateUser.Firstname != "" {
			user.Firstname = updateUser.Firstname
		}
		if updateUser.Surname != "" {
			user.Surname = updateUser.Surname
		}
		if updateUser.Password != "" {
			user.Password, err = bcrypt.GenerateFromPassword([]byte(updateUser.Password), bcrypt.DefaultCost)
			if err != nil {
				s.errorLog(w, r, fmt.Errorf("unable to hash password, %v", err), "", http.StatusInternalServerError)
				return
			}
		}

		// Write changed user in database
		err = s.db.Save(user).Error
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("unable to update user, %v", err), "", http.StatusInternalServerError)
			return
		}

		// Write changed user to response
		w.WriteHeader(http.StatusOK)
		s.writeJSON(userToResponse(*user), w, r)
	}
}
