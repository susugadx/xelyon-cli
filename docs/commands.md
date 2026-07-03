# コマンド一覧

XELYON CLIで使用できる全コマンドのリファレンスです。

## CLI 診断コマンド

### Doctor provider matrix

`doctor` は provider ごとに runtime route、endpoint override、live smoke の意味が違います。共通の確認だけなら `--json` と `--print-request` を使い、`--smoke` 系は live API request を送る場合だけ指定してください。

複数 provider の doctor smoke を横断実行する場合は、対象 provider を明示して `DOCTOR_SMOKE_PROVIDERS="openai groq kimi" make doctor-smoke-matrix` を使います。`DOCTOR_SMOKE_PROVIDERS` が空のときは実 API / ローカル runtime を呼ばず終了します。各 provider の API key や AWS / Ollama の準備は従来の個別 target と同じです。

| Provider | Request target | Live smoke flags | Local / gate flags | Request preview | Endpoint contract | Usage / cost observation |
| --- | --- | --- | --- | --- | --- | --- |
| `deepseek` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `DEEPSEEK_API_URL` is an exact Chat Completions endpoint or intentional proxy path | text / tool token usage and cost when returned |
| `kimi` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--web-search-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `KIMI_API_URL` is an exact Chat Completions endpoint or intentional proxy path | token usage plus built-in web search call count / fee observations |
| `gemini` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--web-search-smoke` | `--capabilities`, `--require-capability` | `--print-request` | text / tool / image use `streamGenerateContent?alt=sse`; native web search uses `generateContent` | SSE / `usageMetadata` usage and cost when returned; web search usage is optional for success |
| `claude` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--thinking-smoke`, `--web-search-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `ANTHROPIC_API_URL` is an exact `/v1/messages` endpoint or intentional proxy path | Messages usage and cost when returned; web search usage is optional for success |
| `groq` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `GROQ_API_URL` is an exact Chat Completions endpoint or intentional proxy path | text / tool token usage and cost when returned |
| `ollama` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `OLLAMA_BASE_URL` is a base URL; concrete `/api/chat` or `/api/tags` endpoints fail | local zero-cost token usage when returned |
| `openrouter` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `OPENROUTER_API_URL` is Chat Completions / proxy; Anthropic Skin `/v1/messages` is derived | selected route token usage and cost when returned |
| `openai` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--retention-smoke` | `--capabilities`, `--require-capability` | `--print-request` | `OPENAI_API_URL` is Chat Completions; `OPENAI_RESPONSES_URL` is Responses | response ID, token usage, cost, and retention chain metadata when returned |
| `openai-subscription` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--retention-smoke`, `--cache-smoke`, `--compact-smoke`, `--thinking-smoke`, `--web-search-smoke` | `--capabilities`, `--require-capability` | `--print-request` | ChatGPT/Codex OAuth subscription endpoint; full-payload Responses-shaped runtime plus dedicated native web_search payload | streaming usage when returned; web search call count when observed; cost is N/A (ChatGPT subscription) |
| `azure` | `--deployment`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--retention-smoke` | `--capabilities`, `--require-capability`, `--print-config` | `--print-request` | `AZURE_OPENAI_BASE_URL` is a resource v1 base URL; smoke uses `<normalized_base_url>/responses` | response ID, token usage, cost, and retention chain metadata when returned |
| `bedrock` | `--model`, `--catalog-model` | `--smoke`, `--tool-smoke`, `--image-smoke`, `--thinking-smoke` | `--capabilities`, `--require-capability` | `--print-request` | AWS region / credentials select Bedrock runtime route; request preview is credential-independent | AWS request ID, token usage, and cost when returned; partial usage makes total cost unavailable |

### `xelyon doctor mcp`

MCP の local config と tool discovery を診断します。デフォルトでは local-only で、`~/.xelyon/mcp.json` が存在する場合だけ読み、ファイル作成や MCP server process の起動は行いません。`mcp.enabled` / `mcp.headless`、server 数、command allowlist、timeout、approval、disabled、include / exclude を確認します。

`--connect` を付けると、対象 MCP server を起動して initialize / `tools/list` まで確認します。MCP tool の `tools/call` は実行しません。runtime 全体の live tool surface total / registered / visible / omitted 数、effective budget、server 別 estimated tokens / schema bytes、omitted reason、`tools.include` / `tools.exclude` と `mcp.surface_budget` の提案も表示します。estimated tokens は analysis に含めた tool の provider tool definition 相当から推定した合計です。schema body は表示しません。`--tools` は `--connect` と組み合わせて raw tool name、XELYON の exported tool name、visible / skipped reason、approval を表示します。`--server` で 1 server に絞れます。JSON 出力は `--json` を使います。

```bash
xelyon doctor mcp
xelyon doctor mcp --json
xelyon doctor mcp --server github
xelyon doctor mcp --connect
xelyon doctor mcp --connect --tools
xelyon doctor mcp --connect --server github --json
```

`doctor mcp` は env value と raw args を出力しません。表示するのは command、arg 数、env key 名、timeout、approval、tool 名、集計済みの token / schema byte 数です。schema body や description 全文は表示しません。

### `xelyon doctor deepseek`

DeepSeek provider の `DEEPSEEK_API_KEY`、`DEEPSEEK_API_URL`、provider 登録、model / `catalog_model` 解決、Chat Completions route、thinking request config、function calling 設定、token / pricing metadata を確認します。route は常に `chat_completions` です。`--smoke` を付けると live text request を送信し、content、usage、概算 cost を表示します。function calling まで確認する場合は `--tool-smoke` を使い、dummy tool call を強制します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。

`--model` は実 request に送る DeepSeek model ID または alias、`--catalog-model` は alias の underlying DeepSeek model として token / pricing / thinking 判定に使います。non-DeepSeek `catalog_model` は warn になり、OpenAI / OpenRouter など別 owner の metadata は token / pricing / capability 判定に使いません。`DEEPSEEK_API_URL` は Chat Completions まで含む完全な endpoint override で、公式 DeepSeek 互換 endpoint は `/chat/completions` で終わります。OpenAI 互換 proxy の `/v1/chat/completions` も指定できますが、doctor では意図的な proxy path として warn になります。`DEEPSEEK_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、text smoke fallback を実行します。`--capabilities` は静的 / catalog ベースの capability snapshot を表示し、`--require-capability` は live request なしで shared capability 名を検証します。`--retention-smoke`、`--image-smoke`、`--thinking-smoke`、`--web-search-smoke` は DeepSeek doctor v1 では提供しません。`--smoke` / `--tool-smoke` は live API request を送るため、設定確認だけなら付けないでください。

```bash
xelyon doctor deepseek
xelyon doctor deepseek --model deepseek-v4-flash
xelyon doctor deepseek --model corp-deepseek-model --catalog-model deepseek-v4-flash
xelyon doctor deepseek --smoke
xelyon doctor deepseek --tool-smoke
xelyon doctor deepseek --capabilities
xelyon doctor deepseek --require-capability chat_completions --require-capability function_calling
xelyon doctor deepseek --print-request
xelyon doctor deepseek --tool-smoke --print-request
xelyon doctor deepseek --smoke --tool-smoke
xelyon doctor deepseek --json
```

手元で doctor 経路だけを実 DeepSeek 環境で確認する場合は、`DEEPSEEK_API_KEY` を設定して `make deepseek-doctor-smoke` を実行します。既定では `deepseek-v4-flash` で text / tool smoke をまとめて実行し、必要なら `DEEPSEEK_DOCTOR_SMOKE_MODEL` で変更できます。

### `xelyon doctor kimi`

Kimi native provider の `MOONSHOT_API_KEY`、`KIMI_API_URL`、provider 登録、model / `catalog_model` 解決、Chat Completions route、token / pricing metadata、未対応機能、`prompt_cache_key` request shape を確認します。`KIMI_API_URL` は Chat Completions まで含む完全な endpoint override で、公式 Moonshot endpoint は `/v1/chat/completions` で終わります。別 path の proxy endpoint も指定できますが、doctor では意図的な proxy path として warn になります。`--catalog-model` は alias の underlying Kimi model として token / pricing 判定に使います。non-Kimi `catalog_model` は warn になり、OpenAI / OpenRouter など別 owner の metadata は token / pricing / capability 判定に使いません。`--print-request` は live request を送らず、redacted bearer header と request body を `request_preview` に表示します。`--smoke` を付けると live Chat Completions request を送信し、streaming、thinking on/off、同一 session の prompt cache key、usage callback を確認します。画像入力の実 API 受理を確認する場合は `--image-smoke` を使い、1x1 PNG を base64 image request として送信します。function calling まで確認する場合は `--tool-smoke`、built-in `$web_search` まで確認する場合は `--web-search-smoke` を使います。K2.7 Code 選択中の built-in `$web_search` smoke / preview は検索 request だけ `kimi-k2.6` に fallback し、usage / cost / 実行ログも `kimi-k2.6` として扱います。web search smoke は `web_search_call_count`、`web_search_call_fee_estimate`、`web_search_usage_observed`、`cached_input_tokens`、検索結果 token 観測値を text / JSON に出します。

`--print-request` は `MOONSHOT_API_KEY` なしで実行でき、text / tool / image / built-in `$web_search` の request body と実 request URL を送信前に確認できます。`--capabilities` は静的 / catalog ベースの capability snapshot を表示し、`--require-capability` は live request なしで shared capability 名を検証します。`--smoke` / `--image-smoke` / `--tool-smoke` / `--web-search-smoke` は live API request を送るため、通常 CI では使いません。`cached_tokens` は Moonshot API が返した場合だけ観測され、0 でも smoke は成功扱いです。`--web-search-smoke` は実検索 call fee が発生し、`$web_search` tool call が 1 件以上観測された場合だけ成功扱いになります。通常の `stop` response で tool call がない場合、request 自体が返っていても smoke は fail します。call fee は token cost とは別料金で、検索結果 tokens は次 request の `prompt_tokens` に含まれるため二重加算しません。endpoint の token usage が返らない場合は `usage` check が warn になり、web search の fee / call count 観測だけでは token usage 観測済みとは扱いません。

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
xelyon doctor kimi --capabilities
xelyon doctor kimi --require-capability image_input --require-capability web_search
xelyon doctor kimi --json
```

