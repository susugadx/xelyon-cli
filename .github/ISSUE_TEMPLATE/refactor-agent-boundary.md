---
name: "Refactor: Agent 境界整理"
about: "Agent 初期化・永続化・観測ロジック・コマンド責務を分離する"
title: "refactor(agent): bootstrap / persistence / observability / command の責務分離"
labels: ["refactor", "tech-debt", "agent"]
assignees: []
---

## 背景
`internal/agent/agent.go` と周辺に、以下の責務が混在している。
- 起動オーケストレーション（runtime/provider/MCP/LSP）
- セッション永続化
- ツール observability 集計
- slash command 実装の一部

変更頻度が高く、機能追加時の競合と回帰リスクが高い。

## 目的
Agent の owner 境界を明確化し、各変更が局所化される構造にする。

## Ownership
- owner: `internal/agent`
- 主境界:
- `bootstrap`: `NewAgentWithRuntime` まわりの初期化責務
- `session_persistence`: history/change storage 保存責務
- `tool_observability`: ツール観測メトリクス責務
- `commands`: slash command の機能別責務

## Contract
- current contract:
- `NewAgentWithRuntime` が起動時に runtime/provider/MCP/LSP を整える
- `Agent` はセッション保存と tool observability を管理する
- target contract:
- 公開挙動を維持したまま、実装オーナーを分割
- 意図的に変えないもの:
- slash command の UI 仕様
- ツール実行や履歴更新の既存フロー

## スコープ
- 対象:
- [ ] `internal/agent/agent.go` の責務分割
- [ ] `commands_misc.go` などコマンド実装の整理
- [ ] observability 関連関数の owner 明確化
- [ ] 必要なテスト追加/更新
- 非対象:
- 新コマンド追加
- Plan Mode の仕様変更

## 作業項目
- [ ] `NewAgentWithRuntime` の初期化処理を bootstrap 系へ切り出し
- [ ] `persistSession` / `Cleanup` まわりを persistence owner へ切り出し
- [ ] search/read_file 観測ロジックを observability owner へ切り出し
- [ ] `commands_misc.go` の機能群をコマンド単位で整理
- [ ] `Agent` 構造体に残す責務を再定義し、コメントで明文化

## DoD
- [ ] `agent.go` が巨大オーナーになっていない（高レベルオーケストレーション中心）
- [ ] 既存コマンド挙動（`/status` `/tokens` `/copy` `/compress` `/think`）が同等
- [ ] 並列実行時のロック整合（`historyMu` / `statsMu` など）が維持される
- [ ] 回帰テストが追加・更新され、責務境界がテストで固定される
- [ ] `make ci-check` が通る

## テスト観点
- [ ] Agent 初期化（MCP 有効/無効、LSP 有効/無効）で挙動が変わらない
- [ ] セッション保存/復元の挙動が維持される
- [ ] observability カウンタの更新条件が変わっていない
- [ ] slash command の入出力と状態更新が維持される
- [ ] headless 実行フローとの整合が維持される

## 検証コマンド
```bash
go test ./internal/agent ./internal/tools/... ./internal/ui
make ci-check
```

## リスク・フォローアップ
- `Agent` は参照箇所が多いため、分割時に循環依存が入りやすい。
- 先に structural cleanup、後で behavior change の順で分離すること。
