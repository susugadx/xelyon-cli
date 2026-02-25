# XELYON CLI

AI搭載のコーディングアシスタントCLI

[![CI](https://github.com/susugadx/xelyon-cli/workflows/CI/badge.svg)](https://github.com/susugadx/xelyon-cli/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 特徴

### 💬 自然言語でコーディング
コマンドを覚える必要なし。日本語で指示するだけ。
- 「このファイルのバグ直して」
- 「テスト書いて」
- 「リファクタリングして」
- 「git commitして」

**差分を見せて確認してから実行（編集・bash・gitなど）**

### 🌐 8種類のLLMプロバイダー
DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq, OpenRouter, Bedrock をシームレスに切り替え。
ローカルLLM（Ollama）も対応で、オフラインでも使用可能。

**OpenAI Responses API 対応**: `gpt-5.2-codex` などの Codex モデルを自動検出し、最適なAPIを選択。
**DeepSeek Reasoner 対応**: `reasoning_content`（思考内容）のストリーミング表示・ツール実行フローでの保持に対応。
**プロバイダー別プロンプト最適化**: Geminiなど特定モデルのルール遵守を強化するプレフィックスを自動注入。
**Gemini FC安全フォールバック**: Function Calling失敗時にターミナル状態を同期的にリセットしてからテキストモードにフォールバック。
**FC rescue JSON修復**: テキストモードで抽出されたツールJSONに生制御文字（改行・タブ等）が含まれる場合、自動修復してパース成功させる。

### 🛠️ 23種類の組み込みツール
- **ファイル操作**: 読み書き、編集、削除、バックアップ復元
- **コード検索**: grep検索、ファイル検索（結果は非テスト→テスト順・定義優先でソート、不正regexはエラー検出）
- **開発支援**: bash（git, テスト, フォーマット等すべて対応）
- **LSP連携**: シンボル検索（定義・参照・実装）

### 📋 確認ベースの安全設計
- 安全なツール（ファイル読み取り等）は自動実行
- 危険なツール（ファイル編集、bash等）は毎回確認
- `--auto-approve`で信頼環境向け自動承認モードも可能
- **Read-Before-Write ガード**: `read_file` せずに `str_replace`(old_str) / `write_file` を実行しようとするとブロック（AI の盲目的な編集を防止）。`search_code` 後は結果の行範囲に対して `str_replace`(line-range) で直接編集可能

### 📋 Plan Mode（オプショナル）
`/plan on` で有効化するとPlan Mode経由で処理されます。
1. **調査フェーズ**: コードベースを調査（SafetyHighツールを自動実行）
2. **単純なQ&A**: 調査のみで回答可能な場合はそのまま終了
3. **計画生成**: 実装が必要な場合、ステップを JSON で出力
4. **承認**: ユーザーが計画を確認・承認
5. **実行**: ステップごとに失敗検知・リトライ付きで順次実行

デフォルトは通常モード（ツール個別確認）。軽いタスクにはオーバーヘッドなく即座に応答。

### 🔄 自動リトライ機能
ツール実行が失敗した場合、自動的にリトライして成功するまで試行します。
- **デフォルト10回**のリトライ（`plan_mode.auto_retry: 10`で設定可能）
- Plan Mode と通常モード両方で有効
- リトライ中: `❌ Failed (retry 1/10)` → `🔄 Retrying...`
- 成功時: `✅ Succeeded (on retry 3)`
- 上限到達時: Selector UI で継続/中止を選択（Plan Mode のみ）

### 🔍 コードレビュー & リファクタリング
`/review` でセキュリティ・テストカバレッジをチェック。
`/refactor` で静的解析ベースのリファクタリング提案。

### 📋 複数行ペースト対応
コードを直接ペーストして処理。Bracketed Paste Mode で改行を含むテキストも1つの入力として認識。
- **そのままペースト**: ターミナルで Cmd+V / Ctrl+V するだけ
- **` ``` ` モード**: 明示的に複数行入力を開始
- **`/paste` コマンド**: Bracketed Paste が使えない環境向け

> WSL等で問題がある場合: `XELYON_BRACKETED_PASTE=0` で無効化可能

### 🖼️ マルチモーダル対応
画像ファイルを指定してUIデザインからコード生成。
エラースクリーンショットから原因分析も可能。

### 🔌 LSP連携（IDE並みのコード理解）
Language Server Protocol (LSP) を活用してIDE並みのコード理解を実現。
- **削除時参照チェック**: ファイル削除前に外部参照を自動検出し警告
- **完了検証**: AI が「完了」「done」と宣言した際、変更ファイルの LSP 診断を自動実行しエラー残存時は修正を続行
- **Completion Hooks**: LSP チェック後にユーザー定義のシェルコマンド（`go test ./...` 等）を自動実行。失敗時は AI が修正を続行（`hooks.on_completion` で設定）
- **自動検出**: プロジェクト内の言語を自動検出し、未インストールのLSPサーバーを提案
- **ワンクリックインストール**: `/lsp install <言語>` でLSPサーバーをインストール
- **23言語対応**:
  - **メイン**: Go, TypeScript/JavaScript (React/JSX), Python, Rust
  - **バックエンド**: Java, C/C++, Ruby, Kotlin, Swift, C#, Scala, PHP, Elixir, Lua
  - **フロントエンド**: CSS/SCSS, HTML, Vue, Svelte
  - **設定/スクリプト**: YAML, TOML, SQL, Bash, Markdown

### 📊 Context Window 管理
長時間の会話でもトークン上限を気にせず作業。
- **`/tokens`**: 現在のトークン使用量と上限を確認
- **自動圧縮**: 80%到達で自動的に履歴を圧縮（デフォルトON）
- **手動圧縮**: `/compress [N]` で履歴を圧縮（最新N件を保持）
- **OpenAI Compact API**: `/compress --compact` でOpenAI独自の圧縮（ユーザーメッセージ保持）
- **80%/90%警告**: 上限接近時に自動で警告表示
- **トークン上限エラー時の提案**: エラー発生時に `/compress` または `/clear` を案内

### 📈 リアルタイムトークン表示
API実測値に基づくトークン使用量とコストをリアルタイム表示。
- **ステータスバー**: プロンプト直前に `● model │ Mode │ tokens/limit │ ~$cost` を表示
- **起動時コンテキスト表示**: ツリー形式で初期コンテキストの内訳を表示
- **リクエスト完了時**: `✓ In: 1,234 + Out: 567 = 1,801 tok (~$0.002)` で使用量を表示
- **Ollama対応**: ローカル実行時はコスト表示を非表示
- **圧縮警告**: `compression.threshold_percent`（デフォルト80%）超過時は黄色で警告表示

### 📝 プロジェクト設定（xelyon.yaml）
プロジェクト固有のルール・コンテキストを構造化 YAML で管理。`/init` でテンプレート作成。

```yaml
# xelyon.yaml（プロジェクトルートに配置）
context: "Go製CLIツール。Cobraベース。"
rules:
  - "変更後は make ci-check を実行"
  - "公開関数にはコメント必須"
hooks:                    # config.yaml の hooks を上書き
  on_completion:
    - "go vet ./... && go test ./..."
  on_step_complete:
    - "echo 'Step {{step_id}}: {{step_status}}'"
  timeout: 120
  max_retry: 3
```

- **context**: AI に注入するプロジェクト説明
- **rules**: 番号付きで system prompt に注入される必須ルール
- **hooks**: 完了時・ステップ完了時フック（`config.yaml` の hooks より優先）

## インストール

```bash
# Homebrew (macOS)
brew install susugadx/tap/xelyon

# または GitHub Releases からダウンロード
# https://github.com/susugadx/xelyon-cli/releases
```

## クイックスタート

### 1. APIキーを設定

```bash
export DEEPSEEK_API_KEY="sk-..."  # または他のプロバイダー
```

### 2. 起動

```bash
xelyon

> main.goを読んで、バグがあれば修正して
```

### 3. 基本コマンド

```bash
/help        # ヘルプ
/use gemini  # プロバイダー切り替え
/think high  # Extended Thinking 有効化
/lsp status  # LSPサーバー状態確認
/lsp detect  # プロジェクト内の言語を検出
/lsp install # LSPサーバーをインストール
/project     # xelyon.yaml を対話式で編集
/exit        # 終了
```

## よく使う機能

### プロバイダー切り替え

```bash
xelyon --provider gemini --model gemini-2.5-flash

# または対話中に
> /use claude
```

### 最大出力トークン数の設定

```yaml
# ~/.xelyon/config.yaml
provider_models:
  claude:
    default_model: claude-sonnet-4-6
    max_output_tokens: 64000   # デフォルト: 64000
  gemini:
    default_model: gemini-3.1-pro-preview
    max_output_tokens: 65536   # デフォルト: 65536
  deepseek:
    default_model: deepseek-chat
    max_output_tokens: 8192    # デフォルト: 8192
```

| プロバイダー | デフォルト max_output_tokens |
|------------|---------------------------|
| claude     | 64000                     |
| bedrock    | 64000                     |
| gemini     | 65536                     |
| openai     | 16384                     |
| deepseek   | 8192                      |
| groq       | 8192                      |
| ollama     | 4096                      |
| openrouter | 64000                     |

### 確認動作のカスタマイズ

```yaml
# ~/.xelyon/config.yaml
tool_confirm:
  auto_approve_safe: true    # SafetyHigh（読み取り）自動承認（デフォルト: true）
  auto_approve_medium: true  # SafetyMedium（書き込み）自動承認（デフォルト: false）
```

| 設定 | 対象ツール | デフォルト |
|------|-----------|-----------|
| `auto_approve_safe` | read_file, list_dir, search_* 等 | true |
| `auto_approve_medium` | str_replace, write_file 等 | false |

```bash
# 全ツール自動承認（信頼できる環境向け、SafetyLow含む）
xelyon --auto-approve
```

### 差分表示設定

```yaml
# ~/.xelyon/config.yaml
diff:
  context_lines: 10    # 差分表示のコンテキスト行数（0で省略なし、デフォルト: 10）
  max_total_lines: 0   # 差分表示の最大行数（0で無制限、デフォルト: 0）
```

### MCP設定

```yaml
# ~/.xelyon/config.yaml
mcp:
  enabled: true    # MCP機能のON/OFF（デフォルト: true）
  headless: false  # ヘッドレスモードでMCPを使うか（デフォルト: false）
```

`enabled: false` にするとMCPサーバーへの接続をスキップし、トークン消費を削減できます。
`~/.xelyon/mcp.json` の設定はそのまま残るため、再度 `enabled: true` にすれば復活します。

### Completion Hooks

AI が完了を宣言すると、LSP 診断の後にここで定義したコマンドを順番に実行します。
コマンド失敗時は AI にエラー内容をフィードバックし修正を続行します（最大 `max_retry` 回）。
Makefile は不要です。普段使っているコマンドをそのまま書けます。

```yaml
# ~/.xelyon/config.yaml
hooks:
  on_completion:
    # Go
    - "go vet ./... && go test ./..."
    # Node.js / TypeScript
    # - "npm test"
    # Python
    # - "pytest"
    # Rust
    # - "cargo test"
    # Makefile がある場合
    # - "make ci-check"
  timeout: 120         # コマンドタイムアウト秒（デフォルト: 60）
  max_retry: 3         # フック失敗時の最大リトライ回数（デフォルト: 3）
```

`xelyon.yaml` にも `hooks` を定義でき、`config.yaml` より優先されます（プロジェクト固有のフック設定に便利）。
変更ファイルは `XELYON_CHANGED_FILES` 環境変数（スペース区切り）で参照できます。
Normal mode / Plan mode の両方で動作します。

### Step Complete Hooks

Plan Mode で各ステップ完了時に実行するコマンドです。
テンプレート変数 `{{step_id}}`, `{{step_description}}`, `{{step_status}}` が使えます。

```yaml
# ~/.xelyon/config.yaml
hooks:
  on_step_complete:
    # ステップごとにテスト実行
    - "go test ./..."
    # 通知（ステップ番号・状態を展開）
    # - "echo 'Step {{step_id}} ({{step_status}}): {{step_description}}'"
```

失敗時は AI にフィードバックして修正を試み、次のステップに進む前に再実行します。

### 設定管理

```bash
> /config         # 対話式設定メニュー（50+設定項目を編集可能）
> /config show    # 全設定を表示（デフォルトとの差分を ⚡ で表示）
```

対話式メニューでは20カテゴリ、50以上の設定項目を編集可能:
- Provider & Model, Compression, Tool Confirm
- Bash Safety, LSP Servers, Plan Mode, MCP など

## ドキュメント

| ドキュメント | 内容 |
|------------|------|
| [コマンド一覧](docs/commands.md) | 全コマンド、24ツール、使用例 |
| [プロバイダー設定](docs/providers.md) | 各プロバイダーのAPIキー取得方法 |
| [設定リファレンス](docs/config.md) | config.yaml と環境変数 |
| [MCP連携](docs/mcp.md) | 外部ツール追加 |
| [LSP連携](docs/lsp.md) | 言語サーバー連携（23言語対応） |
| [使い方詳細](docs/usage.md) | 複数行入力、画像入力、レビュー機能など |

## 開発に参加する

xelyon-cli の開発に参加したい方向け：

```bash
git clone https://github.com/susugadx/xelyon-cli.git
cd xelyon-cli
go build -o xelyon
./xelyon
```

> ⚠️ このリポジトリの `xelyon.yaml` は xelyon-cli 開発用です。

## コントリビュート

PRやIssue歓迎です！詳細は [CONTRIBUTING.md](CONTRIBUTING.md) を参照してください。

## ライセンス

MIT
