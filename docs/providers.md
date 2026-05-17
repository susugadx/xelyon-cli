# プロバイダー設定

XELYON CLIは複数のLLMプロバイダーに対応しています。

## 編集ツールの自動切り替え

XELYON は provider/model に応じて編集ツールを自動で切り替えます。

- OpenAI / Azure OpenAI / Gemini / Kimi 系: `apply_patch`
- Claude / Anthropic / DeepSeek 系: `str_replace` / `write_file` / `delete_file`
- OpenRouter: `anthropic/...` / `deepseek/...` は legacy、`openai/...` / `google/...` / `gemini/...` / `moonshotai/...` は `apply_patch`
- Bedrock: Claude family は legacy 編集ツール。runtime supported な非 Claude Converse モデルは `apply_patch`

`XELYON_EDIT_TOOL` を指定した場合は、この自動判定より環境変数の指定が優先されます。

## 対応プロバイダー

| プロバイダー | 画像入力 | 環境変数 | 公式サイト |
|------------|---------|---------|-----------|
| DeepSeek | ❌ | `DEEPSEEK_API_KEY` | https://platform.deepseek.com |
| Kimi | ✅ | `MOONSHOT_API_KEY` | https://platform.moonshot.ai |
| OpenAI | ✅ | `OPENAI_API_KEY` | https://platform.openai.com |
| Azure OpenAI | ✅ | `AZURE_OPENAI_BASE_URL` + (`AZURE_OPENAI_API_KEY` / `AZURE_OPENAI_AUTH_TOKEN` / `AZURE_OPENAI_AUTH_TOKEN_COMMAND` のいずれか) | https://azure.microsoft.com/products/ai-services/openai-service |
| Gemini | ✅ | `GEMINI_API_KEY` | https://ai.google.dev |
| Claude | ✅ | `ANTHROPIC_API_KEY` | https://console.anthropic.com |
| Groq | ❌ | `GROQ_API_KEY` | https://console.groq.com |
| Ollama | ❌ | `OLLAMA_BASE_URL`（任意） | https://ollama.com |
| OpenRouter | ✅ | `OPENROUTER_API_KEY` | https://openrouter.ai |
| Bedrock | ✅ | `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` | https://aws.amazon.com/bedrock |

## セットアップ

### 1. DeepSeek

```bash
# API キー取得: https://platform.deepseek.com
export DEEPSEEK_API_KEY=sk-...

# 使用例
xelyon --provider deepseek --model deepseek-v4-flash
xelyon --provider deepseek --model deepseek-v4-pro
```

**特徴:**
- **deepseek-v4-flash**: 低コスト・高速・普段使い向き
- **deepseek-v4-pro**: 高精度・重い設計/レビュー向き
- 1M context / 最大 384K output
- streaming / tool calls / JSON output / thinking modes 対応
- 画像入力非対応
- `/thinking off`: `thinking: {"type":"disabled"}` を明示送信
- `/thinking on`: `thinking: {"type":"enabled"}` と `reasoning_effort` を送信（`/thinking xhigh` は DeepSeek では `max`）
- `deepseek-chat` / `deepseek-reasoner` は legacy alias（`deepseek-v4-flash` 相当）です。2026-07-24 廃止予定のため、新規設定では `deepseek-v4-flash` / `deepseek-v4-pro` を使用してください。
- `reasoning_content`（思考内容）はストリーミング表示（💭）され、ツール実行時も保持されます。

設定の到達性は CLI から診断できます。`doctor deepseek` は `DEEPSEEK_API_KEY`、`DEEPSEEK_API_URL`、provider 登録、model / `catalog_model` 解決、Chat Completions route、thinking request config、function calling 設定、token / pricing metadata を確認します。`DEEPSEEK_API_URL` は Chat Completions まで含む完全な endpoint override で、公式 DeepSeek 互換 endpoint は `/chat/completions` で終わります。OpenAI 互換 proxy の `/v1/chat/completions` も指定できますが、doctor では意図的な proxy path として warn になります。`--smoke` を付けると live text request を送って usage / cost を観測し、function calling まで確認したい場合は `--tool-smoke` を使います。`--print-request` は live request を送らずに redacted request body を表示します。

```bash
xelyon doctor deepseek
xelyon doctor deepseek --model deepseek-v4-flash
xelyon doctor deepseek --model corp-deepseek-model --catalog-model deepseek-v4-flash
xelyon doctor deepseek --smoke
xelyon doctor deepseek --tool-smoke
xelyon doctor deepseek --print-request
xelyon doctor deepseek --json
```

`DEEPSEEK_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、text smoke fallback を実行します。`--smoke` / `--tool-smoke` は live API request を送るため、通常 CI では実行しません。手元では `DEEPSEEK_API_KEY` を設定して `make deepseek-doctor-smoke` を実行します。既定モデルは `DEEPSEEK_DOCTOR_SMOKE_MODEL ?= deepseek-v4-flash` です。

### 2. Kimi (Moonshot)

```bash
# API キー取得: https://platform.moonshot.ai
export MOONSHOT_API_KEY=sk-...

# 使用例
xelyon --provider kimi --model kimi-k2.6
xelyon --provider moonshot --model kimi-k2.5
```

**特徴:**
- Moonshot Chat Completions API を native provider として使用
- streaming / tool calls / JSON output / thinking modes 対応
- 256K context / 最大 32K output（K2.6 / K2.5）
- 画像入力対応。`--image` / `image:` から渡された PNG / JPEG / WebP / GIF を `data:image/...;base64,...` の multimodal `image_url` part として送信します。
- `web_search.provider = kimi` / `moonshot`、またはメイン provider が Kimi の場合、Moonshot Chat Completions の built-in `$web_search` を使います。
- `/think off`: K2.6 / K2.5 は `thinking: {"type":"disabled"}` を送信します。`kimi-k2-thinking` には disabled を送信しません。
- `/think on`: K2.6 は `thinking: {"type":"enabled","keep":"all"}`、K2.5 は `thinking: {"type":"enabled"}` を送信し、forced tool choice は `auto` に丸めます。
- `reasoning_content`（思考内容）はストリーミング表示され、ツール実行時も保持されます。
- `KIMI_API_URL` は `/v1/chat/completions` まで含む完全な endpoint override として扱います。別 path の proxy endpoint も指定できますが、doctor では意図的な proxy path として warn になります。

`kimi-k2-thinking` は明示指定された場合のみ 256K context / 最大 32K output の legacy/compat thinking model として扱います。新規利用では thinking on/off が可能な `kimi-k2.6` を推奨します。

Kimi built-in `$web_search` は通常 function tools とは別の `web_search` 専用 route で使います。request には `tools[].type = "builtin_function"` / `function.name = "$web_search"` を入れ、検索 turn では `thinking: {"type":"disabled"}` を送信します。`tool_choice` は送らず、モデルの自動選択に任せます。Moonshot は `$web_search` call fee と Chat Completions token 使用量を別々に課金し、検索結果トークンも次 request の入力 tokens に含まれます。XELYON は API が返す token usage / `cached_tokens` を token cost の source of truth とし、[Kimi API Platform WebSearch Pricing](https://platform.moonshot.ai/docs/pricing/tools.en-US) の `$0.005 / invocation` を外部固定費として別枠で観測します。`finish_reason = "stop"` で `$web_search` tool call が返らない response は call fee なしとして扱います。検索結果 tokens は `tool_call.function.arguments` から best-effort で表示用に読むだけで、`InputTokens` へ二重加算しません。

Memory / code runner、video 入力、file upload / `ms://` 参照は現在の native provider では未対応です。URL 画像は Kimi 公式仕様でも未対応のため、XELYON はローカル画像ファイルを base64 data URL として送ります。

