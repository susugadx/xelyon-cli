# XELYON CLI プロジェクト設定

> ⚠️ **注意**: このファイルは **xelyon-cli 自体の開発用** です。
> xelyon を使いたいだけなら [バイナリをダウンロード](https://github.com/susugadx/xelyon-cli/releases) してください。

## 概要
Go製のAI搭載コーディングアシスタントCLI。複数のLLMプロバイダー（DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq）に対応。

## 技術スタック
- **Go 1.22+**
- **Cobra** - CLIフレームワーク
- **Tree-sitter** - コード構造解析（Repo Map）
- **fatih/color** - ターミナル色付け

## 開発ルール

### コミット
- メッセージは日本語OK
- 具体的に書く（❌「修正」→ ✅「HTTPタイムアウト30秒を追加」）
- 機能単位で小さくコミット

### コード品質
変更後は必ず実行：
```bash
go fmt ./...
go mod tidy
go build -o xelyon
go test ./...
```

### エラーハンドリング
- すべてのI/O操作でエラーチェック必須
- HTTPクライアントには必ずTimeout設定
- context.Contextを使ってキャンセル可能に

## ディレクトリ構造

```
xelyon-cli/
├── main.go              # エントリーポイント
├── cmd/root.go          # Cobraコマンド定義
├── internal/
│   ├── agent/           # エージェント（対話ループ、検証）
│   ├── api/             # LLMプロバイダー（Provider Pattern）
│   ├── tools/           # ツール（Registry方式、38種類）
│   ├── mcp/             # MCP連携（外部ツール統合）
│   ├── lsp/             # LSP連携（言語サーバー統合）
│   ├── repomap/         # Repo Map（Tree-sitter、30言語対応）
│   ├── review/          # コードレビュー機能
│   ├── refactor/        # リファクタリング機能
│   ├── config/          # 設定管理
│   ├── history/         # セッション管理
│   ├── ui/              # スピナー、ページャー
│   ├── cache/           # プロンプトキャッシュ
│   ├── crypto/          # 暗号化
│   ├── audit/           # 監査ログ
│   └── version/         # バージョン管理
├── docs/                # ユーザー向けドキュメント
└── README.md            # ユーザー向け説明
```

## 主要インターフェース

### Provider（LLM）
```go
type Provider interface {
    Name() string
    ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error)
}
```

### OpenAI Responses API
OpenAI の `gpt-5.2-codex` などのモデルは Responses API を使用。
モデル名で自動判定し、適切な API を選択。

```yaml
# ~/.xelyon/config.yaml
openai:
  responses_api_models:
    - gpt-5.2-codex
    - gpt-5.1-codex
    - gpt-5.1-codex-max
    - gpt-5-codex
```

### Tool（ツール）
```go
type Tool interface {
    Name() string
    Description() string
    Parameters() []ToolParameter
    Run(args map[string]string) (string, *FileChange, error)
    Safety() ToolSafety
}
```

### LSP Client
```go
// internal/lsp/client.go
type Client struct {
    servers map[string]*Server      // 言語キー → サーバー
    configs map[string]ServerConfig // 言語キー → 設定
    rootURI string
}

// 主要メソッド
func (c *Client) GetServerForFile(ctx, filePath) (*Server, error)  // 遅延起動
func (c *Client) FindReferences(ctx, file, line, char) ([]Location, error)
func (c *Client) GoToDefinition(ctx, file, line, char) ([]Location, error)
func (c *Client) GetHover(ctx, file, line, char) (*HoverResult, error)
func (c *Client) GetDiagnostics(ctx, filePath) ([]Diagnostic, error)  // エラー・警告取得
func (c *Client) Status() map[string]string
```

## ツール安全レベル

| レベル | 説明 | 例 |
|-------|------|---|
| SafetyHigh | 読み取りのみ、自動承認可 | read_file, grep, list_files, lsp_* |
| SafetyMedium | 変更あり、確認推奨 | write_file, str_replace |
| SafetyLow | 危険、必ず確認 | bash, git_push, delete_file |

## Plan Mode（オプショナル）

### 概要
v0.30.0 より、Plan Mode はオプショナルになりました（Issue #83）。
デフォルトは通常モード（ツール個別確認）で、`/plan on` で Plan Mode を有効化できます。

### 切り替え
```
/plan         # 現在のモード表示
/plan on      # Plan Mode 有効化
/plan off     # 通常モードに戻る
/plan status  # ステータス表示
```

### 通常モード（デフォルト）
- `runNormalMode()` でツール実行ループ
- ツールは `tool_confirm` 設定に従って個別確認
- 軽いタスクにはオーバーヘッドなく即座に応答

### Plan Mode（`/plan on`）
1. **調査フェーズ** (`runInvestigationPhase`)
   - SafetyHighツール（read_file, search_code等）を自由に実行
   - コードベースを理解
   - **単純な Q&A**: 調査のみで回答可能な場合はそのまま終了
   - **バッチ処理・並列実行**: 複数の SafetyHigh ツールが返された場合、バッチでまとめて並列実行（Issue #84）

2. **計画生成フェーズ**
   - 実装が必要な場合、AIに計画JSONを出力させる
   - ユーザーが承認/拒否/フィードバック

3. **実行フェーズ** (`runImplementationPhase`)
   - 各ステップを順次実行
   - **バッチ処理・並列実行**: `depends_on` が同じステップはバッチでまとめて並列実行（Issue #84）
   - 失敗検知 (`containsFailure`)
   - リトライUI (`promptFailureAction`: retry/comment/skip/abort)

### バッチ処理・並列実行設定
```yaml
# ~/.xelyon/config.yaml
plan_mode:
  max_parallel_steps: 3  # バッチ並列実行数（デフォルト: 3）
```

### 関連ファイル
- `internal/agent/plan_mode.go` - Plan Mode 実装
- `internal/agent/agent_chat.go` - `chat()` が `PlanModeEnabled` に応じて分岐
- `internal/agent/plan.go` - 計画構造体
- `internal/agent/agent_commands.go` - `/plan` コマンド実装

## Context Window 管理

### 概要
長時間の会話でトークン上限に達した時の対策を自動化。

### 機能
1. **`/tokens` コマンド**: 現在のトークン使用量を表示
2. **自動圧縮（デフォルトON）**: 80%到達で自動的に履歴を圧縮
3. **手動圧縮**: `/compress [N]` で履歴を圧縮（最新N件を保持）
4. **OpenAI Compact API**: `/compress --compact` でOpenAI独自の圧縮
5. **80%/90%警告**: 上限接近時に警告表示
6. **トークン上限エラー時の提案**: `/compress` または `/clear` を案内

### OpenAI Compact API
OpenAI Responses API の `/responses/compact` エンドポイントを使用した圧縮機能。

**特徴**:
- ユーザーメッセージは**そのまま保持（verbatim）**
- アシスタント応答は暗号化された圧縮データに置換
- ZDR（Zero Data Retention）対応

**使用条件**:
- OpenAI プロバイダーかつ Responses API 対応モデル（gpt-5.2-codex 等）
- 自動圧縮時に `prefer_compact_api: true` で優先使用

### 設定
```yaml
# ~/.xelyon/config.yaml
compression:
  auto_compress: true        # 自動圧縮を有効化（デフォルト: true）
  threshold_percent: 80      # 自動圧縮の閾値（デフォルト: 80%）
  threshold_tokens: 0        # トークン数ベースの閾値（0 = 使用率ベース）
  keep_recent: 10            # 圧縮時に保持する最新メッセージ数
  prefer_compact_api: true   # OpenAI Compact API を優先（デフォルト: true）
```

### 関連ファイル
- `internal/agent/token_limits.go` - モデル別トークン上限
- `internal/agent/auto_compress.go` - 自動圧縮ロジック
- `internal/agent/compress.go` - 圧縮処理（LLMサマリー）
- `internal/agent/compress_compact.go` - OpenAI Compact API 圧縮
- `internal/api/openai_compact.go` - Compact API クライアント
- `internal/agent/token_guard.go` - トークン上限エラー検出

## Extended Thinking

### 概要
複雑なタスクでより深い推論を行うための Extended Thinking（推論モード）に対応。
`/think` コマンドで有効化。

### 対応プロバイダー
| プロバイダー | 対応 | 実装 |
|-------------|------|------|
| Claude | ✅ | `thinking.type` + `thinking.budget_tokens` |
| OpenAI | ✅ | `reasoning_effort` (Chat) / `reasoning.effort` (Responses) |
| Gemini | ✅ | `generationConfig.thinkingConfig.thinkingBudget` |
| DeepSeek | ✅ | モデル自動切替 → `deepseek-reasoner` |
| Groq | ❌ | 警告表示（非対応） |
| Ollama | ⚠️ | モデル依存（R1/QwQ推奨） |

### 対応モデル
- **Claude**: Sonnet 4 以降
- **OpenAI**: gpt-5.2 系
- **Gemini**: 2.5 Pro 系（Flash は非対応）
- **DeepSeek**: 自動で reasoner モデルに切り替わります

### レベル別パラメータ
| Level | Claude (budget_tokens) | OpenAI (effort) | Gemini (budget) |
|-------|------------------------|-----------------|-----------------|
| low | 5,000 | low | 5,000 |
| medium | 10,000 | medium | 10,000 |
| high | 20,000 | high | 20,000 |
| xhigh | 40,000 | high | 40,000 |

### 設定
```yaml
# ~/.xelyon/config.yaml
thinking:
  enabled: false    # デフォルト OFF
  level: medium     # low/medium/high/xhigh
```

### コマンド
```
/think              # 現在の状態表示
/think on           # 有効化（現在のレベルで）
/think off          # 無効化
/think low          # 低レベルで有効化
/think medium       # 中レベルで有効化（デフォルト）
/think high         # 高レベルで有効化
/think xhigh        # 最高レベルで有効化
```

### 関連ファイル
- `internal/config/config_types.go` - ThinkingConfig 構造体
- `internal/agent/agent_commands.go` - `/think` コマンド実装
- `internal/api/claude.go` - ClaudeThinkingConfig
- `internal/api/openai.go` - ReasoningConfig
- `internal/api/gemini_types.go` - GeminiThinkingConfig
- `internal/api/deepseek.go` - モデル切替ロジック

## SystemPromptルール

AIの振る舞いを定義（`internal/agent/prompts.go`）：

1. ツール呼び出しは1回につき1つ
2. 実行前に必ず説明を入れる
3. ファイル編集前に必ず読む
4. bash実行には細心の注意
5. エラー時は3回まで自動リトライ
6. ユーザーの確認なしに危険な操作をしない

## LSP連携

### 概要
Language Server Protocol (LSP) を活用して、IDE並みのコード理解を実現。
AIがコードの参照、定義、型情報を正確に取得できる。

### アーキテクチャ
```
internal/lsp/
├── protocol.go    # JSON-RPC 2.0 & LSP型定義（Diagnostic含む）
├── util.go        # URI変換、言語検出ヘルパー
├── server.go      # 単一LSPサーバープロセス管理（diagnostics通知処理）
├── client.go      # 複数サーバー管理（遅延起動）
├── detect.go      # プロジェクト言語自動検出
├── install.go     # LSPサーバーインストール
└── *_test.go      # ユニットテスト

internal/tools/
├── lsp_tools.go   # LSPツール実装（references, definition, hover, diagnostics）
└── lsp_safety.go  # 削除時参照チェック（シンボル抽出、外部参照検出）
```

### 遅延起動
サーバーは初回使用時に起動（Agent初期化時には起動しない）。
- `GetServerForFile()` 呼び出し時に言語を検出
- 該当言語のサーバーがなければ起動
- 起動済みサーバーは再利用

### 設定
```yaml
# ~/.xelyon/config.yaml
lsp:
  enabled: true  # LSP連携の有効/無効
  servers:
    go:
      command: gopls
    typescript:
      command: vtsls
      args: ["--stdio"]
    python:
      command: pyright-langserver
      args: ["--stdio"]
    rust:
      command: rust-analyzer
      disabled: true  # 個別サーバーの無効化
```

### LSPツール
| ツール | 説明 | パラメータ |
|-------|------|----------|
| `lsp_references` | シンボルの参照箇所検索 | path, line, character |
| `lsp_definition` | 定義位置へジャンプ | path, line, character |
| `lsp_hover` | 型情報・ドキュメント取得 | path, line, character |
| `lsp_diagnostics` | ファイルのエラー・警告取得 | path |
| `lsp_rename` | リネーム変更箇所をプレビュー | path, line, character, new_name |

### 削除時参照チェック
ファイル削除（`delete_file`）時にLSPが有効なら、自動的に外部参照をチェック。

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🗑️  Delete File / ファイル削除
📂 Path / パス: internal/api/handler.go
📏 Size / サイズ: 1234 bytes (45 lines)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️  LSP Warning: This file contains 3 external references!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   HandleUser (2 references):
      - main.go:45
      - routes/api.go:123
   UserHandler (1 references):
      - main.go:67
```

シンボル抽出は正規表現ベース（Go, TypeScript, Python, Rust対応）。

### コマンド
```
/lsp                      # LSPステータス表示（未インストールサーバーの提案付き）
/lsp status               # 同上
/lsp detect               # プロジェクト内の言語を検出して表示
/lsp install <言語>       # 指定言語のLSPサーバーをインストール
/lsp install all          # 未インストールの全サーバーをインストール
```

### 言語自動検出
プロジェクト内のファイル拡張子からサポート言語を自動検出。
- 検出対象: `.go`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.rs`
- 除外ディレクトリ: `.git`, `node_modules`, `vendor`, `__pycache__`, `dist`, `build`, `target`

### LSPサーバーインストール

#### メイン言語（4言語）
| 言語 | パッケージ | インストールコマンド |
|------|----------|-------------------|
| Go | gopls | `go install golang.org/x/tools/gopls@latest` |
| TypeScript/JavaScript | vtsls | `npm i -g @vtsls/language-server typescript` |
| Python | pyright | `pip install pyright` または `npm i -g pyright` |
| Rust | rust-analyzer | `rustup component add rust-analyzer` |

#### バックエンド言語（11言語）
| 言語 | パッケージ | インストールコマンド |
|------|----------|-------------------|
| Java | jdtls | `brew install jdtls` |
| C/C++ | clangd | `brew install llvm` または `apt install clangd` |
| Ruby | solargraph | `gem install solargraph` |
| Kotlin | kotlin-language-server | `brew install kotlin-language-server` |
| Swift | sourcekit-lsp | Xcode/Swift toolchain に含まれる |
| C# | csharp-ls | `dotnet tool install --global csharp-ls` |
| Scala | metals | `brew install coursier/formulas/coursier && cs install metals` |
| PHP | intelephense | `npm i -g intelephense` |
| Elixir | elixir-ls | `brew install elixir-ls` |
| Lua | lua-language-server | `brew install lua-language-server` |

#### フロントエンド言語（4言語）
| 言語 | パッケージ | インストールコマンド |
|------|----------|-------------------|
| CSS/SCSS | vscode-css-language-server | `npm i -g vscode-langservers-extracted` |
| HTML | vscode-html-language-server | `npm i -g vscode-langservers-extracted` |
| Vue | @vue/language-server | `npm i -g @vue/language-server` |
| Svelte | svelte-language-server | `npm i -g svelte-language-server` |

#### 設定/スクリプト言語（5言語）
| 言語 | パッケージ | インストールコマンド |
|------|----------|-------------------|
| YAML | yaml-language-server | `npm i -g yaml-language-server` |
| TOML | taplo | `cargo install taplo-cli --locked` |
| SQL | sqls | `go install github.com/lighttiger2505/sqls@latest` |
| Bash | bash-language-server | `npm i -g bash-language-server` |
| Markdown | marksman | `brew install marksman` |

### エラーハンドリング
- **サーバー未インストール**: フォールバックメッセージを返す + インストール提案
- **タイムアウト**: 30秒でタイムアウト
- **サーバークラッシュ**: 次回使用時に再起動
- **未対応言語**: 言語検出できない場合はエラー

## プロンプトキャッシュ

| プロバイダー | 方式 |
|-------------|------|
| Claude | `cache_control` で明示的キャッシュ |
| OpenAI/DeepSeek | API側で自動キャッシュ |
| Gemini | Implicit Caching（自動） |

## 新機能追加時のチェックリスト

1. [ ] `internal/` に実装
2. [ ] ツールなら `internal/tools/` に追加、Registry登録
3. [ ] コマンドなら `internal/agent/agent_commands.go` に追加
4. [ ] テスト追加
5. [ ] `go fmt && go test ./...` 通過
6. [ ] ドキュメント更新
   - ツール追加 → `docs/commands.md`
   - コマンド追加 → `docs/commands.md`
   - 設定追加 → `docs/config.md`
   - 大きな新機能 → README.md の特徴セクション

## ビルド＆テスト

```bash
# ビルド
go build -o xelyon

# テスト
go test ./...

# カバレッジ
go test -cover ./...

# リント
golangci-lint run
```

## ドキュメント自動生成

設定を追加・変更した場合は、以下のコマンドでドキュメントを更新してください。

```bash
# 設定例とドキュメントを全て自動生成
make gen-all

# 個別に実行
make gen-config  # config.yaml.example を更新
make gen-docs    # docs/config.md の設定例を更新
```

### 仕組み

| コマンド | 入力 | 出力 |
|---------|------|------|
| `make gen-config` | `DefaultConfig()` | `config.yaml.example` |
| `make gen-docs` | `config.yaml.example` | `docs/config.md` の設定例セクション |

`docs/config.md` はマーカーベースの自動更新:
- `<!-- CONFIG-EXAMPLE-START -->` ... `<!-- CONFIG-EXAMPLE-END -->` 間が自動更新
- 詳細説明部分は手動のまま保持

### 設定追加時の手順

1. `internal/config/config_types.go` にフィールド追加（コメント付き）
2. `internal/config/config.go` の `DefaultConfig()` にデフォルト値追加
3. `make gen-all` 実行
4. `docs/config.md` の設定例が自動更新される
5. 詳細説明セクションに新しい設定の説明を追加（手動）

## リリース

```bash
# バージョン更新
# internal/version/version.go の Version を変更

# タグ作成＆プッシュ（GitHub Actionsが自動リリース）
git tag v0.X.0
git push origin v0.X.0
```
