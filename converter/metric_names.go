package converter

// MetricMapping メトリクス名マッピング定義
type MetricMapping struct {
	PrometheusName string
	Type           string // "gauge" or "counter"
	Unit           string // for HELP text
	Help           string // human-readable description for HELP line
	HasStats       bool   // avg/min/max variants
}

// AllowedMetrics Health Auto Exportの名前 → Prometheusメトリクスへのマッピング許可リスト
var AllowedMetrics = map[string]MetricMapping{
	"heart_rate":                    {PrometheusName: "health_heart_rate_bpm", Type: "gauge", Unit: "bpm", Help: "Heart rate in beats per minute", HasStats: true},
	"resting_heart_rate":            {PrometheusName: "health_resting_heart_rate_bpm", Type: "gauge", Unit: "bpm", Help: "Resting heart rate in beats per minute"},
	"heart_rate_variability":        {PrometheusName: "health_heart_rate_variability_ms", Type: "gauge", Unit: "ms", Help: "Heart rate variability (SDNN) in milliseconds"},
	"blood_oxygen":                  {PrometheusName: "health_blood_oxygen_ratio", Type: "gauge", Unit: "ratio", Help: "Blood oxygen saturation as ratio", HasStats: true},
	"respiratory_rate":              {PrometheusName: "health_respiratory_rate_per_minute", Type: "gauge", Unit: "per minute", Help: "Respiratory rate in breaths per minute", HasStats: true},
	"step_count":                    {PrometheusName: "health_steps_total", Type: "gauge", Unit: "steps", Help: "Total step count"},
	"active_energy":                 {PrometheusName: "health_active_energy_kcal", Type: "gauge", Unit: "kcal", Help: "Active energy burned in kilocalories"},
	"basal_energy_burned":           {PrometheusName: "health_basal_energy_kcal", Type: "gauge", Unit: "kcal", Help: "Basal energy burned in kilocalories"},
	"walking_running_distance":      {PrometheusName: "health_walking_distance_meters", Type: "gauge", Unit: "meters", Help: "Walking and running distance in meters"},
	"flights_climbed":               {PrometheusName: "health_flights_climbed_total", Type: "gauge", Unit: "flights", Help: "Total flights of stairs climbed"},
	"sleep_analysis":                {PrometheusName: "health_sleep_duration_seconds", Type: "gauge", Unit: "seconds", Help: "Sleep duration in seconds by stage"},
	"apple_exercise_time":           {PrometheusName: "health_exercise_minutes", Type: "gauge", Unit: "minutes", Help: "Exercise time in minutes"},
	"body_temperature":              {PrometheusName: "health_body_temperature_celsius", Type: "gauge", Unit: "celsius", Help: "Body temperature in degrees Celsius"},
	"environmental_audio_exposure":  {PrometheusName: "health_noise_exposure_db", Type: "gauge", Unit: "dB", Help: "Environmental noise exposure in decibels", HasStats: true},
	"walking_speed":                 {PrometheusName: "health_walking_speed_mps", Type: "gauge", Unit: "m/s", Help: "Walking speed in meters per second", HasStats: true},
}

// PromNameToMapping PrometheusNameからMetricMappingへの逆引きマップ
var PromNameToMapping map[string]MetricMapping

func init() {
	PromNameToMapping = make(map[string]MetricMapping, len(AllowedMetrics))
	for _, m := range AllowedMetrics {
		PromNameToMapping[m.PrometheusName] = m
	}
}
