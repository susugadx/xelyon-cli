# プロバイダー設定

XELYON CLIは複数のLLMプロバイダーに対応しています。

## 対応プロバイダー

| プロバイダー | 画像入力 | 環境変数 | 公式サイト |
|------------|---------|---------|-----------|
| DeepSeek | ❌ | `DEEPSEEK_API_KEY` | https://platform.deepseek.com |
| OpenAI | ✅ | `OPENAI_API_KEY` | https://platform.openai.com |
| Gemini | ✅ | `GEMINI_API_KEY` | https://ai.google.dev |
| Claude | ✅ | `ANTHROPIC_API_KEY` | https://console.anthropic.com |
| Groq | ❌ | `GROQ_API_KEY` | https://console.groq.com |
| Ollama | ❌ | - | https://ollama.com |
| OpenRouter | ✅ | `OPENROUTER_API_KEY` | https://openrouter.ai |
| Bedrock | ✅ | `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` | https://aws.amazon.com/bedrock |

## セットアップ

### 1. DeepSeek

```bash
# API キー取得: https://platform.deepseek.com
export DEEPSEEK_API_KEY=sk-...

# 使用例
xelyon --provider deepseek --model deepseek-chat
xelyon --provider deepseek --model deepseek-chat
xelyon --provider deepseek --model deepseek-reasoner
```

**特徴:**
- 高速・低コスト
- コード生成に特化したモデルあり
- 画像入力非対応
- **deepseek-reasoner**: `reasoning_content`（思考内容）をストリーミング表示（💭）、ツール実行時も思考を保持

### 2. OpenAI

```bash
# API キー取得: https://platform.openai.com/api-keys
export OPENAI_API_KEY=sk-...

# 使用例
xelyon --provider openai --model gpt-4
xelyon --provider openai --model gpt-4-turbo
xelyon --provider openai --model gpt-5.2
xelyon --provider openai --model gpt-5.2-codex
```

**特徴:**
- 高品質な回答
- GPT-4Vで画像入力対応
- 豊富なモデルラインナップ
- **注意: 高コスト**（GPT-5.2: 入力 $1.75/1M, 出力 $14/1M）

#### プロンプトキャッシュに関する注意

XELYON は Responses API の `prompt_cache_key`（ルーティングヒント）と `prompt_cache_retention: "24h"` を送信していますが、GPT-5 系モデルではキャッシュが**不安定**です。

- **GPT-5-nano / GPT-5-mini**: ほぼ機能せず（`cached_tokens=0` が頻発）
- **GPT-5 / GPT-5.1 / GPT-5.2**: 不安定（ヒット率が低い場合あり）
- **GPT-4o / o3-mini**: 正常動作

`prompt_cache_key` はキャッシュ制御ではなく、同じ GPU にルーティングするための**ヒント**です。キャッシュ自体は OpenAI 側で自動的にプレフィックスマッチング（1024 トークン以上）で行われます。

キャッシュヒット時は入力トークン 90% 割引ですが、現時点では GPT-5 系での効果は限定的です。コスト重視の場合は DeepSeek や Bedrock（Claude）の利用を推奨します。

