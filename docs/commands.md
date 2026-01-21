# コマンド一覧

XELYON CLIで使用できる全コマンドのリファレンスです。

## 対話型コマンド

セッション中に `/` で始まるコマンドを入力できます。

### `/help`

利用可能なコマンド一覧を表示します。

```
> /help
```

### `/stats`

現在のセッション統計を表示します（ユーザーメッセージ数、アシスタントメッセージ数、ツール実行回数など）。

```
> /stats
```

### `/save`

現在のセッションを保存します。

```
> /save
```

### `/load`

保存されたセッションを読み込みます。引数なしで最後のセッションを読み込みます。

```
> /load              # 最後のセッションを読み込み
> /load <session-id> # 指定したセッションを読み込み
```

### `/sessions`

保存されたセッション一覧を表示します（最新10件）。

```
> /sessions
```

### `/clear`

会話履歴をクリアします。

```
> /clear
```

### `/history`

現在の会話履歴を表示します。

```
> /history
```

### `/model`

現在のモデルを表示、または切り替えます。

```
> /model              # 現在のモデルを表示
> /model <model-name> # モデルを切り替え
```

例:
```
> /model gpt-4o
> /model claude-sonnet-4-5-20250514
> /model gemini-2.0-flash-exp
```

### `/paste`

ペーストモードを開始します。WSLなどBracketed Paste Modeが動作しない環境で複数行入力を行う際に使用します。

```
> /paste
```

入力が完了したら空行を2回入力（Enter2回）で終了します。

### `/copy`

最後のAI出力をクリップボードにコピーします。

```
> /copy
```

### `/tokens`

トークン使用量とコンテキストウィンドウの状態を表示します。

```
> /tokens
```

**出力例:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Token Usage / トークン使用量
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [████████░░░░░░░░░░░░░░░░░░░░░░] 25.3%

  Current: 50,600 / 200,000 tokens

📋 Breakdown:
    System Prompt: 8,500 tokens (4.3%)
    History:       42,100 tokens (21.1%)  [38 messages]

🤖 Model:
    claude-sonnet-4-20250514 (context: 200,000 tokens)

⚙️  Auto-compress:
    ON (threshold: 80%)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**表示内容:**
- 使用率バー（色分け: 緑 < 80% < 黄 < 90% < 赤）
- 現在のトークン数 / モデルの上限
- 内訳（System Prompt、会話履歴）
- 使用モデルとコンテキストサイズ
- 自動圧縮の状態

### `/compress`

会話履歴を圧縮してトークン数を削減します。

```
> /compress           # LLMサマリーで圧縮（最新10件保持）
> /compress 20        # 最新20件を保持して圧縮
> /compress --compact # OpenAI Compact API で圧縮
> /compress -c        # 同上（短縮形）
```

**動作（デフォルト）:**
1. 古いメッセージをAIで要約
2. 要約を先頭に配置
3. 最新N件のメッセージを保持（デフォルト: 10件）

**Compact API モード (`--compact` / `-c`):**
- OpenAI Responses API の `/responses/compact` エンドポイントを使用
- ユーザーメッセージはそのまま保持（verbatim）
- アシスタント応答は暗号化された圧縮データに置換
- **制限**: OpenAI プロバイダーかつ Responses API 対応モデルのみ

**自動圧縮:** デフォルトで有効（80%到達時に自動実行）
- `prefer_compact_api: true` の場合、Compact API を優先使用

### `/use <provider> [model]`

プロバイダーとモデルを動的に切り替えます。

```
> /use deepseek
> /use openai gpt-4
> /use gemini gemini-2.0-flash-exp
> /use claude claude-sonnet-4-5-20250514
> /use ollama qwen2.5-coder:7b
> /use groq meta-llama/llama-4-scout-17b-16e-instruct
```

### `/providers`

利用可能なプロバイダーとモデル一覧を表示します。

```
> /providers
```

### `/config`

設定を確認・変更します。

```
> /config show              # 全設定を表示（デフォルトとの差分を ⚡ で表示）
> /config set <key> <value> # 設定を変更
> /config reset <key>       # 設定をデフォルトに戻す
```

