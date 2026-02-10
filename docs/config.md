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
        default_model: global.anthropic.claude-opus-4-5-20251101-v1:0
        max_output_tokens: 32768
        anthropic_version: bedrock-2023-05-31
    claude:
        default_model: claude-sonnet-4-5-20250514
        max_output_tokens: 16384
        anthropic_version: "2023-06-01"
    deepseek:
        default_model: deepseek-chat
        max_output_tokens: 16384
    gemini:
        default_model: gemini-3-flash-preview
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
        default_model: anthropic/claude-opus-4.5
        max_output_tokens: 32768

# ============================================================
# 一般設定
# ============================================================
general:
    # 表示言語（ja, en）
    language: ja
    # ツールループ最大回数
    tool_loop_limit: 80

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
    # 圧縮時に保持する直近メッセージ数
    keep_recent: 20
    # プロバイダーのCompact APIを優先使用
    prefer_compact_api: true
    claude_compaction: true
    compaction_trigger: 150000

# ============================================================
# バックアップ設定
# ============================================================
backup:
    # 保持する世代数
    max_generations: 5

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
# プロンプトキャッシュ設定（Claude専用）
# ============================================================
prompt_cache:
    # 有効化
    enabled: true
    # 最大エントリ数
    max_entries: 100
    # キャッシュTTL（秒）
    ttl_seconds: 300

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
    idle_timeout_seconds: 3600
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

# ============================================================
# コード健全性チェック設定
# ============================================================
code_health:
    # 有効化
    enabled: true
    # ファイルの最大行数警告
    max_file_lines: 300
    # 関数の最大行数警告
    max_function_lines: 50
    # 変更時に自動提案
    auto_suggest: true
    on_change:
        - check_file_size
        - check_function_size

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
    # 並列モード有効化
    parallel: false
    # 並列ワーカー数
    max_workers: 3
    # Supervisor用モデル（空=メインモデル）
    supervisor_model: ""
    # Worker用モデル（空=メインモデル）
    worker_model: ""
    # 最大リトライ回数
    max_retry: 10
    # ステップタイムアウト（秒）
    step_timeout: 600
    # 確認レベル: all / dangerous / none
    confirm_level: dangerous

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
    responses_api_models:
        - gpt-5.2-codex
        - gpt-5.1-codex
        - gpt-5.1-codex-max
        - gpt-5-codex
        - gpt-5.2

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
# Serper API Web検索のキャッシュ設定
web_search:
    # キャッシュを有効化（デフォルト: true）
    cache_enabled: true
    # キャッシュTTL秒数（デフォルト: 1800 = 30分）
    cache_ttl: 1800
    # 最大キャッシュ数（デフォルト: 100）
    cache_size: 100
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

#### `keep_recent`
- **型**: integer
- **デフォルト**: `10`
- **説明**: 圧縮時に保持する最新メッセージ数

#### `prefer_compact_api`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: OpenAI Compact API を優先的に使用
- **補足**: OpenAI Responses API 対応モデル使用時のみ有効
- **動作**: 自動圧縮時に `/responses/compact` エンドポイントを呼び出し
- **フォールバック**: Compact API 失敗時は LLM サマリーに自動フォールバック

**自動圧縮の動作:**
1. API呼び出し成功後にトークン使用率をチェック
2. `threshold_percent`（デフォルト80%）を超えた場合に圧縮実行
3. 圧縮時に通知を表示（無言で実行しない）
4. 無効化方法も案内

```
🗜️ Auto-compressing history (82% threshold reached)...
   Before: 162,000 tokens → After: 45,000 tokens
   💡 Disable with: xelyon config set compression.auto_compress false
```

### バックアップ設定 (`backup`)

#### `max_generations`
- **型**: integer
- **デフォルト**: `5`
- **説明**: ファイル編集時のバックアップ世代数
- **補足**: `.bak`, `.bak.1`, `.bak.2` のように保存

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

Plan Modeは計画駆動型の実行モードです。Supervisor-Workerアーキテクチャによる並列実行をサポートしています。