手元で doctor 経路だけを実 Kimi 環境で確認する場合は、`MOONSHOT_API_KEY` を設定して `make kimi-doctor-smoke` を実行します。既定では `kimi-k2.6` で text / tool / image / built-in web search smoke をまとめて実行し、必要なら `KIMI_DOCTOR_SMOKE_MODEL` で変更できます。runtime live test を走らせる場合は `make kimi-smoke`、画像入力だけ確認する場合は `make kimi-image-smoke`、tool calling も含める場合は `make kimi-tool-smoke`、built-in web search は `make kimi-web-search-smoke` を使います。

### `xelyon doctor gemini`

Gemini provider の `GEMINI_API_KEY`、`GEMINI_API_URL`、provider 登録、model / `catalog_model` 解決、`streamGenerateContent?alt=sse` route、function calling、画像入力、thinking、context caching、native web search、token / pricing metadata、`gemini.service_tier` の request / pricing policy を確認します。`--smoke` は live text SSE の最小疎通確認として content、usage、概算 cost を表示します。実運用の function calling 本線まで確認する場合は `--tool-smoke` を使い、request-scoped `ANY` mode で dummy tool call を強制します。画像入力は `--image-smoke`、native web search は `--web-search-smoke` で確認します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted `x-goog-api-key` header、request body を `request_preview` に表示します。`gemini.service_tier` が `flex` / `priority` の場合は preview body にも `service_tier` が出ます。

`--model` は実 request に送る Gemini model ID または alias、`--catalog-model` は alias の underlying Gemini model として token / pricing / thinking / timeout / function calling capability 判定に使います。Gemini の `models/` prefix 付き catalog model はローカル catalog 名へ正規化されます。non-Gemini `catalog_model` は warn になり、OpenAI / OpenRouter など別 owner の metadata は Gemini doctor の token / pricing / capability policy に使いません。既知の function calling 非対応 Gemini は doctor の `function_calling` check が fail になり、CLI/TUI の選択時点と通常 runtime の request 前にも止めます。config validation は `provider_models.gemini.default_model` / `catalog_model` / `model_overrides.*.catalog_model` の既知非対応 Gemini を error にし、既知の代替 model へ autofix できます。unknown alias は互換性のため即 fail しませんが、`--catalog-model` なしでは function calling capability が warn になります。tool smoke / preview では request-scoped `ANY` mode を使って diagnostic tool だけを送り、通常 runtime の `GEMINI_FC_MODE` fallback は変更しません。`GEMINI_API_URL` は runtime と同じ exact endpoint / proxy override で、`--print-request` の `request_preview.requests[].url` が実際の送信先です。text / tool / image は `streamGenerateContent?alt=sse`、native web search は `generateContent` の request shape を同じ URL に送るため、`--web-search-smoke` では `generateContent` を受ける endpoint または両方の shape を受ける proxy を使います。`--timeout` は複数 smoke 全体ではなく各 request に個別適用されます。`gemini.service_tier: flex` では同期 request の response-start timeout を 10 分まで広げます。`priority` が `standard` に downgrade された場合、Gemini の `usageMetadata.serviceTier` を優先し、`x-gemini-service-tier` response header を fallback として cost 概算に反映します。doctor は `service_tier` check と JSON `service_tier` object で configured tier、request body tier、pricing family、smoke 後の実課金 tier を表示します。doctor smoke では実課金 tier が返った場合に usage 表示へ `billing_tier`、JSON usage へ `billing_service_tier` を出します。Batch API は非同期ジョブ API なので Gemini doctor の request preview / smoke 対象外です。`--capabilities` は静的 / catalog ベースの capability snapshot を表示し、`--require-capability` は live request なしで shared capability 名を検証します。`--retention-smoke`、`--thinking-smoke` は Gemini doctor v1 では提供しません。`--smoke` / `--tool-smoke` / `--image-smoke` / `--web-search-smoke` は live API request を送るため、設定確認だけなら付けないでください。

```bash
xelyon doctor gemini
xelyon doctor gemini --model gemini-3.5-flash
xelyon doctor gemini --model corp-gemini-model --catalog-model gemini-3.5-flash
xelyon doctor gemini --smoke
xelyon doctor gemini --tool-smoke
xelyon doctor gemini --image-smoke
xelyon doctor gemini --web-search-smoke
xelyon doctor gemini --capabilities
xelyon doctor gemini --require-capability image_input --require-capability web_search
xelyon doctor gemini --print-request
xelyon doctor gemini --tool-smoke --print-request
xelyon doctor gemini --smoke --tool-smoke --image-smoke --web-search-smoke
xelyon doctor gemini --json
```

手元で doctor 経路だけを実 Gemini 環境で確認する場合は、`GEMINI_API_KEY` を設定して `make gemini-doctor-smoke` を実行します。既定では `gemini-3.1-pro-preview-customtools` で minimal text / function-calling tool / image / web search smoke をまとめて実行し、必要なら `GEMINI_DOCTOR_SMOKE_MODEL` で変更できます。Make target の timeout は `GEMINI_DOCTOR_SMOKE_TIMEOUT ?= 180s` で、各 smoke request に個別適用されます。web search smoke は native `generateContent` の `usageMetadata` が返れば usage / cost を表示し、返らない場合は usage / cost を warn に留めます。summary または source が返れば smoke は成功扱いです。

Gemini live smoke の失敗時は `smoke` check の message / suggestion と `smoke.requests[].error` を見ます。doctor は認証・権限、quota / rate limit / capacity、model unavailable、empty SSE response、`GEMINI_API_URL` route mismatch、tool unsupported、image unsupported、native web search unsupported を分類します。text / tool / image は `streamGenerateContent?alt=sse`、web search は `generateContent` のため、proxy や endpoint override を使う場合は `xelyon doctor gemini --smoke --tool-smoke --image-smoke --web-search-smoke --print-request` で request preview を確認してから live smoke を実行してください。pricing metadata がない場合は smoke 自体は fail せず、`cost` check が warn になります。

### `xelyon doctor claude`

Claude provider の `ANTHROPIC_API_KEY`、`ANTHROPIC_API_URL`、provider 登録、model / `catalog_model` 解決、Anthropic Messages route、function calling、画像入力、thinking request config、context management、Claude compaction、native web search、token / pricing metadata を確認します。`--smoke` を付けると live text request を送信し、content、usage、概算 cost を表示します。function calling まで確認する場合は `--tool-smoke` を使い、diagnostic tool call を強制します。画像入力は `--image-smoke`、thinking request は `--thinking-smoke`、native web search は `--web-search-smoke` で確認します。native web search は通常 Messages request と同じ thinking policy を継承し、adaptive model では `thinking` と `output_config.effort`、legacy thinking model では `budget_tokens` を送ります。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted `x-api-key` header、Anthropic headers、request body を `request_preview` に表示します。

`--model` は実 request に送る Claude model ID または alias、`--catalog-model` は alias の underlying Claude model として token / pricing / thinking / context management 判定に使います。non-Claude `catalog_model` は warn になり、OpenAI / OpenRouter など別 owner の metadata は Claude doctor の token / pricing / capability policy に使いません。`web_search` required capability は request model または trusted Claude `catalog_model` で対応を確認できる場合だけ pass します。`ANTHROPIC_API_URL` は runtime と同じ exact endpoint / proxy override で、公式 Anthropic endpoint では `/v1/messages` を期待します。別 path の proxy endpoint も指定できますが、doctor では意図的な proxy path として warn になり、`--print-request` / live smoke はその URL を実 request 先として使います。`--print-request` は `ANTHROPIC_API_KEY` なしで実行でき、text / tool / image / thinking / native web search の request body を送信前に確認できます。`--capabilities` は静的 / catalog ベースの capability snapshot を表示し、`--require-capability` は live request なしで shared capability 名を検証します。`--retention-smoke` は Claude doctor v1 では提供しません。`--smoke` / `--tool-smoke` / `--image-smoke` / `--thinking-smoke` / `--web-search-smoke` は live API request を送るため、設定確認だけなら付けないでください。

