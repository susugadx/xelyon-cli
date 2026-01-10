# XELYON CLI

AI搭載のコーディングアシスタントCLIツール

## 特徴

- 🌐 **6種類のLLMプロバイダー対応**: DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq
- 🤖 **複数のAIモデル対応**: 各プロバイダーで複数モデルを選択可能
- 💬 **対話型エージェント**: ツールを使って実際にコード編集・Git操作を実行
- 🔍 **Web検索機能**: Serper APIを使ったリアルタイムWeb検索（v0.9.1）
- 🧠 **思考プロセスの可視化**: なぜそのツールを使うのか、理由を説明（v0.9.0）
- 📋 **段階的実行**: 複雑なタスクは計画→確認→実行→検証のフロー（v0.9.0）
- ✅ **品質チェック自動化**: コード変更後に`go fmt`, `go test`を自動実行（v0.9.0）
- 📂 **会話履歴管理**: セッションをJSONL形式で保存・復元
- ⚡ **スピナー表示**: API呼び出し中の視覚的フィードバック
- 📄 **自動ページング**: 長い出力を読みやすく表示
- ↩️ **Undo機能**: ファイル変更の取り消し（最大10件）
- 💾 **自動バックアップ**: 編集時に.bakファイルを自動作成
- ✏️ **安全な編集**: str_replaceツールでdiff表示と確認プロンプト
- 💬 **対話的修正機能（実験的）**: ツール実行前にコメントで修正指示可能（v0.28.0）

## インストール

### 方法1: ビルド済みバイナリ（推奨）

