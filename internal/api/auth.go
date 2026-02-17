package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// apiKeyAuth returns a middleware that checks for a valid API key in the
// Authorization header. If apiKey is empty, auth is disabled entirely.
func apiKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no key is configured, skip auth.
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Allow preflight CORS requests through.
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			token := ""
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimPrefix(auth, "Bearer ")
			} else if qKey := r.URL.Query().Get("key"); qKey != "" {
				// Fallback: accept ?key= for resources loaded by <img>/<video> tags
				// that can't send Authorization headers.
				token = qKey
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// handleLogin validates the provided API key without requiring the auth
// middleware. This lets the frontend check credentials before storing them.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// If auth is disabled, always succeed.
	if s.APIKey == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":           true,
			"auth_enabled": false,
		})
		return
	}

	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if subtle.ConstantTimeCompare([]byte(body.Key), []byte(s.APIKey)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid key",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":           true,
		"auth_enabled": true,
	})
}

// handleAuthCheck lets the frontend know if auth is enabled and if the
// current token is valid, without requiring a POST body.
func (s *Server) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	if s.APIKey == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"auth_enabled":  false,
		})
		return
	}

	token := ""
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}

	valid := subtle.ConstantTimeCompare([]byte(token), []byte(s.APIKey)) == 1
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": valid,
		"auth_enabled":  true,
	})
}