```bash
xelyon doctor claude
xelyon doctor claude --model claude-sonnet-4-6
xelyon doctor claude --model corp-claude-model --catalog-model claude-sonnet-4-6
xelyon doctor claude --smoke
xelyon doctor claude --tool-smoke
xelyon doctor claude --image-smoke
xelyon doctor claude --thinking-smoke
xelyon doctor claude --web-search-smoke
xelyon doctor claude --capabilities
xelyon doctor claude --require-capability image_input --require-capability thinking
xelyon doctor claude --print-request
xelyon doctor claude --tool-smoke --print-request
xelyon doctor claude --smoke --tool-smoke --image-smoke --thinking-smoke --web-search-smoke
xelyon doctor claude --json
```

手元で doctor 経路だけを実 Claude 環境で確認する場合は、`ANTHROPIC_API_KEY` を設定して `make claude-doctor-smoke` を実行します。既定では `claude-sonnet-4-6` で text / tool / image / thinking / web search smoke をまとめて実行し、必要なら `CLAUDE_DOCTOR_SMOKE_MODEL` で変更できます。timeout は `CLAUDE_DOCTOR_SMOKE_TIMEOUT ?= 180s` で変更できます。Claude native web search smoke は summary または source が返れば成功扱いで、現時点では token usage / cost 観測は必須にしません。

### `xelyon doctor groq`

Groq provider の `GROQ_API_KEY`、`GROQ_API_URL`、provider 登録、model / `catalog_model` 解決、Chat Completions route、function calling 設定、token / pricing metadata を確認します。route は常に `chat_completions` です。`--smoke` を付けると live text request を送信し、content、usage、概算 cost を表示します。function calling まで確認する場合は `--tool-smoke` を使い、dummy tool call を強制します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。

`--model` は実 request に送る Groq model ID または alias、`--catalog-model` は alias の underlying Groq model として token / pricing 判定に使います。non-Groq `catalog_model` は warn になり、OpenAI / OpenRouter など別 owner の metadata は token / pricing / capability 判定に使いません。`GROQ_API_URL` は Chat Completions まで含む完全な endpoint override で、公式 Groq endpoint は `/openai/v1/chat/completions` で終わります。OpenAI 互換 proxy の `/v1/chat/completions` も指定できますが、doctor では意図的な proxy path として warn になります。`GROQ_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、text smoke fallback を実行します。`--capabilities` は静的 / catalog ベースの capability snapshot を表示し、`--require-capability` は live request なしで shared capability 名を検証します。`--retention-smoke`、`--image-smoke`、`--thinking-smoke`、`--web-search-smoke` は Groq doctor v1 では提供しません。`--smoke` / `--tool-smoke` は live API request を送るため、設定確認だけなら付けないでください。

```bash
xelyon doctor groq
xelyon doctor groq --model meta-llama/llama-4-scout-17b-16e-instruct
xelyon doctor groq --model corp-groq-model --catalog-model meta-llama/llama-4-scout-17b-16e-instruct
xelyon doctor groq --smoke
xelyon doctor groq --tool-smoke
xelyon doctor groq --capabilities
xelyon doctor groq --require-capability chat_completions --require-capability function_calling
xelyon doctor groq --print-request
xelyon doctor groq --tool-smoke --print-request
xelyon doctor groq --smoke --tool-smoke
xelyon doctor groq --json
```

手元で doctor 経路だけを実 Groq 環境で確認する場合は、`GROQ_API_KEY` を設定して `make groq-doctor-smoke` を実行します。既定では `meta-llama/llama-4-scout-17b-16e-instruct` で text / tool smoke をまとめて実行し、必要なら `GROQ_DOCTOR_SMOKE_MODEL` で変更できます。

### `xelyon doctor ollama`

Ollama provider の `OLLAMA_BASE_URL`、provider 登録、model / `catalog_model` 解決、`/api/tags` による installed model、`/api/chat` route、function calling 設定、token / local zero-cost metadata を確認します。Ollama は API key を使わないため、`auth` check は no-auth local provider として `ok` になります。`--smoke` を付けると live text request をローカル Ollama に送信し、content、usage、概算 cost を表示します。function calling まで確認する場合は `--tool-smoke` を使い、dummy tool call を強制します。`--print-request` を付けると live request を送らず、選択した smoke request の `/api/chat` endpoint、headers、request body を `request_preview` に表示します。

`--model` は実 request に送る Ollama model ID または alias、`--catalog-model` は alias の underlying Ollama model として token policy に使います。`OLLAMA_BASE_URL` は `http://localhost:11434` のような base URL で、`/api/chat` や `/api/tags` そのものを指定すると fail になります。non-Ollama `catalog_model` は warn になり、OpenAI / OpenRouter など別 owner の metadata は Ollama doctor の token policy に使いません。未知のローカルモデル名は request model としては許容されますが、`/api/tags` に存在しない場合は `installed_model` が fail します。`OLLAMA_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、text smoke fallback を実行します。`--capabilities` は静的 / catalog ベースの capability snapshot を表示し、`--require-capability` は generation request なしで shared capability 名を検証します。通常の capability-only 実行では `/api/tags` を叩かないため、JSON は `local_model_available=false` と `local_model_available_known=false` を出し、text detail は `local_model_available=unknown` になります。`--require-capability local_model_available` は model availability gate の source of truth として `/api/tags` discovery を実行します。`--retention-smoke`、`--image-smoke`、`--thinking-smoke`、`--web-search-smoke` は Ollama doctor v1 では提供しません。`--smoke` / `--tool-smoke` はローカル live generation request を送るため、設定確認だけなら付けないでください。

```bash
xelyon doctor ollama
xelyon doctor ollama --model qwen2.5-coder:7b
xelyon doctor ollama --model corp-local-model --catalog-model qwen2.5-coder:7b
xelyon doctor ollama --smoke
xelyon doctor ollama --tool-smoke
xelyon doctor ollama --capabilities
xelyon doctor ollama --require-capability function_calling
xelyon doctor ollama --print-request
xelyon doctor ollama --tool-smoke --print-request
xelyon doctor ollama --smoke --tool-smoke
xelyon doctor ollama --json
```

手元で doctor 経路だけをローカル Ollama で確認する場合は、`ollama serve` を起動し、対象モデルを `ollama pull` してから `make ollama-doctor-smoke` を実行します。既定では `qwen2.5-coder:7b` で text / tool smoke をまとめて実行し、必要なら `OLLAMA_DOCTOR_SMOKE_MODEL` で変更できます。

### `xelyon doctor openrouter`

OpenRouter provider の `OPENROUTER_API_KEY`、`OPENROUTER_API_URL`、provider 登録、model / `catalog_model` 解決、OpenAI-compatible Chat Completions と Anthropic Skin Messages の route 判定、image input support、function calling 設定、token / pricing metadata を確認します。Claude 系 request model で context management が有効な場合は `anthropic_messages` route を選び、それ以外は `chat_completions` route を選びます。`--smoke` を付けると選択された route に live text request を送信し、content、usage、概算 cost を表示します。function calling まで確認する場合は `--tool-smoke` を使い、選択 route の形式で dummy tool call を強制します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。

