package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// sendJSON sends a JSON response with the given status code.
func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// sendError sends an error JSON response.
func sendError(w http.ResponseWriter, statusCode int, message string) {
	sendJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"message": message,
	})
}
