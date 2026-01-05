# XELYON CLI

AI搭載のコーディングアシスタントCLIツール

## 特徴

- 🤖 **複数のAIモデル対応**: DeepSeek V3 / Coder / R1、Claude対応予定
- 💬 **対話型エージェント**: ツールを使って実際にコード編集・Git操作を実行
- 📂 **会話履歴管理**: セッションをJSONL形式で保存・復元
- ⚡ **スピナー表示**: API呼び出し中の視覚的フィードバック
- 📄 **自動ページング**: 長い出力を読みやすく表示

## インストール

```bash
# ビルド
go build -o xelyon

# 環境変数設定
export DEEPSEEK_API_KEY="your-api-key"
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
/clear            - 会話履歴をクリア
/history          - 会話履歴を表示
/model            - 現在のモデルを表示
/help             - ヘルプを表示
/exit, /quit, /q  - 終了
```

### 利用可能なツール

AIが自動で以下のツールを使用します:

- **bash** - シェルコマンド実行
- **read_file** - ファイル読み込み
- **write_file** - ファイル作成・上書き
- **str_replace** - ファイル内の文字列置換（部分編集）
- **list_dir** - ディレクトリ一覧
- **git_status, git_diff, git_add, git_commit, git_push, git_log** - Git操作
- **search_code** - コード内検索（grep）
- **search_file** - ファイル名検索（find）

## プロジェクト設定

プロジェクトルートに`XELYON.md`を置くと、自動的にコンテキストとして読み込まれます。

```markdown
# プロジェクト設定

## 概要
このプロジェクトは...

## コーディングルール
- エラーハンドリング必須
- 日本語コメント可
```

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
