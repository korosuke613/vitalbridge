package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/korosuke613/vitalbridge/converter"
	"github.com/korosuke613/vitalbridge/store"
)

const maxBodySize = 10 * 1024 * 1024 // 10MB

// NewIngestHandler returns an HTTP handler for the ingest endpoint.
func NewIngestHandler(ms *store.MetricsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed, POST required")
			return
		}

		ct := r.Header.Get("Content-Type")
		if ct != "" {
			mediaType, _, err := mime.ParseMediaType(ct)
			if err != nil || mediaType != "application/json" {
				sendError(w, http.StatusUnsupportedMediaType, "unsupported media type, application/json required")
				return
			}
		}

		defer r.Body.Close()
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
		if err != nil {
			sendError(w, http.StatusBadRequest, fmt.Sprintf("failed to read request body: %v", err))
			return
		}
		if int64(len(body)) > maxBodySize {
			sendError(w, http.StatusRequestEntityTooLarge, "request body exceeds 10MB limit")
			return
		}

		samples, err := converter.Convert(body)
		if err != nil {
			sendError(w, http.StatusBadRequest, fmt.Sprintf("failed to convert data: %v", err))
			return
		}

		ms.Update(samples)
		slog.Info("samples received and stored", "count", len(samples))

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Received %d samples", len(samples)),
		})
	}
}
