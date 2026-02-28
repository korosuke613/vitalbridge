package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/korosuke613/vitalbridge/converter"
	"github.com/korosuke613/vitalbridge/store"
)

// NewMetricsHandler Prometheusメトリクスエンドポイントのハンドラを返す
func NewMetricsHandler(ms *store.MetricsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sendError(w, http.StatusMethodNotAllowed, "GETメソッドが必要です")
			return
		}

		samples := ms.GetAll()
		lastReceived, totalSamples, activeMetrics := ms.GetStats()

		var b strings.Builder

		// 運用メトリクス
		b.WriteString("# HELP health_ingest_last_received_timestamp 最後にデータを受信したUNIXタイムスタンプ\n")
		b.WriteString("# TYPE health_ingest_last_received_timestamp gauge\n")
		if !lastReceived.IsZero() {
			fmt.Fprintf(&b, "health_ingest_last_received_timestamp %d\n", lastReceived.Unix())
		} else {
			b.WriteString("health_ingest_last_received_timestamp 0\n")
		}

		b.WriteString("# HELP health_ingest_samples_total 受信したサンプルの累計数\n")
		b.WriteString("# TYPE health_ingest_samples_total counter\n")
		fmt.Fprintf(&b, "health_ingest_samples_total %d\n", totalSamples)

		b.WriteString("# HELP health_ingest_active_metrics 現在保持中のアクティブメトリクス数\n")
		b.WriteString("# TYPE health_ingest_active_metrics gauge\n")
		fmt.Fprintf(&b, "health_ingest_active_metrics %d\n", activeMetrics)

		// メトリクス名ごとにグループ化してHELP/TYPE出力
		type metricGroup struct {
			help    string
			mtype   string
			lines   []string
		}
		groups := make(map[string]*metricGroup)

		for _, sample := range samples {
			g, ok := groups[sample.Name]
			if !ok {
				// 逆引きマップからHELP情報を取得
				help := sample.Name
				if m, found := converter.PromNameToMapping[sample.Name]; found {
					help = m.Help
				}
				g = &metricGroup{
					help:  help,
					mtype: sample.Type,
				}
				groups[sample.Name] = g
			}

			line := formatSample(sample)
			g.lines = append(g.lines, line)
		}

		// メトリクス名でソートして出力
		names := make([]string, 0, len(groups))
		for name := range groups {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			g := groups[name]
			fmt.Fprintf(&b, "# HELP %s %s\n", name, g.help)
			fmt.Fprintf(&b, "# TYPE %s %s\n", name, g.mtype)
			sort.Strings(g.lines)
			for _, line := range g.lines {
				b.WriteString(line)
				b.WriteByte('\n')
			}
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(b.String()))
	}
}

// formatSample メトリクスサンプルをPrometheus text format行に変換
func formatSample(s store.MetricSample) string {
	if len(s.Labels) == 0 {
		return fmt.Sprintf("%s %g %d", s.Name, s.Value, s.Timestamp.UnixMilli())
	}

	keys := make([]string, 0, len(s.Labels))
	for k := range s.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lb strings.Builder
	lb.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			lb.WriteByte(',')
		}
		fmt.Fprintf(&lb, "%s=%q", k, s.Labels[k])
	}
	lb.WriteByte('}')

	return fmt.Sprintf("%s%s %g %d", s.Name, lb.String(), s.Value, s.Timestamp.UnixMilli())
}
