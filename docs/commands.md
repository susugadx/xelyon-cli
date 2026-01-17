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

### `/memory`

ユーザーメモリ機能を管理します。

#### メモリ追加
```
> /memory add <内容>
```

例:
```
> /memory add 私はReactとTypeScriptを使った開発を好みます
```

#### メモリ一覧
```
> /memory list
```

#### メモリ削除
```
> /memory delete <ID>
```

#### メモリクリア
```
> /memory clear          # 全メモリ削除
> /memory clear --project # プロジェクト別メモリのみ削除
```

### `/plan`

Plan Modeのオン/オフを切り替えます。Plan Modeでは、AIが実装前に計画を立てて確認を求めます。

```
> /plan         # トグル（オン⇔オフ）
> /plan on      # 明示的にオン
> /plan off     # 明示的にオフ
```

### `/copy`

最後のAI出力をクリップボードにコピーします。

```
> /copy
```

### `/compress`

会話履歴を圧縮してトークン数を削減します。

```
> /compress
```

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

### `/repomap`

リポジトリのコード構造マップを表示します。Tree-sitterで関数、クラス、構造体などのシンボルを抽出します。

```
> /repomap
```

### `/review`

コードレビューを実行します。セッション中の変更、git diff、または指定したファイルを静的解析してセキュリティ・品質の問題を検出します。

```
> /review                      # セッション中の変更をレビュー
> /review --all                # git diff の全変更をレビュー
> /review path/to/file.go      # 特定ファイルをレビュー
> /review **/*.go              # globパターンでレビュー
```

**フラグ:**
- `--all`, `-a`: git diff の全変更をレビュー
- `--security`, `-s`: セキュリティ問題に焦点
- `--test`, `-t`: テストカバレッジに焦点
- `--fix`, `-f`: 修正提案を生成・適用
- `--yes`, `-y`: 確認をスキップ（自動適用）
- `--ai`: LLMによるAI分析を有効化（静的ルールに加えてAIがコードを分析）
- `--max-issues <n>`: 表示する最大Issue数

**インタラクティブFix機能（--fix）:**
```
> /review --fix path/to/file.go
```
検出された問題に対して、自動修正可能なものはインタラクティブに確認:
- `y`: この修正を適用
- `n`: スキップ
- `a`: 残り全て適用
- `q`: 終了

**検出ルール:**
- セキュリティ: コマンドインジェクション、弱い暗号、HTTPタイムアウト未設定、パストラバーサル
- 一般: 大きすぎるdiff、TODO/FIXME追加、エクスポート関数のドキュメント欠落
- テスト: テストファイル欠落、アサーション欠落
- AI分析（`--ai`有効時）: LLMがコード全体を分析し、静的ルールでは検出できない問題を指摘

**AI分析機能（--ai）:**
```
> /review --ai main.go
> /review --ai --fix **/*.go  # AI分析 + 修正提案
```
- 現在のプロバイダー（DeepSeek, OpenAI, Claude等）を使用してコードを分析
- セキュリティ脆弱性、パフォーマンス問題、潜在的バグ、可読性の問題を検出
- `--fix`と組み合わせると、AIが修正コードも生成

**出力:**
- ターミナルにサマリー表示
- `~/.xelyon/reviews/` にMarkdownレポート保存

### `/refactor`

コードリファクタリング分析を実行します。大きすぎるファイル、長い関数、重複コード、命名の問題を検出し、改善を提案します。

```
> /refactor                       # カレントディレクトリを分析
> /refactor path/to/dir           # 特定ディレクトリを分析
> /refactor **/*.go               # globパターンで分析
```

**フラグ:**
- `--fix`, `-f`: 修正を適用（インタラクティブ確認）
- `--yes`, `-y`: 確認をスキップ（自動適用）
- `--ai`: AI分析を有効化（LLMがリファクタリングコードを生成）
- `--type`, `-t`: 特定タイプのみ検出（split-file, extract-method, dry, rename）
- `--max-file-lines <n>`: 大きいファイルの閾値（デフォルト: 300行）
- `--max-func-lines <n>`: 長い関数の閾値（デフォルト: 50行）

**検出タイプ:**
- `split-file`: 大きすぎるファイル（分割を提案）
- `extract-method`: 長すぎる関数（メソッド抽出を提案）
- `dry`: 重複コード（共通化を提案）
- `rename`: 命名の問題（より説明的な名前を提案）

**使用例:**
```
> /refactor --type split-file     # 大きいファイルのみ検出
> /refactor --ai --fix **/*.go    # AI分析 + 修正適用
> /refactor --max-file-lines 500  # カスタム閾値
```

**`--ai --fix` の動作:**
`--ai`フラグを有効にすると、LLMが実際のリファクタリングコードを生成します。
- `split-file`: ファイル分割プランを生成（新しいファイル名と内容）
- `extract-method`: 長い関数を複数の小さな関数に分割するコードを生成

`--fix`と組み合わせると、生成されたコードをインタラクティブに適用できます。

**テスト検証:**
`--fix`で変更を適用した後、Goファイルの場合は自動的にテスト実行を提案します。
テストが失敗した場合は、`/undo`でロールバックを提案します。

**自動コード健全性チェック（on_change）:**
ファイル変更時に自動的にコード健全性をチェックし、閾値超過時に警告を表示します。
設定は`~/.xelyon/config.yaml`の`code_health`セクションで管理できます。

```yaml
code_health:
  enabled: true           # 健全性チェックを有効化
  max_file_lines: 300     # ファイル行数上限
  max_function_lines: 50  # 関数行数上限
  auto_suggest: true      # 閾値超過時に自動で提案
  on_change:              # 変更時チェック項目
    - check_file_size
    - check_function_size
```

### `/dryrun`

Dry Runモードを切り替えます。有効にすると、AIがツールを呼び出しても実際には実行せず、シミュレート結果を返します。ファイル編集やGit操作を伴う提案を安全に確認したい場合に使用します。

```
> /dryrun         # 現在のステータス表示 + トグル
> /dryrun on      # 明示的にオン
> /dryrun off     # 明示的にオフ
```

**動作:**
- ツール実行は行われません（`tools.Execute()` を呼びません）
- 変更履歴（Undo対象）も作られません
- 履歴には `"[Dry Run] Tool execution simulated"` がツール結果として記録されます

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

# Plan Mode起動
xelyon --plan "ユーザー認証機能を追加"
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

# デバッグモード
export XELYON_DEBUG=1

# 対話的確認モード（ツール実行前に確認）
export XELYON_INTERACTIVE_CONFIRM=1
```

## 使用例

### 基本的な対話

```bash
xelyon
> main.goを読んで、バグがあれば修正して
```

### Plan Modeで実装

```bash
xelyon
> /plan on
> ユーザー認証機能を追加して
```

### プロバイダー切り替え

```bash
xelyon
> /use openai gpt-4
> この問題を詳しく分析して
> /use deepseek
> 同じ問題をもう一度分析して
```

### メモリ活用

```bash
xelyon
> /memory add プロジェクトはNext.js 14 + TypeScript
> /memory add Tailwind CSSを使用
> 新しいコンポーネントを作って
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