設定の到達性は CLI から診断できます。`doctor kimi` は `MOONSHOT_API_KEY`、`KIMI_API_URL`、provider 登録、model / `catalog_model` 解決、Chat Completions route、token / pricing metadata、画像入力対応、未対応機能、`prompt_cache_key` request shape を確認します。`KIMI_API_URL` が `/v1/chat/completions` 以外の path を指す場合は proxy endpoint として warn し、`--print-request` / live smoke はその URL を実 request 先として使います。`--catalog-model` は alias の underlying Kimi model として token / pricing 判定に使います。`--print-request` は live request を送らず、redacted bearer header と request body を `request_preview` に表示します。`--smoke` を付けると live Chat Completions request を送って、streaming、thinking on/off、同一 session の prompt cache key、usage callback を確認します。画像入力の実 API 受理を確認したい場合は `--image-smoke`、function calling は `--tool-smoke`、built-in `$web_search` は `--web-search-smoke` を使います。

```bash
xelyon doctor kimi
xelyon doctor kimi --model kimi-k2.6
xelyon doctor kimi --model corp-kimi-model --catalog-model kimi-k2.6
xelyon doctor kimi --print-request
xelyon doctor kimi --tool-smoke --print-request
xelyon doctor kimi --smoke
xelyon doctor kimi --image-smoke
xelyon doctor kimi --tool-smoke
xelyon doctor kimi --web-search-smoke
xelyon doctor kimi --json
```

`--print-request` は `MOONSHOT_API_KEY` なしで実行できます。live smoke は `MOONSHOT_API_KEY` が必要で、通常 CI では実行しません。手元では以下で実行できます。prompt cache の `cached_tokens` は API が返す場合だけ観測されるため、0 でも smoke は成功扱いです。`kimi-doctor-smoke` は doctor 経路で text / tool / image / built-in web search をまとめて確認します。`kimi-web-search-smoke` は実検索 call fee が発生し、`cached_input_tokens`、`web_search_call_count`、`web_search_call_fee_estimate`、`web_search_usage_observed`、`search_result_total_tokens`（観測できた場合）を表示します。`web_search_usage_observed` は `$web_search` call fee / call count の観測を表し、endpoint の token usage 観測とは別です。`--web-search-smoke` は `$web_search` tool call が 1 件以上観測された場合だけ成功扱いにし、通常の `stop` response で tool call がない場合は fail します。

```bash
export MOONSHOT_API_KEY=sk-...
make kimi-doctor-smoke
make kimi-smoke
make kimi-image-smoke
make kimi-tool-smoke
make kimi-web-search-smoke
```

### 3. OpenAI

```bash
# API キー取得: https://platform.openai.com/api-keys
export OPENAI_API_KEY=sk-...

# 使用例
xelyon --provider openai --model gpt-5.4
xelyon --provider openai --model gpt-5.5
xelyon --provider openai --model gpt-5.5-pro
xelyon --provider openai --model gpt-5.4-mini
xelyon --provider openai --model gpt-5.3-codex
xelyon --provider openai --model gpt-5.2
xelyon --provider openai --model gpt-5.2-codex
```

**特徴:**
- 高品質な回答
- 画像入力対応
- 豊富なモデルラインナップ
- **注意: 高コスト**（GPT-5.5: 入力 $5/1M, 出力 $30/1M、GPT-5.5 Pro: 入力 $30/1M, 出力 $180/1M）

#### プロンプトキャッシュに関する注意

XELYON は Responses API の `prompt_cache_key`（ルーティングヒント）と `prompt_cache_retention: "24h"` を送信していますが、GPT-5 系モデルではキャッシュが**不安定**です。

- **GPT-5-nano / GPT-5-mini**: ほぼ機能せず（`cached_tokens=0` が頻発）
- **GPT-5 / GPT-5.1 / GPT-5.2 / GPT-5.4 / GPT-5.5**: 不安定（ヒット率が低い場合あり）
- **GPT-5.5 Pro**: cached input discount なし（`cached_tokens` が返っても割引単価では計算しません）
- **GPT-4o / o3-mini**: 正常動作

`prompt_cache_key` はキャッシュ制御ではなく、同じ GPU にルーティングするための**ヒント**です。キャッシュ自体は OpenAI 側で自動的にプレフィックスマッチング（1024 トークン以上）で行われます。

キャッシュヒット時は多くの OpenAI モデルで入力トークン割引がありますが、現時点では GPT-5 系での効果は限定的です。GPT-5.5 Pro は公式に cached input discount がないため、XELYON でも通常入力単価で計算します。コスト重視の場合は DeepSeek や Bedrock（Claude）の利用を推奨します。