`--model` は実 request に送る OpenRouter model ID または alias、`--catalog-model` は alias の underlying OpenRouter model ID として token / pricing / upstream model 表示に使います。`--model` が `anthropic/...` や `openai/...` の routed OpenRouter ID の場合、別 owner の `catalog_model` や既知 routed model への別 ID 上書きは warn になり、token / pricing / capability metadata は実 request model 側へ戻します。route 判定は runtime と同じく実 request model で行うため、`catalog_model` だけが `anthropic/claude-*` の alias は Chat Completions route として warn になります。`OPENROUTER_API_URL` は Chat Completions endpoint または互換 proxy path を指定します。`/v1/messages` など Messages endpoint を直接指定すると fail になり、Anthropic Skin route では doctor/runtime が `/v1/messages` を派生します。`OPENROUTER_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、text smoke fallback を実行します。`--capabilities` は静的 / catalog ベースの capability snapshot を表示し、`--require-capability` は live request なしで shared capability 名を検証します。`--retention-smoke`、`--image-smoke`、`--thinking-smoke`、`--web-search-smoke` は OpenRouter doctor v1 では提供しません。`image_input` の通常 check は provider request path を確認し、`--require-capability image_input` は trusted OpenRouter catalog metadata で選択 upstream model の画像対応が分かる場合だけ pass します。画像 live smoke は送りません。`--smoke` / `--tool-smoke` は live API request を送るため、設定確認だけなら付けないでください。

```bash
xelyon doctor openrouter
xelyon doctor openrouter --model anthropic/claude-sonnet-4.6
xelyon doctor openrouter --model openai/gpt-5.4
xelyon doctor openrouter --model corp-openrouter-model --catalog-model openai/gpt-5.4
xelyon doctor openrouter --smoke
xelyon doctor openrouter --tool-smoke
xelyon doctor openrouter --capabilities
xelyon doctor openrouter --require-capability image_input
xelyon doctor openrouter --print-request
xelyon doctor openrouter --tool-smoke --print-request
xelyon doctor openrouter --smoke --tool-smoke
xelyon doctor openrouter --json
```

手元で doctor 経路だけを実 OpenRouter 環境で確認する場合は、`OPENROUTER_API_KEY` を設定して `make openrouter-doctor-smoke` を実行します。既定では `openai/gpt-5.4-mini` で text / tool smoke をまとめて実行し、必要なら `OPENROUTER_DOCTOR_SMOKE_MODEL` で変更できます。

### `xelyon doctor openai`

OpenAI provider の `OPENAI_API_KEY`、`OPENAI_API_URL`、`OPENAI_RESPONSES_URL`、provider 登録、model / `catalog_model` 解決、Responses / Chat Completions route と判定理由、function calling 設定、token / pricing metadata、Responses retention 設定を確認します。`--capabilities` を付けると live request を送らず、Responses API / streaming / function calling / image input / native web search / `previous_response_id` / server compaction / context window / max output / pricing の解決結果を `capabilities` に表示します。`--require-capability` を付けると、解決済みのローカル capability が要求を満たすかを live request なしの `required_capability` check として検証します。対応する名前は `responses_api`、`responses_streaming`、`chat_completions`、`function_calling`、`image_input`、`web_search`、`thinking`、`previous_response_id`、`session_persistence`、`server_compaction`、`local_model_available` です。OpenAI の `responses_streaming` gate は実 `catalog_model` が既知 catalog model として解決できない場合 `unknown` として fail します。`web_search` gate は runtime と同じく Responses API route の model だけ pass し、Chat Completions route では missing になります。`thinking` gate は Responses route だけでなく、通常の request builder が reasoning payload を送る設定または Codex の reasoning fallback を確認できた場合だけ pass します。`--smoke` を付けると live text request を送信し、Responses route では response ID、usage、概算 cost を表示します。Chat Completions route では response ID は返らないため、usage と概算 cost を確認します。function calling まで確認する場合は `--tool-smoke` を使い、dummy tool call を強制します。Responses API の `previous_response_id` chain まで確認する場合は `--retention-smoke` を使い、`responses.store=true` の initial / followup request を連続実行します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。複数 smoke を指定した場合は request を別々に実行または preview し、JSON では `smoke.requests[]` / `request_preview.requests[]` に request 単位の結果を出します。

`OPENAI_FUNCTION_CALLING=0` の場合、`--tool-smoke` は warn skip になり、tool payload / `tool_choice` は送信しません。`--retention-smoke` は Responses API route 専用で、Chat Completions route では live request を送らず fail します。`--model` は実 request に送るモデル名または alias、`--catalog-model` は alias の underlying OpenAI model として token / pricing / route 判定に使います。`OPENAI_API_URL` は Chat Completions まで含む完全な endpoint override で、公式 path は `/v1/chat/completions` です。`OPENAI_RESPONSES_URL` は Responses まで含む完全な endpoint override で、公式 path は `/v1/responses` です。別 path の proxy endpoint も指定できますが、`doctor openai` では意図的な proxy path として warn になり、request preview / live smoke は設定 URL をそのまま使います。`--smoke` / `--tool-smoke` / `--retention-smoke` は live API request を送るため、設定確認だけなら付けないでください。

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

手元で doctor 経路だけを実 OpenAI 環境で確認する場合は、`OPENAI_API_KEY` を設定して `make openai-doctor-smoke` を実行します。既定では `gpt-5.4` で text / tool / retention smoke をまとめて実行し、必要なら `OPENAI_DOCTOR_SMOKE_MODEL` で変更できます。

### `xelyon doctor openai-subscription`

OpenAI Subscription provider の local auth、endpoint、originator、provider 登録、model / `catalog_model` 解決、billing / cost、v2 full-payload runtime policy、native web search capability を確認します。default doctor は local-only で、token refresh や live request は行いません。`--capabilities` は Responses streaming / function calling / native web search / thinking / unsupported chain capability を live request なしで表示し、`--require-capability` は shared capability 名を local gate として評価します。

この provider は ChatGPT/Codex OAuth を使う experimental provider で、OpenAI Platform API key provider ではありません。`OPENAI_API_KEY`、OpenCode auth cache、Codex auth cache は使いません。runtime は `store=false` / full payload / `prompt_cache_key` / streaming / tool loop / subscription Compact API で動作し、`previous_response_id` と `context_management` は expected unsupported / disabled として表示します。native web search は通常 tool loop とは別に `tools: [{"type":"web_search"}]` / `tool_choice: "required"` の dedicated Responses-shaped payload を送り、検索 query は 1 件の Responses input item list に入れます。既存の `thinking.enabled` / `thinking.level` を継承し、Codex catalog model の low reasoning fallback も通常 request と同じです。cost は `N/A (ChatGPT subscription)` です。

```bash
xelyon doctor openai-subscription
xelyon doctor openai-subscription --model gpt-5.4-mini
xelyon doctor openai-subscription --capabilities
xelyon doctor openai-subscription --require-capability responses_streaming --require-capability function_calling --require-capability web_search
xelyon doctor openai-subscription --print-request
xelyon doctor openai-subscription --tool-smoke --print-request
xelyon doctor openai-subscription --web-search-smoke --print-request
xelyon doctor openai-subscription --smoke
xelyon doctor openai-subscription --cache-smoke
xelyon doctor openai-subscription --compact-smoke
xelyon doctor openai-subscription --thinking-smoke
xelyon doctor openai-subscription --web-search-smoke
xelyon doctor openai-subscription --retention-smoke
xelyon doctor openai-subscription --tool-smoke
xelyon doctor openai-subscription --json
```

`--smoke` / `--tool-smoke` / `--cache-smoke` / `--thinking-smoke` / `--web-search-smoke` は live subscription request を送ります。`--web-search-smoke` は `web_search_call` が 1 件以上観測され、summary または source URL が返った場合だけ成功扱いにします。`--compact-smoke` は既定で live verified された subscription Compact endpoint（`https://chatgpt.com/backend-api/codex/responses/compact`）を検証します。`XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT` で検証先を差し替えられ、空文字に明示設定した場合は WARN skip し、runtime の `/compress --compact` も unsupported 扱いになります。`openai_subscription` の `/compress --compact` と `compression.prefer_compact_api` はこの subscription Compact endpoint を使い、OpenAI Platform Compact API や `OPENAI_API_KEY` には fallback しません。unsupported は basic provider failure ではなく WARN です。`--retention-smoke` は v2 runtime が chain を使わず full payload fallback を維持していることを確認します。`--print-request` は live request を送らず、Authorization、account ID、token-like strings、raw prompt body、raw web search query を出さない structural preview を表示します。`--web-search-smoke --print-request` では input item shape と reasoning presence / effort を確認できます。

### `xelyon doctor azure`

Azure OpenAI の base URL、認証、Entra ID token command、deployment 解決、`catalog_model`、Responses route と判定理由、function calling 設定、Responses retention 設定を確認します。`--capabilities` を付けると live request を送らず、Responses API / streaming / function calling / image input / `previous_response_id` / server compaction / context window / max output / pricing の解決結果を `capabilities` に表示します。`--require-capability` を付けると、解決済みのローカル capability が要求を満たすかを live request なしの `required_capability` check として検証します。対応する名前は `responses_api`、`responses_streaming`、`chat_completions`、`function_calling`、`image_input`、`web_search`、`thinking`、`previous_response_id`、`session_persistence`、`server_compaction`、`local_model_available` です。Azure の `responses_streaming` gate は実 `catalog_model` が未解決の場合 `unknown` として fail します。`thinking` gate は Responses route だけでなく、通常の request builder が reasoning payload を送る設定または Codex の reasoning fallback を確認できた場合だけ pass します。`--smoke` を付けると、設定済み deployment に `responses.store=false` の最小 Responses API リクエストを送信し、response ID、usage、概算 cost も表示します。function calling まで確認する場合は `--tool-smoke` を使い、dummy tool call を強制します。Responses API の `previous_response_id` chain まで確認する場合は `--retention-smoke` を使い、`responses.store=true` の initial / followup request を連続実行します。`--print-request` を付けると live request を送らず、選択した smoke request の endpoint、redacted headers、request body を `request_preview` に表示します。

`--deployment` は Azure 側の deployment 名、`--catalog-model` はその deployment の実モデル名です。`AZURE_OPENAI_BASE_URL` は Azure OpenAI resource の v1 base URL で、公式 path は `/openai/v1` です。resource root と `/openai` は `/openai/v1` に正規化され、doctor/runtime は実 request を `<normalized_base_url>/responses` に送ります。`/openai/deployments/<deployment>` や public OpenAI host は fail、`api-version` query は warn で無視します。非標準 path は intentional proxy として warn になり、`--print-request` / live smoke はその path に `/responses` を付けた URL を使います。`--print-config` を付けると、deployment と catalog model から `~/.xelyon/config.yaml` に貼れる YAML 断片だけを出力します。`--smoke` / `--tool-smoke` / `--retention-smoke` は live API request を送るため、設定確認だけなら付けないでください。response ID や usage が返らない場合、または pricing catalog に該当モデルがない場合は warn になりますが、smoke 成功自体は失敗にしません。

