package converter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/korosuke613/vitalbridge/store"
)

// Health Auto Export JSON structures (supports multiple formats)

// wrappedPayload represents the {"data": {"metrics": [...]}} format.
type wrappedPayload struct {
	Data struct {
		Metrics []rawMetric `json:"metrics"`
	} `json:"data"`
}

// metricsPayload represents the {"metrics": [...]} format.
type metricsPayload struct {
	Metrics []rawMetric `json:"metrics"`
}

// rawMetric represents an individual metric entry.
type rawMetric struct {
	Name  string          `json:"name"`
	Units string          `json:"units"`
	Data  []rawDataPoint  `json:"data"`
}

// rawDataPoint represents a data point with flexible fields.
type rawDataPoint struct {
	Date  string   `json:"date"`
	Qty   *float64 `json:"qty"`
	Avg   *float64 `json:"Avg"`
	Min   *float64 `json:"Min"`
	Max   *float64 `json:"Max"`
	Value *float64 `json:"value"`
	// sleep_analysis fields
	Stage     string   `json:"stage"`
	InBed     *float64 `json:"inBed"`
	Asleep    *float64 `json:"asleep"`
}

// Convert converts Health Auto Export JSON into metric samples.
func Convert(payload []byte) ([]store.MetricSample, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty payload")
	}

	metrics, err := parseMetrics(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics found")
	}

	var samples []store.MetricSample

	for _, m := range metrics {
		mapping, ok := AllowedMetrics[m.Name]
		if !ok {
			slog.Debug("skipping unknown metric", "name", m.Name)
			continue
		}

		for _, dp := range m.Data {
			ts := parseTimestamp(dp.Date)

			// base value (qty or value)
			var baseValue *float64
			if dp.Qty != nil {
				baseValue = dp.Qty
			} else if dp.Value != nil {
				baseValue = dp.Value
			}

			if baseValue != nil {
				labels := map[string]string{}
				if dp.Stage != "" {
					labels["stage"] = dp.Stage
				}

				samples = append(samples, store.MetricSample{
					Name:      mapping.PrometheusName,
					Labels:    labels,
					Value:     *baseValue,
					Timestamp: ts,
					Type:      mapping.Type,
				})
			}

			// generate avg/min/max samples when HasStats is true
			if mapping.HasStats {
				if dp.Avg != nil {
					samples = append(samples, store.MetricSample{
						Name:      mapping.PrometheusName,
						Labels:    map[string]string{"stat": "avg"},
						Value:     *dp.Avg,
						Timestamp: ts,
						Type:      mapping.Type,
					})
				}
				if dp.Min != nil {
					samples = append(samples, store.MetricSample{
						Name:      mapping.PrometheusName,
						Labels:    map[string]string{"stat": "min"},
						Value:     *dp.Min,
						Timestamp: ts,
						Type:      mapping.Type,
					})
				}
				if dp.Max != nil {
					samples = append(samples, store.MetricSample{
						Name:      mapping.PrometheusName,
						Labels:    map[string]string{"stat": "max"},
						Value:     *dp.Max,
						Timestamp: ts,
						Type:      mapping.Type,
					})
				}
			}

			// special handling for sleep_analysis
			if m.Name == "sleep_analysis" {
				if dp.InBed != nil {
					samples = append(samples, store.MetricSample{
						Name:      mapping.PrometheusName,
						Labels:    map[string]string{"stage": "in_bed"},
						Value:     *dp.InBed,
						Timestamp: ts,
						Type:      mapping.Type,
					})
				}
				if dp.Asleep != nil {
					samples = append(samples, store.MetricSample{
						Name:      mapping.PrometheusName,
						Labels:    map[string]string{"stage": "asleep"},
						Value:     *dp.Asleep,
						Timestamp: ts,
						Type:      mapping.Type,
					})
				}
			}
		}
	}

	return samples, nil
}

// parseMetrics extracts metrics from multiple JSON formats.
func parseMetrics(payload []byte) ([]rawMetric, error) {
	// 1. {"data": {"metrics": [...]}} format
	var wrapped wrappedPayload
	if err := json.Unmarshal(payload, &wrapped); err == nil && len(wrapped.Data.Metrics) > 0 {
		return wrapped.Data.Metrics, nil
	}

	// 2. {"metrics": [...]} format
	var mp metricsPayload
	if err := json.Unmarshal(payload, &mp); err == nil && len(mp.Metrics) > 0 {
		return mp.Metrics, nil
	}

	// 3. direct array [...] format
	var arr []rawMetric
	if err := json.Unmarshal(payload, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	return nil, fmt.Errorf("unrecognized JSON format")
}

// parseTimestamp parses date strings from Health Auto Export.
func parseTimestamp(s string) time.Time {
	// "2024-01-01 12:00:00 +0900" format
	formats := []string{
		"2006-01-02 15:04:05 -0700",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}

	slog.Warn("failed to parse timestamp, using current time", "raw", s)
	return time.Now()
}
