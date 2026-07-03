# openai_subscription backed XELYON web_search adapter Master Plan

この文書は Codex Goal で `openai_subscription` backed XELYON `web_search` adapter を実装するための内部実装仕様書である。
公開 docs ではなく、実装前の設計・handoff の source of truth として使う。

この計画書作成時点では実装しない。
commit / push / PR 作成もこの計画書作成 task には含めない。

## 0. Purpose

XELYON の `web_search` tool backend として、ChatGPT/Codex subscription backend を使う `openai_subscription` provider-native search adapter を追加する。

完了条件:

- `openai_subscription` が XELYON の native web search provider として解決される。
- `websearch.RegisterWithContext("openai_subscription", ...)` で canonical provider key の adapter が登録される。
- `OPENAI_API_KEY` / OpenAI Platform API fallback を持たず、OAuth token と ChatGPT/Codex subscription endpoint だけで request する。
- `/review web_search_evidence` が既存 `tools/search.SearchWeb` 経路から `openai_subscription` を使える。
- 結果は XELYON `web_search` tool result として `Summary:` / `Sources:` 形式で返り、既存 `provider_history_reduction` / `raw_output_ref` の `xelyon_web_search_tool_result` 経路に乗る。
- `doctor openai-subscription` で capability / request preview / live web-search smoke を確認できる。
- docs / generated config metadata / focused tests / verification commands が揃う。

## 1. Confirmed Decisions

確定事項:

- 方針は XELYON `web_search` tool adapter 固定。
- 通常会話 request に provider-native `tools:[{"type":"web_search"}]` を常時混ぜない。
- `OPENAI_API_KEY` / OpenAI Platform API fallback は禁止。
- `openai_subscription` OAuth token と ChatGPT/Codex subscription endpoint だけ使う。
- `/review web_search_evidence` で使えるようにする。
- 検索結果は XELYON `web_search` tool result として返す。
- 履歴最適化は既存 `provider_history_reduction` / `raw_output_ref` に乗せる。
- provider-native `web_search_call` の lossless replay / compact は今回は対象外。
- doctor capability / smoke / preview を更新対象にする。
- request の第一候補は `tools:[{"type":"web_search"}]` と `tool_choice:"required"`。
- `web_search_preview` は使わない。

live probe 済みの前提:

```text
baseline text: HTTP 200
tools:[{"type":"web_search"}]: HTTP 200
tool_choice:"required" + web_search: HTTP 200
stream events include response.web_search_call.*
output item includes web_search_call and message
web_search_preview: HTTP 400
```

この live probe はこの計画書の入力事実として扱う。
実装時に再 probe する場合も、上の契約を弱めて Platform API fallback へ逃げてはいけない。

## 2. Non-goals

今回はやらない:

- 通常会話の `SubscriptionProvider.ChatWithTools` に `web_search` built-in tool を常時混ぜる。
- OpenAI Platform API provider の `internal/api/providers/openai/web_search.go` を fallback として使う。
- `OPENAI_API_KEY`、`OPENAI_RESPONSES_URL`、Platform `/v1/responses` へ fallback する。
- `web_search_preview` に切り替える。
- provider-native `web_search_call` を通常 conversation history の lossless replay item として保存する。
- provider-native `web_search_call` 専用の compact / raw artifact family を新設する。
- `previous_response_id` / `store=true` / `context_management` を subscription web search に導入する。
- `/review` 専用の search provider 設定や direct `curl` / `python` fetch 経路を追加する。
- raw web search result を review report の citation-capable evidence ref として扱う。
- sensitive token / OAuth credential / Authorization header を doctor JSON、debug、error、artifact、prompt に出す。

## 3. Current Source Findings

### 3.1 Search registry and execution

- `internal/api/websearch/registry.go` が provider-native search adapter registry の owner。
- `RegisterWithContext(providerName, fn)` は provider 名を lower/trim して登録する。
- `SearchWithContext(ctx, providerName, query, model)` は registry に登録済みの adapter を呼ぶ。
- usage callback は `websearch.WithUsageCallback` / `UsageCallbackFromContext` で request context に載る。

`internal/tools/search/web_provider.go`:

- `resolveSearchProvider` は `config.WebSearch.Provider`、main provider config key、main provider の順で native provider を探す。
- native 判定は `llmcatalog.ProviderDescriptorFor(provider).NativeWebSearch`。
- alias owner を保持する設計があり、Kimi `moonshot` / Claude `anthropic` の provider_models owner や cache owner を潰さない tests がある。

`internal/tools/search/web.go`:

- interactive tool は `ExecuteWebSearch`。
- noninteractive API は `SearchWeb`。
- `SearchWeb` は provider 解決、model 解決、cache、usage attribution、`ParseWebSearchResults` をまとめる。
- `/review` はこの `SearchWeb` に乗る。

`internal/tools/search/web_results.go`:

- search result extraction は provider output text から URL を拾う。
- 既存 provider は概ね `Summary:` と `Sources:` を返す。
- `Sources:` の `URL:` line は parser と `/review` external doc fetch への入力になる。

### 3.2 Existing OpenAI Platform web search adapter

`internal/api/providers/openai/web_search.go`:

- OpenAI Platform API key path の adapter。
- `init()` で `websearch.RegisterWithContext("openai", WebSearchWithContext)` する。
- request body は `tools:[{"type":"web_search"}]` と `include:["web_search_call.action.sources"]` を使う。
- non-streaming JSON response の `output` から `web_search_call.action.sources` と message annotations を集める。
- `formatWebSearchResult` は `Summary:` / `Sources:` 形式を作る。

この file は reference にはしてよいが、`openai_subscription` の transport fallback owner ではない。

### 3.3 openai_subscription runtime

`internal/api/providers/openai_subscription/subscription_provider.go`:

- provider registration は `api.RegisterProvider("openai_subscription", ...)`。
- runtime は `openai_responses.RunResponsesRequest` を使う Responses-shaped full payload path。
- `buildChatResponsesRequest` は `stream:true`、`store:false`、`prompt_cache_key` enabled、`previous_response_id` omitted、`context_management` omitted。
- normal chat tools は `subscriptionResponsesTools` / `subscriptionResponsesToolChoice` で function tools を送る。
- この normal chat path に `web_search` built-in tool を混ぜない。

`internal/api/providers/openai_subscription/subscription_transport.go`:

- OAuth credential を `GetSubscriptionCredentialForRequest` で取得する。
- `Authorization: Bearer <OAuth access token>`、`ChatGPT-Account-Id`、`originator: xelyon`、`User-Agent: xelyon/<version>` を設定する。
- `OPENAI_API_KEY` は読まない。

`internal/api/providers/openai_subscription/subscription_validation.go`:

- Platform `/v1/responses` endpoint を明示的に拒否する。
- web search adapter も同じ Platform endpoint 禁止境界を維持する。

`internal/api/providers/openai_subscription/subscription_diagnostics.go`:

- 現状 capability snapshot の `WebSearch` は false。
- doctor は local checks、capabilities、required capability、request preview、smoke を持つ。
- `--web-search-smoke` は現状 command contract で禁止されている。

### 3.4 openai_responses stream parser

`internal/api/providers/openai_responses`:

- `Request` は generic Responses-shaped request DTO。
- `Tool` は function tools 向けで、`Name` が `omitempty` ではない。
- `HandleStreaming` の dispatcher は `response.created`、`response.output_text.delta`、`response.output_item.*`、`response.function_call_arguments.*`、`response.completed` などを扱う。
- 現状 `response.web_search_call.*` の action はない。
- stream state は function_call replay item と assistant message replay item の owner であり、通常 chat continuation と密に結びつく。

今回の adapter では、通常 chat replay を変えず、web search adapter 用の stream parser を `openai_subscription` package 側に置く方針を第一候補にする。
`openai_responses` に共通 helper を export する場合は、normal chat replay へ影響しない pure SSE codec / usage conversion に限定する。

### 3.5 llmcatalog and config metadata

`internal/llmcatalog/provider.go`:

- `openai_subscription` descriptor は `NativeWebSearch:false` のまま。
- `nativeWebSearchProviderOrder` は `kimi`, `openai`, `gemini`, `claude`。
- `NativeWebSearchProviderKeys(true)` が `/config` select options と docs 生成の source of truth になる。

`internal/config/registry_generated.go`:

- `web_search.provider` は生成物。
- 現状選択肢は `kimi`, `moonshot`, `openai`, `gemini`, `claude`, `anthropic`。
- 実装時は source metadata を更新し、`make gen-all` で generated file を更新する。

### 3.6 /review web_search_evidence

`internal/reviewadapter/runner_factory.go`:

- `/review` の `reviewWebSearchRunner.SearchReviewWeb` は `searchtool.SearchWeb` を呼ぶ。
- provider / model / usage attribution は `RunnerFactoryOptions` から渡る。
- つまり `openai_subscription` が `SearchWeb` で native provider として解決できれば、`/review web_search_evidence` は既存 caller path で使える。

`internal/review/evidence/evidence_web_search.go`:

- raw search results は `WebSearchEvidenceQuery.Results` として discovery-only に残る。
- URL ごとに `externaldoc.Fetcher` が HTTPS external doc snippet を取りに行く。
- `ExternalDocs` の fetched snippet が citation-capable evidence の本体。

`internal/review/report/external_doc_validation_test.go`:

- raw `kind:"web_search"` evidence ref は reject される。
- review report で引用できるのは `kind:"external_doc"` の bounded snippet。

### 3.7 Provider history reduction / raw output refs

`internal/providerhistory/web_search_raw_output.go`:

- XELYON `web_search` tool result は `SurfaceXelyonWebSearchToolResult` として raw artifact candidate になる。
- placeholder は `raw_output_ref`、`surface=xelyon_web_search_tool_result`、query hash、selected source URL summary を持つ。
- source credibility は compact summary だけでは上げない。

`internal/rawoutputs/types.go`:

- `SurfaceXelyonWebSearchToolResult` が既に存在する。
- provider-native built-in replay 用の surface も存在するが、今回は使わない。

## 4. Global Contracts

すべての phase で守る契約:

- `openai_subscription` web search は XELYON `web_search` tool adapter であり、normal chat request modifier ではない。
- OAuth-only transport。`OPENAI_API_KEY`、OpenAI Platform endpoint、Platform compact/search API へ fallback しない。
- request identity は `originator: xelyon` と XELYON User-Agent のまま。OpenCode / official Codex identity を名乗らない。
- `store:false`。`store:true` は送らない。
- `previous_response_id` と `context_management` は送らない。
- `max_output_tokens` は subscription endpoint policy と同じく送らない。
- `web_search_preview` は送らない。
- request shape は first candidate として `tools:[{"type":"web_search"}]`、`tool_choice:"required"` を使う。
- `tool_choice:"required"` は request shape の契約であり、runtime 成功判定を `web_search_call` event の有無だけに固定しない。
- 初期実装では `include:["web_search_call.action.sources"]` を送らない。別 live probe で subscription streaming OAuth request が受理すると確認できた場合だけ、後続 tranche で追加を検討する。
- `openai_subscription` family の web search adapter key は registry/cache/usage owner に到達する前に canonical `openai_subscription` へ揃える。
- provider-native `web_search_call` は adapter 内の parse / diagnostics に使うだけで、conversation replay source of truth にしない。
- XELYON `web_search` result text は既存 provider と同じ `Summary:` / `Sources:` contract を守る。
- `/review` raw search results は discovery-only。citation-capable evidence は fetched `external_doc` snippet のみ。
- history optimization は既存 XELYON `web_search` tool result raw output artifact path に任せる。
- token / usage が返る場合だけ `api.UsageCallback` に流す。ChatGPT subscription の API cost は `N/A` / pricing unavailable のまま。
- debug / doctor / error では OAuth token、refresh token、id token、Authorization header、raw account ID を出さない。

## 5. Responsibility Boundaries