```bash
xelyon doctor azure
xelyon doctor azure --deployment my-gpt-5-deployment --catalog-model gpt-5.4 --print-config
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

手元で実 Azure 環境の回帰確認をまとめて走らせる場合は、`AZURE_OPENAI_BASE_URL`、`AZURE_OPENAI_DEPLOYMENT`、`AZURE_OPENAI_CATALOG_MODEL`、認証情報を設定して `make azure-smoke` を実行します。

Azure API error では 401/403/404/429 と tool payload rejected の原因候補を補足します。

### `xelyon doctor bedrock`

AWS Bedrock の region、AWS 認証チェーン、provider 登録、model / `catalog_model` 解決、Claude Messages / ConverseStream route、function calling 設定、token / pricing metadata を確認します。`--smoke` は text smoke、`--tool-smoke` は dummy tool call、`--image-smoke` は tiny PNG 画像入力、`--thinking-smoke` は Extended Thinking request を明示実行します。`--capabilities` は静的 / catalog ベースの capability snapshot を表示し、`--require-capability` は live request なしで shared capability 名を検証します。Bedrock の `function_calling` gate は選択 route に従い、Claude Messages route または streaming tool use 確認済みの ConverseStream model だけ pass します。`thinking` gate は Claude Messages route かつ通常 request config で thinking が有効な場合だけ pass します。`--print-request` を付けると live request を送らず、選択した smoke request の Bedrock operation、conceptual endpoint、redacted AWS SigV4 header、request body を `request_preview` に表示します。

Smoke の JSON では AWS SDK `ResultMetadata` 由来の `request_id`、request 単位の usage、概算 cost を `smoke.requests[]` に出します。summary usage / cost は request 単位の観測値を合算します。request ID、usage、pricing が返らない場合は warn ですが、API smoke 自体が成功していれば fail にはしません。複数 request のうち 1 件でも usage が返らない場合、summary cost は部分値を確定値として表示せず usage unavailable とします。Bedrock では Azure の `response_id` alias は出しません。`BEDROCK_FUNCTION_CALLING=0` の場合、`--tool-smoke` は function calling 無効として warn skip します。ConverseStream route で `--image-smoke` / `--thinking-smoke` を指定した場合は、未対応 request shape として warn skip します。

`--print-request` は AWS credential retrieval と live API request を行わず、`--print-request` 単体では text request preview を 1 件出します。Claude family は `InvokeModelWithResponseStream`、非 Claude は `ConverseStream` の request builder を使います。ConverseStream route の image / thinking preview は smoke と同じ unsupported request shape として skipped entry に残します。

```bash
xelyon doctor bedrock
xelyon doctor bedrock --model global.anthropic.claude-sonnet-4-6
xelyon doctor bedrock --model corp-bedrock-sonnet --catalog-model global.anthropic.claude-sonnet-4-6
xelyon doctor bedrock --smoke
xelyon doctor bedrock --tool-smoke
xelyon doctor bedrock --image-smoke
xelyon doctor bedrock --thinking-smoke
xelyon doctor bedrock --capabilities
xelyon doctor bedrock --require-capability image_input --require-capability thinking
xelyon doctor bedrock --print-request
xelyon doctor bedrock --tool-smoke --print-request
xelyon doctor bedrock --json
```

手元で doctor 経路だけを実 AWS 環境で確認する場合は、AWS 認証チェーンを設定して `make bedrock-doctor-smoke` を実行します。runtime supported モデル全体の継続確認には `make bedrock-smoke` / `make bedrock-smoke-matrix` を使います。

## 対話型コマンド

セッション中に `/` で始まるコマンドを入力できます。
TUI が primary surface です。`--no-tui` の classic REPL は deprecated legacy fallback として残しており、新しい対話型 UI コマンドは TUI 側だけに追加します。
TUI では入力欄で `/` または `/r` のような prefix を入力すると command 候補が表示され、Enter で選択中の command を実行し、Tab で入力欄へ補完できます。

設定系コマンドの責務は分かれています。`/config` は global config (`~/.xelyon/config.yaml`) の編集、`/project` は project config (`xelyon.yaml`) の編集、`/init` は repo-local guidance (`AGENTS.md`) の雛形作成だけを担当します。TUI で `xelyon.yaml` を管理する通常導線は `/project` です。

### `/help`

利用可能なコマンド一覧を表示します。TUI では `/` 候補が primary の command discovery です。

```
> /help
```

### `/status`, `/stats`

現在状態、直近リクエスト、セッション統計をまとめて表示します。`/stats` は互換エイリアスです。
表示には現在の command surface も含まれます。TUI では `TUI primary`、`--no-tui` では `classic legacy fallback` として表示し、classic 側では新しい対話型 UI コマンドを追加しない方針を明示します。
サブエージェントを使ったセッションでは、親とサブのコストを分離表示し、`🤖 Sub-agents` セクションで各サブのトークン使用量・コスト・ツール実行回数も確認できます。
Kimi `$web_search` を使った直近リクエストでは、`WebSearchCalls > 0` の場合だけ、token usage とは別に Web Search Calls、Web Search Fee、観測できた Search Result Tokens が表示されます。
Gemini では `Service Tier` 行に configured tier、request body tier、pricing family を表示し、直近 usage に実課金 tier がある場合は billing tier と billing pricing family も表示します。

```
> /status
> /stats   # alias
```

### `/new`

現在の session を保存して、新しい session を開始します。TUI の表示 transcript は残すため、画面上で直前の流れを見ながら文脈だけ切り替えたい場合に使います。

```
> /new
```

### `/clear`

現在の session を保存して、新しい session を開始します。TUI では表示 transcript、viewport、tool block、agent activity も消します。

```
> /clear
```

### `/resume`

保存済み session を再開します。引数なしでは現在の作業ディレクトリの session picker を開きます。`--all` は他の作業ディレクトリの session も候補に含め、`--last` は現在の作業ディレクトリの最新 session を直接再開します。保存済み provider/model がある session は、global config を保存せずに runtime だけその provider/model へ切り替えます。

```
> /resume
> /resume --all
> /resume --last
> /resume <session-id>
```

### Legacy session commands

`/save`、`/load`、`/sessions` は互換用に残っていますが、TUI の候補と通常 help には表示しません。新しい導線では `/new`、`/clear`、`/resume` を使ってください。

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
> /model gpt-5.5
> /model gpt-5.5-pro
> /model gpt-5.3-codex
> /model claude-sonnet-4-5-20250514
> /model gemini-3.5-flash
```

`gpt-5.5` は OpenAI Responses API の streaming 経路、`gpt-5.5-pro` は streaming unsupported のため non-streaming 経路を使用します。GPT-5.5 Pro は応答に数分かかる場合があります。

### `/copy`

最後のAI出力をクリップボードにコピーします。

```
> /copy
```

### `/attach`

TUI で現在の入力ドラフトにファイルまたは画像を 1 件添付します。パスに空白がある場合は引用符で囲みます。添付は `/attach`・ドラッグ&ドロップ・`Ctrl+V` 画像を合算して 1 ドラフト最大 12 件です。

```
> /attach ./notes.txt
> /attach "screenshots/error shot.png"
```

### `/detach`

添付を番号指定で 1 件外します。番号は入力欄上部の添付行に表示される `#<n>` です。

```
> /detach 2
> /detach #2
```

### `/detach-all`

現在の入力ドラフトにある添付をすべて外します。

```
> /detach-all
```

### `/review`

