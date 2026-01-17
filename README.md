# XELYON CLI

AI搭載のコーディングアシスタントCLI

[![CI](https://github.com/susugadx/xelyon-cli/workflows/CI/badge.svg)](https://github.com/susugadx/xelyon-cli/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 特徴

### 🌐 6種類のLLMプロバイダー
DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq をシームレスに切り替え。
ローカルLLM（Ollama）も対応で、オフラインでも使用可能。

### 🛠️ 34種類の組み込みツール
- **ファイル操作**: 読み書き、編集、コピー、移動、削除、バックアップ復元
- **Git操作**: status, diff, add, commit, push, branch, stash
- **コード検索**: grep検索、ファイル検索、ast-grep（構造的検索）
- **開発支援**: テスト実行、フォーマット、リント

### 📋 Plan Mode（自動実装）
「ユーザー認証を追加して」→ AIが計画を立案 → 承認後に自動実行。
複雑なタスクも一気に完了。

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

### Plan Mode（自動実装）

```bash
> /plan on
> ユーザー認証機能を追加して
# → 計画表示 → y で承認 → 自動実行
```

### 設定確認

```bash
> /config show    # 全設定を表示（デフォルトとの差分を ⚡ で表示）
```

## ドキュメント

| ドキュメント | 内容 |
|------------|------|
| [コマンド一覧](docs/commands.md) | 全コマンド、29ツール、使用例 |
| [プロバイダー設定](docs/providers.md) | 各プロバイダーのAPIキー取得方法 |
| [設定リファレンス](docs/config.md) | config.yaml と環境変数 |
| [MCP連携](docs/mcp.md) | 外部ツール追加 |
| [使い方詳細](docs/usage.md) | 複数行入力、画像入力、レビュー機能など |

## 開発

```bash
git clone https://github.com/susugadx/xelyon-cli.git
cd xelyon-cli
go build -o xelyon
./xelyon
```

> ⚠️ リポジトリの `XELYON.md` は開発用です。あなたのプロジェクトでは `/init` で新規作成してください。

## ライセンス

MIT
