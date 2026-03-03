# PR4: RunHeadless Cleanup Leak — 6-Way Model Comparison

## 実験概要

`RunHeadless` 関数で `agent.Cleanup()` が呼ばれないリソースリークのバグ修正タスクを、6つの異なるモデル/フレームワーク組み合わせに同一の指示文で投げ、実装品質・テスト設計・コストを比較した。

**実験日:** 2026-03-02 〜 2026-03-04

### タスク仕様（全対戦者共通）

```
Goal: Fix resource leak in RunHeadless — ensure Agent cleanup is always called.

Requirements:
1) Fix the leak - Ensure Cleanup is always called after RunHeadless completes, even on error paths
2) Tests - Verify Cleanup is called in headless path, add leak-detection test for repeated invocations
3) Constraints - Minimal diff, do not change RunHeadless return type, gofmt, go test ./... must pass
```

### ブランチ構成

| 対戦者 | ブランチ | リポジトリ |
|---|---|---|
| XELYON + GPT-5.3 | `exp/pr4-xelyon-gpt53` | xelyon-cli |
| Codex CLI + GPT-5.3 | `exp/pr4-codex` | xelyon-cli-codex |
| XELYON + Gemini 3.1 Pro | `exp/pr4-xelyon-gemini` | xelyon-cli |
| XELYON + Sonnet 4.6 | `exp/pr4-xelyon-sonnet` | xelyon-cli |
| Claude Code (Opus 4.6) | `exp/pr4-claude-code` | xelyon-cli |
| XELYON + Opus 4.6 | `exp/pr4-xelyon-opus` | xelyon-cli |

---

## 結果一覧

### 最終スコア

| # | 対戦者 | 差分 | 要件 | テスト | 設計 | コスト | 合計 |
|---|---|---|---|---|---|---|---|
| 1 | XELYON + GPT-5.3 | 5 | 4 | 3 | 3.5 | 5 | **20.5** |
| 2 | Codex CLI + GPT-5.3 | 4.5 | 4.5 | 4 | 3.5 | 4.5 | **21** |
| 3 | XELYON + Gemini 3.1 Pro | 5 | 2.5 | 1.5 | 3 | 4 | **16** |
| 4 | XELYON + Sonnet 4.6 | 3.5 | 5 | 5 | 4.5 | 2 | **20** |
| 5 | Claude Code (Opus 4.6) | 5 | 5 | 4.5 | 4 | 5 | **23.5** |
| 6 | XELYON + Opus 4.6 | 4.5 | 5 | 5 | 4 | 2 | **20.5** |

### 実行統計

| # | 対戦者 | 時間 | コスト | トークン | テスト数 | コンパイルエラー |
|---|---|---|---|---|---|---|
| 1 | XELYON + GPT-5.3 | 3m 06s | $0.32 | 274.9K | 2 | 0 |
| 2 | Codex CLI + GPT-5.3 | 不明 | ~$0.16 | 不明 | 3 | 0 |
| 3 | XELYON + Gemini 3.1 Pro | 7m 58s | $0.63 | 309.0K | 1 | 2 |
| 4 | XELYON + Sonnet 4.6 | 3m 26s | $1.87 | 709.8K | 3 | 0 |
| 5 | Claude Code (Opus 4.6) | 2m 47s | MAX課金内 | 不明 | 3 | 0 |
| 6 | XELYON + Opus 4.6 | 7m 14s | $2.04 | 383.7K | 4 | 0 |

---

## 各対戦者の詳細

### 1. XELYON + GPT-5.3-codex

**ファイル:** `agent_run.go` (+10 -5), `agent_run_test.go` (+72 新規)

**アプローチ:**
- パッケージレベルの関数変数 `cleanupAgent = func(a *Agent) { a.Cleanup() }` を追加
- `RunHeadless` で `defer cleanupAgent(agent)` を使用
- テストでは `cleanupAgent` を上書きして `atomic.Int32` で呼び出し回数をカウント
- 実際の `Cleanup()` はテスト中に実行されない（スタブで置換）

**テスト:**
- `TestRunHeadless_CallsCleanup` — 成功パス
- `TestRunHeadless_RepeatedInvocations` — 5回反復

**欠点:** エラーパステストなし、`IsFunctionCallingEnabled() = false` でマルチターンパス未検証

---

### 2. Codex CLI + GPT-5.3-codex