例:
```
> /config show
> /config set tool_confirm.auto_approve_safe false
> /config reset tool_confirm.auto_approve_safe
```

### `/changes`

セッション中のファイル変更履歴を表示します。

```
> /changes
```

### `/undo`

最後のファイル変更を取り消します（バックアップから復元）。

```
> /undo
```

### `/init`

プロジェクトの設定ファイル（XELYON.md）を対話的に生成します。既存のコードベースをTree-sitterで解析し、技術スタックやディレクトリ構造を自動検出します。

```
> /init
```

**生成される内容:**
- プロジェクト概要
- 技術スタック（自動検出）
- ディレクトリ構造
- コーディング規約
- コードマップ（主要なシンボル一覧）

**対話フロー:**
1. コードベース解析（自動）
2. プロジェクト名入力
3. プロジェクト概要入力
4. コーディング規約入力（オプション）

### `/sync`

既存のXELYON.mdを現在のコードベースと同期します。新しいファイルや削除されたファイル、技術スタックの変更を検出して更新します。

```
> /sync
```

**検出される変更:**
- 技術スタック（追加/削除）
- ファイル（新規作成/削除）
- シンボル数の変化

**フロー:**
1. 現在のXELYON.mdをパース
2. コードベースを再解析
3. 差分を表示
4. 確認後に更新

### `/plan`

Plan Modeを切り替えます。有効にすると、リクエストが「調査→計画→承認→実行」のフローで処理されます。

```
> /plan           # 現在のモード表示
> /plan on        # Plan Mode 有効化
> /plan off       # 通常モードに戻る
> /plan status    # ステータス表示
```

**デフォルト:** OFF（通常モード）

**通常モード（OFF）:**
- ツールを個別に確認しながら実行
- 軽いタスクにはオーバーヘッドなく即座に応答
- `tool_confirm` 設定に従ってツール確認

**Plan Mode（ON）:**
1. **調査フェーズ**: SafetyHighツール（read_file, search_code等）を自由に実行
2. **計画生成**: 実装が必要な場合、ステップをJSONで出力
3. **承認**: ユーザーが計画を確認・承認
4. **実行**: ステップごとに失敗検知・リトライ付きで実行

**ステータス表示:**
```
[Status] waiting_input | Mode: Normal | Ready / 入力待ち
[Status] waiting_input | Mode: 📋 Plan | Ready / 入力待ち
```

### `/think`

Extended Thinking（推論モード）を切り替えます。複雑なタスクでより深い推論を行う際に使用します。

```
> /think              # 現在の状態表示
> /think on           # 有効化（現在のレベルで）
> /think off          # 無効化
> /think low          # 低レベルで有効化
> /think medium       # 中レベルで有効化（デフォルト）
> /think high         # 高レベルで有効化
> /think xhigh        # 最高レベルで有効化
```

**対応プロバイダー:**

| プロバイダー | 対応 | 動作 |
|-------------|------|------|
| Claude | ✅ | thinking.budget_tokens パラメータ |
| OpenAI | ✅ | reasoning_effort パラメータ |
| Gemini | ✅ | thinkingConfig.thinkingBudget |
| DeepSeek | ✅ | deepseek-reasoner モデルに切り替え |
| Groq | ❌ | 警告表示（非対応） |
| Ollama | ⚠️ | モデル依存（R1/QwQ推奨） |

**対応モデル:**
- **Claude**: Sonnet 4 以降
- **OpenAI**: gpt-5.2 系
- **Gemini**: 2.5 Pro 系（Flash は非対応）
- **DeepSeek**: 自動で reasoner モデルに切り替わります

**注意**: Extended Thinking はトークン消費量が増加します。

### `/version`

XELYONのバージョン情報を表示します。

```
> /version
```

**出力例:**
```
XELYON CLI v0.28.3
```

### `/exit`

セッションを終了します。`Ctrl+D`、`Ctrl+C`、または `exit` でも終了できます。

```
> /exit
```

## CLIフラグ

起動時に指定できるオプションです。

### プロバイダー/モデル指定

