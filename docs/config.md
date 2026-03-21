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

20カテゴリ、50以上の設定項目をカテゴリ別に管理できます。変更は即座に `~/.xelyon/config.yaml` に保存されます。

**サポートする設定型:**
- bool（y/n で切り替え）
- int（数値入力）
- string（テキスト入力）
- select（番号選択）
- []string（項目追加/削除）
- map[string]string（エントリ追加/編集/削除）
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
default_provider: deepseek
# デフォルトで使用するモデル
default_model: deepseek-chat
# プロバイダーごとのモデル設定
provider_models:
    bedrock:
        default_model: global.anthropic.claude-sonnet-4-6-v1
        max_output_tokens: 64000
        anthropic_version: bedrock-2023-05-31
    claude:
        default_model: claude-sonnet-4-6
        max_output_tokens: 64000
        anthropic_version: "2023-06-01"
    deepseek:
        default_model: deepseek-chat
        max_output_tokens: 16384
    gemini:
        default_model: gemini-3.1-pro-preview-customtools
        max_output_tokens: 65536
    groq:
        default_model: meta-llama/llama-4-scout-17b-16e-instruct
        max_output_tokens: 8192
    ollama:
        default_model: qwen2.5-coder:7b
        max_output_tokens: 4096
    openai:
        default_model: gpt-5.2
        max_output_tokens: 16384
    openrouter:
        default_model: anthropic/claude-sonnet-4.6
        max_output_tokens: 64000

# ============================================================
# 一般設定
# ============================================================
general:
    # 表示言語（ja, en）
    language: ja
    # ツールループ最大回数（0で無制限）
    tool_loop_limit: 0

# ============================================================
# 会話履歴圧縮設定
# ============================================================
compression:
    # トークン使用率が閾値を超えたら自動圧縮
    auto_compress: true
    # トークン数の閾値（0 = 使用しない）
    threshold_tokens: 0
    # トークン使用率の閾値（%）
    threshold_percent: 80
    # 絶対トークン閾値（デフォルト100000、キャッシュ有無に関係なく発動）
    token_threshold: 0
    # 圧縮用モデル（空 = プロバイダー別デフォルト、main = メインモデル）
    model: ""
    # 圧縮時に保持する直近メッセージ数
    keep_recent: 20
    # プロバイダーのCompact APIを優先使用
    prefer_compact_api: true
    # Claude系の compact_20260112 を有効化
    claude_compaction: true
    # compact のトリガー閾値（デフォルト: 150000）
    compaction_trigger: 150000
    # Claude系の server-side tool clearing を有効化
    clear_tool_uses: true
    # clear_tool_uses のトリガー閾値（デフォルト: 80000）
    clear_tool_uses_trigger: 80000
    # tool_use 側の入力もクリアする
    clear_tool_inputs: false
    provider_thresholds:
        bedrock: 150000
        claude: 150000
        deepseek: 80000
        gemini: 180000
        openai: 100000
        openai:gpt-5.4: 260000
        openai:gpt-5.4-pro: 260000
        openrouter: 120000

# ============================================================
# ループ検知設定
# ============================================================
loop_detection:
    # 同じツール呼び出しの繰り返し回数でループと判定
    threshold: 3

# ============================================================
# APIリトライ設定
# ============================================================
api_retry:
    # リトライ回数
    count: 3
    # 初回リトライ待機時間（秒）
    initial_delay: 1
    # 最大待機時間（秒）
    max_delay: 30
    # タイムアウト（秒）
    timeout: 3600

# ============================================================
# 差分表示設定
# ============================================================
diff:
    # 差分表示時のコンテキスト行数
    context_lines: 10
    # 差分表示の最大行数（0で無制限）
    max_total_lines: 0

# ============================================================
# ツール確認設定
# ============================================================
tool_confirm:
    # 安全なツール（read_file等）を自動承認
    auto_approve_safe: true
    # 中程度のツール（write_file等）を自動承認
    auto_approve_medium: false

# ============================================================
# コマンドエイリアス設定
# ============================================================
# スラッシュコマンドの短縮名を定義
# 例: c → /compress, u → /use, h → /history
command_aliases:
    c: config
    u: use