```text
adapter owner:
  internal/api/providers/openai_subscription
  WebSearchWithContext, request builder, OAuth-only request, SSE parser, Summary/Sources formatter

registry owner:
  internal/api/websearch
  RegisterWithContext / SearchWithContext

provider resolution owner:
  internal/tools/search + internal/llmcatalog
  NativeWebSearch descriptor, provider order, subscription-family search adapter canonicalization, alias owner/canonical key behavior, model resolution, cache/usage attribution

sub-agent / headless caller owner:
  internal/tools/subagent + internal/agent
  spawn_agent runtime context inheritance, headless runner tool execution context, normal tool registry web_search caller path

native web-search option source of truth:
  internal/llmcatalog/provider.go
  NativeWebSearchProviderKeys(includeAliases) and nativeWebSearchProviderOrder; do not ad hoc-filter docs/config generation

normal chat owner:
  internal/api/providers/openai_subscription/subscription_provider.go
  unchanged; no always-on web_search built-in tool

Responses shared helper owner:
  internal/api/providers/openai_responses
  only if a pure helper is needed; do not add web_search replay behavior here in this tranche

/review owner:
  internal/reviewadapter + internal/review/evidence + internal/review/externaldoc
  existing SearchWeb caller path, discovery-only result, external_doc fetch/citation policy

history reduction owner:
  internal/providerhistory + internal/rawoutputs
  existing XELYON web_search tool result artifact-backed compact path

doctor owner:
  cmd/doctor_openai_subscription.go
  internal/clidoctor/doctor_openai_subscription.go
  internal/api/providers/openai_subscription/subscription_diagnostics*.go

docs/generated owner:
  docs/config.md
  docs/providers.md
  docs/commands.md
  docs/design or docs/dev if needed
  internal/config/registry_generated.go via make gen-all
```

## 6. Implementation Priority

1. Phase 0: preflight / test foundation for request shape and provider resolution.
2. Phase 1: `openai_subscription` `WebSearchWithContext` adapter and OAuth-only stream parser.
3. Phase 2: `llmcatalog` native web search descriptor, provider order, config registry generation, alias handling.
4. Phase 3: `/review web_search_evidence` caller-path coverage with `openai_subscription`.
5. Phase 4: sub-agent / headless caller-path coverage.
6. Phase 5: doctor capability, request preview, live web-search smoke, CLI contract matrix.
7. Phase 6: docs / generated metadata / command docs.
8. Phase Final-A: impact audit.
9. Phase Final-B: mandatory behavior-preserving refactor including tests.

### 6.1 Required Implementation Preflight Skills

Before any code implementation starts, classify the work with these skills:

- Primary skill: `xelyon-provider-runtime-change`
  - Reason: this tranche changes provider-native request shape, request validation, streaming parse, model/provider capability, doctor smoke, and provider-facing runtime behavior.
- Required companion when touching generated config/docs provider options: `xelyon-config-contract-change`
  - Reason: `NativeWebSearchProviderKeys(true)`, `web_search.provider` options, generated config metadata, `docs/config.md`, and `/config` surface must stay in sync.
- Required companion when touching OAuth transport, provider request/response parsing, URL extraction, diagnostics, preview, logs, or error output: `security-boundary-change`
  - Reason: subscription web search crosses provider/network boundaries and emits provider-originated text/URLs into tool result text, doctor output, logs, and review discovery.

If implementation discovers a broader public contract change not described in this
plan, stop before coding that part and run `shared-contract-change` preflight.
Do not treat these skills as only stop-condition recovery; they are part of
Phase 0 for the implementation Goal.

## 7. Implementation Sections

### 7.1 openai_subscription WebSearchWithContext Adapter

#### Purpose

Add a provider-native web search adapter under `internal/api/providers/openai_subscription`.

Required exported surface:

```go
func WebSearchWithContext(ctx context.Context, query, model string) (string, error)
```

Required registration:

```go
websearch.RegisterWithContext("openai_subscription", WebSearchWithContext)
```

This registration must satisfy section 8.1 checklist item 1: normal startup/import
paths must activate the adapter without test-only imports or side-effect gaps.

#### Non-goals

- Do not call `openai.WebSearchWithContext`.
- Do not import or use `OPENAI_API_KEY`.
- Do not change `SubscriptionProvider.ChatWithTools` normal request shape.
- Do not persist `web_search_call` output items as normal assistant replay items.

#### Current source findings

- `openai_subscription` package already owns OAuth token store, request prep, endpoint validation, HTTP error redaction, debug preview policy.
- Existing provider tests prove `OPENAI_API_KEY` may be set but must not be used by subscription runtime.
- Current `openai_responses.Tool` struct is function-tool-shaped and would serialize `name:""` for `{"type":"web_search"}` if reused directly.

#### Design contract

- Place adapter in the `openaisubscription` package, not in `openai` or `tools/search`.
- Reuse `SubscriptionProvider.prepareSubscriptionResponsesRequest` for OAuth-only headers.
- Validate model with `ValidateSubscriptionModel` before auth.
- Validate endpoint with `validateSubscriptionResponsesEndpoint` before auth.
- Use a package-local web search request DTO with `Tools []map[string]any` or equivalent exact JSON so `tools` serializes exactly as built-in web search requires.
- Do not reuse `openai_responses.Tool` for built-in `{"type":"web_search"}`. It is function-tool-shaped and may serialize fields such as `name` that do not belong in the built-in request.
- Request payload must include:

```json
{
  "model": "<subscription request model>",
  "input": "<web search prompt or input item>",
  "instructions": "<web search instructions>",
  "stream": true,
  "store": false,
  "tools": [{"type": "web_search"}],
  "tool_choice": "required",
  "prompt_cache_key": "<stable key>"
}
```

Must not include:

```text
previous_response_id
context_management
prompt_cache_retention
max_output_tokens
web_search_preview
include
```

