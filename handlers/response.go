package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// sendJSON JSON応答を送信
func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[Handler] JSON応答エラー: %v", err)
	}
}

// sendError エラー応答を送信
func sendError(w http.ResponseWriter, statusCode int, message string) {
	sendJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"message": message,
	})
}
