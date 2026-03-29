# XELYON CLI

AI搭載のコーディングアシスタントCLI

[![CI](https://github.com/susugadx/xelyon-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/susugadx/xelyon-cli/actions/workflows/ci.yml)
[![E2E](https://github.com/susugadx/xelyon-cli/actions/workflows/e2e.yml/badge.svg)](https://github.com/susugadx/xelyon-cli/actions/workflows/e2e.yml)
[![codecov](https://codecov.io/gh/susugadx/xelyon-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/susugadx/xelyon-cli)
[![Go Version](https://img.shields.io/github/go-mod-go-version/susugadx/xelyon-cli)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/susugadx/xelyon-cli)](https://github.com/susugadx/xelyon-cli/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 特徴

### 💬 自然言語でコーディング
コマンドを覚える必要なし。日本語で指示するだけ。
- 「このファイルのバグ直して」
- 「テスト書いて」
- 「リファクタリングして」
- 「git commitして」

**差分を見せて確認してから実行（編集・bash・gitなど）**

### 🌐 8種類のLLMプロバイダー
DeepSeek, OpenAI, Gemini, Claude, Ollama, Groq, OpenRouter, Bedrock をシームレスに切り替え。
ローカルLLM（Ollama）も対応で、オフラインでも使用可能。

**OpenAI Responses API 対応**: `gpt-5.2-codex` などの Codex モデルを自動検出し、最適なAPIを選択。
**DeepSeek Reasoner 対応**: `reasoning_content`（思考内容）のストリーミング表示・ツール実行フローでの保持に対応。
**プロバイダー別プロンプト最適化**: OpenAI / Gemini では短い実況を促し、Gemini など特定モデルのルール遵守を強化するプレフィックスを自動注入。
**Gemini FCリトライ**: FC失敗時にテキストモードではなくFCモードでリトライ（キャッシュ汚染防止）。idle timeout / thinking timeout / 一般エラーそれぞれで上限付きリトライ。
**FC rescue JSON修復**: テキストモードで抽出されたツールJSONに生制御文字（改行・タブ等）が含まれる場合、自動修復してパース成功させる。

### 🛠️ 組み込みツール
- **ファイル操作**: 編集ツールは provider/model に応じて自動切替。OpenAI / Gemini 系は Codex 互換の `apply_patch`、Claude / DeepSeek 系は旧 `str_replace` / `write_file` / `delete_file` を使います。OpenRouter は model family を見て判定し、`XELYON_EDIT_TOOL=str_replace` などの明示 override がある場合はそれを最優先します
- **コード検索**: `search_code` は language-aware router として動作し、`mode=auto` を既定に symbol-aware / literal / regex の各レーンを内部選択（複数パターン、結果分類、不正regex検出にも対応）
- **シンボル調査**: `search_code` は短い symbol query を優先し、対応言語では定義・caller・参照・関連テストをまとめて返却。Go は first-class に `Config.Build` / `(*Config).Build` や regex っぽい query の rescue も吸収
- **サブエージェント委譲**: `spawn_agent` / `wait_agent` で探索タスクを別コンテキストの軽量モデルへ委譲し、親には最終レポートだけを返す
- **AST基盤（実験的）**: `internal/ast` に Pure Go Tree-sitter（gotreesitter）ベースの共通解析基盤を追加。Phase 1 では Go ファイルのパース、シンボル抽出、行分類を検証段階で提供し、`read_file(symbol=...)` でシンボル範囲の読み出しに利用。legacy `str_replace` では Go ファイル書き込み前に AST 構文検証を行い、問題があれば警告を返す
- **開発支援**: bash（git, テスト, フォーマット等すべて対応）
- **LSP連携**: シンボル検索（定義・参照・実装）

### 📋 確認ベースの安全設計
- 安全なツール（ファイル読み取り等）は自動実行
- 危険なツール（ファイル編集、bash、Web検索等）は毎回確認
- `--auto-approve`で信頼環境向け全ツール自動承認（SafetyLow含む）
- **既定編集フロー**: `search_code` / `read_file` で文脈を集めてから `apply_patch` を構築し、差分確認のうえで適用。legacy edit mode では従来どおり `str_replace` / `write_file` / `delete_file` を使用可能

### 📋 Plan Mode（オプショナル）
`/plan on` で有効化するとPlan Mode経由で処理されます。
1. **調査フェーズ**: コードベースを調査（SafetyHighツールを自動実行）
2. **単純なQ&A**: 調査のみで回答可能な場合はそのまま終了
3. **計画生成**: 実装が必要な場合、ステップを JSON で出力
4. **承認**: ユーザーが計画を確認・承認
5. **コンテキスト軽量化**: 承認後は調査フェーズの履歴をクリアし、承認済みプラン要約だけを残して実装に移行
6. **実行**: ステップごとに失敗検知・リトライ付きで順次実行

デフォルトは通常モード（ツール個別確認）。軽いタスクにはオーバーヘッドなく即座に応答。

### 🔄 自動リトライ機能
ツール実行が失敗した場合、自動的にリトライして成功するまで試行します。
- **デフォルト10回**のリトライ（`plan_mode.auto_retry: 10`で設定可能）
- Plan Mode と通常モード両方で有効
- リトライ中: `❌ Failed (retry 1/10)` → `🔄 Retrying...`
- 成功時: `✅ Succeeded (on retry 3)`
- 上限到達時: Selector UI で継続/中止を選択（Plan Mode のみ）

### ⚡ Parallel Tool Execution
LLMが1回の応答で複数のread-onlyツールを返した場合、並列実行してレイテンシを削減します。
- **parallel-safe ツール**: `read_file`, `list_dir`, `search_code`, `web_search`, `git_status` 等
- **bash**: `ls`, `find`, `rg`, `grep`, `cat`, `git status`, `git diff`, `git log` 等のread-onlyコマンド（allowlist ベース）のみ並列化
- **sequential ツール**: `apply_patch`, `write_file`, `str_replace`, `delete_file`, MCP ツール等の副作用ありツールは従来通り順次実行
- mixed case: parallel-safe 群を先に並列実行 → sequential 群を順次実行 → 結果は元の tool call 順で配送
- 最大並列数: 4（セマフォ制御、固定値）
- 通常モード・Plan Mode 両方で同じポリシーを適用（`executeToolCallsWithParallel` 共通 executor）
- **安全性**: ループ検知・deprecated ツールフィルタリングは実行前（Phase 0）に評価。Investigation Phase は独自ループ（`executeToolOnly`）を使用し並列実行の対象外
- **stdout 制御**: 並列パスでは `ExecuteQuiet()` を使用し wrapper 層の出力（ヘッダー・引数・折りたたみ結果）を抑制。`Tool.Run()` 内部の直接 stdout 出力（例: `read_file` の `📄 Read: ...`）は抑制できない（Tool interface が io.Writer を受け取らないため）。parallel-safe ツールの内部出力は少量のステータス行のみであり、現時点で致命的ではないと判断
- **キャンセル（best effort）**: `context.Context` は goroutine 起動前・実行前にチェック。`Tool.Run()` は ctx を受け取らないため、実行開始後の中断は不可。cancel は best effort であり、fully cancellable ではない
- **ExecutionContext**: process-global 単一変数（`sync.RWMutex` で race-safe）。並列 batch 開始前に Set → goroutine は Read のみ → batch 完了後に Clear。ただし race-safe であることと multi-owner-safe であることは異なり、同一プロセス内で複数 Agent が同時に異なる ExecutionContext を必要とする設計には対応していない。single-agent CLI 前提で許容
- **FC / text-based 差異**: FC（tool_call_id あり）はループ中断時に role=tool で応答。text-based は role=user。text-based の後続ツールにはダミーメッセージを追加しない（intentional spec — 旧 sequential 実装との互換性を by design で維持）

### 🤖 サブエージェント委譲
探索・調査タスクは `spawn_agent` / `wait_agent` で軽量サブエージェントへ委譲できます。
- **コンテキスト分離**: 親に返るのはサブの最終レポートだけ。`read_file` 全文や `search_code` の中間結果は親コンテキストへ再注入されません
- **リアルタイム可視化**: `wait_agent` 実行中はサブエージェントのツール実行を親UIへ逐次表示し、編集系ツールの適用結果も追跡できます
- **既定モデル**: `sub_agent.default_model` が空ならメイン provider の最安モデルを自動選択します。明示設定するとそのモデルを優先します
- **推論強度**: 既定は off（`sub_agent.default_effort` で low / medium / high を指定可能）
- **同時実行数**: 既定 5（`sub_agent.max_concurrent`）
- **再帰禁止**: サブエージェント自身には `spawn_agent` / `wait_agent` を渡しません
- **コスト透明性**: `/status` で親セッションとサブエージェントのトークン使用量・コストを分離表示し、合算コストも確認できます

### 🔍 コードレビュー & リファクタリング
`/review` でセキュリティ・テストカバレッジをチェック。
`/refactor` で静的解析ベースのリファクタリング提案。

### 📋 複数行ペースト対応
コードを直接ペーストして処理。Bracketed Paste Mode で改行を含むテキストも1つの入力として認識。
- **そのままペースト**: ターミナルで Cmd+V / Ctrl+V するだけ
- **` ``` ` モード**: 明示的に複数行入力を開始
- **`/paste` コマンド**: Bracketed Paste が使えない環境向け

> WSL等で問題がある場合: `XELYON_BRACKETED_PASTE=0` で無効化可能

### 🖼️ マルチモーダル対応
画像ファイルを指定してUIデザインからコード生成。
エラースクリーンショットから原因分析も可能。

### 🔌 LSP連携（IDE並みのコード理解）
Language Server Protocol (LSP) を活用してIDE並みのコード理解を実現。
- **削除時参照チェック**: ファイル削除前に外部参照を自動検出し警告
- **完了検証**: AI が「完了」「done」と宣言した際、変更ファイルの LSP 診断を自動実行しエラー残存時は修正を続行
- **Completion Hooks**: LSP チェック後にユーザー定義のシェルコマンド（`go test ./...` 等）を自動実行。失敗時は AI が修正を続行（`hooks.on_completion` で設定）
- **自動検出**: プロジェクト内の言語を自動検出し、未インストールのLSPサーバーを提案
- **ワンクリックインストール**: `/lsp install <言語>` でLSPサーバーをインストール
- **23言語対応**:
  - **メイン**: Go, TypeScript/JavaScript (React/JSX), Python, Rust
  - **バックエンド**: Java, C/C++, Ruby, Kotlin, Swift, C#, Scala, PHP, Elixir, Lua
  - **フロントエンド**: CSS/SCSS, HTML, Vue, Svelte
  - **設定/スクリプト**: YAML, TOML, SQL, Bash, Markdown

### 📊 Context Window 管理
長時間の会話でもトークン上限を気にせず作業。
- **`/tokens`**: 現在のトークン使用量と上限を確認
- **Project Map 自動注入**: 起動時は root manifest 寄りの軽量マップだけを注入し、詳細シンボルは必要時に取得。大規模 repo でも固定コストを抑制
- **自動圧縮**: Context 100K または 80% 到達で自動的に履歴を圧縮（デフォルトON）
- **圧縮専用モデル**: OpenAI は GPT-5 Mini、Gemini は Flash-Lite、Claude/Bedrock は Haiku で低コスト圧縮
- **手動圧縮**: `/compress [N]` で履歴を圧縮（最新N件を保持）
- **OpenAI Compact API**: `/compress --compact` でOpenAI独自の圧縮（ユーザーメッセージ保持）
- **80%/90%警告**: 上限接近時に自動で警告表示
- **トークン上限エラー時の提案**: エラー発生時に `/compress` または `/clear` を案内
- **Assistant narration clearing**: ツール呼び出し時に表示した短い実況テキストは History 保存時に自動削除し、入力トークンの再送を抑制
- **ツール結果の自動truncate**: 3ターン以上前のツール結果（50行超）を送信時に自動圧縮（先頭20行+末尾5行を保持）。元の履歴は保持され、API送信時にのみ適用
- **Claude系の server-side tool clearing**: Claude / Bedrock / OpenRouter(Claude models) では `clear_tool_uses` により古い `tool_use` / `tool_result` ペア構造をサーバー側で削減し、compaction 発動前に入力トークンを節約
- **プロンプトキャッシュ最適化**: Claude/Bedrock利用時、安定区間の末尾userメッセージにBPを配置し、古い履歴のキャッシュHIT率を向上（`prompt_cache.enabled: true`で有効）。Opus 4.6の最低キャッシュトークン数（4096）に対応するため、system promptの最終ブロックにcache_controlを配置
- **Long Context 料金自動判定**: Claude/Gemini Pro で200Kトークン超のリクエスト時、long context 料金ティアを自動適用。キャッシュトークン（cache_read + cache_creation）も含めた総入力トークンでティア判定

### 📈 リアルタイムトークン表示
API実測値に基づくトークン使用量とコストをリアルタイム表示。
- **ステータスバー**: プロンプト直前に `● model │ Mode │ tokens/limit │ ~$cost` を表示
- **起動時コンテキスト表示**: ツリー形式で初期コンテキストの内訳を表示
- **ナビゲーション削減**: Project Map + `search_code` の symbol-aware routing により `read_file` の往復を減らし、編集に集中
- **リクエスト完了時**: `✓ In: 1,234 + Out: 567 = 1,801 tok (~$0.002)` で使用量を表示
- **Ollama対応**: ローカル実行時はコスト表示を非表示
- **圧縮閾値**: `compression.token_threshold`（デフォルト100K）または `compression.threshold_percent`（デフォルト80%）超過時に自動圧縮/警告

### 📝 プロジェクト設定（xelyon.yaml）
プロジェクト固有のルール・コンテキストを構造化 YAML で管理。`/init` でテンプレート作成。

```yaml
# xelyon.yaml（プロジェクトルートに配置）
context: "Go製CLIツール。Cobraベース。"
rules:
  - "変更後は make ci-check を実行"
  - "公開関数にはコメント必須"
conditional:
  - name: Agent internals
    paths:
      - "internal/agent/**/*.go"
    rules:
      - "公開関数・型には日本語コメント必須"
    context: "トークン推定は共通推定器を使う"
ignore:
  patterns:
    - "dist"
    - "*.min.js"
hooks:                    # config.yaml の hooks を上書き
  on_completion:
    - "go vet ./... && go test ./..."
  on_step_complete:
    - "echo 'Step {{step_id}}: {{step_status}}'"
  timeout: 120
  max_retry: 3
```

- **context**: AI に注入するプロジェクト説明
- **rules**: 番号付きで system prompt に注入される必須ルール
- **conditional**: `paths` に一致した時だけ注入する rules/context
- **ignore**: Project Map / `list_dir` / `search_code` で共有する ignore パターン
- **hooks**: 完了時・ステップ完了時フック（`config.yaml` の hooks より優先）
- **Project Map**: 起動時は軽量 manifest が自動生成されるため、`xelyon.yaml` にファイル一覧や関数目次は書かない

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

🗺️  Project map loaded (42 files, 118 symbols)
📋 Context size: ~10.8k tok

> main.goを読んで、バグがあれば修正して
```

### 3. 基本コマンド

```bash
/help        # ヘルプ
/use gemini  # プロバイダー切り替え
/think high  # Extended Thinking 有効化
/lsp status  # LSPサーバー状態確認
/lsp detect  # プロジェクト内の言語を検出
/lsp install # LSPサーバーをインストール
/project     # xelyon.yaml を対話式で編集
/exit        # 終了
```

### 4. NAVモードでテキストをコピー

```text
Esc           入力欄が空なら NAV モードへ
count+j/k/h/l 数字プレフィックス付き移動
w / b / e     単語移動
0 / ^ / $     行頭 / 非空白先頭 / 行末
v             文字単位のビジュアル選択
V             行単位のビジュアル選択
y             選択範囲をコピー
yy            現在行をコピー
Tab           ツールブロックをフォーカス/折りたたみ
q / i / Esc   入力モードへ戻る
```

起動直後のヘッダーや AI 出力、ツール結果も NAV モードからコピーできます。コピー内容は ANSI 色を除いたプレーンテキストです。`3j`、`12G`、`vwe` のような Vim 風操作にも対応しています。

## よく使う機能

### プロバイダー切り替え

```bash
xelyon --provider gemini --model gemini-2.5-flash

# または対話中に
> /use claude
```

### Web検索

`web_search` は OpenAI / Gemini / Claude のネイティブ検索を使います。メインプロバイダーが DeepSeek / Groq / Ollama / OpenRouter / Bedrock の場合は、`web_search.provider` で検索専用プロバイダーを指定できます。

```yaml
# ~/.xelyon/config.yaml
web_search:
  provider: gemini

# 長い web_search 結果だけを軽量モデルで再圧縮（任意）
utility_model:
  enabled: true
  provider: openai
  model: gpt-5.2-mini
  tasks:
    - web_search_compaction
```

`utility_model` は検索結果圧縮などの軽量補助タスク専用です。メイン推論や `compression.model` には影響しません。

Gemini API キーは無料で取得できます: https://aistudio.google.com/apikey

### サブエージェント設定

```yaml
# ~/.xelyon/config.yaml
sub_agent:
  enabled: true
  default_model: ""
  default_effort: off
  max_concurrent: 5
```

`sub_agent.default_model` が空の場合は、メイン provider に応じて OpenAI は `gpt-5.4-nano`、Claude は `claude-haiku-4-5-20251001`、Gemini は `gemini-3.1-flash-lite-preview` などの低コストモデルを自動選択します。親モデルの `default_model` や `thinking` 設定は直接上書きしません。

### 最大出力トークン数の設定

```yaml
# ~/.xelyon/config.yaml
provider_models:
  claude:
    default_model: claude-sonnet-4-6
    max_output_tokens: 64000   # デフォルト: 64000
  gemini:
    default_model: gemini-3.1-pro-preview-customtools
    max_output_tokens: 65536   # デフォルト: 65536
  deepseek:
    default_model: deepseek-chat
    max_output_tokens: 8192    # デフォルト: 8192
```

| プロバイダー | デフォルト max_output_tokens |
|------------|---------------------------|
| claude     | 64000                     |
| bedrock    | 64000                     |
| gemini     | 65536                     |
| openai     | 16384                     |
| deepseek   | 8192                      |
| groq       | 8192                      |
| ollama     | 4096                      |
| openrouter | 64000                     |

### 確認動作のカスタマイズ

```yaml
# ~/.xelyon/config.yaml
tool_confirm:
  auto_approve_safe: true    # SafetyHigh（読み取り）自動承認（デフォルト: true）
  auto_approve_medium: true  # SafetyMedium（書き込み）自動承認（デフォルト: false）
```

### 中間出力の表示レベル

```yaml
# ~/.xelyon/config.yaml
output:
  assistant_updates: ""  # 空なら Normal=phase, Plan=verbose
```

`assistant_updates` は assistant prose の途中表示だけを制御します。`phase` は通常モードの逐次実況を短いフェーズ要約に寄せ、`off` はさらに抑制します。`apply_patch` の diff 表示やツール出力の折りたたみ挙動は変わりません。

| 設定 | 対象ツール | デフォルト |
|------|-----------|-----------|
| `auto_approve_safe` | read_file, list_dir, search_code 等 | true |
| `auto_approve_medium` | str_replace, write_file, web_search 等 | false |

```bash
# 全ツール自動承認（信頼できる環境向け、SafetyLow含む）
xelyon --auto-approve
```

### 差分表示設定

```yaml
# ~/.xelyon/config.yaml
diff:
  context_lines: 10    # 差分表示のコンテキスト行数（0で省略なし、デフォルト: 10）
  max_total_lines: 0   # 差分表示の最大行数（0で無制限、デフォルト: 0）
```

### MCP設定

```yaml
# ~/.xelyon/config.yaml
mcp:
  enabled: true    # MCP機能のON/OFF（デフォルト: true）
  headless: false  # ヘッドレスモードでMCPを使うか（デフォルト: false）
```

`enabled: false` にするとMCPサーバーへの接続をスキップし、トークン消費を削減できます。
`~/.xelyon/mcp.json` の設定はそのまま残るため、再度 `enabled: true` にすれば復活します。
- **ツール単位フィルタリング**: MCPサーバーのツールを `include`/`exclude` で制御。不要なツールを除外してトークン消費を最適化

### Completion Hooks

AI が完了を宣言すると、LSP 診断の後にここで定義したコマンドを順番に実行します。
コマンド失敗時は AI にエラー内容をフィードバックし修正を続行します（最大 `max_retry` 回）。
Makefile は不要です。普段使っているコマンドをそのまま書けます。

```yaml
# ~/.xelyon/config.yaml
hooks:
  on_completion:
    # Go
    - "go vet ./... && go test ./..."
    # Node.js / TypeScript
    # - "npm test"
    # Python
    # - "pytest"
    # Rust
    # - "cargo test"
    # Makefile がある場合
    # - "make ci-check"
  timeout: 120         # コマンドタイムアウト秒（デフォルト: 60）
  max_retry: 3         # フック失敗時の最大リトライ回数（デフォルト: 3）
```

`xelyon.yaml` にも `hooks` を定義でき、`config.yaml` より優先されます（プロジェクト固有のフック設定に便利）。
変更ファイルは `XELYON_CHANGED_FILES` 環境変数（スペース区切り）で参照できます。
Normal mode / Plan mode の両方で動作します。

### Step Complete Hooks

Plan Mode で各ステップ完了時に実行するコマンドです。
テンプレート変数 `{{step_id}}`, `{{step_description}}`, `{{step_status}}` が使えます。

```yaml
# ~/.xelyon/config.yaml
hooks:
  on_step_complete:
    # ステップごとにテスト実行
    - "go test ./..."
    # 通知（ステップ番号・状態を展開）
    # - "echo 'Step {{step_id}} ({{step_status}}): {{step_description}}'"
```

失敗時は AI にフィードバックして修正を試み、次のステップに進む前に再実行します。

### 設定管理

```bash
> /config         # 対話式設定メニュー（50+設定項目を編集可能）
> /config show    # 全設定を表示（デフォルトとの差分を ⚡ で表示）
```

対話式メニューでは20カテゴリ、50以上の設定項目を編集可能:
- Provider & Model, Compression, Tool Confirm
- Bash Safety, LSP Servers, Plan Mode, MCP など

## ドキュメント

| ドキュメント | 内容 |
|------------|------|
| [コマンド一覧](docs/commands.md) | 全コマンド、24ツール、使用例 |
| [プロバイダー設定](docs/providers.md) | 各プロバイダーのAPIキー取得方法 |
| [設定リファレンス](docs/config.md) | config.yaml と環境変数 |
| [MCP連携](docs/mcp.md) | 外部ツール追加 |
| [LSP連携](docs/lsp.md) | 言語サーバー連携（23言語対応） |
| [使い方詳細](docs/usage.md) | 複数行入力、画像入力、レビュー機能など |

## 開発に参加する

xelyon-cli の開発に参加したい方向け：

```bash
git clone https://github.com/susugadx/xelyon-cli.git
cd xelyon-cli
go build -o xelyon
./xelyon
```

> ⚠️ このリポジトリの `xelyon.yaml` は xelyon-cli 開発用です。

## コントリビュート

PRやIssue歓迎です！詳細は [CONTRIBUTING.md](CONTRIBUTING.md) を参照してください。

## ライセンス

MIT