```yaml
plan_mode:
  # 並列実行モード（Phase 3）
  parallel: false          # 並列モード有効化
  max_workers: 3           # 並列ワーカー数

  # モデル設定
  supervisor_model: ""     # Supervisor用モデル（空=メインモデル）
  worker_model: ""         # Worker用モデル（空=メインモデル）

  # リトライ・タイムアウト
  max_retry: 10            # 最大リトライ回数
  step_timeout: 600        # ステップタイムアウト（秒）

  # 確認レベル（並列モード用）
  confirm_level: dangerous # all / dangerous / none
```

#### Supervisor-Workerアーキテクチャ

`parallel: true` を設定すると、Supervisor-Worker型の並列実行モードが有効になります。

```
┌─────────────────────────────────────────────────────────────┐
│                      Supervisor                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐│
│  │ 調査管理    │ │ 計画生成    │ │ 実行管理・リトライ       ││
│  └─────────────┘ └─────────────┘ └─────────────────────────┘│
│                            │                                  │
│  ┌─────────────────────────┴──────────────────────────────┐ │
│  │                 SharedContext                          │ │
│  │  - 調査結果キャッシュ                                  │ │
│  │  - ファイル変更トラッキング                            │ │
│  │  - Worker間依存情報                                    │ │
│  └────────────────────────────────────────────────────────┘ │
│                            │                                  │
│           ┌────────────────┼────────────────┐                │
│           ▼                ▼                ▼                │
│    ┌──────────┐     ┌──────────┐     ┌──────────┐           │
│    │ Worker 1 │     │ Worker 2 │     │ Worker 3 │           │
│    └──────────┘     └──────────┘     └──────────┘           │
└─────────────────────────────────────────────────────────────┘
```

**動作フロー:**
1. **調査フェーズ**: Supervisorが調査クエリを生成し、Workerが並列で調査を実行
2. **計画生成**: 調査結果を集約し、`depends_on`付きの実行計画を生成
3. **実装フェーズ**: 依存関係を解決しながら、Workerが並列でステップを実行

#### 設定項目

##### `parallel`
- **型**: boolean
- **デフォルト**: `false`
- **説明**: 並列実行モードを有効化

##### `max_workers`
- **型**: integer
- **デフォルト**: `3`
- **説明**: 同時に実行するWorkerの最大数

##### `supervisor_model`
- **型**: string
- **デフォルト**: `""`（メインモデルを使用）
- **説明**: Supervisor用のモデル（調査クエリ生成、計画生成）

##### `worker_model`
- **型**: string
- **デフォルト**: `""`（メインモデルを使用）
- **説明**: Worker用のモデル（ステップ実行）

##### `max_retry`
- **型**: integer
- **デフォルト**: `10`
- **説明**: ステップ失敗時の最大リトライ回数

##### `step_timeout`
- **型**: integer
- **デフォルト**: `600`（10分）
- **説明**: 各ステップのタイムアウト秒数

##### `confirm_level`
- **型**: string
- **デフォルト**: `"dangerous"`
- **選択肢**: `all`, `dangerous`, `none`
- **説明**: 並列モードでのツール確認レベル

| 設定 | 計画承認 | 実行中の確認 |
|------|---------|------------|
| all | あり | 全ツール確認 |
| dangerous | あり | 危険ツール（delete_file等）のみ |
| none | あり | なし |

#### 非推奨設定（後方互換性）

以下の設定は非推奨ですが、後方互換性のためサポートされています。

##### `max_parallel_steps`（非推奨）
- **代替**: `max_workers`
- **説明**: `max_workers` が未設定の場合に自動マイグレーション

##### `auto_retry`（非推奨）
- **代替**: `max_retry`
- **説明**: `max_retry` が未設定の場合に自動マイグレーション

#### 失敗時フロー

Worker がステップ実行に失敗した場合、以下の順序で処理されます：

1. **リトライ**: `max_retry` 回まで自動リトライ（失敗理由を SharedContext に追加して別アプローチを促す）
2. **ユーザー確認**: リトライ上限到達後、ユーザーに選択肢を提示
   - リトライ
   - コメント付きリトライ
   - スキップ
   - 中止

※ 確認が必要なツール（`confirm_level` に該当）はリトライをスキップし、即座にユーザー確認へ移行します。

#### 使用例

**基本的な並列モード設定:**
```yaml
plan_mode:
  parallel: true
  max_workers: 3
  confirm_level: dangerous
```

