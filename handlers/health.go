package handlers

import (
	"net/http"
)

// NewHealthHandler ヘルスチェックハンドラを返す
func NewHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendError(w, http.StatusMethodNotAllowed, "GETメソッドが必要です")
			return
		}

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ok",
		})
	}
}
