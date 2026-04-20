---
name: "Refactor: Provider Streaming 共通化"
about: "OpenAI互換ストリーミング処理の重複を共通層へ集約する"
title: "refactor(providers): OpenAI互換 SSE / tool_call 集約処理の共通化"
labels: ["refactor", "tech-debt", "provider"]
assignees: []
---

## 背景
`openrouter` / `deepseek` / `groq` / `openai(completions)` などで、OpenAI互換SSE処理が重複している。
- `data: [DONE]` 終端処理
- delta 結合
- tool_call 分割チャンクの再組み立て
- ストリーム中断時の partial 応答処理

重複により、仕様変更時の追従漏れと回帰が起きやすい。

## 目的
OpenAI互換ストリーミングの共通責務を1箇所に集約し、provider側は差分ロジックのみ持つ構造にする。

## Ownership
- owner: `internal/api/providers`
- 主境界:
- `openai_compat_stream`（共通SSE/tool_call処理）
- 各provider（request構築、provider固有パラメータ、固有警告/フォールバック）

## Contract
- current contract:
- 各providerが独自にSSEを読み取り、最終テキスト/ツール呼び出しを返す
- target contract:
- 共通パーサがSSEの基本挙動を提供し、providerは差分を注入する
- 意図的に変えないもの:
- 各providerの API URL / ヘッダ / request body
- provider別の機能差（thinking警告や独自エンドポイント分岐）

## スコープ
- 対象:
- [ ] SSE共通処理の新規 owner 導入
- [ ] `openrouter` / `deepseek` / `groq` / `openai(completions)` の移行
- [ ] 重複 `toolCallAccumulator` の集約
- [ ] 必要なテスト追加/更新
- 非対象:
- Responses API 固有仕様の変更
- Gemini/Claude の独自ストリーム処理全面置換

## 作業項目
- [ ] 共通SSE読み取り・イベント処理ユーティリティを実装
- [ ] テキストdelta統合と tool_call 再構築を共通化
- [ ] 各providerの `handleStreamingResponse` を thin 化
- [ ] 共通層と各providerの責務境界をコメントで明確化
- [ ] 既存テストを移行し、provider差分が消えていないことを確認

## DoD
- [ ] OpenAI互換provider群の重複実装が解消されている
- [ ] 各providerの外部挙動（戻り値・エラー処理・partial処理）が維持される
- [ ] provider固有差分が共通層に漏れ込んでいない
- [ ] 共通層に対する回帰テストがある
- [ ] `make ci-check` が通る

## テスト観点
- [ ] `data: [DONE]` 終端処理
- [ ] テキストdeltaの順序通り結合
- [ ] tool_call arguments が複数チャンクで正しく再構築される
- [ ] context cancel 時の partial 応答返却方針
- [ ] 非ストリーミング fallback 分岐
- [ ] provider固有（tool_choice 強制など）が維持される

## 検証コマンド
```bash
go test ./internal/api/... ./internal/api/providers/...
make ci-check
```

## リスク・フォローアップ
- 共通化時に provider 固有仕様を吸い込みすぎると逆に責務混在する。
- 共通層は「OpenAI互換SSEの最小共通集合」に限定する。
