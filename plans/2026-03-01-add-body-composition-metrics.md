# 体組成メトリクス追加 & blood_oxygen キー修正

## Context

Ankerスマート体組成計を導入し、Health Auto Export 経由で体組成データが送信されるようになった。
しかし現在の `AllowedMetrics` に体組成メトリクスが存在しないため、全データが黙殺されている。

加えて、`blood_oxygen` のキー名が実際のJSONペイロード（`blood_oxygen_saturation`）と不一致であり、血中酸素データも取りこぼしていることが判明した。

## 変更対象ファイル

- `converter/metric_names.go` — AllowedMetrics マップの修正・追加

## 変更内容

### 1. バグ修正: blood_oxygen キー名不一致

現在のキー `"blood_oxygen"` を `"blood_oxygen_saturation"` に変更する。

実データの単位が `%`（0〜100）であるため、Prometheus名も修正する:
- 旧: `health_blood_oxygen_ratio`（unit: ratio）
- 新: `health_blood_oxygen_percent`（unit: %）

このメトリクスはキーが間違っていたため実質未稼働。Prometheus名変更による破壊的影響なし。
HasStats は既存設定を維持（true）。

### 2. 新規メトリクス追加（6種）

#### 体組成（4種）

| HAE name | Prometheus name | Unit | Help |
|---|---|---|---|
| `weight_body_mass` | `health_weight_kg` | kg | Body weight in kilograms |
| `body_fat_percentage` | `health_body_fat_percent` | % | Body fat percentage |
| `body_mass_index` | `health_bmi` | kg/m² | Body mass index |
| `lean_body_mass` | `health_lean_body_mass_kg` | kg | Lean body mass in kilograms |

いずれも `HasStats: false`（1日1回の計測データ）。

#### 強く推奨メトリクス（2種）

| HAE name | Prometheus name | Unit | Help |
|---|---|---|---|
| `vo2_max` | `health_vo2_max` | ml/(kg·min) | Maximum oxygen uptake |
| `walking_heart_rate_average` | `health_walking_heart_rate_avg_bpm` | bpm | Average heart rate while walking |

いずれも `HasStats: false`。

## 検証方法

```bash
go build -o vitalbridge .
go vet ./...
```

実データ（`HealthAutoExport-2026-03-01-2026-03-01.json`）を使ったインジェストテストで `blood_oxygen_saturation` が取得できることを確認。