# ============================================================
# プロンプトキャッシュ設定（Anthropic API cache_control）
# ============================================================
prompt_cache:
    # 有効化（cache_control ブレークポイントを設定）
    enabled: true
    cache_ttl: 5m

# ============================================================
# ペーストモード設定
# ============================================================
paste:
    # Bracketed Paste Mode を有効化（複数行ペースト対応）
    bracketed_paste: true
    # 最大行数
    max_lines: 10000
    # 最大バイト数
    max_bytes: 1048576
    # タイムアウト（秒）
    timeout_seconds: 60

# ============================================================
# ストリーミング設定
# ============================================================
streaming:
    # アイドルタイムアウト（秒）
    idle_timeout_seconds: 30
    thinking_timeout_seconds: 120
    # ファイル読み込み時にサイズ・行数を表示
    show_file_info: true
    # 検索時に進捗を表示
    show_search_progress: true
    # bashコマンドの出力をリアルタイム表示
    stream_bash_output: true

# ============================================================
# bashツール設定
# ============================================================
bash:
    # 安全レベル: strict / moderate / permissive
    safety_level: permissive
    # 追加の安全コマンド（例: - "npm run"）
    safe_commands: []
    # パイプを許可
    allow_pipe: true
    # リダイレクトを許可
    allow_redirect: true
    # インライン編集を許可（sed -i 等）
    allow_inline_edit: true
list_dir:
    additional_ignore_dirs: []

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
# git_add設定
# ============================================================
git_stage:
    # 複数ファイルをまとめて確認
    batch_confirm: true

# ============================================================
# Plan Mode設定
# ============================================================
plan_mode:
    # 最大リトライ回数
    max_retry: 10
    # ステップタイムアウト（秒）
    step_timeout: 600
    # Plan 承認後に調査フェーズの履歴をクリアして実装を開始
    clear_context_on_approval: true

# ============================================================
# LSP連携設定
# ============================================================
# 23言語のデフォルト設定が内蔵済み。通常は変更不要です。
# 詳細: docs/config.md の「LSP連携設定」セクション
lsp:
    # LSP連携の有効/無効
    enabled: true

# ============================================================
# OpenAI設定
# ============================================================
openai:
    # Responses APIを使用するモデル
    responses_api_models: []

# ============================================================
# Extended Thinking設定
# ============================================================
thinking:
    # 有効化
    enabled: false
    # レベル: low / medium / high / xhigh
    level: medium

# ============================================================
# ツール出力表示設定
# ============================================================
output:
    # 折りたたみ前の最大表示行数
    max_lines: 5

# ============================================================
# Web検索設定
# ============================================================
# ネイティブ Web 検索の実行プロバイダーとキャッシュ設定
# 未設定の場合はメインプロバイダーの検索を使用
# メインが非対応の場合は openai / gemini / claude のいずれかを設定
web_search:
    # 検索プロバイダー（openai / gemini / claude、未設定時はメインプロバイダーを使用）
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
    default_model: ""
    # 既定推論強度（off / low / medium / high）
    default_effort: ""
    # 同時実行上限（デフォルト: 5）
    max_concurrent: 5

# ============================================================
# Utility Model設定
# ============================================================
# main 推論や compression.model とは別の軽量補助モデル設定
# 初期実装では web_search 結果圧縮のような限定タスクだけに使用
utility_model:
    # utility model を有効化
    enabled: false
    tasks:
        - web_search_compaction

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
# フック設定
# ============================================================
# タスク完了時に自動実行するシェルコマンド（LSPチェック後）
# 変更ファイルは XELYON_CHANGED_FILES 環境変数で参照可能
hooks:
    # 完了時に実行するコマンド（例: go test ./...）
    on_completion: []
    on_step_complete: []
    # コマンドタイムアウト（秒）（デフォルト: 60）
    timeout: 60
    # フック失敗時の最大リトライ回数（デフォルト: 3）
    max_retry: 3