**ファイル:** `agent_run.go` (+5 -0), `agent_run_test.go` (+112 新規)

**アプローチ:**
- `runHeadlessCleanup = func(agent *Agent) { agent.Cleanup() }` — XELYON版より命名が明確
- テスト用 mock provider に `response` と `err` フィールドを注入可能な設計

**テスト:**
- `TestRunHeadless_CallsCleanup` — 成功パス
- `TestRunHeadless_CallsCleanupOnError` — エラーパス ✅
- `TestRunHeadless_RepeatedInvocations` — 25回反復

**XELYON版との差:** エラーパステストあり、mock設計がより柔軟、命名が明確。ただし `atomic` 未使用（plain int）。

---

### 3. XELYON + Gemini 3.1 Pro

**ファイル:** `agent_run.go` (+1), `agent_run_test.go` (+36 新規)

**アプローチ:**
- 修正自体は `defer agent.Cleanup()` の1行で正しい
- テストは `TestRunHeadless_Completes` の1本のみ

**問題点:**
- **Cleanup が呼ばれたことを検証する仕組みがない** — コメントで「defer があるから大丈夫」と記述するのみ
- エラーパステストなし
- 反復テストなし
- コンパイルエラーを2回出した（`res.Success`、`res.ErrorDetails` — 存在しないフィールド）
- Deep thinking で124秒フリーズ → Ctrl+C → FC失敗 → 再開

---

### 4. XELYON + Sonnet 4.6

**ファイル:** `agent.go` (+7), `agent_run.go` (+8 -1), `agent_run_test.go` (+129 新規)

**アプローチ — 全対戦者中で最も設計が丁寧:**
- `Agent` struct に `onCleanup func()` フィールドを追加（nil時はゼロコスト）
- `Cleanup()` 末尾で `if a.onCleanup != nil { a.onCleanup() }` — 本物のCleanupが実行された後にフック発火
- `headlessNewAgent` パッケージ変数でテストからAgent生成を差し替え可能
- `setupHeadlessNewAgent(t, provider)` ヘルパーでテストのボイラープレートを共通化

**テスト:**
- `TestRunHeadless_CallsCleanup` — 成功パス、**実際のCleanupが実行される**
- `TestRunHeadless_CallsCleanupOnError` — エラーパス、**エラーメッセージまで検証**
- `TestRunHeadless_RepeatedInvocations` — 5回反復

**独自の強み:**
- フックがチェーン対応（既存フックを壊さない）
- テストで本物の Cleanup が走る（他は全員スタブで置換）
- 思考過程でmcp.Manager のスパイ → 断念、mock NewAgent → 断念、onCleanup → 採用、と複数案を検討

---

### 5. Claude Code (Opus 4.6)

**ファイル:** `agent.go` (+6), `agent_run.go` (+1), `headless_test.go` (+44 追記)

**アプローチ:**
- 修正は `defer agent.Cleanup()` の1行のみ（最小）
- `cleanupHook` グローバル変数を `Cleanup()` 先頭で呼び出し
- **既存の `headless_test.go` にテストを追記**（他は全員新規ファイル作成）
- **既存の `mockProvider` を再利用**（`mockErrorProvider` のみ新規追加）

**テスト:**
- `TestRunHeadless_CallsCleanup` — 成功パス
- `TestRunHeadless_CallsCleanupOnError` — エラーパス
- `TestRunHeadless_RepeatedInvocations` — 5回反復

**独自の強み:**
- コードベースの構造を理解して既存テストファイルに追記する判断
- `atomic.Int32` の Go 1.19+ 新API（`called.Add(1)` / `called.Load()`）

---

### 6. XELYON + Opus 4.6

**ファイル:** `agent.go` (+6), `agent_run.go` (+1), `agent_run_test.go` (+115 拡張)

**アプローチ:**
- 修正は Claude Code と同じ `defer agent.Cleanup()` + `cleanupHook` グローバル
- **唯一 goroutine リーク検出を実装**

**テスト:**
- `TestRunHeadless_Completes` — 基本動作確認
- `TestRunHeadless_CallsCleanup` — 成功パス
- `TestRunHeadless_CallsCleanupOnError` — エラーパス、ステータス検証付き
- `TestRunHeadless_NoLeakOnRepeatedInvocations` — 20回反復 + **goroutine数でリーク検出**