[GitHub Releases](https://github.com/susugadx/xelyon-cli/releases)から環境に合ったバイナリをダウンロード:

- **Linux**: `xelyon_*_linux_amd64.tar.gz` または `xelyon_*_linux_arm64.tar.gz`
- **macOS**: `xelyon_*_darwin_amd64.tar.gz` (Intel) または `xelyon_*_darwin_arm64.tar.gz` (Apple Silicon)
- **Windows**: `xelyon_*_windows_amd64.zip`

```bash
# ダウンロード・展開例（Linux/macOS）
wget https://github.com/susugadx/xelyon-cli/releases/download/v0.27.0/xelyon_0.27.0_linux_amd64.tar.gz
tar -xzf xelyon_0.27.0_linux_amd64.tar.gz
sudo mv xelyon /usr/local/bin/
```

### 方法2: Homebrew（macOS）

```bash
brew install susugadx/tap/xelyon
```

### 方法3: ソースからビルド

```bash
git clone https://github.com/susugadx/xelyon-cli.git
cd xelyon-cli
go build -o xelyon
```

### 環境変数設定

```bash
# 環境変数設定（方法1: .envファイル使用 - 推奨）
cp .env.example .env
# .envファイルを編集してAPIキーを設定

# 環境変数設定（方法2: exportコマンド）
export DEEPSEEK_API_KEY="your-api-key"        # DeepSeek (デフォルト)
export OPENAI_API_KEY="sk-..."                # OpenAI (オプション)
export GEMINI_API_KEY="..."                   # Google Gemini (オプション)
export ANTHROPIC_API_KEY="sk-ant-..."         # Claude (オプション)
export GROQ_API_KEY="gsk_..."                 # Groq (オプション)
# Ollama: ローカル実行のため不要

export SERPER_API_KEY="your-serper-api-key"  # Web検索用（オプション）
```
## サポートされているLLMプロバイダー

XELYON CLIは以下のLLMプロバイダーに対応しています：

| Provider | 環境変数 | 推奨モデル | ストリーミング |
|----------|---------|-----------|--------------|
| **DeepSeek** | `DEEPSEEK_API_KEY` | deepseek-coder | ✅ |
| **OpenAI** | `OPENAI_API_KEY` | gpt-4o | ✅ |
| **Gemini** | `GEMINI_API_KEY` | gemini-2.0-flash-exp | ✅ |
| **Claude** | `ANTHROPIC_API_KEY` | claude-sonnet-4-20250514 | ✅ |
| **Ollama** | (不要) | llama3 | ✅ |
| **Groq** | `GROQ_API_KEY` | llama3-70b-8192 | ✅ |

### プロバイダーの切り替え方法

#### 1. 環境変数で指定（永続的）
```bash
export XELYON_PROVIDER=openai
export OPENAI_API_KEY="sk-..."
./xelyon
```

#### 2. CLIフラグで指定（一時的）
```bash
./xelyon --provider gemini --model gemini-2.0-flash-exp
```

#### 3. 設定ファイルで指定（グローバル設定）
```yaml
# ~/.xelyon/config.yaml
default_provider: claude
default_model: claude-sonnet-4-20250514
```

### Ollama（ローカルLLM）の使い方

```bash
# 1. Ollamaをインストール
curl -fsSL https://ollama.com/install.sh | sh

# 2. Ollamaサーバーを起動
ollama serve

# 3. モデルをダウンロード（初回のみ）
ollama pull llama3

# 4. XELYONでOllamaを使用
export XELYON_PROVIDER=ollama
./xelyon --model llama3
```

インストール済みモデルは自動検出されます。

## 使い方

### 基本的な使い方

```bash
# 対話モード（新規セッション）
./xelyon

# ワンショットモード
./xelyon "このプロジェクトを説明して"

# 前回のセッションを復元
./xelyon --resume

# 自動承認モード（安全・中レベルの操作を自動承認）
./xelyon -y         # または --auto-approve
./xelyon -y --coder # 組み合わせ可能

# モデル選択
./xelyon --coder    # コード特化
./xelyon --think    # 深い推論
./xelyon --claude   # Claude (予定)
```

### --auto-approve フラグ

`-y` または `--auto-approve` フラグで安全・中レベルの操作を自動承認できます:

**自動承認される操作** (SafetyHigh / SafetyMedium):
- 読み取り専用: read_file, list_dir, git_status, git_log, lint, test
- 書き込み（リカバリ可能）: write_file, str_replace, append_file, git_add, git_commit

**常に確認が必要な操作** (SafetyLow):
- 破壊的操作: delete_file, delete_lines, move_file, bash
- リスクの高いGit操作: git_push, git_checkout, git_branch

```bash
# 例: 安全な操作のみ自動承認
./xelyon -y "README.mdを読んでプロジェクト説明を作成"
# → read_fileは自動承認、write_fileも自動承認

# 破壊的操作は常に確認
./xelyon -y "不要なファイルを削除"
# → delete_fileは確認プロンプトが表示される
```

### 対話コマンド

```
/save             - 現在のセッションを保存
/load [id]        - セッションを読み込み（IDなしで最新）
/sessions         - 最近のセッション一覧
/undo             - 直前のファイル変更を取り消し
/stats            - セッション統計情報を表示（時間、メッセージ数、ツール実行、コスト）
/config           - 設定の表示・変更（例: /config model gpt-4o）
/model [name]     - 現在のモデルとプロバイダーを表示、または任意のモデルに切り替え
/repomap          - リポジトリのコード構造マップを表示
/version          - バージョン情報を表示
/clear            - 会話履歴をクリア
/history          - 会話履歴を表示
/help             - ヘルプを表示
/exit, /quit, /q  - 終了
```

### /stats コマンド

セッションの統計情報を表示します:

```
> /stats

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Session Statistics / セッション統計
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⏱️  Time / 経過時間
  Elapsed: 15m 30s

💬 Messages / メッセージ数
  User:      10
  Assistant: 12
  Total:     22

🔧 Tool Executions / ツール実行回数
  Total: 25
  Breakdown:
    - read_file    : 8
    - write_file   : 5
    - git_commit   : 2
    - ...

🤖 Provider / プロバイダー
  Name: DeepSeek
  Model: deepseek-coder

💰 Token Usage & Cost / トークン使用量とコスト
  Input:  12,500 tokens
  Output: 3,200 tokens
  Total:  15,700 tokens
  Estimated Cost: $0.0032 USD

📁 Session File / セッションファイル
  Path: ~/.xelyon/sessions/abc123.json
  Size: 45.2 KB
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**表示内容**:
- 経過時間（セッション開始からの時間）
- メッセージ数（User/Assistant別）
- ツール実行回数とツール別内訳
- プロバイダーとモデル情報
- トークン使用量と推定コスト（プロバイダー別料金表で計算）
- セッションファイルのパスとサイズ

**コスト計算**（$/1M tokens）:
- DeepSeek: input $0.14, output $0.28
- OpenAI GPT-4o: input $2.50, output $10.00
- Claude Sonnet: input $3.00, output $15.00
- Gemini Flash: input $0.075, output $0.30
- Ollama: ローカル実行（$0）

**注**: トークン数はAPI対応が必要です。現在は手動で追加できます（将来のバージョンで自動取得予定）。

### 対話的修正機能（実験的）

ツール実行前の確認プロンプトで `c` (comment) を選択すると、AIに修正指示を送れます:

```
Execute this tool? [y/n/c]: c

💬 Enter your comment (press Enter twice to finish):
> この実装だとエラーハンドリングがない
> もっと堅牢にして
>

🤖 AI: 承知しました。エラーハンドリングを追加します...
```

**使い方**:
- `y` - ツールを実行
- `n` - キャンセル
- `c` - コメント（修正指示）

**制限事項**:
- 最大3回まで修正可能（無限ループ防止）
- 現在は実験的機能（今後Phase 2で画像入力対応予定）

詳細は [Issue #1](https://github.com/susugadx/xelyon-cli/issues/1) を参照。

### 利用可能なツール

AIが自動で以下のツールを使用します（全29ツール）:

#### ファイル編集系（8ツール）
- **read_file** - ファイル読み込み
- **write_file** - ファイル作成・上書き（.bakバックアップ自動作成）
- **str_replace** - ファイル内の文字列置換（部分編集、diff表示、.bakバックアップ）
- **append_file** - ファイル末尾に追加（非破壊的、既存ファイルのみバックアップ）
- **prepend_file** - ファイル先頭に追加（非破壊的、既存ファイルのみバックアップ）
- **insert_after** - パターンマッチした行の後に挿入（2段階マッチング、コンテキスト表示）
- **insert_before** - パターンマッチした行の前に挿入（2段階マッチング、コンテキスト表示）
- **copy_file** - ファイルコピー（上書き時のみ確認、バックアップ作成）

#### ファイル管理系（6ツール）
- **list_dir** - ディレクトリ一覧
- **create_dir** - ディレクトリ作成（親ディレクトリも含む、冪等的）
- **delete_lines** - ファイルから行範囲を削除（破壊的、RED警告、バックアップ作成）
- **delete_file** - ファイルを完全削除（破壊的、プレビュー表示、バックアップ作成）
- **move_file** - ファイル移動・リネーム（アトミック、上書き時は確認＋バックアップ）
- **lint** - リンター実行＆自動修正（Go/JS/Python/Rust対応、auto_fix オプション）

#### Git操作系（9ツール）
- **git_status, git_diff, git_add, git_commit, git_push, git_log** - 基本的なGit操作
- **git_branch** - ブランチ管理（list/create/switch）
- **git_checkout** - ファイル復元またはブランチチェックアウト（破壊的操作、バックアップ作成）
- **git_stash** - 変更を一時退避・復元（save/list/pop/apply/drop）

#### 開発支援系（2ツール）
- **run_test** - テストフレームワークを自動検出して実行（Go/npm/pytest/cargo対応）
- **format** - フォーマッターを自動検出して実行（gofmt/prettier/black/rustfmt対応）

#### 検索系（3ツール）
- **search_code** - コード内検索（grep）
- **search_file** - ファイル名検索（find）
- **web_search** - Web検索（Serper API、上位5件）

#### シェル実行（1ツール）
- **bash** - シェルコマンド実行（sed/awk等の編集コマンドはブロック）

### Undo機能の使い方

ファイル編集（`write_file` / `str_replace`）時に自動的に`.bak`バックアップが作成されます。

```bash
# 編集を実行
> test.txt の "hello" を "world" に置き換えて
✅ Replaced in: test.txt
(test.txt.bak が自動作成される)

# 間違えた場合は取り消し
> /undo
Undo last change?
  File: test.txt
  Tool: str_replace
  Time: 2026-01-06 10:00:05
Continue? (y/n): y
✅ Undone: Replaced in test.txt
   Restored from: test.txt.bak
```

**特徴:**
- 最大10件の変更履歴を保持（メモリ内）
- セッション単位でリセット
- 新規ファイル作成時はバックアップなし

### モデル切り替え

セッション中に再起動なしでモデルを切り替えることができます。

```bash
# 現在のモデルとプロバイダーを確認
> /model
🤖 Current model: gpt-4o
🌐 Provider: OpenAI

Usage: /model <model-name>
Enter any model name supported by your provider.

# モデルを切り替え（任意のモデル名を指定可能）
> /model gpt-4o-mini
✅ Model switched: gpt-4o → gpt-4o-mini
💾 Default model saved to config

# Ollamaの場合はインストール済みモデルも表示
> /model
🤖 Current model: llama3
🌐 Provider: ollama

Usage: /model <model-name>
Enter any model name supported by your provider.

Installed Ollama models:
  - llama3
  - codellama
  - mistral
```

**特徴:**
- 再起動不要でモデルを即座に切り替え
- 任意のモデル名を指定可能（プロバイダーがサポートしていれば使用可）
- Ollamaの場合はインストール済みモデル一覧を表示
- 設定ファイルにも自動保存
- 次回起動時も同じモデルが使われる

### 設定ファイル

設定ファイル `~/.xelyon/config.yaml` でデフォルトモデルなどを設定できます。

```yaml
# XELYON CLI 設定
# Models: deepseek-chat, deepseek-coder, deepseek-reasoner, claude

default_model: deepseek-coder
```

**設定の変更:**
```bash
# 設定を表示
> /config

# デフォルトモデルを変更
> /config model deepseek-coder

# CLIを再起動すると新しいモデルが使用される
```

**特徴:**
- 初回起動時に自動作成
- `/config` コマンドで設定表示・変更
- CLIフラグ（`--coder`, `--think`）が設定ファイルより優先

### MCP（外部ツール連携）

`~/.xelyon/mcp.json` を作成して外部MCPサーバーを登録:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"]
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

起動すると自動的に接続され、AIがMCPツールを使えるようになります。

**利用可能なMCPサーバー例**:
- `@modelcontextprotocol/server-filesystem` - ファイルシステム操作
- `@modelcontextprotocol/server-github` - GitHub操作
- `@modelcontextprotocol/server-git` - Git操作
- その他、MCP互換のサーバーすべて

### Repo Map（コード構造マップ）

起動時にプロジェクトのコード構造を自動解析し、AIに渡します。
これにより、AIが「どこに何があるか」を効率的に把握できます。

**対応言語**: Go, JavaScript, TypeScript, Python

```bash
# 現在のマップを確認
> /repomap
```

**出力例**:
```
### internal/agent/agent.go
  39: func NewAgent(model string) *Agent
  134: func RunInteractive(model string)
  171: func RunOnce(query string, model string)
  199: func (a *Agent) chat(input string)

### internal/tools/tools.go
  62: func ParseToolCall(response string) *ToolCall
  100: func Execute(tc *ToolCall) (string, *FileChange)
```

**特徴**:
- 起動時に自動生成（SystemPromptに追加）
- トークン制限（デフォルト2000トークン）で大規模プロジェクトにも対応
- node_modules、.git、vendor等は自動除外

## プロジェクト設定

プロジェクトルートの`XELYON.md`に、プロジェクトの詳細な設定とアーキテクチャが記載されています。

### XELYON.mdの内容
- プロジェクト概要とアーキテクチャ
- コーディングルールとSystemPromptルール
- 開発ガイド（ツール追加、コマンド追加、デバッグ方法）
- トラブルシューティング
- 既知の制約と今後の拡張案

**開発する際は、まずXELYON.mdを読んでプロジェクトの構造を理解してください。**

XELYON.mdは自動的にコンテキストとして読み込まれます。

## 会話履歴

セッションは`~/.xelyon/history/`に保存されます:

```
~/.xelyon/history/
  ├── 1704567890.jsonl          # メッセージ本体（JSONL）
  └── metadata/
      └── 1704567890.json       # メタデータ（JSON）
```

## アーキテクチャ

```
xelyon-cli/
├── main.go                 # エントリーポイント
├── cmd/
│   └── root.go            # Cobraコマンド定義
├── internal/
│   ├── agent/             # エージェントロジック
│   │   └── agent.go       # 対話ループ、コマンド処理
│   ├── api/               # API クライアント
│   │   ├── deepseek.go    # DeepSeek API（ストリーミング）
│   │   └── xelyon.go      # RAG検索API
│   ├── tools/             # ツール実行エンジン
│   │   └── tools.go       # 15種類のツール実装
│   ├── ui/                # UI コンポーネント
│   │   ├── spinner.go     # ローディングスピナー
│   │   └── pager.go       # 自動ページング
│   ├── history/           # セッション管理
│   │   ├── session.go     # セッション構造
│   │   └── storage.go     # JSONL永続化
│   └── file/              # ファイルI/O
│       ├── reader.go
│       └── writer.go
└── XELYON.md              # プロジェクト設定（自動読み込み）
```

## 技術スタック

- **Go 1.22+**
- **Cobra** - CLIフレームワーク
- **LLM APIs** - DeepSeek, OpenAI, Gemini, Claude (Anthropic), Ollama, Groq
- **fatih/color** - ターミナル色付け
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

## ライセンス

MIT

## バージョン履歴

### v0.28.3 Homebrew Tap自動更新対応 (2026-01-10)
- 🍺 **Homebrew Tap トークン設定**: `HOMEBREW_TAP_TOKEN` 環境変数でFormula自動更新
- 📦 **homebrew-tap README更新**: インストール・アップデート・アンインストール方法を明記

### v0.28.2 CGO不要ビルド対応 (2026-01-10)
- 🏗️ **ビルドタグ実装**: `norepomap` タグでtree-sitter依存を除外
- 🌍 **クロスコンパイル対応**: CGO無効でLinux/macOS/Windows向けビルド可能
- 📝 **スタブ実装**: Repo Map機能を無効化（CGO無効時はメッセージ表示）

### v0.28.1 GoReleaser v6対応 (2026-01-10)
- 🔧 **GitHub Actions修正**: goreleaser-action v5 → v6（version: 2対応）

### v0.28.0 リリース自動化完了 (2026-01-10)
- 🚀 **GoReleaser設定強化**: version: 2 対応、Changelogグループ化
- 📦 **バイナリ情報埋め込み**: Version, Commit, Date をldflags経由で設定
- 🍺 **Homebrew Tap自動更新**: リリース時に `susugadx/homebrew-tap` へFormula自動push
- 📄 **GitHub Releases自動化**: タグpush時に全プラットフォーム向けバイナリを自動生成・アップロード

### v0.27.0 Phase 2テスト実装完了 (2026-01-10)
- ✅ **internal/tools/ テスト追加**: read_file (100%), write_file (90.9%), str_replace (72.2%)
- ✅ **internal/api/ テスト追加**: validator (84-100%), provider (100%)
- 🧪 **テスト結果**: 95 tests passing, 14.7% overall coverage（対象パッケージは95%+）
- 🛠️ **テストヘルパー拡張**: `ReadFile()` 追加、confirm/ValidatePath モック対応

### v0.26.0 テスト実装開始 (2026-01-09)
- 🧪 **テストインフラ構築**: 包括的なテストスイートの基盤を整備
  - `internal/testutil/` パッケージ追加（テストヘルパー関数）
  - `MockConfirm()`, `CreateTempFile()`, `AssertFileExists()`, `SetupTempHome()`

- ✅ **internal/crypto/ テスト完了**: 暗号化機能の完全なテストカバレッジ（81.5%）
  - 暗号化/復号化ラウンドトリップテスト
  - ソルト・ノンスのランダム性検証
  - GCM認証タグ改ざん検出テスト
  - GetOrCreatePassphrase 動作確認（0600パーミッション検証含む）

- ✅ **internal/audit/ テスト完了**: 監査ログ機能の完全なテストカバレッジ（86.4%）
  - JSONL形式ログ記録テスト
  - 機密情報サニタイズ検証（password, token, api_key → [REDACTED]）
  - 出力切り詰め動作確認（500文字制限）
  - 並行アクセス安全性テスト（100 goroutines）
  - ログ記録失敗時の挙動確認（サイレント失敗）

- 🔧 **バグ修正**:
  - `internal/crypto/encryption.go`: `.xelyon`ディレクトリ自動作成を追加
  - `internal/mcp/client.go`: non-constant format string エラー修正

- 📊 **現在のカバレッジ**: 全体 2.4%（テスト実装継続中）

### v0.25.0 Phase 5: LOW優先度機能追加 (2026-01-09)
- 📝 **監査ログ機能**: ツール実行履歴を記録（`XELYON_AUDIT_LOG=1`で有効化）
  - JSONLフォーマット（`~/.xelyon/audit/audit_YYYYMMDD.jsonl`）
  - ツール名、引数、出力、成功/失敗、ファイル変更フラグを記録
  - 機密情報（password, token, api_key）は自動的に[REDACTED]化
  - 監査ログ記録失敗は通常動作を妨げない（サイレント）

- 🔒 **セッション履歴の暗号化**: 会話履歴を保護（`XELYON_ENCRYPT_HISTORY=1`で有効化）
  - AES-256-GCM暗号化方式
  - PBKDF2鍵導出（100,000回イテレーション、OWASP推奨）
  - 暗号化キーは`~/.xelyon/.session_key`に安全に保存（0600パーミッション）
  - 既存の非暗号化データとの互換性を保持

- ✅ **APIレスポンス検証強化**: 不正なレスポンスを検出
  - ストリーミングレスポンスの構造検証（choices配列、delta/messageフィールド）
  - エラーレスポンスの安全なパース
  - ツール呼び出しのスキーマ検証
  - DeepSeekプロバイダーで実装（他プロバイダーは必要に応じて追加可能）

- 🧹 **コード保守性向上**:
  - 非推奨関数のレビュー完了（破壊的変更を避けるため保持）
  - go.mod依存関係更新（golang.org/x/crypto追加）

### v0.24.0 セキュリティ・品質監査完了版 (2026-01-09)
- 🔒 **セキュリティ強化**（Phase 1 CRITICAL完了）
  - **コマンドインジェクション防止**: bash実行時の連結文字検出（`;`, `&&`, `||`, `|`, etc.）
  - **パストラバーサル防止**: 全ファイル操作ツールで`ValidatePath()`による境界チェック
  - **APIキー露出防止**:
    - Gemini: URLパラメータ→ヘッダー送信に変更
    - エラーメッセージサニタイズ（正規表現でAPIキーを[REDACTED]化）
  - **MCP任意コード実行防止**:
    - コマンドホワイトリスト（npx, node, python, etc.）
    - 環境変数サニタイズ（KEY/TOKEN/SECRET除外）
  - **グレースフルシャットダウン**: SIGINT/SIGTERM対応、MCPサーバー自動クローズ

- ⚡ **パフォーマンス改善**（Phase 2 HIGH完了）
  - **HTTPクライアント再利用**: 20MB+メモリ節約（100リクエストあたり）
  - **RepoMap文字列連結最適化**: O(n²)→O(n)改善（strings.Builder使用）
  - **コネクションプーリング**: 全6プロバイダーで有効化

- 🛡️ **信頼性向上**（Phase 2-3完了）
  - **レート制限ハンドリング**: HTTP 429検出＋Retry-Afterヘッダー解析
  - **競合状態修正**: Spinner（nil check）、ToolRegistry（sync.RWMutex）
  - **エラー検出強化**: JSONパース、I/O、セッション保存の警告表示
  - **ファイルパーミッション**: 機密ファイルを0600に変更

- 🔧 **設定可能性向上**（Phase 2完了）
  - **APIエンドポイント設定**: 環境変数でURL変更可能
    - `DEEPSEEK_API_URL`, `OPENAI_API_URL`, `ANTHROPIC_API_URL`, etc.
    - テスト・プロキシ環境での利用が容易に

- 🕐 **タイムアウト制御**（Phase 4完了）
  - **API呼び出しタイムアウト**: 3分で自動キャンセル
  - **リソースリーク防止**: context.WithTimeout＋cancel()

- 📝 **保守性向上**（Phase 4完了）
  - **MCPバージョン一元化**: ハードコード削除、version.Version使用
  - **スレッドセーフ**: ToolRegistry、Spinner

- 📊 **解決した問題**: 26件
  - CRITICAL: 5件（セキュリティ脆弱性）
  - HIGH: 12件（信頼性・パフォーマンス）
  - MEDIUM: 6件（エラーハンドリング）
  - LOW: 3件（保守性）

### v0.19.0 Phase 4 (2026-01-09)
- 🗑️ **4つの破壊的・複雑ツール追加**（Phase 4: 破壊的操作＆リンター統合）
  - **delete_lines**: ファイルから行範囲を削除
    - 1-indexed行番号（N-M形式）
    - 削除前にRED警告＋コンテキスト表示（前後5行）
    - 範囲外の行番号は自動的にファイル末尾にクランプ（エラーにしない）
    - 必須確認プロンプト＋バックアップ作成
  - **delete_file**: ファイルを完全削除
    - 削除前にファイルプレビュー表示（最初20行）
    - RED警告＋必須確認プロンプト
    - バックアップ失敗時は削除を中止（安全第一）
    - Undoでファイル復元可能
  - **move_file**: ファイル移動・リネーム
    - アトミック操作（os.Rename）
    - クロスファイルシステム時はコピー＋削除にフォールバック
    - 上書き時のみ確認プロンプト表示
    - 上書きされる移動先のみバックアップ（移動元はバックアップしない）
  - **lint**: リンター実行＆自動修正
    - 自動検出: Go（golangci-lint/go vet）、JavaScript/TypeScript（eslint）、Python（ruff/pylint）、Rust（clippy）
    - 2段階実行: チェック → 自動修正（auto_fixオプション）
    - 自動修正は確認プロンプト必須
    - 単一ファイル時のみバックアップ作成（ディレクトリ全体の場合はバックアップなし、制限として明示）
- 🛡️ **安全性強化**:
  - すべての破壊的操作にRED警告＋必須確認プロンプト
  - バックアップ失敗時は操作を中止（delete_file）
  - 二言語UI（英語/日本語併記）
  - FileChange追跡でUndo対応
- 🌐 **二言語対応**: 全ツールのUI表示を英語/日本語併記

### v0.18.0 Phase 3 (2026-01-09)
- 📝 **3つの高度なファイル編集ツール追加**（Phase 3: パターンマッチング＆コピー）
  - **insert_after**: パターンマッチした行の後に内容を挿入
    - 2段階パターンマッチング（厳密マッチ → 正規化ホワイトスペースマッチ）
    - コンテキスト表示（マッチ行の前後5行）
    - 複数マッチ時はエラー表示＋全マッチ場所表示
    - バックアップ自動作成
  - **insert_before**: パターンマッチした行の前に内容を挿入
    - insert_afterと同じ機能（挿入位置のみ異なる）
    - 同じ2段階マッチングアルゴリズム
  - **copy_file**: ファイルコピー（パーミッション保持）
    - 上書き時のみ確認プロンプト表示
    - 上書き時はバックアップ自動作成
    - ディレクトリコピーはエラー（ファイルのみ）
- 🔍 **パターンマッチング強化**:
  - パターンが見つからない場合はファイルプレビュー表示（最初50行）
  - 複数マッチ時は全マッチ場所を前後2行のコンテキスト付きで表示
  - 挿入前にマッチ行の前後5行をプレビュー表示
- 🌍 **二言語対応**: すべてのUI表示で英語/日本語併記
- 🛡️ **Undo対応**: 全ツールがFileChange追跡によりUndoコマンドで復元可能

### v0.17.0 Phase 2 (2026-01-09)
- 🔀 **3つのGitツール追加**（Phase 2: Git操作強化）
  - **git_branch**: ブランチ管理（list/create/switch）
    - list: すべてのブランチを表示（ローカル+リモート）
    - create: 新しいブランチを作成（非破壊的）
    - switch: ブランチ切り替え（未コミット変更時は確認プロンプト）
  - **git_checkout**: ファイル復元またはブランチチェックアウト
    - ファイル復元: HEADからファイルを復元（破壊的、バックアップ自動作成、diff表示）
    - ブランチ切り替え: git_branchの代替（未コミット変更時は確認）
  - **git_stash**: 変更を一時退避・復元
    - save: 変更をスタッシュ（メッセージ任意）
    - list: スタッシュ一覧表示
    - pop/apply: スタッシュ適用（プレビュー表示、確認プロンプト）
    - drop: スタッシュ削除（破壊的、確認必須）
- 🛡️ **安全性機能**:
  - 未コミット変更がある場合のブランチ切り替え警告
  - ファイル復元時のバックアップ自動作成
  - 破壊的操作での赤色警告メッセージ
  - スタッシュ適用時のプレビュー表示（最初20行）
- 🌍 **二言語対応**: すべての確認UIで英語/日本語併記

### v0.17.0 Phase 1 (2026-01-09)
- 🛠️ **5つの新ツール追加**（Phase 1: 低リスク・高価値ツール）
  - **append_file**: ファイル末尾に追加（非破壊的、プレビュー表示）
  - **prepend_file**: ファイル先頭に追加（非破壊的、プレビュー表示）
  - **create_dir**: ディレクトリ作成（親ディレクトリも含む、冪等的）
  - **run_test**: テストフレームワークを自動検出して実行
    - 対応: Go (go test), npm/yarn (npm test), pytest, cargo
    - テスト出力を完全表示（切り詰めなし）
  - **format**: フォーマッターを自動検出して実行
    - 対応: gofmt, prettier, black, rustfmt
    - 複数ファイルの一括フォーマット
- 📋 **SystemPrompt改善**: ツールをカテゴリ別に整理
  - File Operations, File Management, Git Operations, Development Tools, Search & Discovery, Shell
- 🎯 **開発支援強化**: テスト実行とコードフォーマットが1コマンドで可能に

### v0.15.0 (2026-01-09)
- 🌐 **マルチプロバイダー対応**: 6種類のLLMプロバイダーに対応
  - **OpenAI**: GPT-4o, GPT-4o-mini, GPT-4-turbo
  - **Gemini**: gemini-2.0-flash-exp, gemini-1.5-pro, gemini-1.5-flash
  - **Claude**: claude-sonnet-4-20250514, claude-opus-4, claude-haiku-3-5
  - **Ollama**: llama3, codellama, mistral等のローカルLLM
  - **Groq**: llama3-70b-8192, llama3-8b-8192, mixtral-8x7b
  - **DeepSeek**: 既存のプロバイダー（後方互換）
- 🔄 **柔軟なプロバイダー切り替え**:
  - 環境変数 `XELYON_PROVIDER` で指定
  - CLIフラグ `--provider openai` で一時的に切り替え
  - 設定ファイル `~/.xelyon/config.yaml` でデフォルト設定
- ⚙️ **設定管理拡張**:
  - プロバイダーごとのデフォルトモデル設定
  - モデル名のハードコード廃止（設定ファイルで管理）
  - Ollama自動モデル検出機能
- 🎯 **使いやすい設計**:
  - 全プロバイダーでストリーミング対応
  - 非ストリーミング時の自動フォールバック
  - 統一されたProvider Interface

### v0.14.0 (2026-01-08)
- 🎨 **ツール実行UX改善**
  - str_replace: 空白正規化マッチング（インデント違いに強く）
    - 行頭の空白を正規化して比較（完全一致優先）
    - タブ/スペース、インデント数の違いを吸収
  - 確認UI: 英語/日本語併記で「何をするか」明示
    - ボックス囲みの見やすい差分表示
    - write_file, str_replace, bash, git_* コマンドすべて改善
  - ツール呼び出し表示: JSON表示廃止、読みやすい形式に

### v0.13.0 (2026-01-07)
- 🗺️ **Repo Map機能**: Tree-sitterベースのコード構造解析
  - Go, JavaScript, TypeScript, Python対応
  - 関数、メソッド、構造体、クラス、インターフェースを抽出
  - 起動時に自動生成、SystemPromptに追加
  - `/repomap` コマンドで確認可能
  - トークン制限対応（デフォルト2000トークン）

### v0.12.0 (2026-01-07)
- 🔌 **MCP対応**: Model Context Protocol クライアント実装
  - 外部MCPサーバーのツールをXELYON CLI内で使用可能
  - `~/.xelyon/mcp.json` で設定
  - 起動時に自動接続、ツールを動的登録
  - filesystem, GitHub等の公式MCPサーバーと連携可能

### v0.11.0 (2026-01-07)
- 🔍 **自動検証機能**: Goファイル変更後の品質チェック
  - `go fmt` + `go test` を自動提案
  - テスト失敗時にrollback提案
  - `y`=両方実行, `f`=fmtのみ, `n`=スキップ
  - Goプロジェクト自動検出（go.mod存在確認）

### v0.10.1 (2026-01-07)
- 📦 **OSS公開準備**: リリース基盤整備
  - LICENSE追加（MIT）
  - CONTRIBUTING.md追加（開発ガイドライン）
  - GitHub Actions CI/CD（.github/workflows/ci.yml）
  - GoReleaser設定（.goreleaser.yml）
  - .gitignore追加
- ✅ **Phase 1完了**: HTTPタイムアウト実装完了
  - go mod tidy実行
  - すべてのHTTPクライアントにタイムアウト設定済み

### v0.10.0 (2026-01-07)
- 🏗️ **アーキテクチャ改善**: Provider/Tool Registryパターン導入
  - **Provider Pattern**: 複数LLMプロバイダー対応の基盤
    - `Provider` interface実装（DeepSeekProvider）
    - `Client` struct with context対応タイムアウト管理
    - 環境変数 `XELYON_PROVIDER` で切り替え可能（将来の拡張用）
  - **Tool Registry Pattern**: 外部ツール統合の基盤
    - `Tool` interface + `Registry` 実装
    - 16個の組み込みツールを Registry に登録
    - MCP等の外部ツール統合準備完了
  - **後方互換性維持**: 既存の関数・APIは全て維持
  - **破壊的変更なし**: v0.9.1からのアップグレードは透過的

### v0.9.1 (2026-01-07)
- 🔍 **Web検索機能**: Serper APIを使ったリアルタイムWeb検索
  - `web_search`ツール追加（上位5件の検索結果を取得）
  - 環境変数: `SERPER_API_KEY`
  - 日本語検索に最適化（地域: jp、言語: ja）
  - AIが必要に応じて自動的に検索を実行

### v0.9.0 (2026-01-06)
- 🧠 **SystemPrompt大幅改善**: 性格定義＋16個の体系化ルール
  - **Core Identity**: 正直さ、透明性、プロフェッショナリズムを明示
  - **4フェーズ構成**: Planning → Execution → Verification → Error Handling
  - 思考プロセスの可視化（なぜこのツールを使うのか説明）
  - 複雑なタスクで事前計画を提示（ユーザー確認を要求）
  - コード品質チェック（`go fmt`, `go test`）の自動化
  - エラー時の代替アプローチ提案
  - 新規ルール6個追加（#2, #4, #10, #12, #13, #16）

### v0.8.0 (2026-01-06)
- 🛡️ **安定性改善**: ループ検知、APIリトライ、エラーハンドリング強化
  - 同じツール呼び出しが3回繰り返されると自動中断
  - APIエラー時に最大2回自動リトライ（指数バックオフ）
  - 長い差分表示を10行で省略（ターミナルが埋まるのを防止）
  - キャンセル時のメッセージ改善（AIが同じ変更を繰り返さない）
  - 複数マッチエラー時にファイルプレビュー表示
  - SystemPromptに3つの新ルール追加（11-13）
- 🔧 **バージョン管理**: `internal/version/version.go`で一元管理
  - `/version`コマンド追加
  - `--version`フラグ対応
  - 起動バナーで自動表示
- 🔄 **モデル切り替え**: `/model`コマンドで再起動なしに切り替え可能
  - 設定ファイルにも自動保存
  - 次回起動時も同じモデルを使用

### v0.6.0 (2026-01-06)
- ✨ **設定ファイル**: `~/.xelyon/config.yaml`でデフォルトモデルを設定
- ✨ **/configコマンド**: 設定の表示・変更が可能
- 📄 **XELYON.md拡充**: 開発ガイド、デバッグ方法、既知の制約を追加
- 📋 **CLAUDE.md簡素化**: XELYON.md参照に統一

### v0.5.0 (2026-01-06)
- ✨ **Undo機能**: `/undo`コマンドでファイル変更を取り消し
- ✨ **自動バックアップ**: 編集時に`.bak`ファイルを自動作成
- ✨ **str_replace強制**: sedコマンドをブロックし、安全な`str_replace`ツールを強制
- ✨ **変更履歴追跡**: 最大10件の変更をメモリ内で追跡
- 🔒 **安全性向上**: 編集コマンド（sed, awk, perl -i）をブロック
- 📋 **SystemPrompt改善**: ファイル編集時の明確なルール追加

### v0.3.0 (2026-01-06)
- ✨ スピナー表示機能
- ✨ 会話履歴の保存/復元（JSONL形式）
- ✨ 自動ページング（100行超え）
- ✨ `/save`, `/load`, `/sessions`コマンド
- ✨ `--resume`フラグ
- ✨ `str_replace`ツール

### v0.2.0
- 対話モード実装
- ツールシステム実装

### v0.1.0
- 初期リリース
