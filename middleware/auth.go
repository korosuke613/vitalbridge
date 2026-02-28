package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// BearerAuth is a middleware that validates Bearer token authentication.
func BearerAuth(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth when API key is empty (development mode)
		if apiKey == "" {
			next(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			slog.Warn("missing authorization header", "method", r.Method, "path", r.URL.Path)
			unauthorized(w)
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			slog.Warn("invalid authorization scheme", "method", r.Method, "path", r.URL.Path)
			unauthorized(w)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			slog.Warn("invalid token", "method", r.Method, "path", r.URL.Path)
			unauthorized(w)
			return
		}

		next(w, r)
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": "Unauthorized",
	})
}