TUIモードで、現在の作業ツリー差分全体をレビューする preset 画面を開きます。
通常は現在の provider/model を使いますが、`review.provider` と `review.model` を設定すると `/review` だけ別の provider/model で実行できます。`review.provider` だけを設定した場合はその provider の既定モデルを使い、`review.model` を設定する場合は `review.provider` も必須です。review モデル呼び出しには通常会話履歴を渡しません。
`review.thinking.mode` は `/review` の thinking 方針です。既定の `inherit` は現在の `/thinking` 状態を引き継ぎます。`off` は `/review` だけ thinking を無効化し、`on` は `/review` だけ thinking を有効化します。`review.thinking.level` は `low` / `medium` / `high` / `xhigh` を指定でき、空の場合は現在の `/thinking` level を使います。
preset の `Review current changes` は追加指示なしの通常レビューです。`Review current changes with custom focus` は同じ current changes 全体に追加の観点・重点項目を渡します。
引数を付けた場合は、そのテキストを custom focus として current changes 全体レビューを即時実行します。
custom focus は対象ファイルや差分範囲を絞るものではありません。特定 finding だけの再検証や focused verification mode はまだ未実装です。
`/review` は `evidence -> probe plan -> probe results -> report -> saturation` の段階を持つ監査可能な review harness です。Go-first の evidence augmentation はありますが、共通 harness は全言語で雑な clean 判定を抑制します。
`XELYON_REVIEW_RUN_ARTIFACTS=1` を設定すると、各段階の debug artifact を実行中はメモリに保持し、終了後に `.xelyon/review-runs/<UTC timestamp>/` へ保存します。保存先 component が symlink の場合は repo 外へ書かず warning にします。artifact には evidence や probe output が含まれ得るため、必要な場合だけ明示的に有効化してください。
`review.web_search_evidence.enabled` を有効化した場合、web search evidence は初期 evidence 収集時の検索と、Pass1 probe plan 後に candidate risk / impact surface から組み立てる追加検索で構成されます。`review.web_search_evidence.max_queries` は両方の合計上限です。弱い入力や generic な risk text だけでは検索を抑制し、query reason に出る intent / expected source type / confidence は検索計画の metadata であって公式判定ではありません。raw web search results は discovery-only で、URL fetch 済みの `external_doc` snippet だけが citation-capable evidence になります。ただし `external_doc` は自動的に公式仕様とは扱わず、Evidence Markdown の `external_support` summary が外部根拠品質の source of truth です。confirmed official spec として扱えるのは、`external_support.official_confirmation=true` で、かつ引用した snippet 内容が明確に支える場合だけです。`source_credibility=official_candidate` は公式候補であり、それ単独では公式確認になりません。`external_support.level` が `none` / `weak` / `partial`、`source_credibility` が `unknown` / `third_party`、fetch failed、truncated、inconclusive の場合は unverified / residual / blocked として分類します。

```
> /review
> /review focus on regressions
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

### `/ledger`

現在の runtime task ledger を表示します。引数なし専用です。TUI と `--no-tui` の classic REPL で利用できます。表示は確認用のコマンド出力だけで、会話履歴、保存済みセッション、prompt、圧縮、provider request には追加されません。

```
> /ledger
```

**表示内容:**
- Changed files
- Touched files
- Evidence
- Recommended reads
- Last failed tests
- Last passed tests

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

### `/provider <provider> [model]`, `/use <provider> [model]`

プロバイダーとモデルを動的に切り替えます。TUI で引数なしの `/provider` を実行すると provider picker を開きます。`/use` は legacy alias です。

切り替え後もローカルの会話履歴と session context は保持されます。OpenAI / Azure OpenAI Responses API の `previous_response_id` など provider 側の continuation state は、新しい provider/model と混ざらないよう切り替え時にリセットされます。文脈自体を切りたい場合は `/new`、画面も消す場合は `/clear` を使います。

```
> /provider
> /provider deepseek
> /provider openai gpt-5.4
> /provider kimi kimi-k2.6
> /provider gemini gemini-3.5-flash
> /provider claude claude-sonnet-4-5-20250514
> /provider ollama qwen2.5-coder:7b
> /provider groq meta-llama/llama-4-scout-17b-16e-instruct
> /use openai gpt-5.4   # legacy alias
```

### `/providers`

プロバイダーの credential status をテキストで表示します。TUI でプロバイダーやモデルを選ぶ場合は `/provider` picker が primary です。

```
> /providers
```

### `/config`

global config (`~/.xelyon/config.yaml`) を確認・変更します。TUI の対話式メニューで設定項目をカテゴリ別に管理できます。project 固有の `xelyon.yaml` は `/project` で編集します。

```
> /config               # 対話式設定メニューを起動
> /config show          # 全設定を表示（デフォルトとの差分を ⚡ で表示）
> /config model <name>  # デフォルトモデルを変更
```

**対話式メニュー:**

`/config` を引数なしで実行すると、以下のような対話式メニューが表示されます：

```
┌─ Configuration ───────────────────────────┐
│ [1] 🤖 Provider & Model                   │
│ [2] 🛡️ Execution Mode                     │
│ [3] 📦 Compression                        │
│ [4] 🗺️ Project Map                        │
│ [5] 🔧 LSP Servers                        │
│ [6] 🔍 Web Search                         │
│ [7] 🧪 Final Checks                       │
│                                           │
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
| []string | 項目追加/削除 | `execution.safe_shell_commands` |
| map[string]struct | サブメニューで編集 | `provider_models`, `lsp.servers` |

**主なカテゴリ:** Provider & Model, Execution Mode, Compression, Paste Mode, Project Map, Agent Instructions, Agent Skills, LSP Servers, Output, Web Search, Sub-agent, MCP Servers, Final Checks など

**変更は即座に保存:** `~/.xelyon/config.yaml` に自動保存されます。

### `/skills`

Agent Skills を選択・確認・診断します。skill catalog は project `.agents/skills`、home `~/.agents/skills`、XELYON 内蔵 skills を読みます。Codex / Claude など他 runtime の system skills は読みません。

`/skills overview` は検出済み skill catalog の概要を会話ログに出力し、`/skills show <name>` は対象の `SKILL.md` 本文と resource 一覧を表示します。互換性のため `/skills list` も overview の alias として受け付けます。

Skill Router は通常の依頼時に bounded hint を model へ渡します。v1 の既定は hint-only で、full `SKILL.md` body は自動読み込みしません。必要な場合だけ model が `activate_skill(name)` を呼びます。

`agents/xelyon.yaml` がある skill は routing metadata として使われます。既存の `SKILL.md` frontmatter は `name` / `description` だけで valid のままです。plain `/skills doctor` は legacy skill に sidecar がないだけでは警告しません。routing metadata や sidecar completeness を見る場合は `/skills doctor --routing` を使います。

`/skills suggest <text>` は debug / authoring 用に Skill Router の full ranked list を表示します。通常ユーザーが毎回使う導線ではありません。v1 は human-readable output のみで、`--json` schema は公開しません。

`/skills usage` は local-only routing usage ledger の要約を表示します。ledger は raw prompt、raw response、diff、file content、secret を保存しません。保存先は `~/.xelyon/skills/router/usage/` 配下で、repo key は project root の hash です。`/skills usage clear` で current repo、`/skills usage clear --all` で全 skill router usage ledger を削除できます。v1 は human-readable output のみです。

```
> /skills overview
> /skills show imagegen
> /skills doctor
> /skills doctor --routing
> /skills suggest "review provider runtime changes"
> /skills usage
> /skills usage clear
> /skills usage clear --all
```

### `/init`

repo-local guidance (`AGENTS.md`) の雛形を作成します。既存 `AGENTS.md` がある場合は上書きしません。

```
> /init
```

`CLAUDE.md` / `.claude/CLAUDE.md` が既にある repo でもコピーや上書きは行いません。必要な場合は `/config` の Agent Instructions で追加選択できます。

XELYON 固有の repo config (`xelyon.yaml`) は `/project` が通常導線です。

### `/project`

TUI で project config (`xelyon.yaml`) の編集画面を開きます。global config (`~/.xelyon/config.yaml`) は `/config` で編集します。

```
> /project
```

**TUI で編集できる項目:**
- `context`（legacy 互換 field。通常の system prompt には注入されません）
- `rules`（legacy 互換 field。新規 guidance は `AGENTS.md` に書きます）
- `ignore.patterns`
- `final_checks.commands`
- `final_checks.timeout`（`final_checks.commands` がある場合）

`conditional` は現時点では preview のみです。`xelyon.yaml` がない場合は、画面内でテンプレートを作成できます。

## TUI 添付の補足

- ドラッグ&ドロップでファイルパスを貼り付けると自動添付されます（最大 12 件）。
- `Ctrl+V` は通常テキスト貼り付けですが、クリップボードテキストが空の場合は画像貼り付けを試みます（Windows/WSL）。
- 送信時、画像はマルチモーダル入力として送信し、ファイルは `Attached context` として本文へ展開されます。
- PDF 添付は `Attached context` へテキスト抽出して展開されます（先頭 20 ページ / 30000 文字まで）。
- PDF の読み取りに失敗した場合、または抽出可能テキストがない場合は、その旨を `Attached context` に明示します。
- クリップボード画像の一時ファイルは、送信・detach・終了時に削除されます。異常終了で残った古い一時ディレクトリは起動時に自動GCされます（24時間超）。

### `/plan`

Plan Modeを切り替えます。有効にすると、リクエストが「調査→計画→承認→実装」のフローで処理されます。

```
> /plan           # 現在のモード表示
> /plan on        # Plan Mode 有効化
> /plan off       # 通常モードに戻る
> /plan toggle    # Plan Mode を切り替え
> /plan status    # ステータス表示
```

TUI のステータスバーには現在値が `Plan: OFF` / `Plan: ON` と表示され、通常フッターの `/plan` ヒントから状態確認と切替コマンドを確認できます。

**デフォルト:** OFF（通常モード）

**通常モード（OFF）:**
- `execution.mode` に従ってツールを自動実行または確認しながら実行
- 軽いタスクにはオーバーヘッドなく即座に応答