Initial implementation must not send `include:["web_search_call.action.sources"]`.
Source extraction must rely on streamed `web_search_call` output items, `action.url`
or equivalent action payload fields when present, and message annotations / text.
Only a separate live probe and follow-up tranche may add `include`, and that follow-up
must preserve the same OAuth-only transport and `Summary:` / `Sources:` output
contract.

#### SSE parsing strategy

Implement a subscription web-search-specific parser that reads SSE `data:` lines and handles at least:

- `response.created`: capture response id for diagnostics only.
- `response.output_text.delta`: append summary text.
- `response.web_search_call.*`: mark web search call observed, capture call status where available.
- `response.output_item.added` / `response.output_item.done` with `item.type=="web_search_call"`: capture source URLs from `item.action.url`, `item.action.sources`, or equivalent action payload fields when present.
- `response.output_item.done` with `item.type=="message"`: capture message content, annotations, and URL-bearing text when present.
- `response.completed` / `response.done`: capture usage if present and finish parsing.
- `error` / `response.failed`: return sanitized error.
- unknown events: ignore unless they are the only signal needed to prove web search ran.

Parser output should be a small internal struct, for example:

```go
type subscriptionWebSearchResult struct {
    Summary string
    Sources []subscriptionWebSearchSource
    WebSearchCallCount int
    ResponseID string
    Usage *api.Usage
}
```

Do not feed `web_search_call` items into `SubscriptionProvider.LastOpenAIResponsesInputItems`.

#### Behavior / output format

Return format must match existing provider adapters:

```text
Summary:
<trimmed summary>

Sources:

1. <title-or-url>
   URL: <url>
```

Rules:

- Empty summary and no sources => `No results found.` or explicit error if web search call was required but never observed.
- Deduplicate sources by trimmed URL.
- Title fallback is URL.
- Summary should be provider text, not a second local summarization pass.
- Do not include OAuth/account/debug metadata in result text.

Runtime success criteria:

- Doctor live web-search smoke is strict: it must fail when no `web_search_call` is observed.
- Normal `SearchWeb` / `ExecuteWebSearch` runtime is tolerant enough to avoid review false negatives.
- Treat the adapter result as successful when at least one of these is true:
  - a `web_search_call` event or output item was observed.
  - structured sources are present from `web_search_call` action payloads or message annotations.
  - non-empty summary text is present and at least one URL can be extracted into the `Sources:` section.
- Summary-only output with no observed `web_search_call` and no URL/source remains a failure or explicit no-results response, not a grounded search result.

#### Safety gates

- If endpoint is Platform `/v1/responses`, fail before auth.
- If OAuth credential is missing/unsafe/malformed, return existing subscription auth error.
- If backend returns HTTP non-200, use `handleSubscriptionHTTPError`.
- If `tool_choice:"required"` or `tools:[{"type":"web_search"}]` is rejected, report capability failure; do not switch to `web_search_preview` or Platform API.
- If no web search call is observed and no URL/source is present, doctor smoke must fail and runtime must not return an ungrounded normal text answer.

#### Tests

Add focused tests under `internal/api/providers/openai_subscription`:

- request uses OAuth bearer token even when `OPENAI_API_KEY` is set.
- request body has `tools[0].type=="web_search"` and `tool_choice=="required"`.
- request omits `previous_response_id`, `context_management`, `prompt_cache_retention`, `max_output_tokens`, `include`, and `web_search_preview`.
- invalid Platform endpoint fails before auth.
- unsupported model fails before auth.
- stream parser captures `response.web_search_call.*`, `response.output_text.delta`, sources, and usage.
- returned usage reaches `websearch.WithUsageCallback` / `api.UsageCallback` when present, while API cost remains ChatGPT subscription `N/A` / pricing unavailable.
- parser dedupes sources from web_search_call action and message annotations.
- parser handles unknown future events without failing.
- runtime succeeds with `web_search_call` observed.
- runtime succeeds with structured sources even if `web_search_call` event naming drifts.
- runtime succeeds with summary plus extractable URL.
- summary-only / no call / no URL causes smoke-classifiable failure.

### 7.2 Provider Resolution, NativeWebSearch, and Alias Policy

#### Purpose

Make `openai_subscription` a first-class native web search provider in XELYON search resolution.

#### Current source findings

- `isNativeSearchProvider` depends on `llmcatalog.ProviderDescriptor.NativeWebSearch`.
- `NativeWebSearchProviderKeys(true)` feeds config registry select options.
- Current `NativeWebSearchProviderKeys(true)` appends every native provider alias, so subscription aliases would leak into config/docs unless the source of truth changes.
- `resolveSearchProvider` intentionally preserves alias owner for Kimi `moonshot` and Claude `anthropic`.
- Current `resolveSearchProvider` returns the normalized input string once `isNativeSearchProvider(provider)` is true.
- `ProviderDescriptorFor("chatgpt")` canonicalizes to the `openai_subscription` descriptor, so `NativeWebSearch:true` would make `chatgpt` look native unless the resolver maps subscription-family search keys to canonical `openai_subscription`.
- `openai_subscription` descriptor has aliases and `CanonicalizeAliasesForConfig:true`.

#### Design contract

- Set `NativeWebSearch:true` on `openai_subscription`.
- Add `openai_subscription` to `nativeWebSearchProviderOrder`.
- Keep canonical provider key `openai_subscription` as the primary adapter key.
- Public `web_search.provider` surface is canonical-only: config UI, generated registry metadata, docs, doctor display, and examples expose only `openai_subscription` for this provider family.
- Hidden compatibility input is allowed: if a user already writes `web_search.provider: chatgpt`, `web_search.provider: openai-subscription`, or `web_search.provider: codex-subscription`, `resolveSearchProvider` may accept it and return canonical `openai_subscription`.
- Do not register `chatgpt`, `openai-subscription`, or `codex-subscription` as search provider aliases.
- Conversation provider aliases remain conversation/provider config concerns; they must not expand the web-search config surface, cache owner labels, docs options, or doctor display names.
- `resolveSearchProvider` must return canonical `openai_subscription` for the entire `openai_subscription` family before calling `websearch.SearchWithContext`.
- This canonicalization applies to hidden explicit `web_search.provider` alias input and main provider aliases, but only for the subscription family.
- Do not solve the registry mismatch by registering subscription aliases in `internal/api/websearch`.
- `NativeWebSearchProviderKeys(true)` must be updated at the `llmcatalog` source of truth so subscription aliases are not returned even when aliases are included for other providers.
- Do not patch docs/config generator output with local filters; generated metadata should inherit the subscription alias suppression from `llmcatalog`.
- Preserve existing alias owner behavior for Kimi / Claude tests.
- Section 8.1 checklist items 3 and 6 are mandatory for this section: aliases are resolver-only hidden compatibility input, and generated config/docs must be refreshed through `make gen-all`.

