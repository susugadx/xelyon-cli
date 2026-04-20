---
name: "Refactor: Config 境界整理"
about: "config の責務を defaults/load/save/migration/override に分離する"
title: "refactor(config): 設定責務の分離（defaults/load/save/migration/overrides）"
labels: ["refactor", "tech-debt", "config"]
assignees: []
---

## 背景
`internal/config/config.go` に以下の責務が混在しており、機能追加時の影響範囲が広い。
- デフォルト定義
- 設定ロード/保存
- 旧キー migration
- YAML パッチ処理
- 環境変数/フラグ override

## 目的
設定処理の責務境界を明確化し、今後の設定追加時に「どこを owner として触るか」を迷わない状態にする。

## Ownership
- owner: `internal/config`
- 主境界:
- `defaults`: デフォルト値の定義
- `loader`: 読み込み・適用順序
- `serializer`: 保存・YAML整形
- `migration`: 旧キー互換
- `overrides`: env/flag 上書き

## Contract
- current contract:
- `LoadConfig` は互換キー migration と defaults 適用を行う
- `SaveConfig` は provider_models の保存形を調整する
- target contract:
- 公開関数シグネチャは維持しつつ、内部実装を責務別に分割する
- 意図的に変えないもの:
- 設定優先順位（明示新キー > 旧キー補完）
- provider_models の absent / explicit empty の意味

## スコープ
- 対象:
- [ ] `internal/config/config.go` の責務分割
- [ ] `internal/config/provider_model_*` 群との境界整理
- [ ] 必要なテスト追加/更新
- 非対象:
- 新しい設定項目の追加
- CLIコマンド仕様変更

## 作業項目
- [ ] `DefaultConfig` 周辺を defaults owner へ抽出
- [ ] `LoadConfig` の処理を loader/migration/default適用の段階に分離
- [ ] `SaveConfig` と YAML パッチ処理を serializer owner へ分離
- [ ] `ApplyEnvironmentOverrides` / `ApplyFlagOverrides` を override owner へ分離
- [ ] `config.go` は公開エントリと最小オーケストレーションのみ残す
- [ ] 既存 call site の import/呼び出しを整理

## DoD
- [ ] `internal/config/config.go` に責務混在が残っていない
- [ ] 公開 API（関数名・引数・戻り値）を壊していない
- [ ] 設定ロード/保存/互換挙動が既存と同等
- [ ] 境界ごとのテストがあり、仕様意図がテストで固定されている
- [ ] `make ci-check` が通る

## テスト観点
- [ ] 新旧キー優先順位（新キー優先、旧キーは補完のみ）
- [ ] `provider_models` の absent / explicit empty / entries の往復保存
- [ ] `lsp.servers` の nil と empty map の区別保持
- [ ] 環境変数 override とフラグ override の適用条件
- [ ] 不正 YAML 読み込み時のエラー挙動

## 検証コマンド
```bash
go test ./internal/config ./internal/api
make ci-check
```

## リスク・フォローアップ
- `LoadConfig`/`SaveConfig` は参照箇所が多いため、分割時に互換挙動を崩しやすい。
- 互換 alias の整理は別 issue で段階的に実施する（この issue では契約維持優先）。