> 参考: [OpenAI Community - Caching is borked for GPT-5 models](https://community.openai.com/t/caching-is-borked-for-gpt-5-models/1359574)

#### Responses API / Codex モデル

`gpt-5.2-codex` などの Codex モデルは自動的に Responses API を使用します。

**対応モデル:**
- `gpt-5.2-codex`
- `gpt-5.1-codex`
- `gpt-5.1-codex-max`
- `gpt-5-codex`

モデル名で自動判定されるため、追加設定は不要です。

**Responses API の特徴:**
- 会話コンテキストをサーバー側で管理
- Compact API による効率的な履歴圧縮
- ZDR（Zero Data Retention）対応

**使用例:**
```bash
xelyon --provider openai --model gpt-5.2-codex
```

### 3. Gemini

```bash
# API キー取得: https://aistudio.google.com/app/apikey
export GEMINI_API_KEY=...

# 使用例
xelyon --provider gemini --model gemini-3-pro-preview
xelyon --provider gemini --model gemini-3-flash-preview
xelyon --provider gemini --model gemini-2.5-flash
xelyon --provider gemini --model gemini-2.0-flash-exp
```

**特徴:**
- 長いコンテキスト対応（1M トークン）
- 画像入力対応
- 無料枠あり

#### Gemini 3 モデル（thinking 対応）

Gemini 3 Pro / Flash は **thinking（推論）が常時 ON** です。XELYON では自動的に `thinkingLevel` パラメータを送信します。

**デフォルト動作:**
- `thinking.enabled: false`（デフォルト）→ Flash: `"minimal"` / Pro: `"low"`（latency 最小化）
- `thinking.enabled: true` → config の `thinking.level` に応じて変換

**対応 thinkingLevel:**
| Level | Gemini 3 Pro | Gemini 3 Flash |
|-------|-------------|----------------|
| `minimal` | ❌ | ✅（デフォルト） |
| `low` | ✅（デフォルト） | ✅ |
| `medium` | ❌（low にフォールバック） | ✅ |
| `high` | ✅ | ✅ |

**Thought Signatures:** Gemini 3 の Function Calling レスポンスには `thoughtSignature`（暗号化された思考プロセス）が含まれます。XELYON はこれをパース・ログ出力しますが、テキストベースの履歴管理のため自動的に処理されます。

**注意:** Gemini 3 Pro の Function Calling には既知のバグ（空レスポンス）が報告されています。問題が発生する場合は `XELYON_DEBUG_GEMINI=1` で詳細ログを確認してください。

### 4. Claude

```bash
# API キー取得: https://console.anthropic.com
export ANTHROPIC_API_KEY=sk-ant-...

# 使用例
xelyon --provider claude --model claude-sonnet-4-5-20250514
xelyon --provider claude --model claude-opus-4
```

**特徴:**
- 長文理解に優れる
- 倫理的な回答
- 画像入力対応

### 5. Groq

```bash
# API キー取得: https://console.groq.com/keys
export GROQ_API_KEY=gsk_...

# 使用例
xelyon --provider groq --model meta-llama/llama-4-scout-17b-16e-instruct
```

**特徴:**
- 超高速推論
- Llama系モデル
- 画像入力非対応
- プロンプトキャッシュ対応（自動、50% OFF、一部モデルのみ）

### 6. Ollama

```bash
# インストール: https://ollama.com/download
# サーバー起動
ollama serve

# モデルダウンロード
ollama pull qwen2.5-coder:7b

# 使用例
xelyon --provider ollama --model qwen2.5-coder:7b
xelyon --provider ollama --model llama3.1:8b
```

**特徴:**
- ローカル実行（APIキー不要）
- プライバシー保護
- 無料
- 画像入力非対応

### 7. OpenRouter

```bash
# API キー取得: https://openrouter.ai
export OPENROUTER_API_KEY=sk-or-...

# 使用例
xelyon --provider openrouter --model anthropic/claude-opus-4.5
```

**特徴:**
- 複数プロバイダーのモデルを1つのAPIキーで利用可能
- OpenAI互換API
- 画像入力対応（モデルによる）

### 8. Bedrock (AWS)

```bash
# AWS 認証情報を設定（以下のいずれか）
# 方法1: 環境変数
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1

# 方法2: AWS CLI プロファイル（~/.aws/credentials）
aws configure

# 方法3: IAM ロール（EC2/ECS上で自動）

# 使用例
xelyon --provider bedrock --model global.anthropic.claude-opus-4-5-20251101-v1:0
xelyon --provider bedrock --model us.anthropic.claude-sonnet-4-20250514-v1:0
xelyon --provider bedrock --model us.anthropic.claude-haiku-4-5-20251001-v1:0
```

**特徴:**
- AWS フルマネージドサービス（中間マージンなし）
- Anthropic ネイティブのプロンプトキャッシュ対応
- IAM ロールによるセキュアな認証
- Extended Thinking 対応
- 画像入力対応

**利用可能なモデル ID:**
| モデル | Bedrock モデル ID |
|--------|------------------|
| Claude Opus 4.5 | `global.anthropic.claude-opus-4-5-20251101-v1:0` |
| Claude Sonnet 4 | `us.anthropic.claude-sonnet-4-20250514-v1:0` |
| Claude Haiku 4.5 | `us.anthropic.claude-haiku-4-5-20251001-v1:0` |

## モデル指定方法

### 1. コマンドラインフラグ（最優先）

```bash
xelyon --provider openai --model gpt-4
```

### 2. 環境変数

```bash
export XELYON_PROVIDER=deepseek
export XELYON_MODEL=deepseek-chat
xelyon
```

### 3. 設定ファイル（`~/.xelyon/config.yaml`）

```yaml
default_provider: deepseek
default_model: deepseek-chat

provider_models:
  deepseek:
    default_model: deepseek-chat
  openai:
    default_model: gpt-4
  gemini:
    default_model: gemini-2.5-flash
  claude:
    default_model: claude-sonnet-4-5-20250514
  ollama:
    default_model: qwen2.5-coder:7b
  groq:
    default_model: meta-llama/llama-4-scout-17b-16e-instruct
  openrouter:
    default_model: anthropic/claude-opus-4.5
  bedrock:
    default_model: global.anthropic.claude-opus-4-5-20251101-v1:0
```

### 4. セッション中の切り替え（`/use`コマンド）

```bash
xelyon
> /use openai gpt-4
> 質問1
> /use deepseek
> 質問2
```

## プロンプトキャッシュ対応状況

| プロバイダー | 方式 | 状態 | 割引率 | 備考 |
|------------|------|------|-------|------|
| **Claude** | 明示的（`cache_control`） | 安定 | 読み取り 90% OFF | `prompt_cache.enabled: true` で有効 |
| **Bedrock** | 明示的（`cache_control`） | 安定 | 読み取り 90% OFF | Claude と同じ仕組み |
| **OpenAI** | 自動（プレフィックス） | **不安定**（GPT-5系） | 読み取り 90% OFF | `prompt_cache_key` はルーティングヒントのみ |
| **DeepSeek** | 自動 | 安定 | 読み取り割引あり | 設定不要 |
| **Gemini** | 自動（暗黙的） | 安定 | - | Gemini 2.5 系で対応 |
| **OpenRouter** | プロバイダー依存 | - | - | Anthropic モデル: 手動 `cache_control` 必要 |
| **Groq** | 自動（プレフィックス） | 安定 | 読み取り 50% OFF | 一部モデルのみ（GPT-OSS, Kimi K2） |
| **Ollama** | - | - | - | ローカル実行のため不要 |

### コスト効率の良い選択肢

長い会話でのコスト効率を重視する場合:

1. **DeepSeek** - 元々低コスト + キャッシュ安定
2. **Bedrock（Claude）** - プロンプトキャッシュが確実に効く + AWS 直接契約で中間マージンなし
3. **Claude（直接）** - プロンプトキャッシュが確実に効く
4. **OpenAI** - 高コスト + キャッシュ不安定のため、コスト重視なら非推奨

## プロバイダー選択のヒント

### コード生成・編集
- **DeepSeek Coder**: 高速・低コスト・高品質
- **Qwen2.5-Coder (Ollama)**: ローカル実行

### 複雑な問題解決
- **Claude Opus 4**: 長文理解・推論
- **GPT-4**: バランスの良い性能

### 高速レスポンス
- **Groq**: 超高速推論
- **Gemini Flash**: バランス良く高速

### 画像解析
- **GPT-4V**: 高品質な画像理解
- **Gemini**: マルチモーダル対応
- **Claude**: 画像+長文の組み合わせ

### AWS インフラとの統合
- **Bedrock**: IAMロール認証、プロンプトキャッシュ、中間マージンなし

### プライバシー重視
- **Ollama**: 完全ローカル実行

## トラブルシューティング

### API キーエラー

```bash
# 環境変数が設定されているか確認
echo $DEEPSEEK_API_KEY

# .zshrc / .bashrc に追加
export DEEPSEEK_API_KEY=sk-...
source ~/.zshrc
```

### Ollama接続エラー

```bash
# サーバーが起動しているか確認
ollama list

# サーバー起動
ollama serve
```

### モデルが見つからない

```bash
# 利用可能なモデル一覧を確認
xelyon
> /providers

# 正しいモデル名を指定
xelyon --provider openai --model gpt-4
```

### レート制限エラー

APIプロバイダーのダッシュボードで使用状況とレート制限を確認してください。

- DeepSeek: https://platform.deepseek.com/usage
- OpenAI: https://platform.openai.com/usage
- Gemini: https://aistudio.google.com
- Claude: https://console.anthropic.com
- Groq: https://console.groq.com

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [設定リファレンス](config.md)
- [LSP連携](lsp.md)
- [MCP連携](mcp.md)
