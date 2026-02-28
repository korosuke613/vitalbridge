package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// BearerAuth Bearer Token認証ミドルウェア
func BearerAuth(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// APIキーが空の場合は認証をスキップ（開発用）
		if apiKey == "" {
			next(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			log.Printf("[Auth] 認証ヘッダーがありません: %s %s", r.Method, r.URL.Path)
			unauthorized(w)
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			log.Printf("[Auth] 無効な認証スキームです: %s %s", r.Method, r.URL.Path)
			unauthorized(w)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			log.Printf("[Auth] 無効なトークンです: %s %s", r.Method, r.URL.Path)
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
