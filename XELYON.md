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

## リリース

```bash
# バージョン更新
# internal/version/version.go の Version を変更

# タグ作成＆プッシュ（GitHub Actionsが自動リリース）
git tag v0.X.0
git push origin v0.X.0
```
