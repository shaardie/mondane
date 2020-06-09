package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shaardie/mondane/database"
	"github.com/stretchr/testify/assert"
)

// Test_server_apiUser tests the whole lifecycle of a user.
func Test_server_apiUser(t *testing.T) {
	// Test user
	testUser := userRequest{
		Email:     "test@example.com",
		Firstname: "Max",
		Password:  "secret",
		Surname:   "Mustermann",
	}
	testUserJSON, _ := json.Marshal(&testUser)

	s := server{
		config: &config{DatabaseDialect: "sqlite3"},
		srv:    &http.Server{},
	}
	t.Log(s.config)
	s.routes()

	// Create user
	req := httptest.NewRequest(http.MethodPost, "/api/user/", bytes.NewReader(testUserJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Result().StatusCode)

	// Get user from database
	user := database.User{}
	s.db.First(&user)
	assert.False(t, user.Activated)

	// Register user
	req = httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/user/register/?token=%v", user.ActivationToken),
		nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Get user from database
	s.db.First(&user)
	assert.True(t, user.Activated)

	// Get session token
	req = httptest.NewRequest(http.MethodPost, "/api/session/", bytes.NewReader(testUserJSON))
	w = httptest.NewRecorder()
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	cookies := w.Result().Cookies()

	// Update user
	req = httptest.NewRequest(
		http.MethodPatch,
		"/api/user/",
		strings.NewReader(`{"firstname": "Moritz"}`))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Get user
	req = httptest.NewRequest(http.MethodGet, "/api/user/", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	resp := userResponse{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, userResponse{
		ID:        1,
		Email:     "test@example.com",
		Firstname: "Moritz",
		Surname:   "Mustermann",
	}, resp)

	// Delete user
	req = httptest.NewRequest(http.MethodDelete, "/api/user/", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	s.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
