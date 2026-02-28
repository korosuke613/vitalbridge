package handlers

import (
	"net/http"
)

// NewHealthHandler returns an HTTP handler for the health check endpoint.
func NewHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed, GET required")
			return
		}

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ok",
		})
	}
}