**goroutine リーク検出コード:**
```go
runtime.GC()
baseGoroutines := runtime.NumGoroutine()
for i := 0; i < 20; i++ {
    RunHeadless(...)
}
runtime.GC()
finalGoroutines := runtime.NumGoroutine()
if finalGoroutines - baseGoroutines > 5 {
    t.Errorf("possible goroutine leak")
}
```

**独自の強み:**
- 全対戦者中で唯一「Cleanupが呼ばれた結果、本当にリソースが解放されたか」をgoroutine数で検証
- 指示文の「leak-detection test」を最も忠実に解釈
- パラレルFC（4ファイル一括読み込み）による効率的なコード探索

---

## 分析

### テスト設計のアプローチ比較

| 方式 | 対戦者 | メリット | デメリット |
|---|---|---|---|
| 関数変数スタブ | GPT-5.3 (両方) | シンプル、本番コード変更少 | 実際のCleanupが実行されない |
| Agent struct フック | Sonnet 4.6 | 本物のCleanupが走る、チェーン対応 | 本番コード変更多い |
| グローバルフック | Claude Code, XELYON+Opus | シンプル、修正1行 | 全Agentで共有、並列テスト時にリスク |
| 検証なし | Gemini 3.1 Pro | 差分ゼロ | テストの価値がない |

### モデル特性

**GPT-5.3-codex:** 指示に忠実で速い。言われたことは確実にやるが、言われてないこと（goroutineリーク検出等）はやらない。コスパ最強。

**Sonnet 4.6:** 「最善の設計を考えてから手を動かす」タイプ。複数アプローチを検討して最も堅牢な案を採用。テストで本物のCleanupを走らせる判断は全対戦者中で最も丁寧。コストは高い。

**Opus 4.6:** Sonnet以上の判断力に加え、goroutineリーク検出のような「指示文の意図を深読みした実装」ができる。ただしコストが最も高い。

**Gemini 3.1 Pro:** コード修正自体は正しいが、テスト設計が雑。構造体フィールドを確認せずにテストを書いてコンパイルエラーを2回出す。Deep thinkingの不安定さも問題。

### フレームワーク差の分析

**同じGPT-5.3での比較（XELYON vs Codex CLI）:** スコア 20.5 vs 21 でほぼ互角。今回のような「追加するだけ」のタスクではフレームワーク差が出にくい。前回の --once 実験（複雑なタスク）ではXELYONが優位だった。

**同じOpus 4.6での比較（XELYON vs Claude Code）:** コスト込みスコア 20.5 vs 23.5 だが、コスト除外ではXELYONが上（テスト本数4 vs 3、goroutineリーク検出あり）。Claude Codeの高スコアはMAX課金でコスト5点満点が効いている。

### タスク特性による有利不利

今回のPR4は「既存コードを読んで、設計判断して、安全に修正する」タスクで、Claude系（Sonnet/Opus）の得意領域。タスクの種類を変えると勢力図が変わると予想される。

---

## 結論

### 推奨モデル選定

| 用途 | 推奨モデル | 理由 |
|---|---|---|
| 日常の実装タスク | GPT-5.3 | $0.32/タスク、3分、要件充足 |
| 設計判断が必要なとき | Sonnet 4.6 | テスト・設計品質が最高、コスト除外なら最強 |
| 大規模リファクタ | Opus 4.6 | goroutineリーク検出まで自発的に実装する深い判断力 |
| 避けるべき | Gemini 3.1 Pro | テストが雑、Deep thinking不安定 |

### XELYONフレームワークの評価

全モデルで要件充足・テスト全パス・`make ci-check` 通過を達成。フレームワーク自体は安定しており、モデルの差は「設計の丁寧さ」や「テストの網羅性」で現れるが、壊れた実装が出たケースはゼロ。マルチプロバイダー対応により、プロバイダー障害時の即座な切り替え（`/use openai`）が可能な点は、単一プロバイダーのClaude Code/Codex CLIに対する明確な優位性。

---

## 採点基準

各評価軸は5点満点。

- **差分の少なさ (5):** 本番コードの変更行数が少ないほど高評価
- **要件充足 (5):** 指示文の全要件（修正・テスト・制約）を満たしているか
- **テストの質 (5):** 成功/エラー/反復の各パス網羅、Cleanup検証の信頼性
- **設計の健全さ (5):** テスタビリティ設計、mock設計、将来の保守性
- **コスト効率 (5):** API費用と実行時間の総合評価