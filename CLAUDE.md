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

### Phase 2: アーキテクチャ改善✅
- [x] internal/api/provider.go 新規作成（Provider interface）（v0.15.0で完了）
- [x] 全プロバイダーをProvider interfaceに統一（v0.15.0で完了）
- [x] 環境変数 XELYON_PROVIDER で LLM切り替え可能に（v0.15.0で完了）
- [x] internal/tools/registry.go 新規作成（Tool interface + Registry）（完了）
- [x] MCP動的ツール登録に対応（v0.12.0で完了）

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

### Phase 6: セキュリティ・品質監査✅（v0.24.0）
- [x] **Phase 1 CRITICAL**: 5件のセキュリティ脆弱性を修正
  - コマンドインジェクション防止
  - パストラバーサル防止
  - APIキー露出防止
  - MCP任意コード実行防止
  - グレースフルシャットダウン実装
- [x] **Phase 2 HIGH**: 12件の信頼性・パフォーマンス問題を修正
  - HTTPクライアント再利用（20MB+メモリ節約）
  - ファイルパーミッション修正（0600）
  - レート制限ハンドリング
  - 競合状態修正（Spinner, ToolRegistry）
  - APIエンドポイント設定可能化
- [x] **Phase 3 MEDIUM**: 6件のエラーハンドリング・効率化を改善
  - エラー検出強化（JSON, I/O, セッション保存）
  - RepoMap文字列連結最適化（O(n²)→O(n)）
- [x] **Phase 4 LOW**: 3件の保守性向上
  - Contextタイムアウト設定（3分）
  - MCPバージョン一元化

### Phase 7: LOW優先度機能追加✅（v0.25.0）
- [x] **監査ログ機能**: ツール実行履歴をJSONL形式で記録
  - `XELYON_AUDIT_LOG=1`で有効化
  - 機密情報（password, token, api_key）は自動的に[REDACTED]化
  - 保存場所: `~/.xelyon/audit/audit_YYYYMMDD.jsonl`
- [x] **セッション履歴の暗号化**: AES-256-GCM暗号化
  - `XELYON_ENCRYPT_HISTORY=1`で有効化
  - PBKDF2鍵導出（100,000回イテレーション）
  - 暗号化キー: `~/.xelyon/.session_key`（0600パーミッション）
- [x] **APIレスポンス検証**: 不正なレスポンス構造を検出
  - ストリーミングレスポンス検証（choices配列、delta/messageフィールド）
  - エラーレスポンスの安全なパース
  - DeepSeekプロバイダーで実装
- [x] **非推奨関数レビュー**: 破壊的変更を避けるため保持
- [x] ドキュメント更新（README.md, XELYON.md）

**合計: 26件の問題を解決**

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