```bash
# プロバイダー指定
xelyon --provider deepseek
xelyon --provider openai
xelyon --provider gemini
xelyon --provider claude
xelyon --provider ollama
xelyon --provider groq

# モデル指定
xelyon --provider openai --model gpt-4
xelyon -m gemini-2.0-flash-exp
```

### 初期プロンプト

```bash
xelyon "main.goを読んで説明して"
```

### 画像入力（マルチモーダル）

```bash
# 画像ファイルを指定して起動
xelyon -i wireframe.png "この画面をReactで実装して"
xelyon --image error.png "このエラーを修正して"

# プロバイダー指定と組み合わせ
xelyon --image screenshot.png --provider gemini "このUIの問題点を教えて"
```

**対応フォーマット**: PNG, JPEG, GIF, WebP
**対応プロバイダー**: Gemini, Claude, OpenAI（DeepSeek, Ollama, Groqは非対応）

### その他のオプション

```bash
# バージョン表示
xelyon --version

# ヘルプ表示
xelyon --help

# セッション再開
xelyon --resume <session-id>

# プロジェクト別セッション
xelyon --project

# 圧縮せずにセッション開始（デバッグ用）
xelyon --no-compress

# ページャーを無効化
xelyon --no-pager

# Headlessモード（JSON出力、対話なし）
xelyon --headless "main.goを読んで説明して"
xelyon --output-format json "バグを修正して"

```

## 環境変数

詳細は [config.md](config.md) を参照してください。

```bash
# API キー
export DEEPSEEK_API_KEY=sk-...
export OPENAI_API_KEY=sk-...
export GEMINI_API_KEY=...
export ANTHROPIC_API_KEY=sk-ant-...
export GROQ_API_KEY=gsk_...

# 対話的確認モード（ツール実行前に確認）
export XELYON_INTERACTIVE_CONFIRM=1
```

## 使用例

### 基本的な対話

```bash
xelyon
> main.goを読んで、バグがあれば修正して
```

### プロバイダー切り替え

```bash
xelyon
> /use openai gpt-4
> この問題を詳しく分析して
> /use deepseek
> 同じ問題をもう一度分析して
```

### 変更管理

```bash
xelyon
> ファイルを編集して
> /changes      # 変更履歴確認
> /undo         # 最後の変更を取り消し
```

## 利用可能なツール

AIが自動で以下のツールを使用します。ユーザーが直接呼び出す必要はありません。

### ファイル操作

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `read_file` | ファイル内容を読み込む | `path` |
| `write_file` | ファイルを新規作成・上書き | `path`, `content` |
| `str_replace` | 文字列置換でファイル編集（old_str優先。old_str空+start_line/end_line指定で行レンジ置換も可） | `path`, `old_str`, `new_str`, `start_line`, `end_line` |
| `append_file` | ファイル末尾に追記 | `path`, `content` |
| `prepend_file` | ファイル先頭に追記 | `path`, `content` |
| `insert_after` | パターンの後に挿入 | `path`, `pattern`, `content` |
| `insert_before` | パターンの前に挿入 | `path`, `pattern`, `content` |
| `copy_file` | ファイルをコピー | `src`, `dest` |
| `move_file` | ファイルを移動 | `src`, `dest` |
| `delete_file` | ファイルを削除 | `path` |
| `delete_lines` | 指定行を削除 | `path`, `start_line`, `end_line` |
| `list_dir` | ディレクトリ一覧取得 | `path` |
| `create_dir` | ディレクトリ作成 | `path` |
| `restore_backup` | バックアップから復元 | `path`, `backup_path` (オプション) |
| `list_backups` | バックアップ一覧表示 | `path` |
| `grep_replace` | 複数ファイルで一括置換 | `pattern`, `replacement`, `path`, `file_pattern`, `dry_run` |
| `diff_files` | 2つのファイルの差分を表示 | `file1`, `file2`, `context` (オプション) |

### HTTP Client

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `http_request` | HTTPリクエストを実行 | `method`, `url`, `headers`, `body`, `timeout` |

**使用例:**
```
# GET リクエスト
{"tool": "http_request", "args": {"method": "GET", "url": "https://api.example.com/users"}}

# POST リクエスト（JSON）
{"tool": "http_request", "args": {"method": "POST", "url": "https://api.example.com/users", "body": "{\"name\": \"test\"}", "headers": "{\"Authorization\": \"Bearer token\"}"}}
```

