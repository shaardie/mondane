// Contains the handler functions to get, create, update or delete hosts.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/shaardie/mondane/database"
)

// hostRequest is for unmarshal json requests.
// It is a gatekeeper to limit the amount of keys,
// which can be set from incoming requests.
type hostRequest struct {
	Ipv4     string `json:"ipv4"`
	Ipv6     string `json:"ipv6"`
	Hostname string `json:"hostname"`
}

// hostResponse is for marshal json responses.
// It is a gatekeeper to limit the information from the database
// to the response.
type hostResponse struct {
	ID       uint   `json:"id"`
	Ipv4     string `json:"ipv4"`
	Ipv6     string `json:"ipv6"`
	Hostname string `json:"hostname"`
}

// hostToResponse creates a hostResponse from the host from the database.
func hostToResponse(host database.Host) hostResponse {
	return hostResponse{
		ID:       host.ID,
		Ipv4:     host.Ipv4,
		Ipv6:     host.Ipv6,
		Hostname: host.Hostname,
	}
}

// hostsToResponse creates a list of hostResponses from a list
// of hosts from the database.
func hostsToResponse(hosts []database.Host) []hostResponse {
	response := make([]hostResponse, len(hosts))
	for index, item := range hosts {
		response[index] = hostToResponse(item)
	}
	return response
}

// createHost returns a handler function creating a new host.
func (s *server) createHost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get User
		user := r.Context().Value(userKey{}).(*database.User)

		// Get host from request
		var newHost hostRequest
		err := json.NewDecoder(r.Body).Decode(&newHost)
		if err != nil {
			s.errorLog(w, r, nil, notProperJSON, http.StatusBadRequest)
			return
		}

		// Add host to database
		host := database.Host{
			UserID:   user.ID,
			Ipv4:     newHost.Ipv4,
			Ipv6:     newHost.Ipv6,
			Hostname: newHost.Hostname,
		}
		err = s.db.Create(&host).Error
		if err != nil {
			s.errorLog(w, r, fmt.Errorf("unable to create host, %v", err), "", http.StatusInternalServerError)
			return
		}

		// Write host into response
		w.WriteHeader(http.StatusCreated)
		s.writeJSON(hostToResponse(host), w, r)
	}
}

// getHosts returns a handler function getting all hosts from a user.
func (s *server) getHosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		user := r.Context().Value(userKey{}).(*database.User)

		// Get related hosts from database
		var hosts []database.Host
		s.db.Model(&user).Related(&hosts)

		// Write hosts into response
		w.WriteHeader(http.StatusOK)
		s.writeJSON(hostsToResponse(hosts), w, r)
	}
}

// getHosts returns a handler function getting a host with `id` from a user.
func (s *server) getHost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		user := r.Context().Value(userKey{}).(*database.User)

		// Get host id
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.errorLog(w, r, nil, "malformed id", http.StatusBadRequest)
		}

		// Get host with host id and user id from database
		var host database.Host
		if s.db.Where("user_id = ? AND id = ?", user.ID, id).First(&host).RecordNotFound() {
			http.NotFound(w, r)
			return
		}

		// Write host in response
		w.WriteHeader(http.StatusOK)
		s.writeJSON(hostToResponse(host), w, r)
	}
}

// updateHost returns a handler function updating the host with `id` from a user.
func (s *server) updateHost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user
		user := r.Context().Value(userKey{}).(*database.User)

		// Get host id
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.errorLog(w, r, nil, "malformed id", http.StatusBadRequest)
		}

		// Get host with host id and user id from database
		var host database.Host
		if s.db.Where("user_id = ? AND id = ?", user.ID, id).First(&host).RecordNotFound() {
			http.NotFound(w, r)
			return
		}

		// Get host changes from request
		var newHost hostRequest
		err = json.NewDecoder(r.Body).Decode(&newHost)
		if err != nil {
			s.errorLog(w, r, nil, notProperJSON, http.StatusBadRequest)
			return
		}

		// Update host values
		if newHost.Ipv4 != "" {
			host.Ipv4 = newHost.Ipv4
		}
		if newHost.Ipv6 != "" {
			host.Ipv6 = newHost.Ipv6
		}
		if newHost.Hostname != "" {
			host.Hostname = newHost.Hostname
		}

		// Write changed host in database
		s.db.Save(&host)

		// Write changed host to response
		w.WriteHeader(http.StatusOK)
		s.writeJSON(hostToResponse(host), w, r)
	}
}