**Plan Mode（ON）:**
1. **調査フェーズ**: SafetyHighツール（read_file, list_dir, search_code等）を自由に実行
2. **計画生成**: 実装が必要な場合、調査結果・根拠・制約と実装ステップをJSONで出力
3. **承認と実装**: ユーザーが計画を確認・承認すると、承認済み計画の要約情報を通常モードへ引き継いで実装を開始

調査フェーズの履歴は通常実装へ持ち越さず、承認済み計画に含まれる調査結果・根拠・制約・ステップだけを通常モードの実装リクエストへ渡します。

**ステータス表示:**
```
[Status] waiting_input | Mode: Plan: OFF | Ready / 入力待ち
[Status] waiting_input | Mode: Plan: ON | Ready / 入力待ち
```

### `/thinking`

Extended Thinking（推論モード）を切り替えます。複雑なタスクでより深い推論を行う際に使用します。
TUI では `/thinking` を選ぶと `on/off/low/medium/high/xhigh` の候補を表示します。

```
> /thinking on           # 有効化（現在のレベルで）
> /thinking off          # 無効化
> /thinking low          # 低レベルで有効化
> /thinking medium       # 中レベルで有効化（デフォルト）
> /thinking high         # 高レベルで有効化
> /thinking xhigh        # 最高レベルで有効化（max）
```

**対応プロバイダー:**

| プロバイダー | 対応 | 動作 |
|-------------|------|------|
| Claude | ✅ | Opus 4.8 / Opus 4.7 / Opus 4.6 / Sonnet 4.6 / Fable 5: adaptive thinking + effort / 4.5以前: budget_tokens |
| OpenAI | ✅ | reasoning_effort パラメータ |
| Gemini | ✅ | thinkingConfig.thinkingBudget |
| DeepSeek | ✅ | V4 `thinking` field + `reasoning_effort` |
| Groq | ❌ | 警告表示（非対応） |
| Ollama | ⚠️ | モデル依存（R1/QwQ推奨） |

**対応モデル:**
- **Claude**: Sonnet 4 以降（Opus 4.8 / Opus 4.7 / Opus 4.6 / Sonnet 4.6 / Fable 5 は adaptive thinking。`/thinking xhigh` は Opus 4.8 / Opus 4.7 / Fable 5 で `xhigh`、Opus 4.6 で `max`、Sonnet 4.6 で `high` effort）
- **OpenAI**: gpt-5.2 系
- **Gemini**: 2.5 Pro 系（Flash は非対応）
- **DeepSeek**: モデル名は維持し、`/thinking off` は `thinking.disabled`、`/thinking on` は `thinking.enabled` + `reasoning_effort` を送ります。`/thinking xhigh` は DeepSeek では `max` に変換されます。`reasoning_content` は💭で表示し、ツール実行フローでも保持します。

**注意**: Extended Thinking はトークン消費量が増加します。

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
xelyon -m gemini-3.5-flash
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
xelyon --image receipt.png --provider kimi "この画像の内容を要約して"
```

**対応フォーマット**: PNG, JPEG, GIF, WebP
**対応プロバイダー**: Kimi, Gemini, Claude, OpenAI, Azure OpenAI（DeepSeek, Ollama, Groqは非対応）
**制限**: `--image` は `--resume` と併用できません。`--headless` / `--output-format json` では対応プロバイダーの場合に使用でき、JSON には bounded image metadata だけを出します。

### その他のオプション

```bash
# バージョン表示
xelyon --version

# ヘルプ表示
xelyon --help

# 1ターンだけ実行して終了（位置引数だけでも one-shot になります）
xelyon --once "main.goを読んで説明して"
xelyon "main.goを読んで説明して"

# query 引数があっても対話 TUI を強制
xelyon --interactive "この続きから相談したい"

# one-shot のヘッダー/ステータス表示を抑制
xelyon --quiet "短く要約して"

# セッション再開（picker / 最新 / 全作業ディレクトリ / ID指定）
xelyon resume
xelyon resume --last
xelyon resume --all
xelyon resume <session-id>

# 互換 alias: 最後のセッションのみ
xelyon --resume

# ツール確認を自動承認
xelyon --auto-approve "テストを直して"

# Headlessモード（JSON出力、対話なし）
xelyon --headless "main.goを読んで説明して"
xelyon --output-format json "バグを修正して"

# バージョンチェックを無効化
xelyon --no-update-check

# legacy classic REPL を使う（deprecated / 互換確認用）
xelyon --no-tui
```

`xelyon resume` は TUI の session picker を開きます。既定では現在の作業ディレクトリに紐づく session と、古い形式で作業ディレクトリ情報を持たない session だけを表示します。`--all` は全作業ディレクトリの session を表示し、`--last` は picker を開かず最新 session を再開します。`--resume` は互換 alias として `xelyon resume --last` 相当です。query 引数や `--image` とは併用できません。`--once` とも併用できません。

`--no-tui` は古い classic REPL との互換確認用です。通常の対話利用では TUI を使ってください。classic REPL には新しい対話型機能や表示更新を追加しません。

`--quiet` は one-shot 実行専用です。`--interactive` や通常の対話セッションでは使用できません。

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

編集系ツールは provider/model に応じて自動で切り替わります。OpenAI / Azure OpenAI / Gemini 系では `apply_patch` を公開し、Kimi / Claude / DeepSeek / Groq / Ollama / Bedrock / unknown 系では legacy `str_replace` / `write_file` / `delete_file` を公開します。OpenRouter は model family を見て、`openai/...` / `google/...` / `gemini/...` だけを `apply_patch` にします。
開発デバッグ用に `XELYON_EDIT_TOOL=str_replace xelyon` または `XELYON_EDIT_TOOL=apply_patch xelyon` で明示 override できます。

### ファイル操作

| ツール名 | 説明 | 主な引数 |
|---------|------|---------|
| `apply_patch` | apply_patch mode 用。Codex 互換の差分パッチでファイルを作成・更新・削除する。1回で複数ファイルを扱える | `patch` |
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
| `gather_context` | 既定の調査入口。symbol-like query では structured impact route を試し、`search_code(intent=impact)` が返す `RecommendedReads` を compact evidence として prefetch する。diagnostics の `resolved_by` / `confidence` / `truncated` / `budget_limit_hit` に応じて prefetch 件数を絞り、ambiguous の場合は speculative prefetch しない | `query`, `path`, `file_filter` |
| `search_code` | 低レベルの expert 検索ツール。通常は `gather_context` を優先し、明示的に search route を制御したい場合だけ使う。`mode=auto` は symbol-aware / literal / regex を language-aware に routing する。`intent=impact` は shared-change impact analysis の入口で、Go、TypeScript `.ts` / `.d.ts`、対象を絞った TSX `.tsx`、JavaScript `.js` / JSX `.jsx` で構造化 impact を優先する。`file_filter=typescript` / `javascript` は broad fallback scope で、targeted structured impact には `ts` / `tsx` / `js` / `jsx` や direct path / glob を使う。結果の diagnostics summary で `resolved_by`、`confidence`、fallback / truncation / budget 状態を確認できる | `pattern`, `intent`, `mode`, `path`, `file_filter` 等 |
| `web_search` | ネイティブWeb検索（`web_search.provider` で Kimi / OpenAI / OpenAI Subscription / Gemini / Claude を選択可能） | `query` |
**注意**: メインプロバイダーがネイティブ検索非対応（DeepSeek / Groq / Ollama / OpenRouter / Bedrock など）の場合は、`config.yaml` で `web_search.provider` を設定してください。メインプロバイダーが Kimi の場合は provider 指定なしで Moonshot built-in `$web_search` を使います。OpenAI Subscription は `xelyon auth openai-subscription login` の OAuth credential を使い、`OPENAI_API_KEY` には fallback しません。K2.7 Code 選択中の Kimi built-in `$web_search` は検索 request だけ `kimi-k2.6` に fallback し、usage / cost / 実行ログも `kimi-k2.6` として扱います。Kimi で `$web_search` が起動すると call fee が発生し、XELYON は token usage と別枠で観測します。詳細は[config.md - Web検索](config.md#web検索)を参照してください。

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
structured impact と diagnostics-aware prefetch の詳細は [Search optimization and structured impact](search.md)、LSP の設定詳細は [LSP連携ガイド](lsp.md) を参照してください。

### 使用例

AIは自然言語の指示に基づいてツールを自動選択します。

```bash
> main.goを読んで
# → read_file が実行される

> バグを修正して
# → gather_context → active edit mode の編集ツールが実行される

> 複数ファイルをまとめて編集して
# → apply_patch mode では apply_patch、legacy edit mode ではファイル単位の編集ツールが実行される

> git statusを見せて
# → bash で git status が実行される

> テストを実行して
# → bash で go test が実行される

> TODOを探して
# → gather_context または search_code が実行される
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

### `/mcp`

現在の対話セッションに読み込まれている MCP runtime 状態を表示します。
`/mcp` と `/mcp status` は同じ表示です。
このコマンドは snapshot-only で、MCP server process の起動、再接続、`tools/list`、`tools/call` は行いません。

```bash
> /mcp
> /mcp status
```

