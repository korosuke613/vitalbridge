# vitalbridge

An ingest service that converts iPhone Health data into Prometheus metrics. Receives JSON via webhook from the iOS app [Health Auto Export](https://apps.apple.com/app/health-auto-export/id1115567461) and exposes metrics for Grafana Alloy to scrape into Grafana Cloud.

## Architecture

```
iPhone (Health Auto Export)
  → POST /api/ingest (Bearer token auth)
    → vitalbridge (JSON → Prometheus conversion, in-memory store)
      → Grafana Alloy scrapes /metrics (60s interval)
        → Grafana Cloud Prometheus + VictoriaMetrics
```

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/ingest` | Bearer token | Webhook receiver |
| `GET` | `/api/health` | None | k8s liveness/readiness probe |
| `GET` | `/metrics` | None | Prometheus exposition format |
| `GET` | `/api/status` | Bearer token | Debug information |

## Supported Metrics

The following Health Auto Export metrics are converted to Prometheus metrics with a `health_` prefix (allowlist-based).

| Health Auto Export | Prometheus | Description | Labels |
|---|---|---|---|
| `heart_rate` | `health_heart_rate_bpm` | Heart rate (bpm) | `stat={avg,min,max}` |
| `resting_heart_rate` | `health_resting_heart_rate_bpm` | Resting heart rate (bpm) | — |
| `heart_rate_variability` | `health_heart_rate_variability_ms` | HRV SDNN (ms) | — |
| `blood_oxygen_saturation` | `health_blood_oxygen_percent` | SpO2 (%) | `stat={avg,min,max}` |
| `respiratory_rate` | `health_respiratory_rate_per_minute` | Breaths per minute | `stat={avg,min,max}` |
| `step_count` | `health_steps_total` | Step count | — |
| `active_energy` | `health_active_energy_kcal` | Active energy burned (kcal) | — |
| `basal_energy_burned` | `health_basal_energy_kcal` | Basal metabolic energy (kcal) | — |
| `walking_running_distance` | `health_walking_distance_meters` | Walk + run distance (m) | — |
| `flights_climbed` | `health_flights_climbed_total` | Flights of stairs climbed | — |
| `sleep_analysis` | `health_sleep_duration_seconds` | Sleep duration (s) | `stage={in_bed,asleep,...}` |
| `apple_exercise_time` | `health_exercise_minutes` | Exercise time (min) | — |
| `body_temperature` | `health_body_temperature_celsius` | Body temperature (°C) | — |
| `environmental_audio_exposure` | `health_noise_exposure_db` | Environmental noise (dB) | `stat={avg,min,max}` |
| `walking_speed` | `health_walking_speed_mps` | Walking speed (m/s) | `stat={avg,min,max}` |
| `weight_body_mass` | `health_weight_kg` | Body weight (kg) | — |
| `body_fat_percentage` | `health_body_fat_percent` | Body fat (%) | — |
| `body_mass_index` | `health_bmi` | BMI (kg/m²) | — |
| `lean_body_mass` | `health_lean_body_mass_kg` | Lean body mass (kg) | — |
| `vo2_max` | `health_vo2_max` | VO2 max (ml/kg/min) | — |
| `walking_heart_rate_average` | `health_walking_heart_rate_avg_bpm` | Walking heart rate avg (bpm) | — |

Operational metrics: `health_ingest_last_received_timestamp`, `health_ingest_samples_total`, `health_ingest_active_metrics`

## Setup

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `HEALTH_INGEST_API_KEY` | Yes | API key for Bearer token authentication |
| `LOG_LEVEL` | No | Log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `TZ` | No | Timezone (default: UTC) |

### Local

```bash
export HEALTH_INGEST_API_KEY="your-secret-key"
go build -o vitalbridge .
./vitalbridge
```

### Docker

```bash
docker build -t vitalbridge .
docker run -p 8080:8080 -e HEALTH_INGEST_API_KEY="your-secret-key" vitalbridge
```

## Configuration

Configured via `config/config.yaml`. Environment variables are expanded using `${VAR_NAME}` syntax.

```yaml
server:
  host: "0.0.0.0"
  port: 8080

auth:
  api_key: "${HEALTH_INGEST_API_KEY}"

metrics:
  ttl_hours: 48                  # Metrics retention period
  cleanup_interval_minutes: 60   # Cleanup interval

log:
  level: "${LOG_LEVEL:-info}"    # debug, info, warn, error
  format: "json"                 # json, text
```

## Testing

```bash
# Health check
curl http://localhost:8080/api/health

# Send sample data
curl -X POST http://localhost:8080/api/ingest \
  -H "Authorization: Bearer your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "metrics": [
        {
          "name": "heart_rate",
          "units": "bpm",
          "data": [{"date": "2025-01-01 12:00:00 +0900", "qty": 72, "Avg": 68, "Min": 55, "Max": 95}]
        },
        {
          "name": "step_count",
          "units": "steps",
          "data": [{"date": "2025-01-01 12:00:00 +0900", "qty": 8500}]
        }
      ]
    }
  }'

# Check metrics
curl http://localhost:8080/metrics

# Check status
curl -H "Authorization: Bearer your-secret-key" http://localhost:8080/api/status
```

## Release Strategy

Releases are triggered exclusively by GitHub Release events — pushing to `main` does not publish any artifacts.

1. Create a GitHub Release (manually or via `/generate-release` skill in Claude Code)
2. The `release.yml` workflow automatically:
   - Builds Go binaries for linux/amd64 and linux/arm64 via [GoReleaser](https://goreleaser.com/)
   - Builds and pushes multi-arch Docker images to `ghcr.io/korosuke613/vitalbridge`

**Tagging rules:**

| Release type | Example tag | Docker tags | `latest` |
|---|---|---|---|
| Stable | `v1.0.0` | `1.0.0`, `1.0`, `latest` | Yes |
| Release candidate | `v1.0.0-rc.1` | `1.0.0-rc.1` | No |

## Kubernetes Deployment

K8s manifests and Flux CD configuration are managed in the [home-server](https://github.com/korosuke613/home-server) repository under `k8s/health-ingest-service/`.

```bash
# Verify via port-forward
kubectl -n health-ingest port-forward svc/health-ingest-service 8080:8080
curl http://localhost:8080/api/health
```

## Health Auto Export Settings

iOS app configuration:

- **Automation Destination**: REST API
- **URL**: `https://health.korosuke613.dev/api/ingest`
- **Method**: POST
- **Headers**: `Authorization: Bearer <api-key>`
- **Export Format**: JSON
- **Automation interval**: 6–15 minutes

## Directory Structure

```
vitalbridge/
├── main.go              # Entry point, HTTP server, graceful shutdown
├── config/
│   ├── config.go        # Config structs, YAML loader, validation
│   └── config.yaml      # Default configuration
├── handlers/
│   ├── ingest.go        # POST /api/ingest
│   ├── health.go        # GET /api/health
│   ├── metrics.go       # GET /metrics (Prometheus text format)
│   ├── status.go        # GET /api/status
│   └── response.go      # JSON response helpers
├── store/
│   └── metrics_store.go # Thread-safe in-memory store (sync.RWMutex)
├── converter/
│   ├── health_export.go # Health Auto Export JSON → MetricSample conversion
│   └── metric_names.go  # Metric name mapping (allowlist)
├── middleware/
│   └── auth.go          # Bearer token auth (crypto/subtle.ConstantTimeCompare)
└── Dockerfile           # Multi-stage build (CGO_ENABLED=0)
```

## License

MIT License
