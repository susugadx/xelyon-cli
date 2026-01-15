# XELYON CLI

> ⚠️ **このプロジェクトは開発中です / This project is under development**
>
> 一部の機能は不安定な場合があります。フィードバックは [Issues](https://github.com/susugadx/xelyon-cli/issues) へお願いします。

AI搭載のコーディングアシスタントCLIツール

[![CI](https://github.com/susugadx/xelyon-cli/workflows/CI/badge.svg)](https://github.com/susugadx/xelyon-cli/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 特徴

- 🌐 **6種類のLLMプロバイダー対応**: DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq
- 💬 **対話型エージェント**: 29種類のツールで実際にコード編集・Git操作を実行
- 📋 **Plan Mode**: AIが実装計画を立てて、承認後に自動実行（並列処理対応）
- 🤖 **Headlessモード**: JSON出力で他のツールから呼び出し可能
- ↩️ **Undo機能**: ファイル変更の取り消し（バックアップから復元）
- 🧠 **メモリ機能**: プロジェクト別・グローバル記憶の永続化
- 📂 **会話履歴管理**: セッションをJSONL形式で保存・復元
- 🗺️ **Repo Map**: Tree-sitterでコード構造を自動解析（RepoMapは永続キャッシュにより2回目以降の起動を高速化）
- 🗃️ **プロンプトキャッシュ**: 基盤（in-memory TTL付きLRU + `prompt_cache` 設定） + Claudeは `cache_control` 対応（OpenAI/Geminiは将来対応）
- 🧯 **コンテキスト肥大化対策**: `go test` 等の長いツール出力は送信前に自動で末尾N行に切り詰め、トークン上限超過(400)で同じ巨大入力を再送して失敗し続ける問題を回避

- 🔌 **MCP対応**: Model Context Protocol による外部ツール連携

## インストール

### Homebrew（macOS）

```bash
brew install susugadx/tap/xelyon
```

### バイナリダウンロード

[GitHub Releases](https://github.com/susugadx/xelyon-cli/releases)から環境に合ったバイナリをダウンロード。

**Linux/macOS:**
```bash
# ダウンロード・展開例
wget https://github.com/susugadx/xelyon-cli/releases/latest/download/xelyon_linux_amd64.tar.gz
tar -xzf xelyon_linux_amd64.tar.gz
sudo mv xelyon /usr/local/bin/
```

**Windows:**
[xelyon_windows_amd64.zip](https://github.com/susugadx/xelyon-cli/releases/latest) をダウンロード・展開。

### ソースからビルド

```bash
git clone https://github.com/susugadx/xelyon-cli.git
cd xelyon-cli
go build -o xelyon
```

## クイックスタート

### 1. APIキーを設定

```bash
# .envファイル使用（推奨）
cp .env.example .env
# .env を編集してAPIキーを設定

# または環境変数で設定
export DEEPSEEK_API_KEY="sk-..."  # DeepSeek（デフォルト）
export OPENAI_API_KEY="sk-..."    # OpenAI
export GEMINI_API_KEY="..."       # Gemini
export ANTHROPIC_API_KEY="sk-ant-..." # Claude
export GROQ_API_KEY="gsk_..."     # Groq
# Ollama: ローカル実行のため不要
```

APIキーの取得方法: [プロバイダー設定ガイド](docs/providers.md)

### 2. 起動して対話

```bash
# インストール後（Homebrew または /usr/local/bin/ に配置）
xelyon

# ローカルビルド（プロジェクトディレクトリから）
./xelyon

> main.goを読んで、バグがあれば修正して
```

### 3. プロバイダー切り替え

```bash
# コマンドライン指定（インストール後）
xelyon --provider gemini --model gemini-2.0-flash-exp

# ローカルビルド
./xelyon --provider gemini --model gemini-2.0-flash-exp

# セッション中に切り替え
> /use openai gpt-4
```

## 基本的な使い方

### Dry Run（安全な試行: ツール実行をシミュレート）

`--dry-run` を有効にすると、AIがツールを呼び出しても実際には実行せず、結果だけをシミュレートします。
ファイル編集やGit操作を伴う提案を「まずは安全に確認したい」場合に使えます。

- ツール実行は行われません（`tools.Execute()` を呼びません）
- 変更履歴（Undo対象）も作られません
- 履歴には `"[Dry Run] Tool execution simulated"` がツール結果として記録されます

例:
```bash
xelyon --dry-run
```


### 対話コマンド

#### コマンドエイリアス（ショートカット）

一部の対話コマンドはショートカット（エイリアス）で呼び出せます。

- `/h` → `/help`
- `/m` → `/memory`
- `/p` → `/paste`

また、`~/.xelyon/config.yaml` の `command_aliases` で追加/上書きできます。

```yaml
command_aliases:
  /h: /help
  /m: /memory
  /hh: /help
```


```bash
/help       # コマンド一覧
/init       # XELYON.md生成（プロジェクト設定ファイル）
/sync       # XELYON.mdを現在のコードと同期
/memory add プロジェクトではReactを使う  # 記憶を追加
/plan on    # Plan Mode有効化
/use gemini # プロバイダー切り替え
/undo       # 最後の変更を取り消し
/exit       # 終了（終了時に会話からルール抽出→XELYON.mdへ追記提案）
```

全コマンド: [コマンドリファレンス](docs/commands.md)

### 複数行入力

プロンプトやコードを複数行で入力できます。

**方法1: ``` マーカー**
```bash
> ```
📝 Multiline input mode (end with ``` on a new line)
  1 | 以下のコードをレビューして
  2 |
  3 | func main() {
  4 |     fmt.Println("Hello")
  5 | }
  6 | ```
✅ Captured 5 lines
```

**方法2: ペースト（Bracketed Paste Mode）**
- コードエディタやブラウザから複数行をコピー＆ペースト
- 自動的に複数行として認識され、途中で実行されません
- `📋 Pasted N lines` と表示されます
- ターミナルエミュレータがBracketed Paste Modeをサポートしている必要があります（大半の現代的なターミナルは対応済み）
  - 対応: iTerm2, Terminal.app, GNOME Terminal, Konsole, Windows Terminal, tmux, screen
  - 非対応の場合は**方法1の ``` マーカー**または**方法3の /paste コマンド**を使用してください

**方法3: /paste コマンド（WSL等のBracketed Paste非対応環境向け）**
```bash
> /paste
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📝 Paste Mode / ペーストモード
   End: empty line x2, 'END', /end, Ctrl+D
   Cancel: /cancel, /c
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[長文をペースト]
[空行]
[空行]
✅ Captured N lines (X.X KB)
```
- 終了方法: 空行2回、`END`、`/end`、Ctrl+D
- キャンセル: `/cancel`、`/c`（入力破棄してプロンプトに戻る）
- エイリアス: `/p`
- 設定: `~/.xelyon/config.yaml` で `paste.max_lines`, `paste.timeout_seconds` 等を変更可能


**補足: 確認プロンプトのコメント入力でも /paste が使えます**
- Plan Mode の `Approve this plan? [y/n/c]` や、ツール実行前/危険操作（SafetyLow）確認で `c` を選んだ後のコメント入力中に、`/paste`（エイリアス: `/p`）を入力すると Paste Mode を起動できます。
- Paste Mode 終了後、入力した内容がそのままコメントとして挿入されます。

**確認UI（y/n/c）とコメント継続**

- デフォルトで有効です（`y/n/c`）。無効化したい場合は `XELYON_INTERACTIVE_CONFIRM=0`（または `false`）を設定してください。

- すべての確認プロンプトは `y/n/c`（yes/no/comment）をサポートします
- `c` を選ぶと、コメント（複数行可）を入力でき、AIがそのコメントを元に提案を修正して作業を継続します
- 対象例: `copy_file`（上書き）, `git_add`, `git_branch`（未コミット変更ありの切り替え）, `git_commit`, `lint`（auto-fix）, `move_file`（上書き） など
- go test失敗時のロールバック確認（Rollback the change?）でも `c` を選ぶとコメント→再提案が可能です
- 誤ってEnterだけ押した場合は無視して入力をリトライします（空入力で即キャンセルになりません）
- `write_file` とバックアップ作成は、既存ファイルがある場合に元のパーミッションを維持します
- ループ防止のため、コメント→再提案→再確認のリトライ回数には上限があります


### キャンセル操作（Ctrl+C）

- **AI応答中**: Ctrl+Cで応答を中断し、プロンプトに戻る
- **アプリ終了**: Ctrl+Cを3秒以内に2回押すと終了
- 1回目のCtrl+Cでは「⚠️ Interrupted. Press Ctrl+C again within 3 seconds to exit.」と表示

### ツール例

AIが自動で以下のツールを使用します:

- **ファイル編集**: `read_file`, `write_file`, `str_replace`（old_strの文字列置換に加え、old_str空+start_line/end_line指定で行レンジ置換も可能）
- **Git操作**: `git_status`, `git_diff`, `git_add`, `git_commit`, `git_push`
- **開発支援**: `run_test`, `format`, `lint`

  - `run_test` 実行中はスピナーを表示し、長時間のテストでも「動作中」であることが分かるようにしています

- **検索**: `search_code`, `search_file`, `web_search`

全ツール: [コマンドリファレンス](docs/commands.md#利用可能なツール)

### Plan Mode

AIが実装計画を立てて、承認後に自動実行します。

```bash
# Plan Modeで起動（インストール後）
xelyon --plan "バグ修正とテストを実行"

# ローカルビルド
./xelyon --plan "バグ修正とテストを実行"

# セッション中に切り替え
> /plan on
> ユーザー認証機能を追加して
```

詳細: [コマンドリファレンス - Plan Mode](docs/commands.md#plan-mode自律実行--並列処理)

### コードレビュー

`/review` コマンドでセッション中の変更をAIがレビューします。

```bash
# セッション中の変更をレビュー
> /review

# すべてのgit変更をレビュー
> /review --all

# セキュリティフォーカス
> /review --security

# テストカバレッジフォーカス
> /review --test

# 修正提案を生成
> /review --fix

# 確認をスキップ
> /review --yes
```

**検出ルール:**
- **セキュリティ**: コマンドインジェクション、パストラバーサル、弱い暗号化、HTTPタイムアウト未設定
- **一般**: 大きな差分、TODO/FIXME追加、ドキュメント不足、機密ファイル変更
- **テスト**: テストカバレッジ不足、アサーション不足

レポートは `~/.xelyon/reviews/` にMarkdown形式で保存されます。

### メモリ機能

プロジェクト別・グローバル記憶を保存できます。

```bash
> /memory add プロジェクトではTypeScriptとReactを使う
> /memory list
> /memory delete <ID>
```

詳細: [コマンドリファレンス - メモリ](docs/commands.md#memory-コマンドnew)

## ドキュメント

- [コマンド一覧](docs/commands.md) - 全コマンド、29ツール、使用例
- [プロバイダー設定](docs/providers.md) - DeepSeek, OpenAI, Gemini, Claude, Groq, Ollamaの設定方法
- [設定リファレンス](docs/config.md) - config.yamlと環境変数（`prompt_cache` 含む）
- [MCP連携](docs/mcp.md) - Model Context Protocolで外部ツール追加

## 技術スタック

- **Go 1.22+**
- **Cobra** - CLIフレームワーク
- **LLM APIs** - DeepSeek, OpenAI, Gemini, Claude (Anthropic), Ollama, Groq
- **Tree-sitter** - コード構造解析

## 開発

```bash
# ビルド
go build -o xelyon

# テスト
go test ./...

# フォーマット
go fmt ./...
```

詳細なアーキテクチャと開発ガイド: [XELYON.md](XELYON.md)

## ライセンス

MIT

## バージョン

最新リリース: [GitHub Releases](https://github.com/susugadx/xelyon-cli/releases)

変更履歴: [CHANGELOG.md](CHANGELOG.md)
