# vitalbridge

iPhoneヘルスケアデータをPrometheusメトリクスに変換するIngestサービス。iOSアプリ [Health Auto Export](https://apps.apple.com/app/health-auto-export/id1115567461) からWebhookでJSONを受信し、Grafana Alloy経由でGrafana Cloudに可視化する。

## アーキテクチャ

```
iPhone (Health Auto Export)
  → POST /api/ingest (Bearer token認証)
    → vitalbridge (JSON→Prometheus変換、インメモリ保持)
      → Grafana Alloy scrapes /metrics (60s間隔)
        → Grafana Cloud Prometheus + VictoriaMetrics
```

## エンドポイント

| Method | Path | Auth | 用途 |
|--------|------|------|------|
| `POST` | `/api/ingest` | Bearer token | Webhook受信 |
| `GET` | `/api/health` | なし | k8s liveness/readiness probe |
| `GET` | `/metrics` | なし | Prometheus exposition format |
| `GET` | `/api/status` | Bearer token | デバッグ情報 |

## 対応メトリクス

Health Auto Exportの以下のメトリクスを `health_` プレフィックスのPrometheusメトリクスに変換する（許可リスト方式）。

| Health Auto Export | Prometheus | Labels |
|---|---|---|
| `heart_rate` | `health_heart_rate_bpm` | `stat={avg,min,max}` |
| `resting_heart_rate` | `health_resting_heart_rate_bpm` | — |
| `heart_rate_variability` | `health_heart_rate_variability_ms` | — |
| `blood_oxygen` | `health_blood_oxygen_ratio` | `stat={avg,min,max}` |
| `respiratory_rate` | `health_respiratory_rate_per_minute` | `stat={avg,min,max}` |
| `step_count` | `health_steps_total` | — |
| `active_energy` | `health_active_energy_kcal` | — |
| `basal_energy_burned` | `health_basal_energy_kcal` | — |
| `walking_running_distance` | `health_walking_distance_meters` | — |
| `flights_climbed` | `health_flights_climbed_total` | — |
| `sleep_analysis` | `health_sleep_duration_seconds` | `stage={in_bed,asleep,...}` |
| `apple_exercise_time` | `health_exercise_minutes` | — |
| `body_temperature` | `health_body_temperature_celsius` | — |
| `environmental_audio_exposure` | `health_noise_exposure_db` | `stat={avg,min,max}` |
| `walking_speed` | `health_walking_speed_mps` | `stat={avg,min,max}` |

運用メトリクス: `health_ingest_last_received_timestamp`, `health_ingest_samples_total`, `health_ingest_active_metrics`

## セットアップ

### 環境変数

| 変数 | 必須 | 説明 |
|------|------|------|
| `HEALTH_INGEST_API_KEY` | Yes | Bearer token認証用APIキー |
| `TZ` | No | タイムゾーン（デフォルト: `Asia/Tokyo`） |

### ローカル実行

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

## 設定

`config/config.yaml` で設定。環境変数は `${VAR_NAME}` 形式で展開される。

```yaml
server:
  host: "0.0.0.0"
  port: 8080

auth:
  api_key: "${HEALTH_INGEST_API_KEY}"

metrics:
  ttl_hours: 48                  # メトリクス保持期間
  cleanup_interval_minutes: 60   # クリーンアップ間隔

log:
  level: "info"                  # debug, info, warn, error
```

## テスト

```bash
# ヘルスチェック
curl http://localhost:8080/api/health

# サンプルデータ送信
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

# メトリクス確認
curl http://localhost:8080/metrics

# ステータス確認
curl -H "Authorization: Bearer your-secret-key" http://localhost:8080/api/status
```

## Kubernetes デプロイ

K8sマニフェストとFlux CD設定は [home-server](https://github.com/korosuke613/home-server) リポジトリの `k8s/health-ingest-service/` で管理。

```bash
# port-forward経由で確認
kubectl -n health-ingest port-forward svc/health-ingest-service 8080:8080
curl http://localhost:8080/api/health
```

## Health Auto Export 設定

iOSアプリ側の設定:

- **Automation Destination**: REST API
- **URL**: `https://health.korosuke613.dev/api/ingest`
- **Method**: POST
- **Headers**: `Authorization: Bearer <api-key>`
- **Export Format**: JSON
- **Automation interval**: 6〜15分

## ディレクトリ構成

```
vitalbridge/
├── main.go              # エントリポイント、HTTPサーバー、graceful shutdown
├── config/
│   ├── config.go        # 設定構造体、YAMLローダー、バリデーション
│   └── config.yaml      # デフォルト設定
├── handlers/
│   ├── ingest.go        # POST /api/ingest
│   ├── health.go        # GET /api/health
│   ├── metrics.go       # GET /metrics (Prometheus text format)
│   ├── status.go        # GET /api/status
│   └── response.go      # JSON応答ヘルパー
├── store/
│   └── metrics_store.go # スレッドセーフなインメモリストア (sync.RWMutex)
├── converter/
│   ├── health_export.go # Health Auto Export JSON → MetricSample変換
│   └── metric_names.go  # メトリクス名マッピング（許可リスト）
├── middleware/
│   └── auth.go          # Bearer token認証 (crypto/subtle.ConstantTimeCompare)
└── Dockerfile           # マルチステージビルド (CGO_ENABLED=0)
```

## ライセンス

MIT License
