# 設定リファレンス

XELYON CLIの設定方法と全オプションのリファレンスです。

## 設定ファイル

設定ファイルは `~/.xelyon/config.yaml` に保存されます。

初回起動時に自動的にデフォルト設定ファイルが作成されます。

## 設定方法

### 対話式メニュー（推奨）

セッション中に `/config` コマンドで対話式設定メニューを起動できます：

```
> /config
```

主要な設定項目をカテゴリ別に管理できます。変更は即座に `~/.xelyon/config.yaml` に保存されます。

**サポートする設定型:**
- bool（y/n で切り替え）
- int（数値入力）
- string（テキスト入力）
- select（番号選択）
- []string（項目追加/削除）
- map[string]struct（サブメニューで編集）

詳細は [コマンド一覧: /config](commands.md#config) を参照してください。

### 直接編集

設定ファイルを直接編集することもできます：

```bash
vi ~/.xelyon/config.yaml
```

## 設定優先順位

設定は以下の優先順位で適用されます（上が優先）:

1. **コマンドラインフラグ** (`--provider`, `--model` など)
2. **環境変数** (`XELYON_PROVIDER`, `DEEPSEEK_API_KEY` など)
3. **設定ファイル** (`~/.xelyon/config.yaml`)
4. **デフォルト値**

## 設定ファイル構造

### 完全な設定例

以下は全ての設定項目とデフォルト値を含む完全な設定例です。

<!-- CONFIG-EXAMPLE-START -->
```yaml
# ============================================================
# プロバイダー設定
# ============================================================
default_provider: deepseek
# デフォルトで使用するモデル
default_model: deepseek-v4-flash
# プロバイダーごとのモデル設定
provider_models:
    azure:
        default_model: azure-gpt-5.4
        max_output_tokens: 16384
    bedrock:
        default_model: global.anthropic.claude-sonnet-4-6
        max_output_tokens: 64000
        anthropic_version: bedrock-2023-05-31
    claude:
        default_model: claude-sonnet-4-6
        max_output_tokens: 64000
        anthropic_version: "2023-06-01"
    deepseek:
        default_model: deepseek-v4-flash
        max_output_tokens: 16384
    gemini:
        default_model: gemini-3.1-pro-preview-customtools
        max_output_tokens: 65536
    groq:
        default_model: meta-llama/llama-4-scout-17b-16e-instruct
        max_output_tokens: 8192
    kimi:
        default_model: kimi-k2.6
        max_output_tokens: 32768
    ollama:
        default_model: qwen2.5-coder:7b
        max_output_tokens: 4096
    openai:
        default_model: gpt-5.4
        max_output_tokens: 16384
    openrouter:
        default_model: anthropic/claude-sonnet-4.6
        max_output_tokens: 64000

# ============================================================
# 一般設定
# ============================================================
general:
    # 表示言語（auto, ja, en）
    ui_language: auto

# ============================================================
# 会話履歴圧縮設定
# ============================================================
compression:
    # 自動圧縮を有効化
    enabled: true
    # 自動圧縮のトークン使用率閾値（%）
    trigger_percent: 80
    # 圧縮時に保持する直近メッセージ数
    keep_recent: 20

# ============================================================
# 実行モード設定
# ============================================================
# ツール実行の承認モードを制御
# balanced: read自動/write確認/verification bash安全自動
# trusted: workspace内通常編集も自動/高影響のみ確認
# full_auto: 原則自動（always_confirm指定は確認）
execution:
    # 実行モード（balanced / trusted / full_auto）
    mode: balanced
    # どのモードでも確認するカテゴリ
    always_confirm: []
    # 追加の安全シェルコマンド（verification / env 用）
    safe_shell_commands: []

# ============================================================
# ペーストモード設定
# ============================================================
paste:
    # Bracketed Paste Mode を有効化（複数行ペースト対応）
    bracketed_paste: true

# ============================================================
# プロジェクト構造マップ設定
# ============================================================
project_map:
    # セッション開始時にプロジェクト構造マップを生成・注入
    enabled: true
    # ProjectMap のベース比率（0.01-0.20、デフォルト: 0.05。大規模 repo では 0.03-0.04 に自動補正）
    context_ratio: 0.05
    # 追加除外ディレクトリ（list_dir と共通）
    additional_ignore_dirs: []

# ============================================================
# Agent Instructions 設定
# ============================================================
# AGENTS.md / CLAUDE.md 互換ガイダンス読み込み設定
# xelyon.yaml の rules とは別レイヤーで扱われます
agent_instructions:
    project:
        # project-local guidance の読み込みモード（off / fallback / always）
        mode: fallback
        # project-local guidance ファイル候補
        files:
            - AGENTS.md
            - CLAUDE.md
            - .claude/CLAUDE.md
        # gitignored / untracked guidance を許可
        include_gitignored: false
    global:
        # global guidance 読み込みを有効化
        enabled: false
        # global guidance ファイル候補
        files:
            - ~/.xelyon/AGENTS.md
            - ~/.xelyon/CLAUDE.md
    # CLAUDE.local.md / AGENTS.local.md など local 系 guidance を許可
    include_local_files: false
    # @path import 行を展開して読み込む（相対パスは当該 guidance file 基準）
    expand_imports: false
    # 1ファイルあたりの最大読み込みバイト数
    max_file_bytes: 20000
    # guidance 全体の最大読み込みバイト数
    max_total_bytes: 60000

# ============================================================
# LSP連携設定
# ============================================================
# 23言語のデフォルト設定が内蔵済み。通常は変更不要です。
# 詳細: docs/config.md の「LSP連携設定」セクション
lsp:
    # LSP連携の有効/無効
    enabled: true

# ============================================================
# ツール出力表示設定
# ============================================================
output:
    # 折りたたみ前の最大表示行数
    max_lines: 5
    # assistant prose の中間表示制御（verbose / phase / off、空でモード別デフォルト）
    assistant_updates: ""

# ============================================================
# Web検索設定
# ============================================================
# ネイティブ Web 検索の実行プロバイダーとキャッシュ設定
# 未設定の場合はメインプロバイダーの検索を使用
# メインが非対応の場合は kimi / moonshot / openai / gemini / claude / anthropic のいずれかを設定
web_search:
    # 検索プロバイダー（kimi / moonshot / openai / gemini / claude / anthropic、未設定時はメインプロバイダーを使用）
    provider: gemini
    # キャッシュを有効化（デフォルト: true）
    cache_enabled: true
    # キャッシュTTL秒数（デフォルト: 3600 = 1時間）
    cache_ttl: 3600
    # 最大キャッシュ数（デフォルト: 50）
    cache_size: 50

# ============================================================
# サブエージェント設定
# ============================================================
# 探索・調査タスクを低コストモデルへ委譲する設定
# spawn_agent / wait_agent の既定値と同時実行数を制御
sub_agent:
    # サブエージェント機能を有効化
    enabled: true
    # 既定モデル（空でメイン provider の最安モデルを自動選択）
    default_model: gpt-5.4-mini
    # 既定推論強度（off / low / medium / high）
    default_effort: ""
    # 同時実行上限（デフォルト: 1）
    max_concurrent: 1

# ============================================================
# MCP設定
# ============================================================
# MCP (Model Context Protocol) サーバー接続の設定
# 個別サーバー設定は ~/.xelyon/mcp.json で管理
mcp:
    # MCP接続を有効化（デフォルト: true）
    enabled: true
    # Headlessモードでも接続（デフォルト: false）
    headless: false

# ============================================================
# Final Checks 設定
# ============================================================
# completed_with_changes 時に自動実行する final checks コマンド
# 変更ファイルは XELYON_CHANGED_FILES 環境変数で参照可能
final_checks:
    # completed_with_changes 時に実行する final checks コマンド（例: go test ./...）
    commands: []
    # final checks コマンドタイムアウト（秒）（デフォルト: 600）
    timeout: 600
```
<!-- CONFIG-EXAMPLE-END -->

## 設定項目詳細

### プロバイダー・モデル設定

#### `default_provider`
- **型**: string
- **デフォルト**: `deepseek`
- **説明**: デフォルトで使用するプロバイダー
- **選択肢**: `deepseek`, `kimi`, `claude`, `openai`, `azure`, `gemini`, `groq`, `ollama`, `openrouter`, `bedrock`

#### `default_model`
- **型**: string
- **デフォルト**: `deepseek-v4-flash`
- **説明**: デフォルトで使用するモデル

#### `provider_models`
- **型**: map
- **説明**: プロバイダーごとのデフォルトモデル設定と出力トークン制限
- **サブキー**:
  - `default_model`: プロバイダーのデフォルトモデル名
  - `max_output_tokens`: プロバイダーの出力トークン上限（デフォルト値）
  - `catalog_model`: `default_model` が deployment 名や社内 alias の場合に、token limit / pricing / context 判定へ使う既知モデル名（オプション）
  - `model_overrides`: 特定モデル別の出力トークン上限設定（オプション）
    - **用途**: 既知モデルマップにないモデル、deployment 名、デフォルト値を上書きしたい場合に指定
    - **サブキー**: `max_output_tokens`, `catalog_model`
    - **優先度**: `model_overrides[model].max_output_tokens` > `catalog_model` を含む既知モデルマップ > `max_output_tokens`
    - **補足**: 既知モデル（`deepseek-v4-flash`, `deepseek-v4-pro`, `kimi-k2.6`, `gpt-5.2`, `gemini-2.5-flash` 等）は自動解決されるため、通常は設定不要
- **例**:
  ```yaml
  provider_models:
    deepseek:
      default_model: deepseek-v4-flash
      # V4 既知モデルは自動解決されるため model_overrides 不要
      max_output_tokens: 16384
      # カスタムモデルのみ指定:
      model_overrides:
        my-deepseek-deployment:
          catalog_model: deepseek-v4-flash
          max_output_tokens: 32768
    kimi:
      default_model: kimi-k2.6
      max_output_tokens: 32768
    openai:
      default_model: corp-gpt-deployment
      catalog_model: gpt-5.4
  ```

料金表にない provider/model は別モデルの料金で概算せず、`/status` などでは `N/A (pricing unavailable)` と表示されます。deployment 名や alias で料金表示を有効にしたい場合は `catalog_model` を指定してください。`catalog_model` は provider の pricing family で解決できる既知モデル名を指定します。OpenRouter alias では `openai/gpt-5.4` のような OpenRouter model ID、Bedrock Claude alias では Bedrock の Claude model ID または Claude catalog model 名を指定してください。Native Kimi alias では `kimi-k2.6` / `kimi-k2.5` のような Kimi catalog model 名を指定します。`pricing.yaml` の `known_models.exact` にある実モデル ID だけが `catalog_model` なしで料金表示され、`rules.contains` は価格選択専用です。OpenRouter の `provider/model` 形式も OpenRouter 側の exact allowlist にある ID だけを料金表示します。

DeepSeek の推奨モデル:
- `deepseek-v4-flash`: 低コスト・高速・普段使い向き
- `deepseek-v4-pro`: 高精度・重い設計/レビュー向き

`deepseek-chat` / `deepseek-reasoner` は `deepseek-v4-flash` 相当の legacy alias です。2026-07-24 廃止予定のため、新規設定では `deepseek-v4-flash` / `deepseek-v4-pro` を使用してください。DeepSeek V4 は 1M context / 最大 384K output です。

Kimi の推奨モデル:
- `kimi-k2.6`: native Kimi provider の既定モデル。高品質な編集/設計向き
- `kimi-k2.5`: サブエージェント既定モデル。低コストな軽作業向き

Kimi K2.6 / K2.5 は 256K context / 最大 32K output です。`kimi-k2-thinking` は明示指定時のみ同じ上限の legacy/compat thinking model として扱います。`moonshot` は `kimi` provider の alias として扱われます。

### 会話履歴圧縮設定 (`compression`)

Context Window（コンテキストウィンドウ）を管理し、トークン上限エラーを防ぎます。

#### `enabled`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: トークン使用率が閾値を超えた際に自動圧縮を実行
- **補足**: コスト削減・エラー防止のためデフォルトON

#### `trigger_percent`
- **型**: integer
- **デフォルト**: `80`
- **説明**: 自動圧縮を実行する使用率閾値（%）
- **補足**: モデルのコンテキストウィンドウに対する使用率。実際の標準閾値は、次リクエストの最大出力トークン分を残すため `context window - max_output_tokens` でも上限をかけます。

#### `keep_recent`
- **型**: integer
- **デフォルト**: `20`
- **説明**: 圧縮時に保持する最新メッセージ数

#### `provider_thresholds`
- **型**: map
- **デフォルト**: 空
- **説明**: provider/model ごとの絶対圧縮閾値を明示 override するための設定
- **補足**: 未設定時は provider 汎用の絶対閾値には fallback しません。通常はモデルの context window、`trigger_percent`、最大出力トークンから自動圧縮閾値を計算します。

**自動圧縮の動作:**
1. API呼び出し成功後、OpenAI/Azure Responses API の `previous_response_id` 継続 request に `context_management.compaction` を実際に載せられた場合のみ local auto-compress をスキップ
2. pricing cliff 回避を評価
3. 明示された `provider_thresholds` を評価
4. Claude Compaction が使える場合は標準の local auto-compress をスキップ
5. 明示された汎用絶対閾値（`token_threshold` / `threshold_tokens`）を評価
6. context window が既知のモデルでは `trigger_percent`（デフォルト80%）と最大出力トークンの headroom で使用率を評価
7. context window が不明なモデルでは、早すぎる圧縮を避けるため標準の local auto-compress をスキップ
8. 圧縮時に通知を表示し、無効化方法も案内

```
🗜️ Auto-compressing history (162K >= 150K threshold, 81% context used)...
   Before: 162,000 tokens → After: 45,000 tokens
   💡 Disable by setting compression.enabled: false in config.yaml
```

### ストリーミング設定 (`streaming`)

> **注意**: `/config` メニューには表示されません。YAML 直接編集で変更できます。

```yaml
streaming:
  idle_timeout_seconds: 30       # アイドルタイムアウト秒
  thinking_timeout_seconds: 120  # thinking 専用タイムアウト秒
  show_file_info: true           # ファイル読み込み時にサイズ・行数表示
  show_search_progress: true     # 検索時に進捗表示
  stream_bash_output: true       # bash 出力をリアルタイム表示
```

### bashツール設定 (`bash`)（レガシー）

> **注意**: `bash` セクションは `execution` に統合されました。既存 YAML は互換読み込みされますが、新規設定では `execution.mode` / `execution.safe_shell_commands` を使用してください。

bashコマンドの安全性制限を設定します。

```yaml
bash:
  # 安全性レベル: strict, moderate, permissive
  safety_level: permissive

  # 追加の安全コマンド（確認なしで実行）
  safe_commands: []

  # パイプを許可
  allow_pipe: true

  # リダイレクトを許可
  allow_redirect: true

  # sed -i等のインライン編集を許可
  allow_inline_edit: true
```

> **Note**: デフォルトは `permissive` ですが、**すべてのコマンドは実行前に確認プロンプトが表示されます**。`sudo` など危険なコマンドは常にブロックされます。より厳格な制限が必要な場合は `moderate` または `strict` に変更してください。

#### `safety_level`
- **型**: string
- **デフォルト**: `permissive`
- **選択肢**: `strict`, `moderate`, `permissive`

| レベル | パイプ `\|` | リダイレクト `>` | sed -i | sudo |
|--------|-------------|------------------|--------|------|
| strict | ❌ | ❌ | ❌ | ❌ |
| moderate | ✅ | ❌ | ❌ | ❌ |
| permissive | ✅ | ✅ | 設定次第 | ❌ |

#### `safe_commands`
- **型**: string[]
- **デフォルト**: `[]`
- **説明**: 確認なしで実行できるコマンドを追加

#### `allow_pipe`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: パイプ (`|`) の使用を許可

#### `allow_redirect`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: リダイレクト (`>`, `>>`, `<`) の使用を許可

#### `allow_inline_edit`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: `sed -i`, `perl -i` 等のインライン編集を許可

**危険なパイプは常にブロック**:
```bash
# 以下のパターンはどのレベルでもブロック
curl ... | sh
cat script.sh | bash
echo password | sudo -S ...
```

### OpenAI設定 (`openai`)（内部）

> **注意**: `/config` メニューには表示されません。Responses API ルーティングはプレフィックスマッチで自動判定されるため、通常は設定不要です。
> カスタムモデルを Responses API で使いたい場合のみ、YAML 直接編集で `responses_api_models` にモデル名を追加してください。`doctor openai --require-capability responses_streaming` で gate する場合は、streaming 可否を判定できるよう `provider_models.openai.catalog_model` または `model_overrides.<model>.catalog_model` に実モデル名も設定してください。

`gpt-5.5` は Responses API の streaming 経路で動作します。`gpt-5.5-pro` は Responses API 対応ですが streaming unsupported のため、XELYON は non-streaming 経路を使用します。GPT-5.5 Pro は cached input discount がなく、応答に数分かかる場合があります。background mode は未対応です。

```yaml
openai:
  responses_api_models:
    - my-custom-responses-model
```

### Responses API retention 設定 (`responses`)（高度な設定）

> **注意**: `/config` メニューには表示されません。ほとんどのユーザーは変更しないでください。
> OpenAI / Azure OpenAI の Responses API 経路で、provider 側に response state を保存するか、保存した response ID を XELYON の session に永続化するかを制御します。

デフォルトはサーバー側 state を使う推奨設定です。通常はこのままにしてください。

```yaml
responses:
  store: true
  persist_response_id: true
  server_compaction:
    enabled: true
    compact_threshold: 0
    local_fallback: true
```

#### `store`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: Responses API request の `store` を制御します。`true` の場合、XELYON は返却された response ID を使って次回 request に `previous_response_id` を送り、会話の続きを provider 側 state に接続します。
- **`false` にする場合**: 新しい response state を provider 側に保存せず、`previous_response_id` も送信しません。各 turn ではローカル履歴、または Compact API の圧縮済み state を request input に含めるため、token 使用量や自動圧縮の挙動が変わる可能性があります。

#### `persist_response_id`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: response ID を XELYON の session に保存し、session reload 後も `previous_response_id` 継続を復元します。
- **`false` にする場合**: 現在のプロセス内では response ID 継続を使いますが、session file には保存しません。`store: false` の場合、この設定は実質的に無効です。

#### `server_compaction.enabled`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: OpenAI / Azure OpenAI の Responses API で `previous_response_id` 継続 request に `context_management.compaction` を載せる機能を有効化します。`enabled` だけでは local auto-compress はスキップされず、実際に request payload へ compaction 設定を載せられた場合のみ local auto-compress をスキップします。
- **`false` にする場合**: `context_management.compaction` を送らず、local auto-compress の通常判定を使います。

#### `server_compaction.compact_threshold`
- **型**: integer
- **デフォルト**: `0`
- **説明**: server-side compaction の**発火閾値**です。圧縮後サイズではありません。
- **`0` の意味**: auto 解決。`compression.trigger_percent` と出力 headroom cap（`context window - max_output_tokens`）から `min(...)` で閾値を計算し、最低 `1000` へ丸めます。
- **validation**: `0` 以外を指定する場合、`1000` 未満は validation error です。
- **補足**: request payload へ `0` は送信しません。auto 解決できない（例: context window 不明）場合は `context_management` 自体を省略します。

#### `server_compaction.local_fallback`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: `context_management.compaction` を request payload に載せられない場合に local auto-compress 判定へ戻すかを制御します。
- **`true`**: local auto-compress の既存判定へフォールバック
- **`false`**: local auto-compress へのフォールバックを行わない

#### `responses.server_compaction` の適用範囲と対象外

- `responses.server_compaction` は OpenAI / Azure OpenAI の Responses API における **`previous_response_id` chain request** 向け機能です。
- 通常の `/responses` request payload に `context_management.compaction` を付与して server-side compaction の発火閾値を渡します。
- Compact API（`/responses/compact` / `/compress --compact`）とは別機能です。互換レイヤーや上位互換ではありません。
- 初回 turn のように `previous_response_id` がない request では、server-side compaction は付与されません。
- `responses.store=false` の場合は `previous_response_id` 自体を送らないため、server-side compaction は適用対象外です。
- `store=false` 前提の stateless input-array chaining で server-side compaction を使う運用は、現時点では未対応です。
- `context_management.compaction` を付与できない場合:
  - `local_fallback=true`: local auto-compress 判定にフォールバックします。
  - `local_fallback=false`: local auto-compress 判定へフォールバックしません（local auto-compress は skip 扱い）。
- `compact_threshold=0` は auto 解決を意味し、API payload に `0` は送信しません。
- `compact_threshold` の正数指定は `1000` 以上のみ有効です（`1..999` は validation error）。

`/clear` はローカル履歴とローカルに保持している response ID を消しますが、provider 側に既に保存された response object の remote delete は行いません。response state を provider 側に残したくない運用では、最初から `store: false` を設定してください。

### LSP連携設定 (`lsp`)

Language Server Protocol (LSP) を使用したコード解析の設定を行います。

詳細は [LSP連携ガイド](lsp.md) を参照してください。

```yaml
lsp:
  enabled: true
  servers:
    go:
      command: gopls
    typescript:
      command: vtsls
      args: ["--stdio"]
    python:
      command: pyright-langserver
      args: ["--stdio"]
    rust:
      command: rust-analyzer
      disabled: true
```

#### `enabled`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: LSP連携の有効/無効

#### `servers`
- **型**: map
- **説明**: 言語ごとのLSPサーバー設定

**サーバー設定項目:**

| 項目 | 型 | 説明 |
|-----|---|------|
| `command` | string | LSPサーバーのコマンド |
| `args` | string[] | コマンドに渡す引数（オプション） |
| `disabled` | boolean | このサーバーを無効化（オプション） |

**対応言語:** Go, TypeScript/JavaScript, Python, Rust, Java, C/C++, Ruby, Kotlin, Swift, C#, Scala, PHP, Elixir, Lua, CSS/SCSS, HTML, Vue, Svelte, YAML, TOML, SQL, Bash, Markdown（23言語）

**遅延起動:** LSPサーバーは初回使用時に起動します（XELYON起動時には起動しません）。

### Extended Thinking設定 (`thinking`)（内部）

> **注意**: `/config` メニューには表示されません。セッション中の切り替えは `/thinking` コマンドを使用してください。
> 既存 YAML の `thinking:` セクションは互換読み込みされます（runtime 初期値として機能）。

**対応モデル:**
- **Claude**: Sonnet 4 以降
- **OpenAI**: GPT-5 系（GPT-5.5 / GPT-5.5 Pro を含む）
- **Gemini**: 2.5 Pro 系（Flash は非対応）
- **DeepSeek**: V4 の `thinking` field で制御します。`/thinking off` は `thinking.disabled`、`/thinking on` は `thinking.enabled` + `reasoning_effort` を送ります。`/thinking xhigh` は DeepSeek では `max` に変換されます。
- **Kimi**: K2.6 / K2.5 は `thinking` field で制御します。`/thinking off` は `thinking.disabled`、`/thinking on` は K2.6 へ `thinking.enabled` + `keep: all`、K2.5 へ `thinking.enabled` を送ります。`kimi-k2-thinking` が明示指定された場合は forced thinking model として扱い、`/thinking off` でも `thinking.disabled` は送信しません。新規利用では `kimi-k2.6` を推奨します。

**コマンドで切り替え（正規ルート）:**

```
/thinking          # 現在の状態を表示
/thinking on       # 有効化（現在のレベルで）
/thinking off      # 無効化
/thinking low      # low レベルで有効化
/thinking medium   # medium レベルで有効化
/thinking high     # high レベルで有効化
/thinking xhigh    # xhigh(max) レベルで有効化
```

**レベル別パラメータ:**

| Level | Claude adaptive (Opus 4.7 / 4.6) | Claude 4.5以前 (budget_tokens) | OpenAI (effort) | Gemini (budget) |
|-------|------------------------------------|-------------------------------|-----------------|-----------------|
| low | low | 5,000 | low | 5,000 |
| medium | medium | 10,000 | medium | 10,000 |
| high | high | 20,000 | high | 20,000 |
| xhigh | Opus 4.7: xhigh / Opus 4.6: max / Sonnet 4.6: high | 40,000 | xhigh | 40,000 |

**Codex モデルの制限:**

OpenAI Codex モデル（`gpt-5.3-codex`, `gpt-5.2-codex`, `gpt-5.1-codex` 等）は reasoning が必須のため、`/thinking off` → `low` レベルにフォールバックします。
GPT-5.5 / GPT-5.5 Pro は `/thinking xhigh` で `reasoning.effort: xhigh` を送信し、`/thinking off` では reasoning を送信しません。

### ツール確認設定 (`tool_confirm`)（レガシー）

> **注意**: `tool_confirm` は `execution` に統合されました。既存 YAML は互換読み込みされますが、新規設定では `execution.mode` を使用してください。
> `auto_approve_medium: true` → `execution.mode: trusted` に自動マイグレーションされます。

### プロンプトキャッシュ設定 (`prompt_cache`)

> **注意**: `/config` メニューには表示されません。YAML 直接編集で変更できます。
> Anthropic API の `cache_control` ブレークポイント制御です。Claude 以外のプロバイダーでは効果がありません。

```yaml
prompt_cache:
  enabled: true    # cache_control BP を設定（デフォルト: true）
  cache_ttl: 5m    # キャッシュ TTL（"5m" または "1h"）
```

### ペーストモード設定 (`paste`)

```yaml
paste:
  bracketed_paste: true  # Bracketed Paste Mode を有効化（デフォルト: true）
```

#### `bracketed_paste`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: Bracketed Paste Mode を有効化。複数行のペーストを一括入力として扱います

### コマンドエイリアス

slash command の alias は command catalog で定義されている組み込み alias だけを使用します。`command_aliases` は旧設定ファイルの読み込み互換のために残っていますが、実行時には使用されません。新しい alias は `~/.xelyon/config.yaml` では追加できません。

## 環境変数

### API キー

```bash
# DeepSeek
export DEEPSEEK_API_KEY=sk-...

# Kimi (Moonshot)
export MOONSHOT_API_KEY=sk-...
# Optional full Chat Completions endpoint override:
export KIMI_API_URL=https://api.moonshot.ai/v1/chat/completions

# OpenAI
export OPENAI_API_KEY=sk-...

# Azure OpenAI
export AZURE_OPENAI_BASE_URL=https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1
export AZURE_OPENAI_API_KEY=...
# Microsoft Entra ID を使う場合
unset AZURE_OPENAI_API_KEY
export AZURE_OPENAI_AUTH_TOKEN=$(az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv)
# 長時間実行や CI で token を自動更新したい場合
unset AZURE_OPENAI_AUTH_TOKEN
export AZURE_OPENAI_AUTH_TOKEN_COMMAND='az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv'
export AZURE_OPENAI_AUTH_TOKEN_COMMAND_TIMEOUT=10s
# この command はローカル shell で実行され、stdout の最初の空でない行を token として使います
# 信頼できる command だけを設定してください

# Gemini
export GEMINI_API_KEY=...

# Claude (Anthropic)
export ANTHROPIC_API_KEY=sk-ant-...

# Groq
export GROQ_API_KEY=gsk_...

# Ollama（環境変数不要・ローカル実行）
```

### Web検索

`web_search` は Kimi / OpenAI / Gemini / Claude のネイティブ検索を使います。

- **`web_search.provider` 未設定**: メインプロバイダーが Kimi / OpenAI / Gemini / Claude の場合、そのままネイティブ検索を使用
- **`web_search.provider` 設定あり**: 指定した検索プロバイダーを使用
- **メインが非対応**: DeepSeek / OpenRouter / Groq / Ollama / Bedrock などでは `web_search.provider` の設定が必要

Gemini native web search は API の `usageMetadata` が返る場合、通常の token usage / cost として表示します。Kimi を使う場合は Moonshot Chat Completions の built-in `$web_search` を text-only の検索 route として使います。検索 request では `thinking: {"type":"disabled"}` を送信し、通常 function tools / 画像 / video / file upload とは混ぜません。Moonshot は `$web_search` call fee と token 使用量を別々に課金します。XELYON は API が返す token usage と `cached_tokens` を token cost の source of truth とし、[Kimi API Platform WebSearch Pricing](https://platform.moonshot.ai/docs/pricing/tools.en-US) の `$0.005 / invocation` を外部固定費として別枠で観測します。call fee は `finish_reason = "tool_calls"` で `tool_call.function.name = "$web_search"` が返った場合だけ観測し、`finish_reason = "stop"` で tool call がない response には加算しません。検索結果 tokens は次 request の `prompt_tokens` に含まれるため、表示用に観測しても token totals へは二重加算しません。

#### 設定例

```yaml
web_search:
  provider: kimi
```

#### 必要なAPIキー

検索に使うプロバイダーに応じて、以下のいずれかを設定してください。

```bash
# OpenAI を検索に使う場合
export OPENAI_API_KEY=sk-...

# Kimi / Moonshot を検索に使う場合
export MOONSHOT_API_KEY=sk-...
# 任意: proxy や互換 endpoint を使う場合も Chat Completions まで含める
export KIMI_API_URL=https://api.moonshot.ai/v1/chat/completions

# Gemini を検索に使う場合
export GEMINI_API_KEY=...

# Claude を検索に使う場合
export ANTHROPIC_API_KEY=sk-ant-...
```

Gemini API キーは無料で取得できます: https://aistudio.google.com/apikey

#### 動作例

```yaml
# メインが DeepSeek でも、検索だけ Kimi を使う
web_search:
  provider: kimi
```

```bash
xelyon
> 最新のGo言語の情報を検索して
```

メインプロバイダーが Kimi / OpenAI / Gemini / Claude の場合は、`web_search.provider` を省略するとそのままネイティブ検索を使用します。

### プロバイダー・モデル指定

```bash
# プロバイダー指定
export XELYON_PROVIDER=deepseek

# モデル指定
export XELYON_MODEL=deepseek-v4-flash
```

### セキュリティ・監査設定

```bash
# 監査ログを有効化
export XELYON_AUDIT_LOG=1
```

ツール実行履歴をJSONL形式で記録します。

- **保存場所**: `~/.xelyon/audit/audit_YYYYMMDD.jsonl`
- **記録内容**: タイムスタンプ、ツール名、引数、出力、成功/失敗
- **セキュリティ**: 機密情報（password, token, api_key, secret）は自動的に`[REDACTED]`化

```bash
# セッション履歴の暗号化を有効化
export XELYON_ENCRYPT_HISTORY=1
```

セッション履歴をAES-256-GCMで暗号化します。

- **暗号化方式**: AES-256-GCM
- **鍵導出**: PBKDF2（100,000回イテレーション、SHA-256）
- **暗号化キー保存場所**: `~/.xelyon/.session_key`（0600パーミッション）
- **注意**: 暗号化キーを紛失すると過去のセッションは復元できません

### 動作設定

```bash
# 対話的確認モード（ツール実行前に確認）
export XELYON_INTERACTIVE_CONFIRM=1

# ループ検知回数（設定ファイル上書き）
export XELYON_LOOP_THRESHOLD=5

# 差分表示行数
export XELYON_DIFF_CONTEXT_LINES=20

# str_replace batch: stdout抑止時でも厳密な行数統計を強制
export XELYON_STR_REPLACE_BATCH_EXACT_LINE_STATS=1
```

`XELYON_STR_REPLACE_BATCH_EXACT_LINE_STATS` は `str_replace` の batch 置換で使う行数統計モードを切り替えます。

- デフォルト（未設定）: `SuppressStdout()==true` の実行では軽量統計を使用（高速）
- `1` / `true` / `yes` / `on`: stdout 抑止時でも厳密統計を使用
- 注意: 厳密統計は大きい入力で CPU / メモリコストが増えるため、検証用途でのみ有効化することを推奨

### Function Calling（ツール呼び出し）

Gemini と Bedrock 以外の各プロバイダーでは Function Calling（ツール呼び出し）機能を無効化できます。
モデルがFunction Callingに対応していない場合や、テキストベースのツール呼び出しに戻したい場合に使用します。
Gemini provider は native function calling を前提にし、内部の text-only request だけ request-scoped tool disable を使います。
Bedrock provider は agent 実行で structured tool calling を必須とするため、streaming tool use が未確認または非対応の Bedrock Converse モデルは runtime supported として扱いません。

```bash
# OpenAI Function Calling 無効化
export OPENAI_FUNCTION_CALLING=0

# DeepSeek Function Calling 無効化
export DEEPSEEK_FUNCTION_CALLING=0

# Azure OpenAI Function Calling 無効化
export AZURE_OPENAI_FUNCTION_CALLING=0

# Groq Function Calling 無効化
export GROQ_FUNCTION_CALLING=0

# Claude Tool Use 無効化
export CLAUDE_FUNCTION_CALLING=0

# Ollama Function Calling 無効化
export OLLAMA_FUNCTION_CALLING=0

# OpenRouter Function Calling 無効化
export OPENROUTER_FUNCTION_CALLING=0
```

| 環境変数 | デフォルト | 説明 |
|---------|-----------|------|
| `OPENAI_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `DEEPSEEK_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `AZURE_OPENAI_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `GROQ_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `CLAUDE_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `OLLAMA_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `OPENROUTER_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |

**使用例:**
```bash
# Ollama で Function Calling 非対応モデルを使用する場合
export OLLAMA_FUNCTION_CALLING=0
xelyon --provider ollama --model phi3
```

### カスタムAPIエンドポイント

プロキシやセルフホスト環境で使用する場合に設定します。

```bash
# DeepSeek
export DEEPSEEK_API_URL=https://your-proxy.com/chat/completions

# Kimi (Moonshot)
export KIMI_API_URL=https://your-proxy.com/v1/chat/completions

# OpenAI
export OPENAI_API_URL=https://your-proxy.com/v1/chat/completions

# Claude (Anthropic)
export ANTHROPIC_API_URL=https://your-proxy.com/v1/messages

# Gemini
export GEMINI_API_URL=https://your-proxy.com/v1beta/models/YOUR_MODEL:streamGenerateContent?alt=sse

# Groq
export GROQ_API_URL=https://your-proxy.com/openai/v1/chat/completions

# OpenRouter
export OPENROUTER_API_URL=https://your-proxy.com/v1/chat/completions
```

DeepSeek の `DEEPSEEK_API_URL` は Chat Completions まで含む完全な endpoint override です。公式 DeepSeek 互換 endpoint は `/chat/completions` で終わります。OpenAI 互換 proxy が `/v1/chat/completions` を公開する場合も指定できますが、`doctor deepseek` では意図的な proxy path として warn になります。

Kimi の `KIMI_API_URL` は Chat Completions まで含む完全な endpoint override です。公式 Moonshot endpoint は `/v1/chat/completions` で終わります。別 path の proxy endpoint も指定できますが、`doctor kimi` では意図的な proxy path として warn になります。

Claude の `ANTHROPIC_API_URL` は Messages まで含む完全な endpoint override です。公式 Anthropic endpoint は `/v1/messages` で終わります。別 path の proxy endpoint も指定できますが、`doctor claude` では意図的な proxy path として warn になります。

Groq の `GROQ_API_URL` は Chat Completions まで含む完全な endpoint override です。公式 Groq endpoint は `/openai/v1/chat/completions` で終わります。OpenAI 互換 proxy が `/v1/chat/completions` を公開する場合も指定できますが、`doctor groq` では意図的な proxy path として warn になります。

OpenRouter の `OPENROUTER_API_URL` は Chat Completions endpoint または互換 proxy path を指定します。Claude 系 model で Anthropic Skin route を使う場合も `/v1/messages` は provider が派生するため、`OPENROUTER_API_URL` に直接 `/v1/messages` を指定しないでください。

## 設定ファイルの編集

### 手動編集

```bash
vi ~/.xelyon/config.yaml
```

### 設定ファイルの場所確認

```bash
ls -la ~/.xelyon/
```

### 設定のリセット

```bash
rm ~/.xelyon/config.yaml
xelyon  # 次回起動時にデフォルト設定が再作成される
```

## 使用例

### 1. 自動圧縮の閾値を調整

```yaml
compression:
  enabled: true          # デフォルトON
  trigger_percent: 70    # 70%でも圧縮（保険として残す）
  keep_recent: 15        # 最新15件を保持
  provider_thresholds:
    deepseek: 600000     # 明示 override が必要な場合だけ指定
```

### 1b. 自動圧縮を無効化

```yaml
compression:
  enabled: false         # 手動で /compress を使用
```

### 2. プロバイダーごとにモデルを変更

```yaml
provider_models:
  openai:
    default_model: gpt-4-turbo
  gemini:
    default_model: gemini-2.0-flash-exp
```

## トラブルシューティング

### 設定ファイルが読み込まれない

```bash
# YAMLシンタックスエラーをチェック
cat ~/.xelyon/config.yaml

# 設定ファイルを削除して再作成
rm ~/.xelyon/config.yaml
xelyon
```

### 環境変数が適用されない

```bash
# 環境変数が設定されているか確認
env | grep XELYON

# シェル設定ファイルを再読み込み
source ~/.zshrc  # または ~/.bashrc
```

### ツール出力表示設定 (`output`)

ツール出力の折りたたみ表示を設定します。長い出力を省略して見やすくします。

```yaml
output:
  max_lines: 5           # 折りたたみ前の最大表示行数
  assistant_updates: ""  # 空なら Normal=phase, Plan=verbose
```

#### `max_lines`
- **型**: integer
- **デフォルト**: `5`
- **環境変数**: `XELYON_OUTPUT_MAX_LINES`
- **説明**: ツール出力の折りたたみ前に表示する最大行数

**表示例:**

```
🔧 Tool: bash
   Command: git status
⎿  ブランチ main
   Your branch is up to date with 'origin/main'.
   Changes not staged for commit:
   ... +20 lines
```

**適用対象:**
- `bash` コマンド出力
- `list_dir` ディレクトリ一覧
- その他の長い出力を返すツール

#### `assistant_updates`
- **型**: string
- **許可値**: `verbose`, `phase`, `off`
- **デフォルト**: `""`（Normal Mode は `phase`、Plan Mode は `verbose`）
- **説明**: assistant prose の中間表示を制御します。`phase` は着手・フェーズ切替・エラー・最終応答だけに寄せ、`off` は中間 prose をさらに抑えます。`apply_patch` やツール出力の折りたたみルールには影響しません。

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [プロバイダー設定](providers.md)
- [LSP連携](lsp.md)
- [MCP連携](mcp.md)
