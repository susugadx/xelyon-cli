# XELYON CLI

AI搭載のコーディングアシスタントCLI

[![CI](https://github.com/susugadx/xelyon-cli/workflows/CI/badge.svg)](https://github.com/susugadx/xelyon-cli/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 特徴

- 🌐 **6種類のLLMプロバイダー**: DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq
- 🛠️ **29種類のツール**: ファイル編集、Git操作、コード検索を自動実行
- 📋 **Plan Mode**: AIが計画を立てて承認後に自動実行
- ↩️ **Undo**: ファイル変更の取り消し

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