**Worker に軽量モデルを指定:**
```yaml
plan_mode:
  parallel: true
  max_workers: 3
  worker_model: "claude-3-haiku-20240307"
  max_retry: 5
```

**完全自動実行（確認なし）:**
```yaml
plan_mode:
  parallel: true
  max_workers: 5
  confirm_level: none
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

| Level | Claude (budget_tokens) | OpenAI (effort) | Gemini (budget) |
|-------|------------------------|-----------------|-----------------|
| low | 5,000 | low | 5,000 |
| medium | 10,000 | medium | 10,000 |
| high | 20,000 | high | 20,000 |
| xhigh | 40,000 | high | 40,000 |

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
- **説明**: SafetyHigh ツール（read_file, list_dir, search_file, search_code, git_status, git_log, git_diff 等）を確認なしで実行

#### `auto_approve_medium`
- **型**: boolean
- **デフォルト**: `false`
- **説明**: SafetyMedium ツール（str_replace, write_file, append_file, copy_file, git_add, git_commit 等）を確認なしで実行

**安全性レベル一覧:**

| レベル | ツール例 | 説明 |
|--------|---------|------|
| SafetyHigh | read_file, list_dir, search_* | 読み取り専用 |
| SafetyMedium | str_replace, write_file, git_commit | 書き込み（リカバリ可能） |
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
- **説明**: System PromptやRepo Mapのキャッシュを有効化（Claude使用時のコスト削減）

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

### コード健全性チェック設定 (`code_health`)

```yaml
code_health:
  enabled: true           # 健全性チェックを有効化
  max_file_lines: 300     # ファイル行数上限
  max_function_lines: 50  # 関数行数上限
  auto_suggest: true      # 閾値超過時に自動提案
  on_change:              # 変更時チェック項目
    - check_file_size
    - check_function_size
```

#### `enabled`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: コード健全性チェックを有効化

#### `max_file_lines`
- **型**: integer
- **デフォルト**: `300`
- **説明**: ファイル行数の上限（超過時に警告）

#### `max_function_lines`
- **型**: integer
- **デフォルト**: `50`
- **説明**: 関数行数の上限（超過時に警告）

#### `auto_suggest`
- **型**: boolean
- **デフォルト**: `true`
- **説明**: 閾値超過時に自動でリファクタリングを提案

#### `on_change`
- **型**: string[]
- **デフォルト**: `["check_file_size", "check_function_size"]`
- **説明**: ファイル変更時に実行するチェック項目

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

### Web検索（Serper API）

**オプション機能**: `web_search`ツールを使用する場合のみ必要です。

```bash
# Serper API キー
export SERPER_API_KEY=your_serper_api_key_here
```

#### Serper APIとは

[Serper](https://serper.dev)は、Google検索結果を取得できるAPIサービスです。

**特徴**:
- Google検索結果を高速に取得
- 無料枠: 2,500クエリ/月
- 有料プラン: $50/月〜（100,000クエリ）

#### APIキーの取得方法

1. [https://serper.dev](https://serper.dev)にアクセス
2. GitHubアカウントでサインアップ
3. ダッシュボードからAPIキーを取得
4. `.env`ファイルまたは環境変数に設定

```bash
# .envファイルに追加
echo "SERPER_API_KEY=your_api_key_here" >> .env

# または環境変数で設定
export SERPER_API_KEY=your_api_key_here
```

#### 使用例

```bash
xelyon
> 最新のGo言語の情報を検索して

# AIがweb_searchツールを使って検索結果を取得
```

**注意**: APIキーが未設定の場合、`web_search`ツールは使用できませんが、他のツールは正常に動作します。

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

# Serper (Web検索)
export SERPER_API_URL=https://your-proxy.com/search
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

### 1. 自動圧縮の閾値を変更

```yaml
compression:
  auto_compress: true       # デフォルトON
  threshold_percent: 70     # 70%で圧縮（デフォルト: 80%）
  keep_recent: 15           # 最新15件を保持（デフォルト: 10）
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
- `search_code` / `search_file` 検索結果
- その他の長い出力を返すツール

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [プロバイダー設定](providers.md)
- [LSP連携](lsp.md)
- [MCP連携](mcp.md)