```
<!-- CONFIG-EXAMPLE-END -->

## 設定項目詳細

### プロバイダー・モデル設定

#### `default_provider`
- **型**: string
- **デフォルト**: `deepseek`
- **説明**: デフォルトで使用するプロバイダー
- **選択肢**: `deepseek`, `openai`, `gemini`, `claude`, `ollama`, `groq`

#### `default_model`
- **型**: string
- **デフォルト**: `deepseek-chat`
- **説明**: デフォルトで使用するモデル

#### `provider_models`
- **型**: map
- **説明**: プロバイダーごとのデフォルトモデル設定と出力トークン制限
- **サブキー**:
  - `default_model`: プロバイダーのデフォルトモデル名
  - `max_output_tokens`: プロバイダーの出力トークン上限（デフォルト値）
  - `model_overrides`: 特定モデル別の出力トークン上限設定（オプション）
    - **用途**: 既知モデルマップにないモデルや、デフォルト値を上書きしたい場合に指定
    - **優先度**: `model_overrides[model]` > 既知モデルマップ > `max_output_tokens`
    - **補足**: 既知モデル（`deepseek-chat`, `gpt-5.2`, `gemini-2.5-flash` 等）は自動解決されるため、通常は設定不要
- **例**:
  ```yaml
  provider_models:
    deepseek:
      default_model: deepseek-chat
      max_output_tokens: 16384
      # 既知モデルは自動解決されるため model_overrides 不要
      # カスタムモデルのみ指定:
      model_overrides:
        my-custom-model:
          max_output_tokens: 32768
  ```

### 会話履歴圧縮設定 (`compression`)

Context Window（コンテキストウィンドウ）を管理し、トークン上限エラーを防ぎます。

#### `auto_compress`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: トークン使用率が閾値を超えた際に自動圧縮を実行
- **補足**: コスト削減・エラー防止のためデフォルトON

#### `threshold_percent`
- **型**: integer
- **デフォルト**: `80`
- **説明**: 自動圧縮を実行する使用率閾値（%）
- **補足**: モデルのコンテキストウィンドウに対する使用率

#### `threshold_tokens`
- **型**: integer
- **デフォルト**: `0`
- **説明**: 自動圧縮を実行するトークン閾値（絶対値）
- **補足**: `0` = 使用率ベース（`threshold_percent`を使用）

#### `token_threshold`
- **型**: integer
- **デフォルト**: `0`
- **説明**: Context トークン数のカスタム絶対閾値
- **補足**: `0` = 無効。明示設定した場合のみ安全弁として使用

#### `model`
- **型**: string
- **デフォルト**: `""`
- **説明**: 圧縮時に使用するモデル名
- **補足**: 空文字はプロバイダー別デフォルト、`main` は現在のメインモデルを使用

#### `keep_recent`
- **型**: integer
- **デフォルト**: `20`
- **説明**: 圧縮時に保持する最新メッセージ数

#### `prefer_compact_api`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: OpenAI Compact API を優先的に使用
- **補足**: OpenAI Responses API 対応モデル使用時のみ有効
- **動作**: 自動圧縮時に `/responses/compact` エンドポイントを呼び出し
- **フォールバック**: Compact API 失敗時は LLM サマリーに自動フォールバック

#### `claude_compaction`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: Claude 系プロバイダーで `compact_20260112` を有効化
- **補足**: Claude / Bedrock / OpenRouter(Claude models) の対応モデルでのみ有効
- **補足**: `clear_tool_uses` とは独立設定。`false` でも `clear_tool_uses: true` なら tool clearing は有効

#### `compaction_trigger`
- **型**: integer
- **デフォルト**: `150000`
- **説明**: `compact_20260112` を発動する input token 閾値
- **補足**: 最小 `50000`。それ未満は runtime で `50000` に補正

#### `clear_tool_uses`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: Claude 系プロバイダーで `clear_tool_uses_20250919` を有効化
- **補足**: 古い `tool_use` / `tool_result` ペア構造を server-side で削除し、compaction 前に入力トークンを節約
- **補足**: `claude_compaction` が `false` でも単独で有効化可能

#### `clear_tool_uses_trigger`
- **型**: integer
- **デフォルト**: `80000`
- **説明**: `clear_tool_uses_20250919` を発動する input token 閾値
- **補足**: 通常は `compaction_trigger` より小さくして、`clear_tool_uses` → `compact` の順で発動させる
- **補足**: 最小 `50000`。それ未満は runtime で `50000` に補正

#### `clear_tool_inputs`
- **型**: boolean
- **デフォルト**: `false`
- **説明**: `clear_tool_uses` 実行時に `tool_use` 側の入力引数も削除
- **補足**: デフォルトでは `tool_result` の clearing のみ。セッション再開時の履歴再構築への影響を避けるため、通常は `false` 推奨

**自動圧縮の動作:**
1. API呼び出し成功後に pricing cliff とプロバイダー別閾値を評価
2. `previous_response_id` や Claude Compaction が使える場合は自動圧縮をスキップ
3. `token_threshold` を明示設定している場合のみ、その値を追加の安全弁として評価
4. それ以外は `threshold_percent`（デフォルト80%）や `threshold_tokens` を評価
5. 圧縮時に通知を表示し、無効化方法も案内

```
🗜️ Auto-compressing: context 151K exceeds 150K custom threshold...
   Before: 162,000 tokens → After: 45,000 tokens
   💡 Disable with: xelyon config set compression.auto_compress false
