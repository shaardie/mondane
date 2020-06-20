package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/shaardie/mondane/database"
	pb "github.com/shaardie/mondane/mail/api"
)

type mailMock struct{}

func (mailMock) SendMail(ctx context.Context, in *pb.SendMailRequest, opts ...grpc.CallOption) (*pb.SendMailResponse, error) {
	return &pb.SendMailResponse{}, nil
}

func testServer() *server {
	s := server{
		config: &config{DatabaseDialect: "sqlite3"},
		mail:   mailMock{},
		srv:    &http.Server{},
	}
	s.routes()
	return &s
}

// Test_server_apiUser tests the whole lifecycle of a user.
func Test_server_apiUser(t *testing.T) {
	// Test user
	testUser := userRequest{
		Email:     "test@example.com",
		Firstname: "Max",
		Password:  "super-secret",
		Surname:   "Mustermann",
	}
	testUserJSON, _ := json.Marshal(&testUser)

	s := server{
		config: &config{DatabaseDialect: "sqlite3"},
		mail:   mailMock{},
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

func Test_server_createUser(t *testing.T) {
	tests := []struct {
		name string
		user userRequest
		want int
	}{
		{
			name: "complete user",
			user: userRequest{
				Email:     "max.mustermann@example.com",
				Firstname: "Max",
				Surname:   "Mustermann",
				Password:  "secret",
			},
			want: http.StatusCreated,
		},
		{
			name: "minimal user",
			user: userRequest{
				Email:    "max.mustermann2@example.com",
				Password: "secret",
			},
			want: http.StatusCreated,
		},
		{
			name: "missing password",
			user: userRequest{
				Email: "max.mustermann3@example.com",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "duplicate email",
			user: userRequest{
				Email:    "max.mustermann2@example.com",
				Password: "secret",
			},
			want: http.StatusBadRequest,
		},
	}

	s := testServer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userJSON, err := json.Marshal(&tt.user)
			if err != nil {
				t.Errorf("Unable to marshal user %v", tt.user)
			}
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/user/",
				bytes.NewReader(userJSON))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Result().StatusCode)
		})
	}
}

func Test_server_registerUser(t *testing.T) {
	tests := []struct {
		name  string
		user  *database.User
		token string
		want  int
	}{
		{
			name: "register user",
			user: &database.User{
				Email:           "max.mustermann@example.com",
				Password:        []byte{},
				ActivationToken: "testtoken",
			},
			token: "testtoken",
			want:  http.StatusOK,
		},
		{
			name:  "unknown token",
			token: "failtoken",
			want:  http.StatusBadRequest,
		},
	}
	s := testServer()
	if err := s.initResourses(); err != nil {
		t.Errorf("Unable to init server %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.user != nil {
				s.db.Create(tt.user)
			}
			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/api/user/register/?token=%v", tt.token),
				nil)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Result().StatusCode)
		})
	}
}