表示するのは runtime 有効状態、読み込み済み config の有無、server 数、接続済み server、disabled / blocked / not connected server、registered / visible / omitted tool 数、effective budget、tool surface のサンプル、server 別 token / schema byte 数、omitted reason、`tools.include` / `tools.exclude` と `mcp.surface_budget` の提案です。
env value、raw args、server error detail、tool schema body、description 全文は表示しません。
設定ファイルそのものや live 接続を確認する場合は `xelyon doctor mcp`、実際に initialize / `tools/list` まで確認する場合は `xelyon doctor mcp --connect` を使います。

## 高度な機能

### Headlessモード

対話なしで machine-readable JSON だけを stdout に出力します。`--headless` と `--output-format json` はどちらも headless mode に解決されます。CI でそのまま使う例は [Headless CI guide](ci.md) を参照してください。

#### 利用例

```bash
# 基本実行
xelyon --headless "main.goを読んで概要を説明して"
xelyon --output-format json "バグを修正して"

# prompt file から入力
xelyon --headless --prompt-file prompt.md

# stdin から入力
cat prompt.md | xelyon --headless -
cat prompt.md | xelyon --headless --prompt-file -

# 画像入力を含める
xelyon --headless --image screenshot.png "このスクリーンショットを確認して"

# jq と組み合わせる
xelyon --headless "バグを修正して" | jq -r '.response'

# CI 推奨 command
xelyon --headless --prompt-file prompt.md --exit-code-policy ci --fail-on-tool-error --read-only

# workspace mutation を禁止する review-only 実行
xelyon --headless --prompt-file review-prompt.md --exit-code-policy ci --fail-on-tool-error --read-only
```

#### JSON 出力例

```json
{
  "schema_version": "xelyon.headless.v1",
  "status": "success",
  "provider": "gemini",
  "model": "gemini-3.5-flash",
  "response": "このファイルは...",
  "input": {
    "source": "args",
    "bytes": 42,
    "image": {
      "path": "screenshot.png",
      "mime_type": "image/png",
      "bytes": 12345,
      "provider_supported": true
    }
  },
  "summary": {
    "changed_files": ["internal/example.go"],
    "commands": [
      {
        "command": "go test ./internal/agent -run TestHeadless -count=1",
        "exit_code": 0,
        "status": "passed",
        "source": "tool"
      }
    ],
    "final_checks": [
      {
        "command": "make verify-fast",
        "exit_code": 0,
        "status": "passed"
      }
    ]
  },
  "exit_policy": "legacy",
  "recommended_exit_code": 0,
  "duration_ms": 1234,
  "timestamp": "2026-05-25T12:00:00+09:00",
  "cost": 0.00012
}
```

#### JSON / exit policy contract

`schema_version` は `xelyon.headless.v1` です。prompt 本文は JSON に出しません。`input.source` は `args`、`prompt_file`、`stdin` のいずれかで、`--prompt-file prompt.md` の場合は `input.prompt_file` に指定 path が入ります。prompt file と stdin は 1 MiB まで読み込み、空入力、directory、存在しない file、query 引数との併用は `status:"error"` / `error.type:"config_error"` / `failure_reason:"usage_error"` になります。`--image` は headless / JSON mode でも使えます。JSON には `input.image.path`、`input.image.mime_type`、`input.image.bytes`、`input.image.provider_supported` の bounded metadata だけを出し、raw image bytes や base64 body は出しません。画像非対応 provider では `failure_reason:"unsupported_capability"`、画像 file が読めない場合は `failure_reason:"usage_error"` になります。

`error.type` は既存互換の分類で、CI 向け分類は `failure_reason` に出ます。成功時は `failure_reason` を省略し、`recommended_exit_code` は `0` です。既定の `exit_policy` は `legacy` で、error は従来どおり process exit code `1` です。CI で詳細 code が必要な場合は `--exit-code-policy ci` を指定します。

| `failure_reason` | `--exit-code-policy ci` の `recommended_exit_code` |
| --- | --- |
| `usage_error` | `2` |
| `config_error` | `3` |
| `provider_setup_required` | `3` |
| `tool_error` | `4` |
| `final_check_failed` | `5` |
| `api_error` | `6` |
| `cancelled` | `7` |
| `read_only_violation` | `8` |
| `unsupported_capability` | `9` |
| `tool_loop_limit` | `1` |
| `unknown_error` | `1` |

既定では failed tool call が `tool_calls[].success=false` に残っていても、最終応答があれば全体は success のままです。CI で failed tool call も失敗扱いにする場合は `--fail-on-tool-error` を指定します。

`summary` は runtime observation がある場合だけ出ます。`summary.changed_files` は tool の `FileChange` を task ledger で repo-relative に正規化したものです。`summary.commands` は bash tool で実行したコマンドの `command`、`exit_code`、`status`、`source:"tool"` を出します。`summary.final_checks` は変更ファイルがある headless 実行で `final_checks.commands` が設定されている場合に実行され、`command`、`exit_code`、`status` を出します。コマンド出力本文は JSON summary には入れません。final check が失敗した場合は `status:"error"`、`error.type:"final_check_failed"`、`failure_reason:"final_check_failed"` になります。

Gemini / OpenAI Subscription native web search の token usage は、provider が usage を返した場合に通常の token usage として `tokens` / `cost` に含まれます。OpenAI Subscription の cost は ChatGPT subscription 扱いのため `pricing_unavailable` のままです。Kimi `$web_search` を使った場合、既存の `cost` は token cost + web search call fee の合計を維持し、`web_search` object に `calls`、`fee_estimate`、`result_tokens` を分けて出します。検索結果 tokens は次 request の `prompt_tokens` に含まれる前提の表示用観測値で、headless JSON の token totals には再加算しません。

#### Read-only / dry-run contract

`--read-only` は headless 実行で workspace mutation を禁止します。`--dry-run` は v1 では `--read-only` と同じ no-write mode です。どちらも `--headless` または `--output-format json` 専用で、それ以外では `usage_error` になります。

read-only mode は write tool、bash tool、`run_skill_script`、MCP tool、`spawn_agent` / `wait_agent` を非表示または拒否します。拒否された tool call は実ツールを実行せず、`tool_calls[]` に `success:false` と `Error:` output を残します。拒否された bash は `summary.commands` に `status:"failed"` / `exit_code:-1` / `source:"tool"` として残ります。`--fail-on-tool-error` なしでは、拒否 tool call があっても最終応答があれば全体 status は success のままです。`--fail-on-tool-error` ありでは `status:"error"`、`error.type:"read_only_violation"`、`failure_reason:"read_only_violation"` に昇格し、denied call 後に tool loop limit へ到達した場合も `read_only_violation` を保持します。

read-only headless run は runner 到達前の config bootstrap から no-write になり、first-run HOME でも `~/.xelyon/config.yaml` / `~/.xelyon/AGENTS.md` を作成しません。session history / change history / audit log storage、persistent tool cache、startup artifact cleanup、startup ProjectMap build / prompt injection / `~/.xelyon/cache/projectmap` persistence、LSP client startup / warmup も実行しません。provider-history raw output artifact projection は候補/report を残しますが、artifact materialization と artifact-backed provider-facing replacement は行いません。skill-router runtime hint は `git status --porcelain` signal を使わず、recommendation / activation usage ledger も保存しません。

`mcp.headless: true` の環境でも read-only mode では MCP server に接続せず、`mcp_*` tool call も拒否します。XML 形式の `<mcp_...>` attempt も、Markdown code block 外にあり、対応する閉じタグがある場合だけ denied tool call として `tool_calls[]` に記録します。MCP tool の read/write capability metadata は v1 では未分類のため、read-only mode では fail-closed にします。

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
- [Headless CI guide](ci.md)
- [Search optimization and structured impact](search.md)
- [LSP連携](lsp.md)
- [MCP連携](mcp.md)

### `/rawoutputs`

raw output artifact store の read-only diagnostics を表示します。provider history reduction の `raw_output_artifacts` が作成した refs / artifacts / verify 状態 / GC 見込みを、raw body を表示せずに確認するためのコマンドです。

```
> /rawoutputs [summary|verify|refs|gc --dry-run]
```

- `/rawoutputs` または `/rawoutputs summary`: 有効 mode、artifact root、root source、session、refs/artifacts/bytes、live ref source 数を表示します。
- `/rawoutputs verify`: manifest と object を読み、missing object、hash mismatch、decrypt failure、path failure、quarantine/tombstone/collected refs の件数を表示します。
- `/rawoutputs refs`: 最大 20 件の ref metadata を表示します。raw output body は表示しません。
- `/rawoutputs gc --dry-run`: 現在 runtime が把握している caller-provided live refs だけを source of truth として、tombstone / collect / keep の見込み件数を表示します。

`/rawoutputs gc --apply`、`delete`、`repair` は実装していません。このコマンドは状態確認専用で、store や history を変更しません。

## 未ドキュメント化コマンド（自動追加）

<!-- TODO: 以下のコマンドに詳細な説明を追加してください -->

### `/setup`

Show first-run setup checklist

```
> /setup
```
