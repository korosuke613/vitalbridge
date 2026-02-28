# ghacron のログ形式・開発ルールを vitalbridge に移植する

## Context

vitalbridge は現在、旧式の `log` パッケージで日本語ログを出力しており、ログレベル設定が config に存在するが実際にはフィルタリングされていない。ghacron プロジェクトでは `log/slog` による構造化ログ（JSON デフォルト）、英語メッセージ、CLAUDE.md による開発ルール（Conventional Commits 等）が確立されている。これらのパターンを vitalbridge にも適用し、統一されたコードベース規約を実現する。

## 変更対象ファイル一覧

| # | ファイル | 変更内容 |
|---|---------|---------|
| 1 | `config/config.go` | `LogConfig` に `Format` フィールド追加、`SlogLevel()` メソッド追加、日本語→英語 |
| 2 | `config/config.yaml` | `log.format: "json"` 追加 |
| 3 | `main.go` | `log` → `slog` 移行、`init()` 削除、`initLogger()` 追加、日本語→英語 |
| 4 | `handlers/ingest.go` | `log` → `slog`、日本語→英語 |
| 5 | `handlers/response.go` | `log` → `slog`、日本語→英語 |
| 6 | `handlers/health.go` | 日本語→英語（log 使用なし） |
| 7 | `handlers/metrics.go` | 日本語→英語（log 使用なし、HELP テキスト含む） |
| 8 | `handlers/status.go` | 日本語→英語（log 使用なし） |
| 9 | `middleware/auth.go` | `log` → `slog`（Warn レベル）、日本語→英語 |
| 10 | `converter/health_export.go` | `log` → `slog`、日本語→英語 |
| 11 | `converter/metric_names.go` | 日本語コメント→英語（log 使用なし） |
| 12 | `store/metrics_store.go` | 日本語コメント→英語（log 使用なし） |
| 13 | `CLAUDE.md` | 新規作成（ghacron ベース） |
| 14 | `.claude/settings.local.json` | 新規作成 |
| 15 | `.gitignore` | 新規作成（`.claude/settings.local.json` を含む） |

## 実装手順

### Step 1: `config/config.go` — 基盤変更

`LogConfig` を拡張し、`slog` レベル変換メソッドを追加する。他の全ファイルがこれに依存する。

- `LogConfig` に `Format string \`yaml:"format"\`` フィールド追加
- `SlogLevel() slog.Level` メソッド追加（ghacron の `config/config.go` と同一パターン）
- `validate()` で `Format` のバリデーション追加（`"json"`, `"text"`, `""` を許可）
- `Format` が空の場合は `"json"` をデフォルトとして設定
- ログレベルのバリデーションを switch 文に簡素化
- 全ての日本語文字列（コメント・エラーメッセージ）を英語に変換

### Step 2: `config/config.yaml` — format フィールド追加

```yaml
log:
  level: "info"
  format: "json"
```

### Step 3: `main.go` — slog 移行の中核

- `init()` 関数を完全削除（`log.SetFlags` は不要、`time.Local = Asia/Tokyo` も削除）
- import の `"log"` を `"log/slog"` に置換、`"strings"` を追加
- `main()` 冒頭にブートストラップロガー設置: `slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))`
- config.Load 成功後に `initLogger(&cfg.Log)` を呼び出し
- `initLogger` 関数を追加（ghacron の `main.go:102-115` と同一パターン）
- 全 `log.*` 呼び出しを `slog.*` に変換（構造化キー・値ペア付き、英語メッセージ）
- `log.Fatalf` → `slog.Error` + `os.Exit(1)`（slog に Fatal 相当はないため）
- flag ヘルプ文字列を英語に変換
- 全コメントを英語に変換

### Step 4: `handlers/ingest.go`

- import `"log"` → `"log/slog"`
- `log.Printf("[Ingest] %d サンプルを...")` → `slog.Info("samples received and stored", "count", len(samples))`
- 全 `sendError` の日本語メッセージを英語に
- コメントを英語に

### Step 5: `handlers/response.go`

- import `"log"` → `"log/slog"`
- `log.Printf("[Handler] JSON応答エラー...")` → `slog.Error("failed to encode JSON response", "error", err)`
- コメントを英語に

### Step 6: `middleware/auth.go`

