package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MetricSample 個々のメトリクスサンプル
type MetricSample struct {
	Name      string
	Labels    map[string]string
	Value     float64
	Timestamp time.Time
	Type      string // "gauge" or "counter"
}

// MetricsStore スレッドセーフなインメモリメトリクスストア
type MetricsStore struct {
	mu             sync.RWMutex
	samples        map[string]MetricSample
	lastReceivedAt time.Time
	totalSamples   int64
}

// NewMetricsStore 新しいメトリクスストアを作成
func NewMetricsStore() *MetricsStore {
	return &MetricsStore{
		samples: make(map[string]MetricSample),
	}
}

// buildKey メトリクス名とラベルから一意キーを生成
func buildKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%s=%q", k, labels[k])
	}
	b.WriteByte('}')
	return b.String()
}

// Update 複数サンプルを一括更新
func (s *MetricsStore) Update(samples []MetricSample) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sample := range samples {
		key := buildKey(sample.Name, sample.Labels)
		s.samples[key] = sample
	}

	s.lastReceivedAt = time.Now()
	s.totalSamples += int64(len(samples))
}

// GetAll 全メトリクスを返す
func (s *MetricsStore) GetAll() []MetricSample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]MetricSample, 0, len(s.samples))
	for _, sample := range s.samples {
		result = append(result, sample)
	}
	return result
}

// CleanExpired TTL超過データを削除
func (s *MetricsStore) CleanExpired(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-ttl)
	for key, sample := range s.samples {
		if sample.Timestamp.Before(cutoff) {
			delete(s.samples, key)
		}
	}
}

// GetStats 統計情報を返す
func (s *MetricsStore) GetStats() (lastReceived time.Time, totalSamples int64, activeMetrics int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastReceivedAt, s.totalSamples, len(s.samples)
}
