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

### 🛠️ 35種類の組み込みツール
- **ファイル操作**: 読み書き、編集、コピー、移動、削除、バックアップ復元
- **Git操作**: status, diff, add, commit, push, branch, stash
- **コード検索**: grep検索、ファイル検索、ast-grep（構造的検索）
- **開発支援**: テスト実行、フォーマット、リント

### 📋 2つの実行モード

**通常モード**（デフォルト）
- 安全なツール（ファイル読み取り等）は自動実行
- 危険なツール（ファイル編集、bash等）は毎回確認

**Plan Mode**（`/plan on`で有効化）
- 調査ツール（ファイル読み取り、検索等）は即座に実行
- 実装ツール（ファイル編集等）を使う前に計画を表示
- 承認後、実装ステップを自動実行（Claude Code風）

### 🔍 コードレビュー & リファクタリング
`/review` でセキュリティ・テストカバレッジをチェック。
`/refactor --ai --fix` で自動修正まで実行。

### ↩️ 安全なUndo機能
すべてのファイル変更は自動バックアップ。
`/undo` または「さっきの変更を戻して」で即座に復元。

### 🖼️ マルチモーダル対応
画像ファイルを指定してUIデザインからコード生成。
エラースクリーンショットから原因分析も可能。

### 🗺️ Repo Map（30言語対応）
Tree-sitterによる高精度なコード構造解析。
Go, TypeScript, Python, Rust, Java, C/C++, Ruby, Kotlin, Swift, C#, Scala, PHP, Elixir, Lua, CSS/SCSS, HTML, Vue, Svelte, YAML, TOML, SQL, Bash, Markdown, Dockerfile等に対応。

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
/help       # ヘルプ
/undo       # 変更取り消し
/use gemini # プロバイダー切り替え
/plan on    # Plan Mode有効化
/exit       # 終了
```

## よく使う機能

### プロバイダー切り替え

```bash
xelyon --provider gemini --model gemini-2.0-flash-exp

# または対話中に
> /use claude
```

### Plan Mode（Claude Code風）

```bash
> /plan on
> ユーザー認証機能を追加して
# → 調査フェーズ（read_file, search_code等を自動実行）
# → 実装計画を表示
# → y で承認 → 実装ステップを自動実行
```

### 確認動作のカスタマイズ

```yaml
# ~/.xelyon/config.yaml

# 安全なツールも毎回確認したい場合
tool_confirm:
  auto_approve_safe: false
```

```bash
# 全ツール自動承認（信頼できる環境向け）
xelyon --auto-approve
```

### 設定確認

```bash
> /config show    # 全設定を表示（デフォルトとの差分を ⚡ で表示）
```

## ドキュメント

| ドキュメント | 内容 |
|------------|------|
| [コマンド一覧](docs/commands.md) | 全コマンド、35ツール、使用例 |
| [プロバイダー設定](docs/providers.md) | 各プロバイダーのAPIキー取得方法 |
| [設定リファレンス](docs/config.md) | config.yaml と環境変数 |
| [MCP連携](docs/mcp.md) | 外部ツール追加 |
| [使い方詳細](docs/usage.md) | 複数行入力、画像入力、レビュー機能など |

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
