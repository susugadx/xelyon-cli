# コマンド一覧

XELYON CLIで使用できる全コマンドのリファレンスです。

## 対話型コマンド

セッション中に `/` で始まるコマンドを入力できます。

### `/help`

利用可能なコマンド一覧を表示します。

```
> /help
```

### `/status`, `/stats`

現在状態、直近リクエスト、セッション統計をまとめて表示します。`/stats` は互換エイリアスです。
サブエージェントを使ったセッションでは、親とサブのコストを分離表示し、`🤖 Sub-agents` セクションで各サブのトークン使用量・コスト・ツール実行回数も確認できます。

```
> /status
> /stats   # alias
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

### `/paste`, `/p`

ペーストモードを開始します。WSLなどBracketed Paste Modeが動作しない環境で複数行入力を行う際に使用します。

```
> /paste
> /p  # 短縮形
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

**自動圧縮:** デフォルトで有効（Context 100K または 80% 到達時に自動実行）
- `prefer_compact_api: true` の場合、Compact API を優先使用
- `model: ""` の場合、圧縮時はプロバイダー別の低コストモデルを自動選択

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

設定を確認・変更します。対話式メニューで50以上の設定項目をカテゴリ別に管理できます。

```
> /config               # 対話式設定メニューを起動
> /config show          # 全設定を表示（デフォルトとの差分を ⚡ で表示）
> /config model <name>  # デフォルトモデルを変更
```

**対話式メニュー:**

`/config` を引数なしで実行すると、以下のような対話式メニューが表示されます：

```
┌─ Configuration (1/2) ─────────────────────┐
│ [1] 🤖 Provider & Model                   │
│ [2] 📦 Compression                        │
│ [3] 🔄 Loop Detection                     │
│ [4] 🌐 API Settings                       │
│ [5] 📝 Diff Display                       │
│ [6] ✅ Tool Confirm                       │
│ [7] 🔗 Command Aliases                    │
│ [8] 💨 Prompt Cache                       │
│ [9] 📋 Paste Mode                         │
│                                           │
│ [n] Next page                             │
│ [q] Cancel                                │
└───────────────────────────────────────────┘
```

**サポートする設定型:**

| 型 | 編集方法 | 例 |
|----|----------|-----|
| bool | y/n で切り替え | `thinking.enabled` |
| int | 数値入力 | `api_retry.count` |
| string | テキスト入力 | `default_model` |
| select | 番号選択 | `default_provider` (deepseek/claude/openai/...) |
| []string | 項目追加/削除 | `bash.safe_commands` |
| map[string]string | エントリ追加/編集/削除 | `command_aliases` |
| map[string]struct | サブメニューで編集 | `provider_models`, `lsp.servers` |

**20以上のカテゴリ:** Provider & Model, Compression, Loop Detection, API Settings, Diff Display, Tool Confirm, Command Aliases, Prompt Cache, Paste Mode, Streaming, Bash Safety, Project Map, Git Settings, Plan Mode, LSP Servers, OpenAI, Thinking, Output, Web Search, Utility Model など

**変更は即座に保存:** `~/.xelyon/config.yaml` に自動保存されます。

### `/init`

プロジェクトの設定ファイル（xelyon.yaml）のテンプレートを作成します。

```
> /init
```

**テンプレートに含まれるフィールド:**
- `context` — プロジェクトの概要・背景情報
- `rules` — 必須ルール（AI が必ず従うルール）
- `conditional` — `paths` に一致した時だけ注入する rules/context
- `ignore` — Project Map / `list_dir` / `search_code` で共有する ignore パターン
- `hooks` — 完了時フック・ステップ完了時フック（省略時は config.yaml の hooks を使用）