- import `"log"` → `"log/slog"`
- 認証失敗ログを `slog.Warn` に昇格（セキュリティイベントのため）
  - `"missing authorization header"`, `"invalid authorization scheme"`, `"invalid token"` + `"method"`, `"path"` 属性
- コメントを英語に

### Step 7: `converter/health_export.go`

- import `"log"` → `"log/slog"`
- 未知メトリクスのスキップログを `slog.Debug` に降格（高頻度・想定内イベントのため）
- タイムスタンプパース失敗を `slog.Warn` に
- 全日本語文字列（エラーメッセージ・コメント）を英語に

### Step 8: ログ呼び出しなしファイルの日本語→英語変換

以下はコメントとエラー文字列のみの変更:

- `converter/metric_names.go` — 3つの日本語コメント
- `store/metrics_store.go` — 8つの日本語コメント
- `handlers/health.go` — コメント1つ、エラー文字列1つ
- `handlers/metrics.go` — コメント多数、HELP テキスト3つ
- `handlers/status.go` — コメント1つ、エラー文字列1つ

### Step 9: `CLAUDE.md` 新規作成

ghacron の CLAUDE.md をベースに vitalbridge 用に適応:
- Language Rules（.go ファイルは英語、CLAUDE.md/plans は日本語OK）
- Conventional Commits ルール
- Common Commands（build, run, test, vet, docker）
- Architecture Overview（vitalbridge 固有の内容）

### Step 10: `.claude/settings.local.json` 新規作成

```json
{
  "permissions": {
    "allow": [
      "Bash(go build:*)",
      "Bash(go test:*)",
      "Bash(go vet:*)",
      "Bash(go mod tidy:*)",
      "Bash(go doc:*)",
      "Bash(go run:*)"
    ]
  }
}
```

### Step 11: `.gitignore` 新規作成

```
.claude/settings.local.json
```

## 設計判断

| 判断 | 理由 |
|------|------|
| `init()` 完全削除 | `log.SetFlags` は slog で不要。`time.Local = Asia/Tokyo` は slog JSON の UTC タイムスタンプと矛盾し、グローバル副作用はコンテナ環境で不適切 |
| 未知メトリクスログを `slog.Debug` に | 許可リスト外メトリクスは高頻度・想定内。Info では運用ノイズが大きい |
| 認証失敗ログを `slog.Warn` に | セキュリティイベントは Info より高い可視性が必要 |
| config.yaml で format のデフォルトを json に | ghacron と統一。Grafana Loki 等での構造化ログ前提 |
| `Format` 未指定時は validate() で `"json"` を補完 | 既存の config.yaml（format フィールドなし）との後方互換性 |

## ニンジャ・クラン並行実行の提案

Step 1-3 は順序依存（config → main）だが、Step 4-8 は互いに独立しており並行実行可能。Step 9-11 も独立。以下のクラン編成を提案:

- **グランドマスター**: 全体統括、Step 1-3 をジッコウ後にニンジャを spawn
- **ニンジャA**: Step 4-5（handlers の slog 移行）
- **ニンジャB**: Step 6-7（middleware + converter の slog 移行）
- **ニンジャC**: Step 8（コメントのみ変更ファイル群）
- **ニンジャD**: Step 9-11（CLAUDE.md, settings, .gitignore 新規作成）

## 検証方法

```bash
# ビルド確認
go build -o /dev/null .

# vet 確認
go vet ./...

# 起動確認（ctrl+c で停止）
HEALTH_INGEST_API_KEY=test go run main.go -config config/config.yaml

# ログ形式の確認（JSON 出力であること）
HEALTH_INGEST_API_KEY=test go run main.go -config config/config.yaml 2>&1 | head -5

# テキスト形式の確認（config.yaml の format を text に変更して）
# → slog.TextHandler の key=value 形式で出力されること

# ingest エンドポイントのテスト
curl -X POST http://localhost:8080/api/ingest \
  -H "Authorization: Bearer test" \
  -H "Content-Type: application/json" \
  -d '{"metrics": [{"name": "heart_rate", "data": [{"date": "2024-01-01 12:00:00 +0900", "qty": 72}]}]}'
# → ログに slog.Info "samples received and stored" count=1 が出力されること

# 日本語文字列が残っていないことの確認
grep -r '[ぁ-ん\|ァ-ヴ\|亜-熙]' --include='*.go' .
# → 一致なしであること
```
