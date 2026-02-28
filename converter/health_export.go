package converter

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/korosuke613/vitalbridge/store"
)

// Health Auto Export JSON構造体（複数フォーマット対応）

// wrappedPayload {"data": {"metrics": [...]}} 形式
type wrappedPayload struct {
	Data struct {
		Metrics []rawMetric `json:"metrics"`
	} `json:"data"`
}

// metricsPayload {"metrics": [...]} 形式
type metricsPayload struct {
	Metrics []rawMetric `json:"metrics"`
}

// rawMetric 個々のメトリクスエントリ
type rawMetric struct {
	Name  string          `json:"name"`
	Units string          `json:"units"`
	Data  []rawDataPoint  `json:"data"`
}

// rawDataPoint データポイント（柔軟なフィールド）
type rawDataPoint struct {
	Date  string   `json:"date"`
	Qty   *float64 `json:"qty"`
	Avg   *float64 `json:"Avg"`
	Min   *float64 `json:"Min"`
	Max   *float64 `json:"Max"`
	Value *float64 `json:"value"`
	// sleep_analysis用
	Stage     string   `json:"stage"`
	InBed     *float64 `json:"inBed"`
	Asleep    *float64 `json:"asleep"`
}

// Convert Health Auto Export JSONをメトリクスサンプルに変換
func Convert(payload []byte) ([]store.MetricSample, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("空のペイロードです")
	}

	metrics, err := parseMetrics(payload)
	if err != nil {
		return nil, fmt.Errorf("JSONのパースに失敗しました: %w", err)
	}

	if len(metrics) == 0 {
		return nil, fmt.Errorf("メトリクスが含まれていません")
	}

	var samples []store.MetricSample

	for _, m := range metrics {
		mapping, ok := AllowedMetrics[m.Name]
		if !ok {
			log.Printf("[Converter] 未知のメトリクスをスキップ: %s", m.Name)
			continue
		}

		for _, dp := range m.Data {
			ts := parseTimestamp(dp.Date)

			// 基本値（qty or value）
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

			// HasStats=true の場合は avg/min/max も生成
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

			// sleep_analysis特殊処理
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

// parseMetrics 複数のJSONフォーマットに対応してメトリクスを抽出
func parseMetrics(payload []byte) ([]rawMetric, error) {
	// 1. {"data": {"metrics": [...]}} 形式
	var wrapped wrappedPayload
	if err := json.Unmarshal(payload, &wrapped); err == nil && len(wrapped.Data.Metrics) > 0 {
		return wrapped.Data.Metrics, nil
	}

	// 2. {"metrics": [...]} 形式
	var mp metricsPayload
	if err := json.Unmarshal(payload, &mp); err == nil && len(mp.Metrics) > 0 {
		return mp.Metrics, nil
	}

	// 3. 直接配列 [...] 形式
	var arr []rawMetric
	if err := json.Unmarshal(payload, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	return nil, fmt.Errorf("認識可能なJSONフォーマットではありません")
}

// parseTimestamp Health Auto Exportの日付文字列をパース
func parseTimestamp(s string) time.Time {
	// "2024-01-01 12:00:00 +0900" 形式
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

	log.Printf("[Converter] 日付のパースに失敗、現在時刻を使用: %s", s)
	return time.Now()
}
