# CLAUDE.md

プロジェクト設定は **xelyon.yaml** を参照してください。

## 開発ルール（必ず守ること）

### ドキュメント更新
機能追加・変更時は**必ず** **README.md** を同時に更新（使い方、コマンド説明、バージョン履歴）。

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

### ドキュメント自動生成
設定を追加・変更した場合は必ず実行：
```bash
make gen-all  # config.yaml.example と docs/config-generated.md を更新
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
- RAG連携（XELYON Web API）は [Issue #10](https://github.com/susugadx/xelyon-cli/issues/10) に移行

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
- [x] ドキュメント更新（README.md）

**合計: 26件の問題を解決**

### Phase 7: テスト実装✅（v0.26.0-v0.27.0）
- [x] **Phase 1: 優先度高テスト**（v0.26.0）
  - internal/crypto/encryption_test.go（8 tests, 81.5% coverage）
  - internal/audit/logger_test.go（7 tests, 86.4% coverage）
  - internal/tools/delete_file_test.go（8 tests）
  - internal/tools/delete_lines_test.go（7 tests）
  - internal/tools/move_file_test.go（8 tests）
  - internal/testutil/testutil.go（テストヘルパー）

- [x] **Phase 2: 優先度中テスト**（v0.27.0）
  - internal/tools/read_file_test.go（11 tests, 100% coverage）
  - internal/tools/write_file_test.go（8 tests, 90.9% coverage）
  - internal/tools/str_replace_test.go（10 tests, 72.2% coverage）
  - internal/api/validator_test.go（23 tests, 84-100% coverage）
  - internal/api/provider_test.go（10 tests, 100% coverage）

**テスト結果**: 95 tests passing, 14.7% overall coverage（対象パッケージは95%+）

### Phase 8: リリース自動化✅（v0.28.0-v0.28.3）
- [x] **GoReleaser設定強化**（v0.28.0）
  - version: 2 対応
  - ldflags: Version, Commit, Date をバイナリに埋め込み
  - Changelog グループ化（Features/Bug Fixes/Others）
  - Homebrew Tap 自動更新設定

- [x] **GitHub Actions修正**（v0.28.1）
  - goreleaser-action: v5 → v6（version: 2 対応）

- [x] **CGO不要ビルド対応**（v0.28.2）
  - クロスコンパイル対応（Linux/macOS/Windows）

- [x] **Homebrew Tap トークン設定**（v0.28.3）
  - HOMEBREW_TAP_TOKEN 環境変数設定
  - 自動Formula更新（homebrew-tap リポジトリ）

**サポート環境**: Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)

### Phase 9: Plan Mode 統一✅（v0.29.0）
- [x] **Issue #82**: Plan Mode オンリーに統一
  - `chat()` 関数を簡略化、`RunPlanMode()` を常に呼び出し
  - `shouldEnterPlanMode()` デッドコード削除
  - 単純な Q&A も Plan Mode 経由で処理（調査フェーズで回答）
  - テスト更新（shouldEnterPlanMode テスト削除）
  - ドキュメント更新（README.md, XELYON.md, CLAUDE.md）

### Phase 10: Context Window 管理改善✅（Issue #96）
- [x] **`/tokens` コマンド**: トークン使用量の可視化
  - モデル別上限マップ（`token_limits.go`）
  - 使用量バー表示（色分け: 緑/黄/赤）
  - システムプロンプト/履歴の内訳表示
- [x] **80%/90% 警告システム**: API呼び出し前に警告表示
- [x] **自動圧縮（デフォルトON）**: 80%到達で自動圧縮
  - `auto_compress.go` 新規作成
  - 圧縮時に通知表示 + 無効化方法の案内
  - 設定: `compression.auto_compress: true`
- [x] **トークン上限エラー時の提案**: `/compress` または `/clear` を案内
- [x] ドキュメント更新（README.md, CLAUDE.md）

### Phase 11: MCP設定 + ツールトークン可視化
- [x] **MCP ON/OFF設定**: `config.yaml` に `mcp.enabled`（デフォルト: true）追加
- [x] **ヘッドレスMCP設定**: `mcp.headless`（デフォルト: false）追加
  - `--headless` モードでMCP接続をスキップ（トークン節約）
  - `mcp.headless: true` でヘッドレスでもMCP有効
- [x] **ツールトークン可視化**: `printContextSize()` を Registry ベースに書き換え
  - FC プロバイダー: Registry の JSON 定義からトークン推定
  - ツール数表示: `Tools (17)` / `Tools (17+26 MCP)`
- [x] **`/tokens` コマンド強化**: ツール定義トークンの内訳表示追加
- [x] **`EstimateTokens()` 改善**: FC プロバイダーのツール定義トークンを加算
- [x] ドキュメント更新（README.md, docs/mcp.md, CLAUDE.md）

### Phase 12.5: Completion Hooks（Stop Hook）
- [x] **`hooks.on_completion` 設定**: `config.yaml` で完了時フックコマンドを定義
  - LSP 診断後にシェルコマンドを順番に実行
  - 失敗時は AI にフィードバックして修正を続行
  - `XELYON_CHANGED_FILES` 環境変数で変更ファイルを参照可能
  - LSP 検証は1回限り（ループ防止）、フックは毎回実行（修正後の再チェック）
- [x] **`hooks.max_retry` 設定**: フック失敗時の最大リトライ回数（デフォルト: 3）
  - リトライ上限到達で警告表示し完了を許可（無限ループ防止）
  - Normal mode / Plan mode（順次・並列）の全パスで動作
  - `runCompletionHooksWithRetry()` ヘルパーで Plan mode に対応
- [x] **テスト**: 12テスト（既存6 + max_retry系6）

### Phase 12: Read-Before-Write Guard
- [x] **ReadTracker 実装**: `internal/tools/read_tracker.go`
  - `read_file`/`read_files` で読んだファイルパスを追跡
  - 書き込み系ツール実行前に「読み済みか」をチェック
  - セッションリセット時（NewAgent, /clear）にクリア
- [x] **ガード対象ツール**: `str_replace`, `write_file`（既存ファイルのみ）, `grep_replace`
- [x] **テスト**: ReadTracker 単体テスト + 各ツールのガードテスト追加

### Phase 13: XELYON.md → xelyon.yaml 移行
- [x] **ProjectConfig 構造体**: `internal/config/project.go` 新規作成
  - `Context`/`Rules`/`Hooks` を YAML で構造化管理
  - `LoadProjectConfig()`: cwd → 親方向に `xelyon.yaml` 探索
  - `ResolveHooks()`: xelyon.yaml hooks 優先 → config.yaml フォールバック
- [x] **prompt 層**: `BuildRulesBlockFromList()` 追加（`[]string` → 番号付き必須ルール）
- [x] **agent 注入統一**: 全4箇所（Interactive/Resume/Headless/Image）で統一パターン
  - `injectProjectConfig()` + `applyProjectConfig()` ヘルパー
  - Interactive/Headless/Image 間の非対称注入を解消
- [x] **`/init` 更新**: xelyon.yaml テンプレート生成
- [x] **system prompt テキスト更新**: "XELYON.md" → "project config" 参照に統一
- [x] **テスト**: 11 tests（config/project_test.go）+ 既存テスト更新
- [x] **XELYON.md フォールバック完全廃止**: IsLegacy, loadProjectConfigFromMD, ExtractSection 等を全削除、XELYON.md ファイル削除

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

## プロバイダー別システムプロンプト注入

```go
// internal/agent/system_prompt.go
func BuildProviderSystemPrompt(base, providerName string) string
func getProviderPrefix(provider string) string
```

- `NewAgent()` でシステムプロンプト構築時に `BuildProviderSystemPrompt()` を通す
- Gemini: `read_file BEFORE str_replace` ルールを冒頭に強制注入
- 他プロバイダー: プレフィックスなし（後から追加可能な map 構造）
- テスト: `internal/agent/system_prompt_test.go`（8 tests）

---

## 注意事項

- bash実行は危険コマンドをブロック（blockedCommands参照）
- 長い差分は10行で省略表示
- 同じツール呼び出し3回でループ検知・中断
- **Read-Before-Write ガード**: `read_file` せずに `str_replace`/`write_file`/`grep_replace` するとブロック（`internal/tools/read_tracker.go`）
