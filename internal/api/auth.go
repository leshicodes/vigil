package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

const cookieName = "vigil_session"

// apiKeyAuth returns a middleware that checks for a valid API key.
// It checks (in order): Authorization header → session cookie.
// If apiKey is empty, auth is disabled entirely.
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
			tokenFromCookie := false

			// 1. Check Authorization: Bearer header.
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimPrefix(auth, "Bearer ")
			}

			// 2. Fallback: check session cookie.
			if token == "" {
				if c, err := r.Cookie(cookieName); err == nil {
					token = c.Value
					tokenFromCookie = true
				}
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
				// If the bad token came from a cookie, clear it so the
				// browser stops sending it on every request (especially
				// <img> tags which can't handle 401s gracefully).
				if tokenFromCookie {
					http.SetCookie(w, &http.Cookie{
						Name:     cookieName,
						Value:    "",
						Path:     "/",
						MaxAge:   -1,
						HttpOnly: true,
						SameSite: http.SameSiteStrictMode,
					})
				}
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

// handleLogin validates the provided API key and sets a session cookie.
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

	// Set HttpOnly session cookie - browser sends it automatically on every
	// request, including <img> tags. No key in URLs ever.
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    body.Key,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":           true,
		"auth_enabled": true,
	})
}

// handleLogout clears the session cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	json.NewEncoder(w).Encode(map[string]string{
		"status": "logged out",
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

	// Check header first, then cookie.
	token := ""
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}
	if token == "" {
		if c, err := r.Cookie(cookieName); err == nil {
			token = c.Value
		}
	}

	valid := subtle.ConstantTimeCompare([]byte(token), []byte(s.APIKey)) == 1
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": valid,
		"auth_enabled":  true,
	})
}
