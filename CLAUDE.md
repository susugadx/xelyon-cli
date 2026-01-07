# CLAUDE.md

このプロジェクトの詳細設定は **XELYON.md** を参照してください。

## 開発ルール（必ず守ること）

### ドキュメント更新
機能追加・変更時は**必ず**以下を同時に更新：
- **README.md**: 使い方、コマンド説明、バージョン履歴
- **XELYON.md**: アーキテクチャ、内部設計、SystemPromptルール

ドキュメント更新なしのコミットは禁止。

### コミットルール
- メッセージは日本語OK
- 具体的に書く（❌「修正」→ ✅「HTTPタイムアウト30秒を追加」）
- 機能単位で小さくコミット

### コード品質
変更後は必ず実行：
```bash
go fmt ./...
go mod tidy
go build -o xelyon
```

可能なら：
```bash
go test ./...
```

### エラーハンドリング
- すべてのI/O操作でエラーチェック必須
- HTTPクライアントには必ずTimeout設定
- context.Contextを使ってキャンセル可能に

---

## 現在のタスクリスト（優先順）

### Phase 1: バグ修正（即座に）✅
- [x] go mod tidy → go.sum生成・コミット（v0.10.1で完了）
- [x] internal/api/deepseek.go: http.Client に Timeout: 30 * time.Second 追加（v0.10.0で完了）
- [x] internal/api/xelyon.go: 同様にTimeout追加（v0.10.0で完了）
- [x] internal/api/serper.go: 同様にTimeout追加（v0.9.1で完了）

### Phase 2: アーキテクチャ改善
- [ ] internal/api/provider.go 新規作成（Provider interface）
- [ ] internal/api/client.go 新規作成（Client struct + タイムアウト管理）
- [ ] internal/api/deepseek.go を DeepSeekProvider struct に改修
- [ ] 環境変数 XELYON_PROVIDER で LLM切り替え可能に
- [ ] internal/tools/registry.go 新規作成（Tool interface + Registry）
- [ ] internal/tools/builtin.go 新規作成（既存ツールをRegistry登録）
- [ ] tools.go の switch文を Registry.Execute() に置き換え

### Phase 3: 品質向上✅
- [x] internal/agent/verify.go 新規作成（変更後の自動検証）（v0.11.0で完了）
- [x] Goファイル変更後に gofmt + go test を提案（v0.11.0で完了）
- [x] テスト失敗時にrollback提案（v0.11.0で完了）

### Phase 4: OSS公開準備✅
- [x] LICENSE ファイル追加（MIT）（v0.10.1で完了）
- [x] CONTRIBUTING.md 追加（v0.10.1で完了）
- [x] .github/workflows/ci.yml 追加（build + test）（v0.10.1で完了）
- [x] .goreleaser.yml 追加（バイナリ配布）（v0.10.1で完了）
- [x] .gitignore 追加（v0.10.1で完了）

### Phase 5: 差別化機能✅
- [x] MCP対応（ToolRegistryに外部ツール登録）（v0.12.0で完了）
- [x] Repo Map実装（go-tree-sitter）（v0.13.0で完了）
- [ ] RAG連携（XELYON Web API）- Web版リリース後

---

## アーキテクチャ概要

```
xelyon-cli/
├── cmd/root.go          # CLIエントリーポイント
├── internal/
│   ├── agent/           # エージェントロジック（対話ループ）
│   ├── api/             # LLMプロバイダー（DeepSeek/Claude/OpenAI）
│   ├── tools/           # ツール実行（Registry方式）
│   ├── config/          # 設定管理
│   ├── history/         # セッション管理
│   ├── ui/              # スピナー、ページャー
│   └── version/         # バージョン管理
├── CLAUDE.md            # このファイル（Claude Code用）
├── XELYON.md            # 詳細設計ドキュメント
└── README.md            # ユーザー向けドキュメント
```

---

## Provider インターフェース設計（実装時参考）

```go
// internal/api/provider.go
type Provider interface {
    Name() string
    ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error)
}

// internal/api/client.go
type Client struct {
    Provider Provider
    Timeout  time.Duration  // デフォルト30秒
}
```

## ToolRegistry 設計（実装時参考）

```go
// internal/tools/registry.go
type Tool interface {
    Name() string
    Run(args map[string]string) (output string, change *FileChange)
}

type Registry struct {
    tools map[string]Tool
}

func (r *Registry) Register(t Tool)
func (r *Registry) Execute(tc *ToolCall) (string, *FileChange)
```

---

## 注意事項

- bash実行は危険コマンドをブロック（blockedCommands参照）
- ファイル編集時は必ず.bakバックアップ作成
- 長い差分は10行で省略表示
- 同じツール呼び出し3回でループ検知・中断
