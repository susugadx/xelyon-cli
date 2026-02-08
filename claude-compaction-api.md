# Claude Compaction API 実装計画

## 概要

Claude の Compaction API をXELYON CLI に統合する。
OpenAI の Compact API とは根本的にアーキテクチャが異なる。

| | OpenAI Compact | Claude Compaction |
|---|---|---|
| エンドポイント | 別の `/responses/compact` | 通常の `/v1/messages` |
| トリガー | クライアントが明示的に呼ぶ | サーバーが自動判定 |
| 結果 | 暗号化された `compacted` アイテム | テキスト要約の `compaction` ブロック |
| 渡し方 | `input[]` に `type: "compacted"` | `messages[]` に `compaction` ブロックを含めるだけ |
| ヘッダー | 不要 | `anthropic-beta: compact-2026-01-12` |
| モデル | OpenAI 全般 | Claude Opus 4.6 のみ |

**つまり Claude 版は「リクエストにパラメータ追加 + レスポンスの新ブロック型を処理」だけ。**
別 API を叩く必要がなく、既存のチャットフローに透過的に組み込める。

---

## API 仕様（公式ドキュメントより）

### リクエスト
```json
{
  "model": "claude-opus-4-6",
  "max_tokens": 4096,
  "messages": [...],
  "context_management": {
    "edits": [
      {
        "type": "compact_20260112",
        "trigger": {
          "type": "input_tokens",
          "value": 150000
        }
      }
    ]
  }
}
```

必須ヘッダー:
- `anthropic-beta: compact-2026-01-12`

### レスポンス（compaction 発火時）
```json
{
  "content": [
    {
      "type": "compaction",
      "content": "Summary of the conversation: ..."
    },
    {
      "type": "text",
      "text": "Based on our conversation so far..."
    }
  ]
}
```

### 後続リクエスト
`compaction` ブロックを含む assistant メッセージをそのまま `messages[]` に渡す。
API は compaction ブロックより前のメッセージを自動ドロップする。

---

## 実装方針

### アプローチ: 透過的統合

Claude の Compaction は「チャット中に勝手に圧縮してくれる」設計。
XELYON 側は:
1. リクエストに `context_management` を付ける
2. レスポンスの `compaction` ブロックを捨てずに保持する
3. 後続リクエストでそのまま渡す

**`CompactCapable` インターフェースは使わない。** OpenAI とアーキテクチャが違いすぎる。

---

## 変更箇所

### 変更1: Request 構造体に `context_management` フィールド追加

**`internal/api/providers/claude/claude.go`**

```go
// ContextManagement は Compaction API の設定
type ContextManagement struct {
    Edits []ContextEdit `json:"edits"`
}

// ContextEdit は context_management.edits の要素
type ContextEdit struct {
    Type    string         `json:"type"`              // "compact_20260112"
    Trigger *CompactTrigger `json:"trigger,omitempty"`
}

// CompactTrigger は compaction のトリガー条件
type CompactTrigger struct {
    Type  string `json:"type"`  // "input_tokens"
    Value int    `json:"value"` // トークン数（最低 50000）
}
```

**Request と MultimodalRequest に追加:**
```go
type Request struct {
    Model             string             `json:"model"`
    Messages          []AnthropicMessage `json:"messages"`
    System            interface{}        `json:"system,omitempty"`
    MaxTokens         int                `json:"max_tokens"`
    Stream            bool               `json:"stream"`
    Thinking          *ThinkingConfig    `json:"thinking,omitempty"`
    Tools             []ClaudeTool       `json:"tools,omitempty"`
    ContextManagement *ContextManagement `json:"context_management,omitempty"` // NEW
}

type MultimodalRequest struct {
    Model             string             `json:"model"`
    Messages          []interface{}      `json:"messages"`
    System            interface{}        `json:"system,omitempty"`
    MaxTokens         int                `json:"max_tokens"`
    Stream            bool               `json:"stream"`
    Thinking          *ThinkingConfig    `json:"thinking,omitempty"`
    Tools             []ClaudeTool       `json:"tools,omitempty"`
    ContextManagement *ContextManagement `json:"context_management,omitempty"` // NEW
}
```

### 変更2: `anthropic-beta` ヘッダーに `compact-2026-01-12` を追加

**`internal/api/providers/claude/claude.go`** の `executeRequest`:

```go
// Anthropic Beta - compaction beta を追加
betaHeaders := make([]string, 0)
if len(pCfg.AnthropicBeta) > 0 {
    betaHeaders = append(betaHeaders, pCfg.AnthropicBeta...)
}
// Compaction が有効な場合は beta ヘッダーを追加
if p.compactionEnabled {
    betaHeaders = append(betaHeaders, "compact-2026-01-12")
}
if len(betaHeaders) > 0 {
    req.Header.Set("anthropic-beta", strings.Join(betaHeaders, ","))
}
```

### 変更3: Provider に compaction 設定を追加

**`internal/api/providers/claude/claude.go`** の Provider 構造体:

```go
type Provider struct {
    api.BaseProvider
    mcpTools          []api.ToolDefinition
    usageCallback     api.UsageCallback
    compactionEnabled bool  // NEW: Compaction API を使用するか
    compactionTrigger int   // NEW: トリガー閾値（トークン数、デフォルト 150000）
}
```

### 変更4: ChatWithTools で `context_management` を設定

**`internal/api/providers/claude/claude.go`** の `ChatWithTools`:

```go
reqBody := Request{
    Model:     model,
    Messages:  messages,
    System:    api.BuildSystemField(systemPrompt),
    MaxTokens: api.GetMaxOutputTokens("claude", model),
    Stream:    true,
}

// Compaction API（Opus 4.6 のみ）
if p.compactionEnabled && isCompactionSupported(model) {
    trigger := p.compactionTrigger
    if trigger == 0 {
        trigger = 150000
    }
    reqBody.ContextManagement = &ContextManagement{
        Edits: []ContextEdit{
            {
                Type: "compact_20260112",
                Trigger: &CompactTrigger{
                    Type:  "input_tokens",
                    Value: trigger,
                },
            },
        },
    }
}
```

ヘルパー関数:
```go
// isCompactionSupported は Compaction API 対応モデルか判定
// 現時点では Opus 4.6 のみ
func isCompactionSupported(model string) bool {
    return strings.Contains(model, "opus-4-6") || strings.Contains(model, "opus-4-5")
}
```

### 変更5: ストリーミングで `compaction` ブロックを処理

**`internal/api/providers/claude/claude.go`** の `handleStreamingResponse`:

`content_block_start` の処理に `compaction` タイプを追加:

```go
case "content_block_start":
    if event.ContentBlock != nil {
        switch event.ContentBlock.Type {
        case "tool_use":
            toolUses[event.Index] = &toolUseAccumulator{
                ID:   event.ContentBlock.ID,
                Name: event.ContentBlock.Name,
            }
        case "compaction":
            // compaction ブロックの開始を記録
            // content_block_delta の text_delta でサマリーテキストが送られてくる
            // → compactionAccumulator に蓄積
            compactionBlocks[event.Index] = &strings.Builder{}
        }
    }
    return "", false, nil
```

`content_block_delta` に compaction 蓄積を追加:
```go
case "content_block_delta":
    if event.Delta == nil {
        return "", false, nil
    }
    if event.Delta.Type == "text_delta" {
        // compaction ブロックの場合は蓄積（表示しない）
        if acc, ok := compactionBlocks[event.Index]; ok {
            acc.WriteString(event.Delta.Text)
            return "", false, nil
        }
        return event.Delta.Text, false, nil
    }
    // ...
```

`content_block_stop` で compaction を出力:
```go
case "content_block_stop":
    // compaction ブロックの完了
    if acc, ok := compactionBlocks[event.Index]; ok {
        // compaction サマリーを特別な形式で出力
        // agent 側で認識できるようにマーカーを付ける
        compactionText := acc.String()
        compactionOutput.WriteString(compactionText)
        delete(compactionBlocks, event.Index)
    }
    // tool_use ブロックの完了（既存）
    // ...
```

最終的なレスポンス構築:
```go
// compaction が発生した場合、レスポンスの先頭にマーカーを付加
if compactionOutput.Len() > 0 {
    // agent_tool_executor がこのマーカーを認識して
    // History に compaction 情報を保存する
    result := "[COMPACTION]\n" + compactionOutput.String() + "\n[/COMPACTION]\n" + content
    if toolCallsOutput.Len() > 0 {
        result += toolCallsOutput.String()
    }
    return result, nil
}
```

### 変更6: AnthropicMessage に compaction ブロック対応

**`internal/api/providers/claude/convert.go`**:

`ConvertToAnthropicMessages` で compaction ブロックを認識:

```go
// compaction ブロックを含む assistant メッセージの場合、
// content を構造化して渡す
if strings.Contains(msg.Content, "[COMPACTION]") {
    // compaction サマリー部分を抽出
    compactionContent, textContent := extractCompaction(msg.Content)
    parts := []interface{}{}
    if compactionContent != "" {
        parts = append(parts, map[string]string{
            "type":    "compaction",
            "content": compactionContent,
        })
    }
    if textContent != "" {
        parts = append(parts, map[string]string{
            "type": "text",
            "text": textContent,
        })
    }
    // 構造化された content で送信
    messages = append(messages, AnthropicMessage{
        Role:    "assistant",
        Content: parts, // interface{} なので構造体も可
    })
}
```

ただし現在の `AnthropicMessage.Content` は `interface{}` なので構造化データも送れる。

### 変更7: config に compaction 設定追加

**`internal/config/config_types.go`** の `CompressionConfig`:

```go
type CompressionConfig struct {
    AutoCompress       bool `yaml:"auto_compress"`
    ThresholdTokens    int  `yaml:"threshold_tokens"`
    ThresholdPercent   int  `yaml:"threshold_percent"`
    KeepRecent         int  `yaml:"keep_recent"`
    PreferCompactAPI   bool `yaml:"prefer_compact_api"`
    ClaudeCompaction   bool `yaml:"claude_compaction"`    // NEW: Claude Compaction API 有効化
    CompactionTrigger  int  `yaml:"compaction_trigger"`   // NEW: トリガー閾値（デフォルト 150000）
}
```