```

### ループ検知設定 (`loop_detection`)

#### `threshold`
- **型**: integer
- **デフォルト**: `3`
- **説明**: 同じツール呼び出しが繰り返された場合に警告する回数
- **補足**: 無限ループ防止機能

### APIリトライ設定 (`api_retry`)

#### `count`
- **型**: integer
- **デフォルト**: `3`
- **説明**: API呼び出し失敗時のリトライ回数

#### `initial_delay`
- **型**: integer
- **デフォルト**: `1`
- **説明**: 初回リトライまでの待機秒数

#### `max_delay`
- **型**: integer
- **デフォルト**: `30`
- **説明**: リトライ時の最大待機秒数
- **補足**: Exponential Backoff（指数バックオフ）方式

#### `timeout`
- **型**: integer
- **デフォルト**: `300`（5分）
- **説明**: API呼び出しのタイムアウト秒数
- **補足**: 長時間のレスポンスが必要な場合は増やす（例: 7200 = 2時間）

### 差分表示設定 (`diff`)

#### `context_lines`
- **型**: integer
- **デフォルト**: `10`
- **説明**: ファイル編集時の差分表示行数
- **補足**: `0` で省略なし（全行表示）

### ストリーミング設定 (`streaming`)

```yaml
streaming:
  idle_timeout_seconds: 30
  show_file_info: true
  show_search_progress: true
  stream_bash_output: true
```

#### `idle_timeout_seconds`
- **型**: integer
- **デフォルト**: `30`
- **説明**: ストリーミングレスポンスのアイドルタイムアウト秒数
- **補足**: データ受信がこの秒数続くとタイムアウト。データが来続けている間はタイムアウトしない

#### `show_file_info`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: ファイル読み込み時にサイズと行数を表示
- **例**: `📖 Reading main.go (2.3KB, 150 lines)`

#### `show_search_progress`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: 検索中にリアルタイムで進捗を表示
- **例**: `🔍 Searching... 42 matches found`

#### `stream_bash_output`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: bashコマンドの出力をリアルタイムでストリーミング表示
- **補足**: `false` の場合、コマンド完了後に一括表示

### bashツール設定 (`bash`)

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

### git_stage

git_addツールの動作を設定します。

```yaml
git_stage:
  # 複数ファイルのバッチ確認UIを有効化
  batch_confirm: true
```

#### `batch_confirm`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: 複数ファイルをステージングする際のバッチ確認UIを有効化

**バッチ確認UI**:
```
📦 Git Stage Batch / Gitバッチステージング
   3 files / 3ファイル
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  - file1.go
  - file2.go
  - file3.go