### Git操作

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `git_status` | 現在の変更状況を表示 | - |
| `git_diff` | 差分を表示 | `path` (オプション) |
| `git_add` | ステージに追加（複数ファイルはバッチ確認UI） | `path`（スペース/カンマ区切りで複数可） |
| `git_commit` | コミット作成 | `message` |
| `git_push` | リモートにプッシュ | - |
| `git_log` | コミット履歴を表示 | - |
| `git_branch` | ブランチ一覧表示 | - |
| `git_checkout` | ブランチ切り替え | `branch` |
| `git_stash` | 変更を一時退避 | `action` (save/pop/list) |

### 検索

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `search_code` | コード内を正規表現検索 | `pattern`, `path` |
| `search_file` | ファイル名検索 | `pattern`, `path` |
| `ast_grep` | 構造的コード検索（ast-grep使用） | `pattern`, `path`, `lang` |
| `web_search` | Web検索（Serper API、要APIキー） | `query` |

**注意**: `web_search`を使用するには`SERPER_API_KEY`環境変数の設定が必要です。詳細は[config.md - Web検索（Serper API）](config.md#web検索serper-api)を参照してください。

### 開発支援

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `bash` | シェルコマンド実行 | `command` |
| `run_test` | テスト実行 | `path` |
| `format` | コードフォーマット | `path` |
| `lint` | Lint実行 | `path`, `auto_fix` |

### 使用例

AIは自然言語の指示に基づいてツールを自動選択します。

```bash
> main.goを読んで
# → read_file が実行される

> バグを修正して
# → read_file → str_replace → write_file が実行される

> 10-20行目を指定の内容に置き換えて
# → str_replace（old_str を空にして start_line/end_line を指定） が実行される

> git statusを見せて
# → git_status が実行される

> テストを実行して
# → run_test が実行される

> search_codeで"TODO"を探して
# → search_code が実行される
```

**str_replace の補足**
- `old_str` が非空のときは従来どおり文字列置換を行い、`start_line`/`end_line` は無視されます
- 行レンジ置換は `old_str: ""` かつ `start_line` と `end_line` を両方指定した場合のみ有効です（1-indexed, inclusive）

### MCP対応ツール

XELYONはMCP（Model Context Protocol）に対応しており、`~/.xelyon/mcp.json`で外部ツールを追加できます。

詳細は [MCP連携ガイド](mcp.md) を参照してください。

## 高度な機能

### Headlessモード

対話なしでJSON形式で結果を出力します。他のツールやスクリプトから呼び出す際に便利です。

```bash
# JSON出力
xelyon --headless "main.goを読んで概要を説明して"

# 出力例
{
  "query": "main.goを読んで概要を説明して",
  "response": "このファイルは...",
  "success": true
}

# jqと組み合わせて
xelyon --headless "バグを修正して" | jq -r '.response'

# CI/CDパイプラインで使用
xelyon --output-format json "テストを実行して" | jq '.success'
```

### Repo Map（コード構造解析）

Tree-sitterを使ってプロジェクトのコード構造を自動解析します。

**対応言語**: Go, JavaScript, TypeScript, Python

**抽出される情報**:
- 関数、メソッド
- 構造体、クラス、インターフェース
- 型定義

**自動生成**: 起動時にプロジェクトをスキャンし、AIに提示されます。

**除外パターン**: `node_modules`, `.git`, `vendor`, `dist`, `build` などは自動除外。

```bash
# Repo Mapを確認
xelyon
> このプロジェクトの構造を教えて

# AIがRepo Mapを使って構造を説明
```

### 対話的確認モード

ツール実行前に毎回確認プロンプトを表示します（デバッグ・学習用）。

```bash
export XELYON_INTERACTIVE_CONFIRM=1
xelyon

> main.goを編集して

# 各ツール実行前に確認
# [y] 実行 / [n] スキップ / [c] フィードバック付き / [s] 終了
```

## 関連ドキュメント

- [プロバイダー設定](providers.md)
- [設定リファレンス](config.md)
- [MCP連携](mcp.md)
