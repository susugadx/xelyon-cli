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

# 会話履歴圧縮
compression:
  auto_compress: false      # 自動圧縮を有効化
  threshold_tokens: 40000   # 自動圧縮のトークン閾値
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

# 差分表示
diff:
  context_lines: 10         # 差分表示行数（0で省略なし）
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

#### `auto_compress`
- **型**: boolean
- **デフォルト**: `false`
- **説明**: トークン数が閾値を超えた際に自動圧縮を実行

#### `threshold_tokens`
- **型**: integer
- **デフォルト**: `40000`
- **説明**: 自動圧縮を実行するトークン閾値

#### `keep_recent`
- **型**: integer
- **デフォルト**: `10`
- **説明**: 圧縮時に保持する最新メッセージ数

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

### 差分表示設定 (`diff`)

#### `context_lines`
- **型**: integer
- **デフォルト**: `10`
- **説明**: ファイル編集時の差分表示行数
- **補足**: `0` で省略なし（全行表示）

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

### デバッグ・動作設定

```bash
# デバッグモード
export XELYON_DEBUG=1

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

### DeepSeek API URL（カスタムエンドポイント）

```bash
export DEEPSEEK_API_URL=https://your-custom-endpoint.com/chat/completions
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

### 1. 自動圧縮を有効化

```yaml
compression:
  auto_compress: true
  threshold_tokens: 30000
  keep_recent: 15
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