Stage all? (y/n/s=select) / 全てステージング？ (y/n/s=個別選択):
```

- `y`: 全ファイルをステージング
- `n`: キャンセル
- `s`: 個別選択モード（1ファイルずつ確認）

### Plan Mode設定 (`plan_mode`)

Plan Modeは計画駆動型の実行モードです。調査フェーズで情報を収集し、計画を生成・承認後にステップごとに順次実行します。

```yaml
plan_mode:
  max_retry: 10            # 最大リトライ回数
  step_timeout: 600        # ステップタイムアウト（秒）
```

#### 設定項目

##### `max_retry`
- **型**: integer
- **デフォルト**: `10`
- **説明**: ステップ失敗時の最大リトライ回数

##### `step_timeout`
- **型**: integer
- **デフォルト**: `600`（10分）
- **説明**: 各ステップのタイムアウト秒数

#### 失敗時フロー

ステップ実行に失敗した場合、以下の順序で処理されます：

1. **リトライ**: `max_retry` 回まで自動リトライ
2. **ユーザー確認**: リトライ上限到達後、ユーザーに選択肢を提示
   - リトライ
   - コメント付きリトライ
   - スキップ
   - 中止

#### 使用例

```yaml
plan_mode:
  max_retry: 5
  step_timeout: 300
```

### OpenAI設定 (`openai`)

OpenAI固有の設定を行います。

```yaml
openai:
  responses_api_models:
    - gpt-5.2-codex
    - gpt-5.1-codex
    - gpt-5.1-codex-max
    - gpt-5-codex
```

#### `responses_api_models`
- **型**: string[]
- **デフォルト**: `["gpt-5.2-codex", "gpt-5.1-codex", "gpt-5.1-codex-max", "gpt-5-codex"]`
- **説明**: Responses API を使用するモデルのリスト

**動作:**
- リストに含まれるモデル名が指定された場合、自動的に Responses API (`/v1/responses`) を使用
- それ以外のモデルは従来の Chat Completions API (`/v1/chat/completions`) を使用
- モデル名の一部一致で判定（`gpt-5.2-codex-max` も対象になる）

**Responses API の特徴:**
- 会話コンテキストをサーバー側で管理
- Compact API による効率的な圧縮
- ZDR（Zero Data Retention）対応

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

### Extended Thinking設定 (`thinking`)

> **重要**: この機能は対応モデルでのみ動作します。
> - **Claude**: Sonnet 4 以降
> - **OpenAI**: gpt-5.2 系
> - **Gemini**: 2.5 Pro 系（Flash は非対応）
> - **DeepSeek**: 自動で reasoner モデルに切り替わります

```yaml
thinking:
  enabled: false    # デフォルト OFF
  level: medium     # low/medium/high/xhigh
```

#### `enabled`
- **型**: boolean
- **デフォルト**: `false`
- **説明**: Extended Thinking を有効化

#### `level`
- **型**: string
- **デフォルト**: `medium`
- **選択肢**: `low`, `medium`, `high`, `xhigh`
- **説明**: 推論の深さレベル

**レベル別パラメータ:**

| Level | Claude 4.6 (effort) | Claude 4.5以前 (budget_tokens) | OpenAI (effort) | Gemini (budget) |
|-------|---------------------|-------------------------------|-----------------|-----------------|
| low | low | 5,000 | low | 5,000 |
| medium | medium | 10,000 | medium | 10,000 |
| high | high | 20,000 | high | 20,000 |
| xhigh | max (Opus) / high | 40,000 | xhigh | 40,000 |

> Claude Opus 4.6 / Sonnet 4.6 では `type: "adaptive"` + `output_config.effort` を使用。
> それ以前のモデルでは従来の `type: "enabled"` + `budget_tokens` を使用。
> `xhigh` の `max` は Opus 4.6 のみ対応（Sonnet 4.6 では `high` にフォールバック）。

**コマンドで切り替え:**

```
/think          # 現在の状態を表示
/think on       # 有効化（現在のレベルで）
/think off      # 無効化
/think low      # low レベルで有効化
/think medium   # medium レベルで有効化
/think high     # high レベルで有効化
/think xhigh    # xhigh レベルで有効化
```

設定ファイルを変更せずに、セッション中にリアルタイムで切り替えられます。

**Codex モデルの制限:**

OpenAI Codex モデル（`gpt-5.2-codex`, `gpt-5.1-codex` 等）は reasoning が必須のため、`/think off` を実行しても完全に無効化されません：

- `/think off` → `low` レベルにフォールバック（警告メッセージ表示）
- Codex 以外のモデルでは通常通り無効化されます

### ツール確認設定 (`tool_confirm`)

```yaml
tool_confirm:
  auto_approve_safe: true    # SafetyHigh（read_file等）を確認なしで実行
  auto_approve_medium: false # SafetyMedium（str_replace等）を確認なしで実行