**注意:**
- コード構造の詳細な記載は不要
- Project Map は起動時に軽量 manifest を自動注入するため、ファイル一覧や関数目次は書かない
- `hooks.on_completion` を定義すると、AIが変更後に必ず実行します
- `hooks.on_step_complete` を定義すると、Plan Mode の各ステップ完了時に実行します（テンプレート変数: `{{step_id}}`, `{{step_description}}`, `{{step_status}}`）

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
1. **調査フェーズ**: SafetyHighツール（read_file, list_dir, search_code等）を自由に実行
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
| Claude | ✅ | 4.6: adaptive thinking + effort / 4.5以前: budget_tokens |
| OpenAI | ✅ | reasoning_effort パラメータ |
| Gemini | ✅ | thinkingConfig.thinkingBudget |
| DeepSeek | ✅ | deepseek-reasoner モデルに切り替え |
| Groq | ❌ | 警告表示（非対応） |
| Ollama | ⚠️ | モデル依存（R1/QwQ推奨） |

**対応モデル:**
- **Claude**: Sonnet 4 以降（Opus 4.6 / Sonnet 4.6 は adaptive thinking）
- **OpenAI**: gpt-5.2 系
- **Gemini**: 2.5 Pro 系（Flash は非対応）
- **DeepSeek**: 自動で reasoner モデルに切り替わります（`reasoning_content` を💭で表示、ツール実行フローでも保持）

**注意**: Extended Thinking はトークン消費量が増加します。

### `/lsp`

LSPサーバーのステータスを表示・管理します。

```
> /lsp              # ステータス表示
> /lsp status       # 同上
> /lsp detect       # プロジェクト内の言語を検出
> /lsp install go   # 指定言語のLSPサーバーをインストール
> /lsp install all  # 未インストールの全サーバーをインストール
```

**出力例:**
```
LSP Server Status / LSPサーバー状態
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✅ Go: gopls (running)
  ✅ TypeScript: vtsls (running)
  ⏸️ Python: pyright (installed, idle)
  ❌ Rust: rust-analyzer (not installed)
```

**サブコマンド:**

| コマンド | 説明 |
|---------|------|
| `/lsp` または `/lsp status` | 現在のLSPサーバー状態を表示 |
| `/lsp detect` | プロジェクト内の言語を検出して表示 |
| `/lsp install <言語>` | 指定言語のLSPサーバーをインストール |
| `/lsp install all` | 未インストールの全サーバーをインストール |

**対応言語:** Go, TypeScript/JavaScript, Python, Rust, Java, C/C++, Ruby, Kotlin, Swift, C#, Scala, PHP, Elixir, Lua, CSS/SCSS, HTML, Vue, Svelte, YAML, TOML, SQL, Bash, Markdown（23言語）

詳細は [LSP連携ガイド](lsp.md) を参照してください。

### `/version`

XELYONのバージョン情報を表示します。

```
> /version
```

**出力例:**
```
XELYON CLI v0.28.3
```

### `/exit`, `/quit`, `/q`

セッションを終了します。`Ctrl+D`、`Ctrl+C`、または `exit` でも終了できます。