Do not globally canonicalize `resolveSearchProvider` in a way that breaks Kimi / Claude alias owner tests unless those tests are intentionally redesigned with equivalent owner preservation.

#### Tests

- `llmcatalog.NativeWebSearchProviderKeys(true)` includes `openai_subscription` and does not include `chatgpt`, `openai-subscription`, or `codex-subscription`.
- `llmcatalog.NativeWebSearchProviderKeys(true)` still includes existing non-subscription aliases that are intentionally exposed, such as Kimi / Claude aliases.
- `web_search.provider` generated select options expose only `openai_subscription` for the subscription provider family.
- `resolveSearchProvider` returns registered key `openai_subscription` when explicit `cfg.WebSearch.Provider = "openai_subscription"`.
- Hidden compatibility test: `resolveSearchProvider` returns registered key `openai_subscription` when explicit `cfg.WebSearch.Provider` is `chatgpt`, `openai-subscription`, or `codex-subscription`; cache and usage owner labels still use canonical `openai_subscription/<model>`.
- `resolveSearchProvider` returns registered key `openai_subscription` when main provider config is an `openai_subscription` conversation alias and `web_search.provider` is unset.
- direct `websearch.SearchWithContext(ctx, "chatgpt", ...)` remains unregistered unless a later shared-contract change intentionally expands the public registry surface.
- `SearchWeb` with `cfg.WebSearch.Provider = "openai_subscription"` calls the subscription adapter.
- cache and usage attribution owner are `openai_subscription/<model>`.
- Kimi / Claude alias owner tests still pass.

### 7.3 /review web_search_evidence Integration

#### Purpose

Allow `/review web_search_evidence` to use `openai_subscription` through the existing search abstraction.

#### Current source findings

- `/review` caller path is `reviewadapter.reviewWebSearchRunner` -> `tools/search.SearchWeb`.
- `review.web_search_evidence.enabled` controls collection.
- raw web search result URLs are discovery-only.
- fetched `external_doc` snippets are citation-capable.

#### Design contract

- Do not add `/review`-specific provider config.
- Do not add direct network fetches outside `externaldoc.Fetcher`.
- Do not add `kind:"web_search"` evidence refs.
- `external_support` summary remains the source of truth for external support quality.
- Usage attribution should continue through `tools.UsageAttributionCallback`.

#### Tests

- `internal/reviewadapter` caller-path test: with config `web_search.provider=openai_subscription`, `SearchReviewWeb` returns provider `openai_subscription`, URL results, truncation state, and usage attribution.
- `internal/review/evidence` tests should not need provider-specific changes unless result shape exposes a new edge case.
- Existing report validation rejecting raw `web_search` evidence refs must remain unchanged.

### 7.4 Sub-agent / Headless Caller Path

#### Purpose

Ensure `spawn_agent` / sub-agent / headless execution can use the same
`openai_subscription` web search provider path as the parent runtime.

#### Current source findings

- `internal/tools/subagent/register.go` reads `execCtx.EffectiveProvider()`, `execCtx.EffectiveContext()`, and `execCtx.EffectiveConfig()` before spawning.
- `internal/tools/subagent/manager.go` prepares spawn config from the parent provider/config/model, then starts a sub-agent runtime.
- `internal/agent/headless_runner.go` creates the sub-agent headless runtime and installs the normal tool registry visibility policy.
- `internal/agent/agent_tool_executor.go` builds `tools.ExecutionContext` with `ProviderName`, `ProviderConfigKey`, `Model`, `Config`, and request context for normal tool execution.
- A sub-agent web search therefore reaches the regular `web_search` tool and `tools/search.SearchWeb`, not a separate sub-agent search adapter.

#### Design contract

- Do not add a sub-agent-specific search provider config.
- Do not special-case `openai_subscription` in `internal/tools/subagent`.
- Sub-agent/headless web search must inherit the parent effective config, including `web_search.provider`, unless the existing sub-agent model/provider override contract intentionally changes it.
- When inherited `web_search.provider` is `openai_subscription`, the sub-agent web search must reach `SearchWeb` with canonical provider `openai_subscription` and usage/cache owner `openai_subscription/<model>`.
- Hidden compatibility alias input follows the same resolver rule as parent execution: sub-agent/headless must canonicalize subscription-family aliases before registry/cache/usage ownership.
- Tool visibility/read-only policy should not be changed solely for this tranche unless existing tests show the web_search tool is incorrectly unavailable.
- Section 8.1 checklist item 4 is mandatory: this caller-path coverage must be fake-driven and must not use live network.

#### Tests

- Add or update an `internal/tools/subagent` caller-path test proving `spawn_agent` passes parent config containing `web_search.provider=openai_subscription` into the spawned runtime context.
- Add or update an `internal/agent` headless/sub-agent test proving a spawned headless agent executing `web_search` reaches `SearchWeb` with canonical provider `openai_subscription`.
- Use fake websearch adapter / fake provider plumbing for these tests. Do not require subscription credentials, OAuth tokens, or live backend access.
- Include hidden compatibility coverage if the test harness can set `web_search.provider=chatgpt` or another subscription alias: the resulting provider owner must still be `openai_subscription`.
- Assert usage attribution/cache owner labels use canonical `openai_subscription/<model>`, not raw alias strings.

### 7.5 Doctor Capability, Preview, and Web-search Smoke

#### Purpose

