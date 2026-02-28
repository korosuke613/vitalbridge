package handlers

import (
	"net/http"
	"time"

	"github.com/korosuke613/vitalbridge/store"
)

// NewStatusHandler returns an HTTP handler for the status endpoint.
func NewStatusHandler(ms *store.MetricsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendError(w, http.StatusMethodNotAllowed, "method not allowed, GET required")
			return
		}

		lastReceived, totalSamples, activeMetrics := ms.GetStats()

		lastReceivedStr := ""
		if !lastReceived.IsZero() {
			lastReceivedStr = lastReceived.Format(time.RFC3339)
		}

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Service status",
			"data": map[string]interface{}{
				"last_received":  lastReceivedStr,
				"total_samples":  totalSamples,
				"active_metrics": activeMetrics,
				"timestamp":      time.Now().Format(time.RFC3339),
			},
		})
	}
}
