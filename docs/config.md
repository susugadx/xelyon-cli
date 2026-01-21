# 設定リファレンス

XELYON CLIの設定方法と全オプションのリファレンスです。

## 設定ファイル

設定ファイルは `~/.xelyon/config.yaml` に保存されます。

初回起動時に自動的にデフォルト設定ファイルが作成されます。

## 設定優先順位

設定は以下の優先順位で適用されます（上が優先）:

1. **コマンドラインフラグ** (`--provider`, `--model` など)
2. **環境変数** (`XELYON_PROVIDER`, `DEEPSEEK_API_KEY` など)
3. **設定ファイル** (`~/.xelyon/config.yaml`)
4. **デフォルト値**

## 設定ファイル構造

### 完全な設定例

```yaml
# プロバイダー・モデル設定
default_provider: deepseek
default_model: deepseek-coder

provider_models:
  deepseek:
    default_model: deepseek-coder
  openai:
    default_model: gpt-5.2
  gemini:
    default_model: gemini-2.5-flash
  claude:
    default_model: claude-sonnet-4-5-20250514
  ollama:
    default_model: qwen2.5-coder:7b
  groq:
    default_model: meta-llama/llama-4-scout-17b-16e-instruct

# 会話履歴圧縮（Context Window管理）
compression:
  auto_compress: true       # 自動圧縮を有効化（デフォルト: ON）
  threshold_percent: 80     # 自動圧縮の使用率閾値（%）
  threshold_tokens: 0       # トークン閾値（0 = 使用率ベース）
  keep_recent: 10           # 保持する最新メッセージ数

# バックアップファイル
backup:
  max_generations: 5        # 保持する世代数

# ループ検知
loop_detection:
  threshold: 3              # 同じツール呼び出しの許容回数

# APIリトライ
api_retry:
  count: 3                  # リトライ回数
  initial_delay: 1          # 初回待機秒数
  max_delay: 30             # 最大待機秒数
  timeout: 300              # タイムアウト秒（5分）

# 差分表示
diff:
  context_lines: 10         # 差分表示行数（0で省略なし）

# ストリーミング設定
streaming:
  idle_timeout_seconds: 30  # アイドルタイムアウト秒（データ受信がない場合）

# ツール確認
tool_confirm:
  auto_approve_safe: true    # SafetyHigh（読み取り）は確認なしで実行
  auto_approve_medium: false # SafetyMedium（書き込み）は確認必要（デフォルト）

# プロンプトキャッシュ
prompt_cache:
  enabled: true             # キャッシュを有効化（推奨）
  max_entries: 100          # 最大エントリ数
  ttl_seconds: 300          # TTL（秒）

# ペーストモード
paste:
  max_lines: 10000          # 最大行数
  max_bytes: 1048576        # 最大バイト数（1MB）
  timeout_seconds: 60       # タイムアウト秒

# コード健全性チェック
code_health:
  enabled: true             # 有効化
  max_file_lines: 300       # ファイル行数上限
  max_function_lines: 50    # 関数行数上限
  auto_suggest: true        # 自動提案

# bashツール設定
bash:
  safety_level: moderate      # strict, moderate, permissive
  safe_commands:              # 追加の安全コマンド
    - "npm run"
    - "npm test"
    - "cargo build"
    - "make"
  allow_pipe: true            # パイプを許可
  allow_redirect: false       # リダイレクトを許可
  allow_inline_edit: false    # sed -i等を許可

# git_add設定
git_stage:
  batch_confirm: true         # 複数ファイルのバッチ確認UI

# Plan Mode設定
plan_mode:
  max_parallel_steps: 3       # 並列実行数（デフォルト: 3）

# コマンドエイリアス
command_aliases:
  t: test
  r: review
```

## 設定項目詳細

### プロバイダー・モデル設定

#### `default_provider`
- **型**: string
- **デフォルト**: `deepseek`
- **説明**: デフォルトで使用するプロバイダー
- **選択肢**: `deepseek`, `openai`, `gemini`, `claude`, `ollama`, `groq`

#### `default_model`
- **型**: string
- **デフォルト**: `deepseek-coder`
- **説明**: デフォルトで使用するモデル

#### `provider_models`
- **型**: map
- **説明**: プロバイダーごとのデフォルトモデル設定

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

#### `idle_timeout_seconds`
- **型**: integer
- **デフォルト**: `30`
- **説明**: ストリーミングレスポンスのアイドルタイムアウト秒数
- **補足**: データ受信がこの秒数続くとタイムアウト。データが来続けている間はタイムアウトしない

### bashツール設定 (`bash`)

bashコマンドの安全性制限を設定します。

```yaml
bash:
  # 安全性レベル: strict, moderate, permissive
  safety_level: moderate

  # 追加の安全コマンド（確認なしで実行）
  safe_commands:
    - "npm run"
    - "npm test"
    - "cargo build"
    - "make"

  # パイプを許可（moderate以上でデフォルト有効）
  allow_pipe: true

  # リダイレクトを許可
  allow_redirect: false

  # sed -i等のインライン編集を許可
  allow_inline_edit: false
```

#### `safety_level`
- **型**: string
- **デフォルト**: `moderate`
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
- **デフォルト**: `false`
- **説明**: リダイレクト (`>`, `>>`, `<`) の使用を許可

#### `allow_inline_edit`
- **型**: boolean
- **デフォルト**: `false`
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

```yaml
plan_mode:
  max_parallel_steps: 3  # バッチ並列実行数（デフォルト: 3）
```

#### `max_parallel_steps`
- **型**: integer
- **デフォルト**: `3`
- **説明**: Plan Mode の調査・実装フェーズでバッチ処理として並列実行するツール/ステップの最大数

**バッチ処理・並列実行の動作:**
- **調査フェーズ**: AI が複数の SafetyHigh ツール（read_file 等）を返した場合、バッチでまとめて並列実行
- **実装フェーズ**: `depends_on` が同じステップ（依存関係のないステップ）をバッチでまとめて並列実行
- ループ検知: 同じツールセットが3回連続で返された場合はエラー終了

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
  enabled: false        # キャッシュを有効化
  max_entries: 100      # 最大エントリ数
  ttl_seconds: 300      # TTL（秒）
```

#### `enabled`
- **型**: boolean
- **デフォルト**: `false`
- **説明**: System PromptやRepo Mapのキャッシュを有効化

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
  t: test        # /t → /test
  r: review      # /r → /review
  c: commit      # /c → /commit
  u: use         # /u → /use
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
export XELYON_MODEL=deepseek-coder
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

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [プロバイダー設定](providers.md)