```
> /exit
> /quit  # エイリアス
> /q     # 短縮形
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

## 利用可能なツール

AIが自動で以下のツールを使用します。ユーザーが直接呼び出す必要はありません。

デフォルトでは編集系ツールとして `apply_patch` のみを公開します。
開発デバッグ用に `XELYON_EDIT_TOOL=str_replace xelyon` で起動すると、legacy `str_replace` / `write_file` / `delete_file` を再度公開できます。

### ファイル操作

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `apply_patch` | Codex 互換の差分パッチでファイルを作成・更新・削除する既定編集ツール。1回で複数ファイルを扱える | `patch` |
| `read_file` | ファイル内容を読み込む。Go ファイルは `symbol` で関数/型/メソッド単位の読み出しも可能 | `path`, `symbol`, `start_line`, `end_line`, `paths` |
| `write_file` | legacy edit mode 用。ファイルを新規作成・上書き | `path`, `content` |
| `str_replace` | legacy edit mode 用。文字列置換でファイル編集（old_str優先。old_str空+start_line/end_line指定で行レンジ置換も可。Go ファイルは書き込み前に AST 構文チェックを実施） | `path`, `old_str`, `new_str`, `start_line`, `end_line` |
| `delete_file` | legacy edit mode 用。ファイルを削除 | `path` |
| `list_dir` | ディレクトリ一覧取得 | `path` |

**Note**: ファイル操作（mkdir, cp, mv, diff等）は `bash` ツールで実行可能です。

`read_file` の `symbol` 例:

```json
{"tool":"read_file","args":{"path":"internal/agent/agent.go","symbol":"maybeAutoCompress"}}
```

複数シンボルを読む場合はカンマ区切りで指定します。

### Git操作

すべてのGit操作は `bash` ツールで実行します。

```bash
# 例
bash: git status
bash: git diff
bash: git add -A && git commit -m "message"
bash: git checkout -b feature-branch
```

### コード調査・検索

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `inspect_symbol` | 既知シンボルの定義・caller・参照・テストをまとめて取得（Go ファイル対応） | `symbol`, `path`, `mode` |
| `search_code` | コード検索（正規表現・複数パターン・結果分類） | `pattern`, `path`, `file_type` 等 |
| `web_search` | ネイティブWeb検索（`web_search.provider` で OpenAI / Gemini / Claude を選択可能） | `query` |
**注意**: メインプロバイダーがネイティブ検索非対応（DeepSeek / Groq / Ollama / OpenRouter など）の場合は、`config.yaml` で `web_search.provider` を設定してください。詳細は[config.md - Web検索](config.md#web検索)を参照してください。

### 開発支援

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `bash` | シェルコマンド実行 | `command` |

テスト、フォーマット、Lint はすべて bash で実行:
```bash
bash: go test ./...
bash: go fmt ./...
bash: golangci-lint run
```

### LSP（言語サーバー）

LSPは診断（エラー検知）、削除時参照チェック、Plan依存分析で内部的に利用されます。
コード検索には `search_code` ツールを使用してください。
詳細は [LSP連携ガイド](lsp.md) を参照してください。

### 使用例

AIは自然言語の指示に基づいてツールを自動選択します。

```bash
> main.goを読んで
# → read_file が実行される

> バグを修正して
# → read_file / search_code → apply_patch が実行される

> 複数ファイルをまとめて編集して
# → apply_patch が1回で複数ファイルに適用される

> git statusを見せて
# → bash で git status が実行される

> テストを実行して
# → bash で go test が実行される

> TODOを探して
# → search_code が実行される
```

**apply_patch の補足**
- `*** Begin Patch` / `*** End Patch` の境界が必須です
- `*** Add File:` / `*** Update File:` / `*** Delete File:` を1つ以上含めます
- `@@` と前後3行のコンテキストで変更位置を特定します
- 1つの patch で複数ファイルの追加・更新・削除をまとめて実行できます
- パスは相対パスのみを使います

**legacy str_replace の補足**
- `old_str` が非空のときは従来どおり文字列置換を行い、`start_line`/`end_line` は無視されます
- 行レンジ置換は `old_str: ""` かつ `start_line` と `end_line` を両方指定した場合のみ有効です（1-indexed, inclusive）
- Go ファイルでは置換後の内容を AST パースし、構文エラーが見つかった場合は警告を表示して tool result にも付与します
- AST 警告が出ても `str_replace` 自体は停止せず、ユーザー確認済みまたは auto-approve の場合はそのまま書き込みます

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
- [LSP連携](lsp.md)
- [MCP連携](mcp.md)

## 未ドキュメント化コマンド（自動追加）

<!-- TODO: 以下のコマンドに詳細な説明を追加してください -->

### `/project`

Edit xelyon.yaml interactively (rules, hooks)

```
> /project
```


## 未ドキュメント化コマンド（自動追加）

<!-- TODO: 以下のコマンドに詳細な説明を追加してください -->
