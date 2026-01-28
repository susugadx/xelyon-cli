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

### 🌐 6種類のLLMプロバイダー
DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq をシームレスに切り替え。
ローカルLLM（Ollama）も対応で、オフラインでも使用可能。

**OpenAI Responses API 対応**: `gpt-5.2-codex` などの Codex モデルを自動検出し、最適なAPIを選択。

### 🛠️ 24種類の組み込みツール
- **ファイル操作**: 読み書き、編集、削除、バックアップ復元
- **Git操作**: commit, checkout（status/diff/add等はbashで）
- **コード検索**: grep検索、ファイル検索、ast-grep（構造的検索）
- **開発支援**: テスト実行、フォーマット、リント、bash
- **LSP連携**: 参照検索、定義ジャンプ、ホバー情報

### 📋 確認ベースの安全設計
- 安全なツール（ファイル読み取り等）は自動実行
- 危険なツール（ファイル編集、bash等）は毎回確認
- `--auto-approve`で信頼環境向け自動承認モードも可能

### 📋 Plan Mode（オプショナル）
`/plan on` で有効化するとPlan Mode経由で処理されます。
1. **調査フェーズ**: コードベースを調査（SafetyHighツールを**バッチ処理で並列実行**）
2. **単純なQ&A**: 調査のみで回答可能な場合はそのまま終了
3. **計画生成**: 実装が必要な場合、ステップを JSON で出力
4. **承認**: ユーザーが計画を確認・承認
5. **実行**: ステップごとに失敗検知・リトライ付きで実行（**依存関係のないステップはバッチで並列実行**）

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

### ↩️ 安全なUndo機能
すべてのファイル変更は自動バックアップ。
`/undo` または「さっきの変更を戻して」で即座に復元。

### 📋 複数行ペースト対応
コードを直接ペーストして処理。Bracketed Paste Mode で改行を含むテキストも1つの入力として認識。
- **そのままペースト**: ターミナルで Cmd+V / Ctrl+V するだけ
- **` ``` ` モード**: 明示的に複数行入力を開始
- **`/paste` コマンド**: Bracketed Paste が使えない環境向け

> WSL等で問題がある場合: `XELYON_BRACKETED_PASTE=0` で無効化可能

### 🖼️ マルチモーダル対応
画像ファイルを指定してUIデザインからコード生成。
エラースクリーンショットから原因分析も可能。

### 🗺️ Repo Map（30言語対応）
Tree-sitterによる高精度なコード構造解析。
Go, TypeScript, Python, Rust, Java, C/C++, Ruby, Kotlin, Swift, C#, Scala, PHP, Elixir, Lua, CSS/SCSS, HTML, Vue, Svelte, YAML, TOML, SQL, Bash, Markdown, Dockerfile等に対応。

### 📚 Skills（自動知識ロード）
タスクに応じて関連知識を自動ロード。CI、Git、Docker等の操作をスムーズに。
カスタムスキルで独自の知識も追加可能。

### 🔌 LSP連携（IDE並みのコード理解）
Language Server Protocol (LSP) を活用してIDE並みのコード理解を実現。
- **参照検索**: シンボルのすべての参照箇所を検索
- **定義ジャンプ**: 関数や型の定義位置を特定
- **ホバー情報**: 型情報やドキュメントを取得
- **診断情報**: ファイルのエラー・警告を取得（`lsp_diagnostics`）
- **リネームプレビュー**: シンボルのリネーム変更箇所をプレビュー（`lsp_rename`）
- **削除時参照チェック**: ファイル削除前に外部参照を自動検出し警告
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

### 📝 プロジェクト設定（XELYON.md）
プロジェクトごとのルールをAIが学習。
終了時に会話から抽出したルールを自動提案。

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
/undo        # 変更取り消し
/use gemini  # プロバイダー切り替え
/think high  # Extended Thinking 有効化
/lsp status  # LSPサーバー状態確認
/lsp detect  # プロジェクト内の言語を検出
/lsp install # LSPサーバーをインストール
/exit        # 終了
```

## よく使う機能

### プロバイダー切り替え

```bash
xelyon --provider gemini --model gemini-2.0-flash-exp

# または対話中に
> /use claude
```

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

### 設定確認

```bash
> /config show    # 全設定を表示（デフォルトとの差分を ⚡ で表示）
```

## ドキュメント

| ドキュメント | 内容 |
|------------|------|
| [コマンド一覧](docs/commands.md) | 全コマンド、24ツール、使用例 |
| [プロバイダー設定](docs/providers.md) | 各プロバイダーのAPIキー取得方法 |
| [設定リファレンス](docs/config.md) | config.yaml と環境変数 |
| [MCP連携](docs/mcp.md) | 外部ツール追加 |
| [LSP連携](docs/lsp.md) | 言語サーバー連携（23言語対応） |
| [使い方詳細](docs/usage.md) | 複数行入力、画像入力、レビュー機能など |
| [Skills](docs/skills.md) | 組み込み・カスタムスキル |

## 開発に参加する

xelyon-cli の開発に参加したい方向け：

```bash
git clone https://github.com/susugadx/xelyon-cli.git
cd xelyon-cli
go build -o xelyon
./xelyon
```

> ⚠️ このリポジトリの `XELYON.md` は xelyon-cli 開発用です。

## コントリビュート

PRやIssue歓迎です！詳細は [CONTRIBUTING.md](CONTRIBUTING.md) を参照してください。

## ライセンス

MIT