Make `doctor openai-subscription` report local web-search adapter capability and,
when requested, separately prove that the subscription backend currently accepts
the web search request shape.

#### Current source findings

- `cmd/doctor_openai_subscription.go` does not expose `--web-search-smoke`.
- `cmd/doctor_contract_matrix_test.go` currently forbids `web-search-smoke` for openai-subscription.
- `SubscriptionDiagnosticOptions` does not have `WebSearchSmoke`.
- capability snapshot sets `WebSearch:false`.
- request preview currently renders text/tool/cache/thinking/compact request shapes.

#### Design contract

- Add `--web-search-smoke` to `doctor openai-subscription`.
- Add `WebSearchSmoke` to `OpenAISubscriptionOptions` and `SubscriptionDiagnosticOptions`.
- `RunSmoke` should include web search smoke when requested and not `--print-request`.
- capability snapshot should report `WebSearch:true` as a local/static capability when the XELYON adapter and request shape exist.
- `WebSearch:true` does not prove the subscription backend will accept the request forever; live backend acceptance belongs to `web_search_smoke`.
- `--require-capability web_search` should pass for `openai_subscription` after this tranche based on local capability, while `--web-search-smoke` remains the live backend check.
- `--print-request --web-search-smoke` must not send a live request and must show sanitized structural preview:
  - request name
  - route
  - method / redacted URL
  - redacted headers
  - body shape including `tools[0].type=web_search`, `tool_choice=required`, `stream=true`, `store=false`
  - no prompt text, token, raw account id, Authorization value
- live `--web-search-smoke` should require:
  - request sent to subscription endpoint with OAuth transport
  - at least one web search call observed
  - final summary or sources non-empty
  - usage displayed if returned
  - cost remains `N/A (ChatGPT subscription)`

#### Failure policy

- Missing auth / unsafe permission / invalid endpoint remain readiness skips/failures as current doctor behavior dictates.
- If backend rejects `web_search` or `tool_choice:"required"`, classify as web search smoke failure with a suggestion to rerun without `--web-search-smoke`; do not mark basic provider text smoke as failed unless basic smoke was requested and failed.
- If `web_search_preview` would work or fail, ignore it; it is out of scope and must not appear as fallback.

#### Tests

- `cmd/doctor_contract_matrix_test.go` updates required/forbidden flags and docs row.
- `internal/clidoctor/doctor_openai_subscription_test.go` or new focused test covers option wiring.
- `internal/api/providers/openai_subscription` diagnostics tests cover preview body and live mock smoke:
  - `web_search_payload=true`
  - `tool_choice=required`
  - `WebSearch` capability true
  - no token leakage
  - smoke failure when no `web_search_call` observed
- capability JSON distinguishes local `web_search=true` from live `web_search_smoke` success/failure.

### 7.6 Docs and Generated Metadata

#### Purpose

Document `openai_subscription` as a selectable search provider without implying Platform API fallback or cost avoidance.

#### Required updates

- `docs/config.md`
  - Web検索 section: add canonical `openai_subscription` to native search providers.
  - Do not document `chatgpt`, `openai-subscription`, or `codex-subscription` as `web_search.provider` values.
  - Required credential section: use `xelyon auth openai-subscription login`, not `OPENAI_API_KEY`.
  - State that subscription web search uses ChatGPT/Codex subscription backend and no Platform API fallback.
- `docs/providers.md`
  - OpenAI Subscription section: add `--web-search-smoke`, request shape, and no fallback policy.
  - Distinguish local/static `WebSearch:true` capability from live `--web-search-smoke` backend acceptance.
  - Provider docs should not claim stable production automation support.
- `docs/commands.md`
  - `web_search` tool provider list.
  - `doctor openai-subscription` command flags if generated docs require it.
- `internal/config/registry_generated.go`
  - regenerate through `make gen-all`; do not hand-edit generated output.
- `scripts/internal/configmeta/sections_test.go` and `internal/config/registry_metadata_test.go`
  - update expected options if needed.

#### Non-goals

- Do not add a new `openai_subscription.web_search` config section.
- Do not add a new `review.web_search_evidence.provider`.
- Do not present subscription endpoint as OpenAI API billing replacement.

### 7.7 Provider History and Raw Output Ref Contract

#### Purpose

Ensure new subscription search results remain ordinary XELYON `web_search` tool results for existing provider-facing history reduction.

#### Design contract

- `ExecuteWebSearch` / `SearchWeb` returns raw result text in the same format as other providers.
- No provider-native `web_search_call` replay item is added to chat history.
- `providerhistory` sees tool name `web_search` and result content, so existing `recordProviderHistoryWebSearchArtifactCandidate` applies.
- `SurfaceXelyonWebSearchToolResult` remains the surface.
- `SurfaceProviderNativeBuiltinReplay` is not used for this tranche.

#### Tests

- Existing `internal/providerhistory/web_search_projection_test.go` should pass unchanged.
- Add a caller-path test only if subscription result format needs a fixture to prove `ParseWebSearchResults` and raw output placeholder compatibility.

## 8. Mode / Policy / Defaults

- `openai_subscription` web search is opt-in through `web_search.provider` or selected as main provider when provider resolution says native web search is available.
- Public docs/examples should show only `web_search.provider=openai_subscription`; subscription aliases are hidden compatibility input only.
- It is not enabled as an always-on normal conversation tool.
- Existing web search cache policy applies.
- Existing `review.web_search_evidence.enabled` opt-in applies.
- Existing provider history reduction defaults apply.
- No new stable config key is planned.
- No migration is required unless generated select metadata changes.

Provider selection examples:

```yaml
web_search:
  provider: openai_subscription
```

```yaml
review:
  web_search_evidence:
    enabled: true
```

## 8.1 Implementation Pitfalls / Must-not-miss Checklist

Implementation must explicitly check these items before Final-A:

- [ ] `websearch.RegisterWithContext("openai_subscription", ...)` is active in the normal startup/import path.
  Do not rely on test-only imports, one-off registration in a command path, or an init order that only works in focused tests.