> 参考: [OpenAI Community - Caching is borked for GPT-5 models](https://community.openai.com/t/caching-is-borked-for-gpt-5-models/1359574)

#### Responses API / Codex モデル

`gpt-5.3-codex` / `gpt-5.2-codex` などの Codex モデルは自動的に Responses API を使用します。

GPT-5.5 系も OpenAI provider では Responses API を使用します。

**対応モデル:**
- `gpt-5.5`（Responses API + streaming）
- `gpt-5.5-pro`（Responses API、streaming unsupported のため non-streaming 経路）
- `gpt-5.3-codex`
- `gpt-5.2-codex`
- `gpt-5.1-codex`
- `gpt-5.1-codex-max`
- `gpt-5-codex`

モデル名で自動判定されるため、追加設定は不要です。

**Responses API の特徴:**
- 会話コンテキストをサーバー側で管理
- `previous_response_id` chain request で `context_management.compaction` による server-side compaction を利用可能
- Compact API（`/responses/compact` / `/compress --compact`）は別機能
- ZDR（Zero Data Retention）対応

XELYON は既定で Responses API の `store: true` と `previous_response_id` 継続を使います。これは通常の推奨設定です。provider 側に response state を保存したくない運用だけ、`~/.xelyon/config.yaml` で `responses.store: false` を設定してください。詳しくは [Responses API retention 設定](config.md#responses-api-retention-設定-responses高度な設定) を参照してください。

`responses.server_compaction` は Compact API の完全上位互換ではありません。`previous_response_id` を使う OpenAI/Azure Responses request 向けに compaction 発火閾値を設定する機能で、載せられない場合は `local_fallback` 設定に従って local auto-compress へ戻ります。

**GPT-5.5 Pro の注意:**
- streaming は公式に unsupported のため、XELYON は non-streaming Responses 経路を使用します。
- 応答に数分かかる場合があります。background mode は今回未対応です。
- function calling / structured outputs は Responses API 経路で利用します。

**使用例:**
```bash
xelyon --provider openai --model gpt-5.3-codex
xelyon --provider openai --model gpt-5.2-codex
xelyon --provider openai --model gpt-5.5
xelyon --provider openai --model gpt-5.5-pro
```

設定の到達性は CLI から診断できます。`doctor openai` は `OPENAI_API_KEY`、`OPENAI_API_URL`、`OPENAI_RESPONSES_URL`、provider 登録、model / `catalog_model` 解決、Responses / Chat Completions route と判定理由、function calling 設定、token / pricing metadata、Responses retention 設定を確認します。`--capabilities` を付けると live request を送らず、Responses API / streaming / function calling / image input / `previous_response_id` / server compaction / context window / max output / pricing の解決結果を `capabilities` に表示します。`--require-capability` を付けると、解決済みのローカル capability が要求を満たすかを live request なしの `required_capability` check として検証します。対応する名前は `responses_api`、`responses_streaming`、`chat_completions`、`function_calling`、`image_input`、`previous_response_id`、`session_persistence`、`server_compaction` です。OpenAI の `responses_streaming` gate は実 `catalog_model` が既知 catalog model として解決できない場合 `unknown` として fail します。`--smoke` を付けると live text request を送って、Responses route では response ID、usage、概算 cost を表示します。Chat Completions route では response ID は返らないため、usage と概算 cost を確認します。function calling まで確認したい場合は `--tool-smoke` を使い、dummy tool call を強制します。Responses API の `previous_response_id` chain まで確認したい場合は `--retention-smoke` を使い、`responses.store=true` の initial / followup request を連続実行します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。複数 smoke を指定した場合は request を別々に実行または preview し、JSON では `smoke.requests[]` / `request_preview.requests[]` に request 単位の結果を出します。

```bash
xelyon doctor openai
xelyon doctor openai --model gpt-5.4
xelyon doctor openai --model corp-openai-deployment --catalog-model gpt-5.4
xelyon doctor openai --smoke
xelyon doctor openai --tool-smoke
xelyon doctor openai --retention-smoke
xelyon doctor openai --capabilities
xelyon doctor openai --model corp-openai-deployment --catalog-model gpt-5.4 --require-capability responses_api --require-capability previous_response_id
xelyon doctor openai --print-request
xelyon doctor openai --tool-smoke --print-request
xelyon doctor openai --retention-smoke --print-request
xelyon doctor openai --smoke --tool-smoke
xelyon doctor openai --smoke --tool-smoke --retention-smoke
xelyon doctor openai --json
```

`OPENAI_FUNCTION_CALLING=0` の場合、`--tool-smoke` は function calling 無効として warn skip し、tool payload / `tool_choice` は送信しません。`--retention-smoke` は Responses API route 専用で、Chat Completions route では live request を送らず fail します。`OPENAI_API_URL` は Chat Completions まで含む完全な endpoint override で、公式 path は `/v1/chat/completions` です。`OPENAI_RESPONSES_URL` は Responses まで含む完全な endpoint override で、公式 path は `/v1/responses` です。別 path の proxy endpoint も指定できますが、`doctor openai` では意図的な proxy path として warn になり、request preview / live smoke は設定 URL をそのまま使います。`--smoke` / `--tool-smoke` / `--retention-smoke` は live API request を送るため、通常 CI では実行しません。手元では `OPENAI_API_KEY` を設定して `make openai-doctor-smoke` を実行します。既定モデルは `OPENAI_DOCTOR_SMOKE_MODEL ?= gpt-5.4` です。

### 4. Azure OpenAI

会社環境向けの最短セットアップと問い合わせ前チェックは [Azure OpenAI 利用メモ](azure-openai.md) も参照してください。

```bash
# Azure OpenAI resource の v1 base URL と認証情報を設定
export AZURE_OPENAI_BASE_URL=https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1
export AZURE_OPENAI_API_KEY=...

# Microsoft Entra ID を使う場合は API key の代わりに bearer token を設定
unset AZURE_OPENAI_API_KEY
export AZURE_OPENAI_AUTH_TOKEN=$(az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv)

# 会社環境などで token refresh したい場合は command を設定
unset AZURE_OPENAI_AUTH_TOKEN
export AZURE_OPENAI_AUTH_TOKEN_COMMAND='az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv'

# model には Azure 側の deployment 名を指定
xelyon --provider azure --model my-gpt-5-deployment
```

**特徴:**
- Responses API (`/openai/v1/responses`) を使用
- API key 認証は `api-key` ヘッダー
- Microsoft Entra ID 認証は `Authorization: Bearer` ヘッダー
- `AZURE_OPENAI_AUTH_TOKEN_COMMAND` を設定すると、token 未設定時または 401 受信時に bearer token を再取得します
- 画像入力 / function calling 対応
- `model` は Azure の deployment 名
- OpenAI provider 用の `prompt_cache_key` / `prompt_cache_retention` は送信しません
- `responses.store` / `responses.persist_response_id` は OpenAI provider と同じ設定を使用します

`model` / `provider_models.azure.default_model` には Azure 側の **deployment 名**を入れます。deployment 名が実モデル名と異なる場合は、token limit / pricing / capability 判定用に `catalog_model` を設定してください。`catalog_model` は `gpt-5.4`、`gpt-5.5-pro`、`gpt-5.3-codex` のような実モデル名で、deployment 名ではありません。

`AZURE_OPENAI_BASE_URL` は Azure OpenAI resource の v1 base URL です。resource root と `/openai` は `/openai/v1` に正規化され、runtime / doctor smoke / `--print-request` は `<normalized_base_url>/responses` を実 request endpoint として使います。`/openai/deployments/<deployment>` まで含めた旧 Azure endpoint や public OpenAI host は fail になります。`api-version` query は Azure OpenAI v1 Responses path では使わないため warn で無視します。会社 proxy などで非標準 path を指定した場合は intentional proxy として warn になり、その path に `/responses` を付けた URL が request preview / live smoke に使われます。

API key 認証の最小設定:

```bash
xelyon doctor azure --deployment my-codex-deployment --catalog-model gpt-5.3-codex --print-config
```

```yaml
default_provider: azure

provider_models:
  azure:
    default_model: my-codex-deployment
    catalog_model: gpt-5.3-codex
```

Microsoft Entra ID 認証でも YAML は同じです。環境変数だけ API key ではなく bearer token にします。長時間実行や CI では、固定 token より `AZURE_OPENAI_AUTH_TOKEN_COMMAND` を推奨します。取得した token は process memory にだけ保持し、config/session には保存しません。

```bash
unset AZURE_OPENAI_API_KEY
export AZURE_OPENAI_AUTH_TOKEN=$(az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv)
```

```bash
unset AZURE_OPENAI_API_KEY
unset AZURE_OPENAI_AUTH_TOKEN
export AZURE_OPENAI_AUTH_TOKEN_COMMAND='az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv'
export AZURE_OPENAI_AUTH_TOKEN_COMMAND_TIMEOUT=10s
```

`AZURE_OPENAI_AUTH_TOKEN_COMMAND` はローカル shell で実行され、stdout の最初の空でない行を bearer token として扱います。信頼できる command だけを設定してください。command は token 未設定時と 401 応答後の 1 回だけの retry 時に実行されます。

複数 deployment を使う場合は、deployment ごとに `model_overrides` で catalog model を固定できます。

```yaml
provider_models:
  azure:
    default_model: my-gpt-5-deployment
    catalog_model: gpt-5.4
    model_overrides:
      my-gpt-5-pro-deployment:
        catalog_model: gpt-5.5-pro
```

`responses.store` / `responses.persist_response_id` は通常変更しないでください。既定値は Responses API の `previous_response_id` 継続と session reload を安定させるための推奨設定です。provider 側に response state を残せない運用だけ、[Responses API retention 設定](config.md#responses-api-retention-設定-responses高度な設定) を確認してから変更してください。

よくある設定ミス:

- `AZURE_OPENAI_BASE_URL` に `https://api.openai.com/v1` を入れる。Azure provider では Azure OpenAI resource の URL が必要です。
- `AZURE_OPENAI_BASE_URL` に `/openai/deployments/<deployment>` まで入れる。XELYON は `/openai/v1/responses` を使うため、base URL は `/openai/v1` で止めます。
- `AZURE_OPENAI_BASE_URL` に `api-version` query を付ける。Azure OpenAI v1 Responses path では query を使わず、XELYON は query を無視します。
- `AZURE_OPENAI_API_KEY` に OpenAI の `sk-...` key を入れる。Azure OpenAI resource key か Microsoft Entra ID bearer token を使ってください。
- `default_model` と `catalog_model` を逆にする。`default_model` は deployment 名、`catalog_model` は実モデル名です。

設定の到達性は CLI から診断できます。`doctor azure` は base URL、認証方式、deployment 解決、`catalog_model` とそれに紐づく token / pricing / capability 判定、Responses route と判定理由、function calling 設定、Responses retention 設定を確認します。`--capabilities` を付けると live request を送らず、Responses API / streaming / function calling / image input / `previous_response_id` / server compaction / context window / max output / pricing の解決結果を `capabilities` に表示します。`--require-capability` を付けると、解決済みのローカル capability が要求を満たすかを live request なしの `required_capability` check として検証します。対応する名前は `responses_api`、`responses_streaming`、`chat_completions`、`function_calling`、`image_input`、`previous_response_id`、`session_persistence`、`server_compaction` です。Azure の `responses_streaming` gate は実 `catalog_model` が未解決の場合 `unknown` として fail します。`--smoke` を付けると `responses.store=false` の最小リクエストを送って、実 deployment への到達性、Responses API の response ID、usage、概算 cost を検証します。function calling まで確認したい場合は `--tool-smoke` を使い、dummy tool call を強制します。Responses API の `previous_response_id` chain まで確認したい場合は `--retention-smoke` を使い、`responses.store=true` の initial / followup request を連続実行します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。

```bash
xelyon doctor azure
xelyon doctor azure --deployment my-codex-deployment --catalog-model gpt-5.3-codex --print-config
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --smoke
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --tool-smoke
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --retention-smoke
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --capabilities
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --require-capability responses_api
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --print-request
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --retention-smoke --print-request
xelyon doctor azure --json
```

smoke の text / JSON report には `response_id`、`usage_observed`、`usage.input_tokens` / `usage.output_tokens` / `usage.thinking_tokens` / `usage.cached_input_tokens` / `usage.cache_creation_tokens`、`cost.usd`、`cost.pricing_unavailable` が含まれます。`--retention-smoke` では request 単位の `retention_payload` と `previous_response_id` も出します。`--print-request` では `request_preview.requests[]` に request 単位の `body`、redacted `headers`、endpoint、`previous_response_id` placeholder を出します。response ID や usage が返らない場合、または pricing catalog に該当モデルがない場合は warn になりますが、到達性の smoke 成功自体は維持します。cost は `internal/cost/pricing.yaml` と `catalog_model` を source of truth にした概算です。

Azure OpenAI の API error は、HTTP status に応じて原因候補を補足します。401/403 は認証・権限、404 は base URL または deployment 名、429 は quota / rate limit / capacity、tool payload rejected は `AZURE_OPENAI_FUNCTION_CALLING=0` の案内を表示します。

より深い実 Azure 環境の smoke test は以下で実行できます。`AZURE_OPENAI_PRO_DEPLOYMENT` を指定した場合は GPT-5.5 Pro 系の non-streaming 経路も検証します。

```bash
export AZURE_OPENAI_BASE_URL=https://YOUR-RESOURCE-NAME.openai.azure.com/openai/v1
export AZURE_OPENAI_DEPLOYMENT=my-gpt-5-deployment
export AZURE_OPENAI_CATALOG_MODEL=gpt-5.4
export AZURE_OPENAI_API_KEY=...
# または AZURE_OPENAI_AUTH_TOKEN=...
# または AZURE_OPENAI_AUTH_TOKEN_COMMAND='az account get-access-token --resource https://cognitiveservices.azure.com --query accessToken -o tsv'

make azure-smoke
```

GitHub Actions では `Azure Smoke` workflow を手動実行できます。通常 CI では走らず、repository secrets に `AZURE_OPENAI_BASE_URL`、`AZURE_OPENAI_DEPLOYMENT`、`AZURE_OPENAI_API_KEY` / `AZURE_OPENAI_AUTH_TOKEN` / `AZURE_OPENAI_AUTH_TOKEN_COMMAND` のいずれかがある場合だけ live smoke を実行します。workflow input の `tool_smoke` を有効にすると doctor smoke で dummy tool call も強制します。

### 5. Gemini

```bash
# API キー取得: https://aistudio.google.com/app/apikey
export GEMINI_API_KEY=...

# 使用例
xelyon --provider gemini --model gemini-3-pro-preview
xelyon --provider gemini --model gemini-3.1-pro-preview-customtools
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

設定の到達性は CLI から診断できます。`doctor gemini` は `GEMINI_API_KEY`、`GEMINI_API_URL`、provider 登録、model / `catalog_model` 解決、`streamGenerateContent?alt=sse` route、function calling 設定、画像入力、thinking、context caching、native web search、token / pricing metadata を確認します。`--smoke` を付けると live text SSE request を送って usage / cost を観測し、function calling まで確認したい場合は `--tool-smoke` を使います。画像入力の実 API 受理は `--image-smoke`、native web search は `--web-search-smoke` で確認します。`--print-request` は live request を送らず、redacted `x-goog-api-key` header と request body を表示します。live smoke の失敗は `smoke` check の message / suggestion と request 単位の `smoke.requests[].error` に分類して表示します。

```bash
xelyon doctor gemini
xelyon doctor gemini --model gemini-3.1-pro-preview-customtools
xelyon doctor gemini --model corp-gemini-model --catalog-model gemini-3.1-pro-preview-customtools
xelyon doctor gemini --smoke
xelyon doctor gemini --tool-smoke
xelyon doctor gemini --image-smoke
xelyon doctor gemini --web-search-smoke
xelyon doctor gemini --print-request
xelyon doctor gemini --json
```

tool smoke / preview では request-scoped `ANY` mode を使って diagnostic tool だけを送りますが、通常 runtime の `GEMINI_FC_MODE` fallback は変更しません。`GEMINI_API_URL` は exact endpoint / proxy override として扱われ、doctor の `request_preview.requests[].url` が実際の送信先です。text / tool / image は `streamGenerateContent?alt=sse`、native web search は `generateContent` の request shape を同じ URL に送るため、`--web-search-smoke` では `generateContent` を受ける endpoint または両方の shape を受ける proxy を使います。non-Gemini `catalog_model` は warn になり、OpenAI / OpenRouter など別 owner の metadata は Gemini doctor の token / cost policy に使いません。`--smoke` / `--tool-smoke` / `--image-smoke` / `--web-search-smoke` は live API request を送るため、通常 CI では実行しません。手元では `GEMINI_API_KEY` を設定して `make gemini-doctor-smoke` を実行します。既定モデルは `GEMINI_DOCTOR_SMOKE_MODEL ?= gemini-3.1-pro-preview-customtools`、timeout は `GEMINI_DOCTOR_SMOKE_TIMEOUT ?= 180s` です。web search smoke は native `generateContent` の `usageMetadata` が返れば usage / cost を表示し、返らない場合は usage / cost を warn に留めます。summary または source が返れば smoke は成功扱いです。

Gemini doctor は live smoke 失敗時に、認証・権限、quota / rate limit / capacity、model unavailable、empty SSE response、endpoint route mismatch、tool unsupported、image unsupported、native web search unsupported を分類して suggestion を出します。pricing metadata unavailable は live request 成功後の観測不足なので fail ではなく `cost` warn です。`GEMINI_API_URL` を使う場合は、route mismatch が疑わしいときに `--print-request` の URL と body を source of truth として確認してください。

### 6. Claude

```bash
# API キー取得: https://console.anthropic.com
export ANTHROPIC_API_KEY=sk-ant-...

# 使用例
xelyon --provider claude --model claude-sonnet-4-6
xelyon --provider claude --model claude-opus-4-7
xelyon --provider claude --model claude-opus-4-6
```

**特徴:**
- 長文理解に優れる
- 倫理的な回答
- 画像入力対応

**Extended Thinking:**
- Opus 4.7 / Opus 4.6 / Sonnet 4.6: `type: "adaptive"` + `output_config.effort`
- それ以前のモデル: `type: "enabled"` + `budget_tokens`
- `xhigh` レベルは Opus 4.7 で `xhigh`、Opus 4.6 で `max`、Sonnet 4.6 では `high` にフォールバック
- Claude Compaction は Opus 4.7 / Opus 4.6 / Opus 4.5 / Sonnet 4.6 で有効化対象

設定の到達性は CLI から診断できます。`doctor claude` は `ANTHROPIC_API_KEY`、`ANTHROPIC_API_URL`、provider 登録、model / `catalog_model` 解決、Anthropic Messages route、function calling、画像入力、thinking request config、context management、Claude compaction、native web search、token / pricing metadata を確認します。`--catalog-model` は alias の underlying Claude model として token / pricing / thinking / context management 判定に使います。`--print-request` は live request を送らず、redacted `x-api-key` header、`anthropic-version` / `anthropic-beta`、request body を `request_preview` に表示します。`--smoke` を付けると live Messages request を送って usage / cost を観測します。画像入力は `--image-smoke`、function calling は `--tool-smoke`、thinking request は `--thinking-smoke`、native web search は `--web-search-smoke` を使います。

```bash
xelyon doctor claude
xelyon doctor claude --model claude-sonnet-4-6
xelyon doctor claude --model corp-claude-model --catalog-model claude-sonnet-4-6
xelyon doctor claude --print-request
xelyon doctor claude --tool-smoke --print-request
xelyon doctor claude --smoke
xelyon doctor claude --tool-smoke
xelyon doctor claude --image-smoke
xelyon doctor claude --thinking-smoke
xelyon doctor claude --web-search-smoke
xelyon doctor claude --json
```

`ANTHROPIC_API_URL` は runtime と同じ exact endpoint / proxy override で、公式 Anthropic endpoint では `/v1/messages` を期待します。別 path の proxy endpoint も指定できますが、doctor では意図的な proxy path として warn になり、`--print-request` / live smoke はその URL を実 request 先として使います。non-Claude `catalog_model` は warn になり、OpenAI / OpenRouter など別 owner の metadata は Claude doctor の token / cost policy に使いません。`--smoke` / `--tool-smoke` / `--image-smoke` / `--thinking-smoke` / `--web-search-smoke` は live API request を送るため、通常 CI では実行しません。手元では `ANTHROPIC_API_KEY` を設定して `make claude-doctor-smoke` を実行します。既定モデルは `CLAUDE_DOCTOR_SMOKE_MODEL ?= claude-sonnet-4-6`、timeout は `CLAUDE_DOCTOR_SMOKE_TIMEOUT ?= 180s` です。Claude native web search smoke は summary または source が返れば成功扱いで、現時点では token usage / cost 観測は必須にしません。

### 7. Groq

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

設定の到達性は CLI から診断できます。`doctor groq` は `GROQ_API_KEY`、`GROQ_API_URL`、provider 登録、model / `catalog_model` 解決、Chat Completions route、function calling 設定、token / pricing metadata を確認します。`GROQ_API_URL` は Chat Completions まで含む完全な endpoint override で、公式 Groq endpoint は `/openai/v1/chat/completions` で終わります。OpenAI 互換 proxy の `/v1/chat/completions` も指定できますが、doctor では意図的な proxy path として warn になります。`--smoke` を付けると live text request を送って usage / cost を観測し、function calling まで確認したい場合は `--tool-smoke` を使います。`--print-request` は live request を送らずに redacted request body を表示します。

```bash
xelyon doctor groq
xelyon doctor groq --model meta-llama/llama-4-scout-17b-16e-instruct
xelyon doctor groq --model corp-groq-model --catalog-model meta-llama/llama-4-scout-17b-16e-instruct
xelyon doctor groq --smoke
xelyon doctor groq --tool-smoke
xelyon doctor groq --print-request
xelyon doctor groq --json
```

`GROQ_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、text smoke fallback を実行します。`--smoke` / `--tool-smoke` は live API request を送るため、通常 CI では実行しません。手元では `GROQ_API_KEY` を設定して `make groq-doctor-smoke` を実行します。既定モデルは `GROQ_DOCTOR_SMOKE_MODEL ?= meta-llama/llama-4-scout-17b-16e-instruct` です。

### 8. Ollama

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

設定の到達性は CLI から診断できます。`doctor ollama` は `OLLAMA_BASE_URL`、provider 登録、model / `catalog_model` 解決、`/api/tags` の installed model、`/api/chat` route、function calling 設定、token / local zero-cost metadata を確認します。`--smoke` を付けると live text request をローカル Ollama に送り、usage / cost を観測します。function calling まで確認したい場合は `--tool-smoke` を使います。`--print-request` は live request を送らずに request body を表示します。

```bash
xelyon doctor ollama
xelyon doctor ollama --model qwen2.5-coder:7b
xelyon doctor ollama --model corp-local-model --catalog-model qwen2.5-coder:7b
xelyon doctor ollama --smoke
xelyon doctor ollama --tool-smoke
xelyon doctor ollama --print-request
xelyon doctor ollama --json
```

`OLLAMA_BASE_URL` は `http://localhost:11434` のような base URL を指定します。`/api/chat` や `/api/tags` そのものを指定すると doctor は fail します。`OLLAMA_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、text smoke fallback を実行します。手元では `ollama serve` と `ollama pull "$(OLLAMA_DOCTOR_SMOKE_MODEL)"` を済ませてから `make ollama-doctor-smoke` を実行します。既定モデルは `OLLAMA_DOCTOR_SMOKE_MODEL ?= qwen2.5-coder:7b` です。

### 9. OpenRouter

```bash
# API キー取得: https://openrouter.ai
export OPENROUTER_API_KEY=sk-or-...

# 使用例
xelyon --provider openrouter --model anthropic/claude-sonnet-4.6
```

**特徴:**
- 複数プロバイダーのモデルを1つのAPIキーで利用可能
- OpenAI互換API。Claude 系 model は context management 有効時に Anthropic Skin `/v1/messages` route を使用
- 画像入力対応（モデルによる）

設定の到達性は CLI から診断できます。`doctor openrouter` は `OPENROUTER_API_KEY`、`OPENROUTER_API_URL`、provider 登録、model / `catalog_model` 解決、OpenAI-compatible Chat Completions と Anthropic Skin Messages の route 判定、image input support、function calling 設定、token / pricing metadata を確認します。Claude 系 request model で context management が有効な場合は `/v1/messages` の `anthropic_messages` route、それ以外は Chat Completions route を表示します。`catalog_model` は alias の token / pricing / upstream model 表示に使います。実 request model が既知の routed OpenRouter ID、または別 owner の routed ID である場合、mismatch した `catalog_model` は warn になり、token / pricing は実 request model 側へ戻します。route 判定は runtime と同じく実 request model で行います。

```bash
xelyon doctor openrouter
xelyon doctor openrouter --model anthropic/claude-sonnet-4.6
xelyon doctor openrouter --model openai/gpt-5.4
xelyon doctor openrouter --model corp-openrouter-model --catalog-model openai/gpt-5.4
xelyon doctor openrouter --smoke
xelyon doctor openrouter --tool-smoke
xelyon doctor openrouter --print-request
xelyon doctor openrouter --json
```

`OPENROUTER_API_URL` は Chat Completions endpoint または互換 proxy path を指定します。`/v1/messages` など Messages endpoint を直接指定すると fail になり、Anthropic Skin route では doctor/runtime が `/v1/messages` を派生します。`--tool-smoke` は選択 route の形式で dummy tool call を強制します。`OPENROUTER_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、text smoke fallback を実行します。`--print-request` は live request を送らず、OpenRouter 固有の `HTTP-Referer` / `X-Title` と redacted bearer header、選択 route の request body を表示します。`image_input` は provider-level support のローカル check で、OpenRouter doctor v1 は画像 live smoke を送りません。`--smoke` / `--tool-smoke` は live API request を送るため、通常 CI では実行しません。手元では `OPENROUTER_API_KEY` を設定して `make openrouter-doctor-smoke` を実行します。既定モデルは `OPENROUTER_DOCTOR_SMOKE_MODEL ?= openai/gpt-5.4-mini` です。

### 10. Bedrock (AWS)

現在の Bedrock provider は Claude on Bedrock を `InvokeModelWithResponseStream` + Claude Messages 形式で実行します。Claude 以外の Bedrock モデルは `ConverseStream` 経路を使いますが、xelyon の agent 実行では structured tool calling が必須のため、streaming tool use 対応を確認済みのモデルだけを runtime supported として扱います。Converse 経路の画像入力、thinking/reasoning、prompt cache は未対応です。Converse 経路ではモデル上限が catalog で既知の場合のみ `maxTokens` を送信し、未知モデルでは Bedrock 側のデフォルト上限に委ねます。

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
xelyon --provider bedrock --model global.anthropic.claude-sonnet-4-6
xelyon --provider bedrock --model global.anthropic.claude-opus-4-5-20251101-v1:0
xelyon --provider bedrock --model us.anthropic.claude-haiku-4-5-20251001-v1:0
xelyon --provider bedrock --model amazon.nova-pro-v1:0
xelyon --provider bedrock --model moonshotai.kimi-k2.5

# 設定診断
xelyon doctor bedrock
xelyon doctor bedrock --model global.anthropic.claude-sonnet-4-6 --smoke
xelyon doctor bedrock --model amazon.nova-pro-v1:0 --smoke --tool-smoke
xelyon doctor bedrock --print-request
xelyon doctor bedrock --tool-smoke --print-request
xelyon doctor bedrock --json

# 実 API smoke test
make bedrock-smoke
make bedrock-doctor-smoke

# runtime supported モデルの matrix smoke
make bedrock-smoke-matrix

# streaming tool-use unsupported/unverified な Converse モデルの probe
make bedrock-smoke-probe
```

`xelyon doctor bedrock` は AWS region / 認証チェーン、model / `catalog_model` 解決、Claude Messages / ConverseStream route、function calling 設定、token / pricing metadata を確認します。`--smoke` は text smoke、`--tool-smoke` は dummy tool call、`--image-smoke` は tiny PNG 画像入力、`--thinking-smoke` は Extended Thinking request を明示実行します。`--print-request` は live request を送らず、request name、route、Bedrock operation、conceptual endpoint、redacted AWS SigV4 header、request body を `request_preview` に表示します。Smoke 結果は AWS SDK `ResultMetadata` 由来の `request_id`、usage、pricing catalog に基づく概算 cost を request 単位で text / JSON に出します。request ID、usage、pricing が返らない場合は warn ですが、API smoke 成功自体は fail にしません。複数 request のうち 1 件でも usage が返らない場合、summary cost は部分値を確定値として表示せず usage unavailable とします。`BEDROCK_FUNCTION_CALLING=0` の場合、`--tool-smoke` は function calling 無効として warn skip します。Converse route の image / thinking smoke は未対応 request shape として warn skip します。

`--print-request` は AWS credential retrieval と live API request を行わず、単体では text request preview を 1 件出します。Claude family は runtime と同じ Claude Messages body builder から `InvokeModelWithResponseStream` preview を作り、非 Claude は runtime と同じ `buildConverseStreamInput` から `ConverseStream` preview を作ります。Converse route の image / thinking request は smoke と同じ unsupported request shape として skipped entry に残します。

Bedrock smoke test は `XELYON_BEDROCK_SMOKE=1` のときだけ実 API を呼びます。Claude route は text / tool use / image / thinking、Converse route は text + usage / tool use を確認します。既定モデルを変える場合は `XELYON_BEDROCK_SMOKE_CLAUDE_MODEL` または `XELYON_BEDROCK_SMOKE_CONVERSE_MODEL` を指定してください。`XELYON_BEDROCK_SMOKE_CONVERSE_MODEL` には streaming tool use 対応モデルだけを指定してください。複数モデルを継続検証する場合は `BEDROCK_SMOKE_CONVERSE_MODELS` を上書きして `make bedrock-smoke-matrix` を実行します。詳しい運用手順は [Bedrock Provider 運用](bedrock.md) を参照してください。

**特徴:**
- AWS フルマネージドサービス（中間マージンなし）
- IAM ロールによるセキュアな認証
- Claude 経路はプロンプトキャッシュ / Extended Thinking / 画像入力対応
- Converse 経路は streaming tool use 対応モデルで text streaming / tool use / usage callback 対応

**利用可能なモデル ID:**
| モデル | Bedrock モデル ID |
|--------|------------------|
| Claude Opus 4.5 | `global.anthropic.claude-opus-4-5-20251101-v1:0` |
| Claude Sonnet 4.6 | `global.anthropic.claude-sonnet-4-6` |
| Claude Haiku 4.5 | `us.anthropic.claude-haiku-4-5-20251001-v1:0` |
| Amazon Nova Pro | `amazon.nova-pro-v1:0` |
| Moonshot Kimi K2.5 | `moonshotai.kimi-k2.5` |

Claude Sonnet 4.6 は `global.anthropic.claude-sonnet-4-6` のほか、`us.anthropic.claude-sonnet-4-6` / `eu.anthropic.claude-sonnet-4-6` / `au.anthropic.claude-sonnet-4-6` の Geo Inference Profile ID も利用できます。

料金表にある Bedrock モデルでも、streaming tool use が未確認または非対応の Converse モデルは xelyon runtime では unsupported として扱います。Text response や non-streaming tool use が可能なだけでは agent 実行対象として自動 fallback しません。

## モデル指定方法

### 1. コマンドラインフラグ（最優先）

```bash
xelyon --provider openai --model gpt-5.4
```

### 2. 環境変数

```bash
export XELYON_PROVIDER=deepseek
export XELYON_MODEL=deepseek-v4-flash
xelyon
```

### 3. 設定ファイル（`~/.xelyon/config.yaml`）

```yaml
default_provider: deepseek
default_model: deepseek-v4-flash

provider_models:
  deepseek:
    default_model: deepseek-v4-flash
  kimi:
    default_model: kimi-k2.6
  openai:
    default_model: gpt-5.4
  azure:
    default_model: my-gpt-5-deployment
    catalog_model: gpt-5.4
  gemini:
    default_model: gemini-3.1-pro-preview-customtools
  claude:
    default_model: claude-sonnet-4-6
  ollama:
    default_model: qwen2.5-coder:7b
  groq:
    default_model: meta-llama/llama-4-scout-17b-16e-instruct
  openrouter:
    default_model: anthropic/claude-sonnet-4.6
  bedrock:
    default_model: global.anthropic.claude-sonnet-4-6
```

### 4. セッション中の切り替え（`/use`コマンド）

```bash
xelyon
> /use openai gpt-5.4
> 質問1
> /use deepseek
> 質問2
```

## プロンプトキャッシュ対応状況

| プロバイダー | 方式 | 状態 | 割引率 | 備考 |
|------------|------|------|-------|------|
| **Claude** | 明示的（`cache_control`） | 安定 | 読み取り 90% OFF | `prompt_cache.enabled: true` で有効 |
| **Bedrock(Claude)** | 明示的（`cache_control`） | 安定 | 読み取り 90% OFF | Claude と同じ仕組み |
| **OpenAI** | 自動（プレフィックス） | **不安定**（GPT-5系） | モデル依存 | `prompt_cache_key` はルーティングヒントのみ。GPT-5.5 Pro は cached input discount なし |
| **DeepSeek** | 自動 | 安定 | 読み取り割引あり | 設定不要 |
| **Kimi** | 自動（`prompt_cache_key`） | 対応 | モデル依存 | usage の `cached_tokens` を表示 |
| **Gemini** | 自動（暗黙的） | 安定 | - | Gemini 2.5 系で対応 |
| **OpenRouter** | プロバイダー依存 | - | - | Anthropic モデル: 手動 `cache_control` 必要 |
| **Groq** | 自動（プレフィックス） | 安定 | 読み取り 50% OFF | 一部モデルのみ（GPT-OSS, Kimi K2） |
| **Ollama** | - | - | - | ローカル実行のため不要 |

### コスト効率の良い選択肢

長い会話でのコスト効率を重視する場合:

1. **DeepSeek V4 Flash** - 低コスト + キャッシュ安定
2. **Kimi K2.6 / K2.5** - 低コスト + Chat Completions の prompt cache key 対応
3. **Bedrock（Claude）** - プロンプトキャッシュが確実に効く + AWS 直接契約で中間マージンなし
4. **Claude（直接）** - プロンプトキャッシュが確実に効く
5. **OpenAI** - 高コスト + キャッシュ不安定のため、コスト重視なら非推奨

## 料金表示

`/status`、ステータスバー、headless JSON の `cost` は `internal/cost/pricing.yaml` と組み込みの既知ルールに基づく推定値です。価格表にない provider/model は別モデルの料金で代用せず、UI では `N/A (pricing unavailable)`、ステータスバーでは `cost N/A` と表示します。headless JSON では `pricing_unavailable: true` を返します。Gemini native web search の `usageMetadata` は通常の token usage として `tokens` / `cost` に含め、Kimi `$web_search` の call fee など token 料金とは別枠の固定費は `cost` に含めますが、検索結果 tokens は token totals に再加算しません。headless JSON では Kimi web search の観測値を `web_search` object の `calls`、`fee_estimate`、`result_tokens` にも分けて出します。

カスタム deployment 名や社内 alias を使う場合は、`provider_models.<provider>.catalog_model` または `model_overrides.<model>.catalog_model` に provider の pricing family で解決できる既知モデル名を指定すると、そのモデルの token limit / pricing / context 判定を使えます。OpenRouter alias では `openai/gpt-5.4` のような OpenRouter model ID、Bedrock Claude alias では Bedrock の Claude model ID または Claude catalog model 名を指定してください。Native Kimi alias では `kimi-k2.6` / `kimi-k2.5` のような Kimi catalog model 名を指定します。`pricing.yaml` の `known_models.exact` にある実モデル ID だけが `catalog_model` なしで料金表示され、`rules.contains` は価格選択専用です。OpenRouter の `provider/model` 形式も OpenRouter 側の exact allowlist にある ID だけを料金表示します。

Bedrock は AWS Price List の US East (N. Virginia) text token 価格が確認できた exact model ID / inference profile ID だけ料金表示します。`global.*` ID は AWS が別料金を出している場合、Global Cross-region の text token 価格を使います。Bedrock の Claude direct / inference profile ID は Bedrock 料金を優先し、`claude-sonnet-4-6` のような抽象 catalog 名だけ Claude 料金へ委譲します。Amazon Nova、Anthropic Claude、Meta Llama、Mistral、Cohere Command R、AI21 Jamba、Writer Palmyra、DeepSeek、Qwen、MiniMax、NVIDIA Nemotron、OpenAI gpt-oss、Google Gemma、Moonshot Kimi、Z.AI GLM の対応済み ID は料金表示されます。embedding / image / video / query 単価の inference profile は text token 料金ではないため `N/A` のままです。リージョン別価格、Batch / Flex / Priority / Provisioned Throughput、画像・音声・動画 token、query/unit ベースの rerank はまだ推定対象外です。

## プロバイダー選択のヒント

### コード生成・編集
- **DeepSeek V4 Flash**: 高速・低コスト・普段使い
- **DeepSeek V4 Pro**: 高精度・重い設計/レビュー向き
- **Kimi K2.6**: 低コストで大きい文脈を扱う編集向き
- **Qwen2.5-Coder (Ollama)**: ローカル実行

### 複雑な問題解決
- **Claude Opus 4**: 長文理解・推論
- **GPT-5.4**: バランスの良い性能

### 高速レスポンス
- **Groq**: 超高速推論
- **Kimi K2.5**: 低コストなサブエージェント/軽作業向き
- **Gemini Flash**: バランス良く高速

### 画像解析
- **Kimi**: Moonshot Chat Completions の native multimodal image input
- **OpenAI**: 高品質な画像理解
- **Azure OpenAI**: Azure 上の GPT deployment で画像入力
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
xelyon --provider openai --model gpt-5.4
```

### レート制限エラー

APIプロバイダーのダッシュボードで使用状況とレート制限を確認してください。

- DeepSeek: https://platform.deepseek.com/usage
- Kimi: https://platform.moonshot.ai
- OpenAI: https://platform.openai.com/usage
- Azure OpenAI: Azure Portal の Azure OpenAI resource
- Gemini: https://aistudio.google.com
- Claude: https://console.anthropic.com
- Groq: https://console.groq.com

## 関連ドキュメント

- [コマンド一覧](commands.md)
- [設定リファレンス](config.md)
- [LSP連携](lsp.md)
- [MCP連携](mcp.md)