```

#### `auto_approve_safe`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: SafetyHigh ツール（read_file, list_dir, git_status, git_log, git_diff, search_code 等）を確認なしで実行

#### `auto_approve_medium`
- **型**: boolean
- **デフォルト**: `false`
- **説明**: SafetyMedium ツール（str_replace, write_file, web_search 等）を確認なしで実行

**安全性レベル一覧:**

| レベル | ツール例 | 説明 |
|--------|---------|------|
| SafetyHigh | read_file, list_dir, search_* | 読み取り専用 |
| SafetyMedium | str_replace, write_file | 書き込み（リカバリ可能） |
| SafetyLow | delete_file, bash, git_push | 破壊的操作（常に確認必須） |

### プロンプトキャッシュ設定 (`prompt_cache`)

```yaml
prompt_cache:
  enabled: true         # キャッシュを有効化
  max_entries: 100      # 最大エントリ数
  ttl_seconds: 300      # TTL（秒）
```

#### `enabled`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: System Promptのキャッシュを有効化（Claude使用時のコスト削減）

#### `max_entries`
- **型**: integer
- **デフォルト**: `100`
- **説明**: キャッシュの最大エントリ数（LRU方式）

#### `ttl_seconds`
- **型**: integer
- **デフォルト**: `300`（5分）
- **説明**: キャッシュのTTL（有効期限）

### ペーストモード設定 (`paste`)

```yaml
paste:
  max_lines: 10000       # 最大行数
  max_bytes: 1048576     # 最大バイト数（1MB）
  timeout_seconds: 60    # タイムアウト秒
```

#### `max_lines`
- **型**: integer
- **デフォルト**: `10000`
- **説明**: ペーストモードで受け付ける最大行数

#### `max_bytes`
- **型**: integer
- **デフォルト**: `1048576`（1MB）
- **説明**: ペーストモードで受け付ける最大バイト数

#### `timeout_seconds`
- **型**: integer
- **デフォルト**: `60`
- **説明**: ペーストモードのタイムアウト秒数

### コマンドエイリアス設定 (`command_aliases`)

よく使うコマンドにエイリアスを設定できます。

```yaml
command_aliases:
  c: compress    # /c → /compress
  u: use         # /u → /use
  h: history     # /h → /history
  p: plan        # /p → /plan
```

## 環境変数

### API キー

```bash
# DeepSeek
export DEEPSEEK_API_KEY=sk-...

# OpenAI
export OPENAI_API_KEY=sk-...

# Gemini
export GEMINI_API_KEY=...

# Claude (Anthropic)
export ANTHROPIC_API_KEY=sk-ant-...

# Groq
export GROQ_API_KEY=gsk_...

# Ollama（環境変数不要・ローカル実行）
```

### Web検索

`web_search` は OpenAI / Gemini / Claude のネイティブ検索を使います。

- **`web_search.provider` 未設定**: メインプロバイダーが OpenAI / Gemini / Claude の場合、そのままネイティブ検索を使用
- **`web_search.provider` 設定あり**: 指定した検索プロバイダーを使用
- **メインが非対応**: DeepSeek / OpenRouter / Groq / Ollama / Bedrock などでは `web_search.provider` の設定が必要

#### 設定例

```yaml
web_search:
  provider: gemini
```

#### 必要なAPIキー

検索に使うプロバイダーに応じて、以下のいずれかを設定してください。

```bash
# OpenAI を検索に使う場合
export OPENAI_API_KEY=sk-...

# Gemini を検索に使う場合
export GEMINI_API_KEY=...

