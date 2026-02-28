# ghacron と同一のリリース方式を vitalbridge に導入する

## Context

vitalbridge は現在 `build-and-push.yml` で main プッシュ時に Docker イメージを ghcr.io へプッシュするのみ。バージョニング、バイナリ配布、GitHub Release 連動が存在しない。

ghacron のリリース方式（GitHub Release 作成 → GoReleaser でバイナリ配布 + Docker マルチアーキテクチャイメージのビルド・プッシュ）を vitalbridge にも導入する。

## 変更一覧

### 1. `main.go` のバージョン変数を ldflags 注入可能にする

- **ファイル**: `main.go:21`
- `const Version = "0.1.0"` → `var version = "dev"` に変更
- 参照箇所（`Version` → `version`）を全て書き換え（L29, L45）
- `-version` フラグ出力も `vitalbridge v%s` 形式に変更

### 2. `.goreleaser.yml` を新規作成

ghacron の `.goreleaser.yml` をベースに vitalbridge 用に調整:

```yaml
version: 2
project_name: vitalbridge

before:
  hooks:
    - go mod tidy

builds:
  - id: vitalbridge
    main: .
    binary: vitalbridge
    env:
      - CGO_ENABLED=0
    goos:
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - id: vitalbridge
    builds:
      - vitalbridge
    format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

changelog:
  disable: true

release:
  github:
    owner: korosuke613
    name: vitalbridge
  draft: false
  prerelease: auto
```

### 3. `Dockerfile` を修正

- `VERSION` ビルド引数を追加
- ldflags に `-X main.version=${VERSION}` を追加

```dockerfile
ARG VERSION=dev
...
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o vitalbridge .
```

※ ランタイムベースイメージ（Alpine）、config.yaml コピー、appuser 等の vitalbridge 固有要素はそのまま維持

### 4. `.github/workflows/release.yml` を新規作成

ghacron の `release.yml` を vitalbridge 用に調整。3つのジョブ構成:

- **goreleaser**: GoReleaser でバイナリビルド・GitHub Release へアップロード
- **build-and-push**: Docker マルチアーキテクチャイメージのビルド（amd64 + arm64 並列）
- **merge-manifests**: マルチアーキテクチャマニフェスト作成・タグ付け

トリガー: `on: release: types: [published]`

タグ戦略（ghacron と同一）:
| リリースタイプ | 例 | Docker タグ |
|---|---|---|
| 安定版 | `v1.0.0` | `1.0.0`, `1.0`, `latest` |
| プレリリース | `v1.0.0-rc.1` | `1.0.0-rc.1` のみ |

### 5. `.github/workflows/build-and-push.yml` を削除し、`ci.yml` に置き換え

- `build-and-push.yml` を削除
- 新規 `ci.yml` を作成（push to main + PR でトリガー）:
  - `go vet`
  - `go build`（ビルドチェックのみ、アーティファクトは保存しない）
  - `go test`（テストがあれば実行）

## 変更対象ファイル

| ファイル | 操作 |
|---|---|
| `main.go` | 編集 |
| `Dockerfile` | 編集 |
| `.goreleaser.yml` | 新規作成 |
| `.github/workflows/release.yml` | 新規作成 |
| `.github/workflows/ci.yml` | 新規作成 |
| `.github/workflows/build-and-push.yml` | 削除 |

## 検証方法

1. `goreleaser check` で `.goreleaser.yml` の構文を検証
2. `go build -ldflags="-s -w -X main.version=test" -o /dev/null .` でバージョン注入を確認
3. GitHub Release を作成し、ワークフローが正常に起動することを確認（実際の動作確認はリモート側）
