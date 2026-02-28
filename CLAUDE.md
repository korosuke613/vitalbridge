# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

コミットメッセージ、プルリクエストは英語で書くこと。

## Language Rules

- `.go`ファイル内の文字列（ログメッセージ、エラーメッセージ、コメント）はすべて英語で記述すること
- CLAUDE.md、`plans/`、`.claude/` 配下のファイルは日本語OK（ランタイムに影響しないため）

## Commit Convention

コミットメッセージは [Conventional Commits](https://www.conventionalcommits.org/) に従うこと。

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

- **type**: `feat`, `fix`, `docs`, `refactor`, `test`, `ci`, `chore`, `perf`, `build`
- **scope**: 使用しない
- **description**: 変更の要約（英語、小文字始まり、末尾ピリオドなし）
- **breaking change**: 破壊的変更がある場合は `!` を付与（例: `feat!: rename config key`）
  - デフォルト値の変更、設定キーのリネーム、APIレスポンス構造の変更など、既存ユーザーが同じ設定のままアップグレードしたときに挙動が変わるものはすべて破壊的変更

## Common Commands

```bash
# Build
go build -o vitalbridge .

# Run (requires API key)
HEALTH_INGEST_API_KEY=test go run main.go -config config/config.yaml

# Test all
go test ./...

# Test single package
go test ./converter/

# Vet
go vet ./...

# Dependencies
go mod tidy

# Docker build
docker build -t vitalbridge .
```

### API Testing (requires running instance)

```bash
# Health check
curl http://localhost:8080/api/health

# Ingest data
curl -X POST http://localhost:8080/api/ingest \
  -H "Authorization: Bearer test" \
  -H "Content-Type: application/json" \
  -d '{"metrics": [{"name": "heart_rate", "data": [{"date": "2024-01-01 12:00:00 +0900", "qty": 72}]}]}'

# Prometheus metrics
curl http://localhost:8080/metrics

# Service status
curl -H "Authorization: Bearer test" http://localhost:8080/api/status
```

## Architecture Overview

iPhoneヘルスケア（Health Auto Export）のデータを受信し、Prometheusメトリクス形式で公開するGoスタンドアロンサービス。

### Startup Flow

```
main.go: -version flag → bootstrap slog (JSON) → config.Load (YAML + env vars)
  → re-init slog → store.NewMetricsStore → http.NewServeMux (routes)
  → http.Server → cleanup goroutine → signal wait → graceful shutdown
```

### Core Packages

| Package | 責務 |
|---------|------|
| `config/` | YAML設定ファイル読み込み、環境変数展開、バリデーション |
| `converter/` | Health Auto Export JSON → Prometheus MetricSample 変換 |
| `handlers/` | HTTPハンドラ（ingest, health, metrics, status） |
| `middleware/` | Bearer Token認証ミドルウェア |
| `store/` | スレッドセーフなインメモリメトリクスストア（TTL付き） |

### Key Design Decisions

- **Allowlist-based metric conversion**: `converter/metric_names.go` の `AllowedMetrics` マップに定義されたメトリクスのみ変換
- **Multiple JSON formats**: `{"data":{"metrics":[...]}}`, `{"metrics":[...]}`, `[...]` の3形式に対応
- **In-memory store with TTL**: 外部DB不要、`CleanExpired` goroutine で期限切れデータを自動削除
- **Structured logging**: `log/slog` による JSON 構造化ログ（Grafana Loki 等での解析を想定）