# Claude を検索に使う場合
export ANTHROPIC_API_KEY=sk-ant-...
```

Gemini API キーは無料で取得できます: https://aistudio.google.com/apikey

#### 動作例

```yaml
# メインが DeepSeek でも、検索だけ Gemini を使う
web_search:
  provider: gemini
```

```bash
xelyon
> 最新のGo言語の情報を検索して
```

メインプロバイダーが OpenAI / Gemini / Claude の場合は、`web_search.provider` を省略するとそのままネイティブ検索を使用します。

### プロバイダー・モデル指定

```bash
# プロバイダー指定
export XELYON_PROVIDER=deepseek

# モデル指定
export XELYON_MODEL=deepseek-chat
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

# APIリトライ回数
export XELYON_API_RETRY_COUNT=5

# API初回待機秒数
export XELYON_API_RETRY_INITIAL_DELAY=2

# API最大待機秒数
export XELYON_API_RETRY_MAX_DELAY=60

# 差分表示行数
export XELYON_DIFF_CONTEXT_LINES=20
```

### Function Calling（ツール呼び出し）

各プロバイダーのFunction Calling（ツール呼び出し）機能を無効化できます。
モデルがFunction Callingに対応していない場合や、テキストベースのツール呼び出しに戻したい場合に使用します。

```bash
# OpenAI Function Calling 無効化
export OPENAI_FUNCTION_CALLING=0

# DeepSeek Function Calling 無効化
export DEEPSEEK_FUNCTION_CALLING=0

# Gemini Function Calling 無効化
export GEMINI_FUNCTION_CALLING=0

# Groq Function Calling 無効化
export GROQ_FUNCTION_CALLING=0

# Claude Tool Use 無効化
export CLAUDE_FUNCTION_CALLING=0

# Ollama Function Calling 無効化
export OLLAMA_FUNCTION_CALLING=0
```

| 環境変数 | デフォルト | 説明 |
|---------|-----------|------|
| `OPENAI_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `DEEPSEEK_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `GEMINI_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `GROQ_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `CLAUDE_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |
| `OLLAMA_FUNCTION_CALLING` | `1`（有効） | `0` で無効化 |

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
export DEEPSEEK_API_URL=https://your-proxy.com/v1/chat/completions

# OpenAI
export OPENAI_API_URL=https://your-proxy.com/v1/chat/completions

# Claude (Anthropic)
export ANTHROPIC_API_URL=https://your-proxy.com/v1/messages

# Gemini
export GEMINI_API_URL=https://your-proxy.com/v1beta/models

# Groq
export GROQ_API_URL=https://your-proxy.com/openai/v1/chat/completions
```

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

### 1. 自動圧縮の閾値と圧縮モデルを調整

```yaml
compression:
  auto_compress: true       # デフォルトON
  token_threshold: 150000   # 明示設定した場合のみ custom threshold として使用
  threshold_percent: 70     # 70%でも圧縮（保険として残す）
  model: ""                # プロバイダー別デフォルト圧縮モデル
  keep_recent: 15           # 最新15件を保持
```

### 1b. 自動圧縮を無効化

```yaml
compression:
  auto_compress: false      # 手動で /compress を使用
```

または:
```bash
xelyon config set compression.auto_compress false
```

### 2. APIリトライを増やす

```yaml
api_retry:
  count: 5
  initial_delay: 2
  max_delay: 60
```

または環境変数で:

```bash
export XELYON_API_RETRY_COUNT=5
```

### 3. ループ検知を緩和

```yaml
loop_detection:
  threshold: 5
```

### 4. 差分を全行表示

```yaml
diff:
  context_lines: 0
```

### 5. プロバイダーごとにモデルを変更

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

### APIリトライが動かない

`config.yaml` で `api_retry.count` が `0` になっていないか確認:

```yaml
api_retry:
  count: 3  # 0 以外を指定
```

### ツール出力表示設定 (`output`)

ツール出力の折りたたみ表示を設定します。長い出力を省略して見やすくします。

```yaml
output:
  max_lines: 5  # 折りたたみ前の最大表示行数
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

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [プロバイダー設定](providers.md)
- [LSP連携](lsp.md)
- [MCP連携](mcp.md)
