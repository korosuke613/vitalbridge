# ghacron と同一のリリース方式を vitalbridge に導入する

## Context

vitalbridge は現在 `build-and-push.yml` で main プッシュ時に Docker イメージを ghcr.io へプッシュするのみ。バージョニング、バイナリ配布、GitHub Release 連動が存在しない。

ghacron のリリース方式（GitHub Release 作成 → GoReleaser でバイナリ配布 + Docker マルチアーキテクチャイメージのビルド・プッシュ）を vitalbridge にも導入する。

## 変更一覧

### 1. `main.go` のバージョン変数を ldflags 注入可能にする ✅

- `const Version = "0.1.0"` → `var version = "dev"` に変更
- 参照箇所（`Version` → `version`）を全て書き換え
- `-version` フラグ出力を `vitalbridge v%s` 形式に変更

### 2. `.goreleaser.yml` を新規作成 ✅

- ghacron ベース、GoReleaser v2 最新記法（`formats` / `ids`）に更新
- linux/amd64 + linux/arm64 対象
- ldflags で `main.version` にバージョン注入

### 3. `Dockerfile` を修正 ✅

- `ARG VERSION=dev` 追加
- ldflags に `-X main.version=${VERSION}` 追加
- ランタイムベースイメージ（Alpine）等の vitalbridge 固有要素は維持

### 4. `.github/workflows/release.yml` を新規作成 ✅

- ghacron の `release.yml` を vitalbridge 用に調整。3ジョブ構成:
  - **goreleaser**: バイナリビルド・GitHub Release へアップロード
  - **build-and-push**: Docker マルチアーキテクチャイメージのビルド（amd64 + arm64 並列）
  - **merge-manifests**: マルチアーキテクチャマニフェスト作成・タグ付け
- トリガー: `on: release: types: [published]`
- タグ戦略: 安定版は `latest` + `major.minor`、プレリリースはバージョンタグのみ

### 5. `.github/workflows/build-and-push.yml` を削除し、`ci.yml` に置き換え ✅

- `build-and-push.yml` を削除
- `ci.yml` を新規作成（push to main + PR でトリガー）: `go vet` / `go build` / `go test`

### 6. `/generate-release` スキルを新規作成 ✅

- `.claude/skills/generate-release/SKILL.md` を ghacron から移植
- Conventional Commits 解析 → セマンティックバージョン算出 → `gh release create` の自動化スキル

### 7. `README.md` に Release Strategy セクション追記 ✅

- ghacron と同一構成の Release Strategy セクションを Kubernetes Deployment の直前に挿入
- Docker イメージ名を `ghcr.io/korosuke613/vitalbridge` に差し替え

## 変更対象ファイル

| ファイル | 操作 | 状態 |
|---|---|---|
| `main.go` | 編集 | ✅ |
| `Dockerfile` | 編集 | ✅ |
| `.goreleaser.yml` | 新規作成 | ✅ |
| `.github/workflows/release.yml` | 新規作成 | ✅ |
| `.github/workflows/ci.yml` | 新規作成 | ✅ |
| `.github/workflows/build-and-push.yml` | 削除 | ✅ |
| `.claude/skills/generate-release/SKILL.md` | 新規作成 | ✅ |
| `README.md` | Release Strategy セクション追記 | ✅ |

## 検証結果

1. `goreleaser check` → ✅ 構文検証パス
2. `go build -ldflags="-s -w -X main.version=test" -o /dev/null .` → ✅ ビルド成功
3. GitHub Release 作成によるワークフロー起動 → リモート側で確認