**`internal/config/config.go`** のデフォルト値:
```go
Compression: CompressionConfig{
    AutoCompress:      true,
    ClaudeCompaction:  true,  // デフォルト ON（Opus 4.6 使用時のみ発動）
    CompactionTrigger: 150000,
    // ...
}
```

### 変更8: Provider 初期化時に config を反映

**`internal/api/providers/claude/claude.go`** の `New` または初期化箇所:

```go
func New(apiKey string) *Provider {
    cfg := config.GetGlobalConfig()
    return &Provider{
        BaseProvider: api.BaseProvider{
            APIKey:     apiKey,
            APIURL:     "https://api.anthropic.com/v1/messages",
            HTTPClient: &http.Client{Timeout: config.DefaultHTTPTimeout},
        },
        compactionEnabled: cfg.Compression.ClaudeCompaction,
        compactionTrigger: cfg.Compression.CompactionTrigger,
    }
}
```

### 変更9: Agent 側での compaction ブロック処理

**`internal/agent/agent_chat.go`** または新ファイル:

agent の通常レスポンス処理（`handleNormalResponse` 等）で `[COMPACTION]` マーカーを検出:

```go
// compaction が含まれていた場合
if strings.Contains(response, "[COMPACTION]") {
    compactionSummary, cleanResponse := parseCompactionResponse(response)
    if compactionSummary != "" {
        cyan.Println("📦 Context compacted by Claude")
        // History にはマーカー込みで保存（次回リクエストで API に渡すため）
    }
    response = cleanResponse // 表示用はクリーンなテキスト
}
```

---

## 変更対象ファイル一覧

| ファイル | 変更内容 |
|---|---|
| `internal/api/providers/claude/claude.go` | `ContextManagement` 等の型追加、Request/MultimodalRequest にフィールド追加、Provider に compaction フィールド追加、ChatWithTools で設定、ストリーミングで compaction ブロック処理、beta ヘッダー追加 |
| `internal/api/providers/claude/convert.go` | compaction ブロックを含む assistant メッセージの変換対応 |
| `internal/config/config_types.go` | `CompressionConfig` に `ClaudeCompaction`, `CompactionTrigger` 追加 |
| `internal/config/config.go` | デフォルト値設定 |

---

## 注意事項

### 1. Opus 4.6 限定
Compaction API は現時点で Opus 4.6 のみ対応。`isCompactionSupported()` でモデルチェック必須。

### 2. 既存の auto_compress との共存
- Claude + Opus 4.6: サーバーサイド Compaction（API が自動管理）
- Claude + 他モデル: 既存の LLM サマリー圧縮
- OpenAI: 既存の Compact API
- その他: 既存の LLM サマリー圧縮

`maybeAutoCompress` は Compaction が有効な場合は**スキップ**すべき。
サーバーが自動管理するので、クライアント側で二重に圧縮する必要がない。

```go
// auto_compress.go に追加
// Claude Compaction が有効な場合はスキップ
if claudeProvider, ok := a.CurrentProvider.(*claude.Provider); ok {
    if claudeProvider.IsCompactionEnabled() {
        return false
    }
}
```

### 3. History の永続化
compaction ブロックを含む History をセッションに保存する際、
`[COMPACTION]...[/COMPACTION]` マーカーがそのまま保存される。
セッション復元時に次の API コールで正しく渡される。

### 4. `anthropic-beta` ヘッダーの管理
config の `anthropic_beta` と compaction の beta が衝突しないよう、
マージして送信する（カンマ区切り）。

---

## テスト

### ビルド確認
```bash
go test ./internal/api/providers/claude/...
go test ./internal/agent/...
go build -o xelyon
```

### 手動検証
```bash
# Compaction 有効で長い会話をシミュレート
xelyon --provider claude --model claude-opus-4-6 "大きなプロジェクトを分析して"
# → 長いツールループ後に compaction が自動発火することを確認

# Compaction 無効
xelyon config set compression.claude_compaction false
xelyon --provider claude --model claude-opus-4-6 "hello"
# → context_management が送信されないことを確認

# 非 Opus モデルでは発動しないことを確認
xelyon --provider claude --model claude-sonnet-4-20250514 "hello"
```

---

## 実装順序

1. 型定義追加（ContextManagement, ContextEdit, CompactTrigger）
2. config 追加（ClaudeCompaction, CompactionTrigger）
3. Request/MultimodalRequest にフィールド追加
4. Provider に compaction フィールド追加 + 初期化
5. ChatWithTools で context_management 設定
6. executeRequest で beta ヘッダー追加
7. handleStreamingResponse で compaction ブロック処理
8. convert.go で compaction ブロック送信対応
9. auto_compress.go で Claude Compaction 時スキップ
