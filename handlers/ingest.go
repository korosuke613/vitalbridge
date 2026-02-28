package handlers

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"

	"github.com/korosuke613/vitalbridge/converter"
	"github.com/korosuke613/vitalbridge/store"
)

const maxBodySize = 10 * 1024 * 1024 // 10MB

// NewIngestHandler Ingestエンドポイントのハンドラを返す
func NewIngestHandler(ms *store.MetricsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "POSTメソッドが必要です")
			return
		}

		ct := r.Header.Get("Content-Type")
		if ct != "" {
			mediaType, _, err := mime.ParseMediaType(ct)
			if err != nil || mediaType != "application/json" {
				sendError(w, http.StatusUnsupportedMediaType, "Content-Type: application/json が必要です")
				return
			}
		}

		defer r.Body.Close()
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
		if err != nil {
			sendError(w, http.StatusBadRequest, fmt.Sprintf("リクエストボディの読み取りに失敗しました: %v", err))
			return
		}
		if int64(len(body)) > maxBodySize {
			sendError(w, http.StatusRequestEntityTooLarge, "リクエストボディが10MBを超えています")
			return
		}

		samples, err := converter.Convert(body)
		if err != nil {
			sendError(w, http.StatusBadRequest, fmt.Sprintf("データ変換に失敗しました: %v", err))
			return
		}

		ms.Update(samples)
		log.Printf("[Ingest] %d サンプルを受信・保存しました", len(samples))

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Received %d samples", len(samples)),
		})
	}
}
