# XELYON CLI

AI搭載のコーディングアシスタントCLIツール

## 特徴

- 🤖 **複数のAIモデル対応**: DeepSeek V3 / Coder / R1、Claude対応予定
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

## インストール

```bash
# ビルド
go build -o xelyon

# 環境変数設定
export DEEPSEEK_API_KEY="your-api-key"
export SERPER_API_KEY="your-serper-api-key"  # Web検索用（オプション）
```

## 使い方

### 基本的な使い方

```bash
# 対話モード（新規セッション）
./xelyon

# ワンショットモード
./xelyon "このプロジェクトを説明して"

# 前回のセッションを復元
./xelyon --resume

# モデル選択
./xelyon --coder    # コード特化
./xelyon --think    # 深い推論
./xelyon --claude   # Claude (予定)
```

### 対話コマンド

```
/save             - 現在のセッションを保存
/load [id]        - セッションを読み込み（IDなしで最新）
/sessions         - 最近のセッション一覧
/undo             - 直前のファイル変更を取り消し
/config           - 設定の表示・変更（例: /config model deepseek-coder）
/model [name]     - 現在のモデルを表示、または再起動なしでモデル切り替え
/version          - バージョン情報を表示
/clear            - 会話履歴をクリア
/history          - 会話履歴を表示
/help             - ヘルプを表示
/exit, /quit, /q  - 終了
```

### 利用可能なツール

AIが自動で以下のツールを使用します:

- **bash** - シェルコマンド実行（sed/awk等の編集コマンドはブロック）
- **read_file** - ファイル読み込み
- **write_file** - ファイル作成・上書き（.bakバックアップ自動作成）
- **str_replace** - ファイル内の文字列置換（部分編集、diff表示、.bakバックアップ）
- **list_dir** - ディレクトリ一覧
- **git_status, git_diff, git_add, git_commit, git_push, git_log** - Git操作
- **search_code** - コード内検索（grep）
- **search_file** - ファイル名検索（find）
- **web_search** - Web検索（Serper API、上位5件）

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
# 現在のモデルを確認
> /model
🤖 Current model: DeepSeek Coder (code-focused)

# モデルを切り替え
> /model deepseek-chat
✅ Model switched: DeepSeek Coder (code-focused) → DeepSeek V3 (balanced)
💾 Default model saved to config

# 次の質問から新しいモデルが使われる
> こんにちは
```

**特徴:**
- 再起動不要でモデルを即座に切り替え
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
- **DeepSeek API** - AI推論（V3, Coder, R1）
- **fatih/color** - ターミナル色付け

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