- [ ] Do not reuse `openai_responses.Tool` for the built-in `{"type":"web_search"}` request.
  Build the subscription web-search request with a package-local DTO that emits exact JSON for `tools:[{"type":"web_search"}]` and `tool_choice:"required"`.
- [ ] Do not register subscription aliases in the `websearch` registry.
  `chatgpt`, `openai-subscription`, and `codex-subscription` remain hidden compatibility input only; `resolveSearchProvider` must canonicalize them to `openai_subscription` before registry lookup, cache owner, and usage owner handling.
- [ ] Sub-agent/headless caller-path tests must not use live network.
  Use fake websearch adapter / fake provider plumbing to prove `spawn_agent` inherits parent config and reaches normal `web_search` execution through `SearchWeb` with canonical `openai_subscription` ownership.
- [ ] If token usage is returned, pass it through the existing usage callback path.
  Do not convert ChatGPT subscription usage into billable OpenAI Platform API cost; API cost remains `N/A` / pricing unavailable.
- [ ] Because `NativeWebSearchProviderKeys`, config registry metadata, and docs are in the contract, run `make gen-all` after implementation changes that affect provider options or generated docs.
  Do not hand-edit generated config/docs output and stop there.

Scope guard:

- `include:["web_search_call.action.sources"]` remains out of scope for the initial request.
- `web_search_preview` remains out of scope.
- Normal chat always-on built-in web search remains out of scope.

## 9. Tests

Focused tests to add or update:

- `go test ./internal/api/providers/openai_subscription -run 'Test.*WebSearch|TestDiagnose.*WebSearch|TestSubscription.*WebSearch'`
- `go test ./internal/api/websearch`
- `go test ./internal/tools/search -run 'TestResolveSearchProvider|TestSearchWeb|TestExecuteWebSearch'`
- `go test ./internal/llmcatalog ./scripts/internal/configmeta ./internal/config`
- `go test ./internal/reviewadapter ./internal/review/evidence ./internal/review/report`
- `go test ./internal/tools/subagent ./internal/agent -run 'Test.*SubAgent.*WebSearch|TestRunHeadless.*WebSearch'`
- `go test ./internal/clidoctor ./cmd -run 'Test.*OpenAISubscription|TestDoctorContract'`
- `go test ./internal/providerhistory -run 'TestProject.*WebSearch'`

Test categories:

- section 8.1 checklist coverage.
- OAuth-only transport and Platform fallback rejection.
- exact request shape.
- stream parser positive and negative paths.
- Summary/Sources formatting and URL dedupe.
- provider resolution / alias / cache owner / usage attribution.
- `/review` caller path.
- sub-agent / headless caller path.
- doctor capability / preview / smoke.
- docs/generated metadata sync.
- no normal chat web_search built-in payload regression.

## 10. Verification Commands

During implementation:

```sh
gofmt -w <changed-go-files>
go test ./internal/api/providers/openai_subscription
go test ./internal/api/websearch ./internal/tools/search
go test ./internal/reviewadapter ./internal/review/evidence ./internal/review/report
go test ./internal/tools/subagent ./internal/agent -run 'Test.*SubAgent.*WebSearch|TestRunHeadless.*WebSearch'
go test ./internal/clidoctor ./cmd
go test ./internal/llmcatalog ./scripts/internal/configmeta ./internal/config
go test ./internal/providerhistory -run 'TestProject.*WebSearch'
make gen-all
git diff --check
```

Do not skip `make gen-all` after touching `NativeWebSearchProviderKeys`, config
registry metadata, generated docs, or provider option docs. This is a section
8.1 checklist gate, not an optional cleanup.

Before commit, per repo rule:

```sh
make verify-fast
make ci-check
```

Live smoke is not normal CI. Run manually only with logged-in subscription auth:

```sh
xelyon doctor openai-subscription --web-search-smoke --print-request
xelyon doctor openai-subscription --web-search-smoke
xelyon doctor openai-subscription --require-capability web_search
```

`/review` is primarily slash-command/runtime surface, so do not add an unverified
CLI smoke command to this plan. Use the unit/caller tests above for automated
coverage. If manual smoke is needed, run it from an interactive XELYON session
with `review.web_search_evidence.enabled=true` and `web_search.provider` set to
`openai_subscription`.

## 11. Stop Conditions

Section 6.1 skills are required at implementation start. The list below is for
additional reclassification if implementation uncovers a larger or different
contract than this plan describes.

Stop and reclassify before continuing if any of these happen:

- Implementing web search requires changing normal `ChatWithTools` to always include provider-native built-in web search.
- Subscription backend only works by using `web_search_preview`.
- Subscription backend only works by using `OPENAI_API_KEY` or Platform `/v1/responses`.
- `response.web_search_call.*` replay needs to become part of general conversation history for correctness.
- A change to `openai_responses` stream parser risks altering normal function call replay / reasoning replay semantics.
- Alias canonicalization breaks existing Kimi / Claude owner tests.
- `/review` needs a new search provider config or direct fetch path to make this work.
- Raw search results are being promoted to citation-capable review evidence refs.
- Doctor preview or debug output would expose OAuth tokens, account IDs, raw prompt text, or Authorization headers.
- Generated config metadata cannot be updated without adding a new stable config key.

When a stop condition triggers, do not keep widening the same diff.
Report the owner conflict and split a new task using the appropriate skill:

- shared request/config/capability contract: `shared-contract-change`
- provider runtime behavior / token / request path: `xelyon-provider-runtime-change`
- config / generated metadata / docs provider option contract: `xelyon-config-contract-change`
- security boundary: `security-boundary-change`
- state/cache/session lifecycle: `go-state-lifecycle-change`
- package split: `package-boundary-map` then `package-boundary-refactor`

## 12. Phase Final-A: Impact Audit

After implementation and focused tests, perform an impact audit before final report.

Required checks:

- Section 8.1 checklist is complete and reflected in tests or explicit no-change evidence.
- `OPENAI_API_KEY` set in tests never affects subscription web search Authorization.
- Platform endpoint forbidden tests cover web search path.
- `web_search_preview` does not appear in request builder, docs as fallback, or smoke recovery.
- `include:["web_search_call.action.sources"]` is absent from initial request builder and docs.
- normal chat request tests still show no built-in `web_search` tool.
- `openai_responses` function_call / reasoning replay tests still pass if touched.
- Kimi / Claude alias owner tests still pass.
- `openai_subscription` search provider surface is canonical-only; `chatgpt`, `openai-subscription`, and `codex-subscription` are not exposed as `web_search.provider` options.
- explicit or inherited subscription aliases resolve to canonical `openai_subscription` before registry/cache/usage ownership; no path returns raw `chatgpt` / `openai-subscription` / `codex-subscription` to `websearch.SearchWithContext`.
- sub-agent/headless web search inherits parent config and reaches `SearchWeb` with canonical `openai_subscription` provider/cache/usage owner.
- Runtime adapter success criteria cover `web_search_call` observed, structured sources present, and summary plus URL present.
- Doctor smoke remains stricter than normal runtime and fails when no `web_search_call` is observed.
- `WebSearch:true` doctor capability is local/static and does not replace live web-search smoke.
- `/review` raw `web_search` evidence ref rejection still passes.
- providerhistory `web_search` raw output artifact tests still pass.
- doctor JSON/text preview redaction covers web search request.
- generated config metadata and docs agree on provider options.

If a correctness issue is found, fix it in Final-A.
Do not continue to Final-B while behavior is known broken.

## 13. Phase Final-B: Mandatory Refactor Gate

This implementation touches provider-facing request shape, web search projection, doctor diagnostics, config metadata, and review evidence caller paths.
Final-B is mandatory.

Final-B inventory:

- production files changed
- test files changed
- new request DTO / stream parser / formatter helpers
- changed provider descriptor / config metadata
- doctor request preview / smoke result fields
- docs/generated updates

Refactor decision:

- MUST extract package-local helper if request building, SSE parsing, and formatting mix in one long function.
- MUST keep OAuth transport preparation in existing subscription transport owner.
- MUST keep normal chat request builder free of built-in web search logic.
- SHOULD share formatter logic only if it does not create a cross-provider dependency knot.
- SHOULD split tests if one file becomes a mixed fixture for auth, stream parser, doctor, and provider resolution.
- NO broad `openai_responses` replay refactor unless required by focused tests.

Exit checklist:

- Owner map reviewed.
- New helper boundaries named by responsibility, not generic `utils`.
- Tests are readable and fail on the intended contract.
- No duplicated request-shape assertions that should be a helper.
- No behavior-preserving cleanup left in the same package that would obviously reduce future review findings.
- Residual debt is reported with file/function names.

## 14. Implementer Freedom

Implementation may choose:

- exact file name under `internal/api/providers/openai_subscription`, for example `subscription_web_search.go`.
- exact internal parser struct names.
- whether parser uses `api.ParseStreamingResponse` or a package-local scanner, as long as cancellation and body limits stay safe.
- whether source extraction handles only known probe shape first or also OpenAI Platform-like annotations, as long as unknown events are safe.

Implementation may not change:

- OAuth-only transport.
- canonical `websearch.RegisterWithContext("openai_subscription", ...)`.
- `tools:[{"type":"web_search"}]` and `tool_choice:"required"` as the first request shape.
- canonical-only public `web_search.provider` surface for `openai_subscription`; hidden compatibility alias input is resolver-only.
- no `include:["web_search_call.action.sources"]` in the initial request shape.
- no `web_search_preview`.
- no normal chat always-on built-in web search.
- no provider-native `web_search_call` lossless replay / compact in this tranche.
- `/review` discovery-only raw search result policy.

## 15. Open Decisions

No user-facing product decision remains open.

Implementation-time decisions:

- Exact package-local parser/helper names.
- Exact source extraction breadth beyond observed probe shapes, as long as unknown event handling stays safe.

Deferred to a separate live-probe / follow-up tranche:

- Adding `include:["web_search_call.action.sources"]`.
- Promoting hidden subscription aliases to public `web_search.provider` options, docs examples, doctor display names, or `websearch` registry adapters.
- Provider-native normal chat `web_search_call` replay or compact.

## 16. Goal Handoff Prompt

Use this prompt for a fresh implementation Goal:

```text
/goal Implement docs/dev/openai-subscription-web-search-master-plan.md end to end.

Use docs/dev/openai-subscription-web-search-master-plan.md as the source of truth. Start with Phase 0 pre-implementation refactor / test foundation, then implement the planned sections, then run the impact audit and mandatory post-implementation refactor described in the plan.

Before coding, run the required preflight classification from section 6.1: use `xelyon-provider-runtime-change` as the primary skill, use `xelyon-config-contract-change` for generated config/docs provider option changes, and use `security-boundary-change` for OAuth/provider request, response parsing, URL extraction, diagnostics, preview, logs, or error output.

Before implementation closeout, complete the section 8.1 Implementation Pitfalls / Must-not-miss Checklist. Treat unchecked checklist items as blockers for Final-A.

Pay special attention to section 7.2: `openai_subscription` search provider registration is canonical-only, but `resolveSearchProvider` must canonicalize subscription-family aliases to `openai_subscription` before registry/cache/usage ownership. Do not register `chatgpt`, `openai-subscription`, or `codex-subscription` as web_search adapters.

Also include section 7.4 coverage: sub-agent/headless `web_search` must inherit parent config and reach `SearchWeb` with canonical `openai_subscription` provider/cache/usage ownership.

Final-B is mandatory, not optional. After tests pass, run post-implementation-refactor; if tests, fixtures, fakes, table tests, or assertion helpers changed, also run test-boundary-refactor. Same-file repeated findings, large files, or generic helpers mixing semantic roles trigger a file/test split audit before strict review.

If the plan and latest source structure conflict, preserve the safety contracts and adapt to the existing owner boundaries. Re-read the plan after resume or context compaction. Do not commit or push unless explicitly requested.
```
