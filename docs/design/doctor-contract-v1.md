# Doctor Surface Inventory and Contract v1

この文書は provider doctor / live smoke を全 provider に広げる前の Phase 0 inventory と contract v1 の設計メモです。

目的は「全 provider に同じ高度機能を無理に載せる」ことではなく、OSS として予測できる診断 surface をそろえることです。Provider 固有の機能差は残しつつ、共通の flag、JSON 形、status 意味、test 境界を先に決めます。

## Scope

対象:

- CLI doctor surface: `xelyon doctor <provider>`
- live smoke / request preview / JSON output
- provider model / endpoint / auth / route / usage / cost / capability diagnostics
- docs と tests に固定すべき user-visible contract

対象外:

- この Phase 0 では runtime request path は変更しない
- `doctor all`、support bundle export、policy engine、enterprise dashboard は別 phase
- live provider credentials を必要とする smoke の通常 CI 化はしない

## Current Inventory

### Command Surface

| Provider | Doctor | `--json` | `--smoke` | `--tool-smoke` | `--image-smoke` | `--thinking-smoke` | `--web-search-smoke` | `--retention-smoke` | `--capabilities` | `--require-capability` | `--print-request` | Provider-only |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| OpenAI | yes | yes | yes | yes | no | no | no | yes | yes | yes | yes | none |
| Azure OpenAI | yes | yes | yes | yes | no | no | no | yes | yes | yes | yes | `--deployment`, `--print-config` |
| Kimi | yes | yes | yes | yes | yes | no | yes | no | no | no | yes | none |
| Bedrock | yes | yes | yes | yes | yes | yes | no | no | no | no | yes | none |
| DeepSeek | yes | yes | yes | yes | no | no | no | no | no | no | yes | none |
| Gemini | yes | yes | yes | yes | yes | no | yes | no | no | no | yes | none |
| Claude / Anthropic | yes | yes | yes | yes | yes | yes | yes | no | no | no | yes | none |
| Groq | yes | yes | yes | yes | no | no | no | no | no | no | yes | none |
| Ollama | yes | yes | yes | yes | no | no | no | no | no | no | yes | none |
| OpenRouter | yes | yes | yes | yes | no | no | no | no | no | no | yes | none |

### Current Behavior Notes

OpenAI:

- Checks `OPENAI_API_KEY`, `OPENAI_API_URL`, `OPENAI_RESPONSES_URL`, provider registration, model / `catalog_model`, route, function calling, catalog policy, retention settings.
- Supports local `--capabilities`, `--require-capability`, and `--print-request`.
- Live smoke supports text, tool, and Responses retention chain.
- Main owner packages: `cmd/doctor_openai.go`, `internal/api/providers/openai/diagnostics*.go`, `internal/providerdiag`.

Azure OpenAI:

- Checks `AZURE_OPENAI_BASE_URL`, API key / Entra ID auth, deployment, `catalog_model`, route, function calling, catalog policy, retention settings.
- Supports local `--capabilities`, `--require-capability`, `--print-request`, and Azure-specific `--print-config`.
- Live smoke supports text, tool, and Responses retention chain.
- Main owner packages: `cmd/doctor.go`, `cmd/doctor_azure_config.go`, `internal/api/providers/azure/diagnostics*.go`, `internal/providerdiag`.

Kimi:

- Checks `MOONSHOT_API_KEY`, `KIMI_API_URL`, provider registration, model / `catalog_model`, Chat Completions route, token / pricing metadata, image capability, unsupported native features, and prompt cache request shape.
- Supports `--print-request`.
- Live smoke supports text, image, tool, and built-in `$web_search`.
- Does not yet support `--capabilities` or `--require-capability`.
- Main owner packages: `cmd/doctor_kimi.go`, `internal/api/providers/kimi/diagnostics*.go`, `internal/providerdiag`.

Bedrock:

- Checks AWS region / credentials, provider registration, model / `catalog_model`, route, function calling, catalog policy.
- Live smoke supports text, tool, image, and thinking request types. Unsupported request shapes are reported as skipped.
- Supports `--print-request`.
- Request preview is credential-independent and records request name, route, Bedrock operation, conceptual runtime endpoint, model ID, redacted AWS SigV4 header, and request body without sending network traffic.
- Claude family preview uses the same Claude Messages body builder as `InvokeModelWithResponseStream`; non-Claude preview uses the same `buildConverseStreamInput` builder as `ConverseStream`.
- Does not yet support `--capabilities` or `--require-capability`.
- Main owner packages: `cmd/doctor_bedrock.go`, `internal/api/providers/bedrock/diagnostics*.go`.

Groq:

- Checks `GROQ_API_KEY`, `GROQ_API_URL`, provider registration, model / `catalog_model`, Chat Completions route, function calling, catalog policy.
- Supports `--print-request`.
- Live smoke supports text and tool request types. `GROQ_FUNCTION_CALLING=0` skips tool smoke with warn and runs text smoke fallback.
- Does not support `--capabilities`, `--require-capability`, image, thinking, web search, or retention smoke in v1.
- Main owner packages: `cmd/doctor_groq.go`, `internal/api/providers/groq/diagnostics*.go`, `internal/providerdiag`.

DeepSeek:

- Checks `DEEPSEEK_API_KEY`, `DEEPSEEK_API_URL`, provider registration, model / `catalog_model`, Chat Completions route, thinking request config, function calling, catalog policy.
- Supports `--print-request`.
- Live smoke supports text and tool request types. `DEEPSEEK_FUNCTION_CALLING=0` skips tool smoke with warn and runs text smoke fallback.
- Does not support `--capabilities`, `--require-capability`, image, web search, or retention smoke in v1. Thinking is reported from normal request config rather than a separate `--thinking-smoke`.
- Main owner packages: `cmd/doctor_deepseek.go`, `internal/api/providers/deepseek/diagnostics*.go`, `internal/providerdiag`.

OpenRouter:

- Checks `OPENROUTER_API_KEY`, `OPENROUTER_API_URL`, provider registration, model / `catalog_model`, OpenAI-compatible Chat Completions vs Anthropic Skin route, function calling, image input support, and catalog policy.
- Supports `--print-request`.
- Live smoke supports text and tool request types on the selected runtime route. `OPENROUTER_FUNCTION_CALLING=0` skips tool smoke with warn and runs text smoke fallback.
- Does not support `--capabilities`, `--require-capability`, image, thinking, web search, or retention smoke in v1. `image_input` is a local provider-level check only.
- Route selection follows the runtime request model. A configured alias whose `catalog_model` is Claude still reports Chat Completions unless the request model itself is `anthropic/claude-*`.
- A direct routed OpenRouter request model cannot be re-described by a different known routed `catalog_model`; mismatch is warn and token / pricing policy falls back to the request model when local metadata is available.
- Main owner packages: `cmd/doctor_openrouter.go`, `internal/api/providers/openrouter/diagnostics*.go`, `internal/providerdiag`.

Gemini:

- Checks `GEMINI_API_KEY`, `GEMINI_API_URL`, provider registration, model / `catalog_model`, `streamGenerateContent?alt=sse` route, function calling, image input, thinking, context caching, native web search, and catalog policy.
- Supports `--print-request`.
- Live smoke supports text, tool, image, and native web search request types. Text / tool / image use `streamGenerateContent?alt=sse`; web search uses native `generateContent`.
- Native web search smoke observes usage / cost from `generateContent` `usageMetadata` when available, but usage is not required for the smoke success condition.
- `GEMINI_API_URL` is an exact endpoint / proxy override. Endpoint diagnostics are route-aware: selected text / tool / image requests expect `streamGenerateContent?alt=sse`, while selected native web search requests expect `generateContent`.
- Tool smoke / preview forces request-scoped Gemini function calling mode `ANY` for the diagnostic tool only. Normal runtime still uses `GEMINI_FC_MODE` fallback.
- Non-Gemini `catalog_model` values are warn and do not use OpenAI / OpenRouter / other owner metadata for token or cost policy.
- Live smoke failures classify auth / authorization, quota / rate limit / capacity, model unavailable, empty SSE response, endpoint route mismatch, tool unsupported, image unsupported, and native web search unsupported in the `smoke` check suggestion. Request-level errors stay in `smoke.requests[].error`.
- Pricing metadata unavailable is a `cost` warn after successful usage observation, not a smoke failure.
- Does not support `--capabilities`, `--require-capability`, retention smoke, or separate thinking smoke in v1.
- Main owner packages: `cmd/doctor_gemini.go`, `internal/api/providers/gemini/diagnostics*.go`, `internal/providerdiag`.

Claude / Anthropic:

- Checks `ANTHROPIC_API_KEY`, `ANTHROPIC_API_URL`, provider registration, model / `catalog_model`, Anthropic Messages route, function calling, image input, thinking request config, context management, Claude compaction, native web search, and catalog policy.
- Supports `--print-request`.
- Live smoke supports text, tool, image, thinking, and native web search request types through the Claude runtime request builders. Native web search uses the Anthropic Messages endpoint with `web-search-2025-03-05` beta.
- Request preview is credential-independent and records redacted `x-api-key`, `anthropic-version`, optional `anthropic-beta`, endpoint, route, and request body without sending network traffic.
- Non-Claude `catalog_model` values are warn and do not use OpenAI / OpenRouter / other owner metadata for token or cost policy.
- `CLAUDE_FUNCTION_CALLING=0` skips tool smoke with warn and runs text smoke fallback.
- Native web search smoke currently treats summary or source as the success condition and does not require token usage / cost observation.
- Does not support `--capabilities`, `--require-capability`, or retention smoke in v1.
- Main owner packages: `cmd/doctor_claude.go`, `internal/api/providers/claude/diagnostics*.go`, `internal/providerdiag`.

Ollama:

- Checks `OLLAMA_BASE_URL`, provider registration, model / `catalog_model`, installed model availability from `/api/tags`, `/api/chat` route, function calling, and catalog policy.
- Supports `--print-request`.
- Live smoke supports text and tool request types through the Ollama runtime request builder. `OLLAMA_FUNCTION_CALLING=0` skips tool smoke with warn and runs text smoke fallback.
- Has no API key. `auth` reports explicit no-auth local provider status.
- `OLLAMA_BASE_URL` must be a base URL. Concrete `/api/chat` or `/api/tags` endpoint values fail endpoint diagnostics so smoke does not send a guaranteed wrong `.../api/chat/api/chat` request.
- Non-Ollama `catalog_model` values are warn and do not use OpenAI / OpenRouter / other owner metadata for token policy. Unknown local request models are allowed but fail `installed_model` when absent from `/api/tags`.
- Does not support `--capabilities`, `--require-capability`, image, thinking, web search, or retention smoke in v1.
- Main owner packages: `cmd/doctor_ollama.go`, `internal/api/providers/ollama/diagnostics*.go`, `internal/providerdiag`.

Missing doctor providers:

- None for canonical providers in the current matrix. Future work is contract consolidation, not a missing v1 baseline.

## Contract v1

### Shared CLI Contract

Every canonical provider doctor should eventually support:

- `xelyon doctor <provider>`
- `--json`
- `--model <model>` where provider uses model IDs directly
- `--deployment <deployment>` only where provider has a deployment abstraction, currently Azure
- `--catalog-model <model>` when runtime model identity may differ from local catalog identity
- `--smoke`
- `--timeout`
- `--print-request`

Specialized flags stay provider-specific:

- OpenAI / Azure: `--retention-smoke`
- Kimi: `--web-search-smoke`
- Bedrock: `--thinking-smoke`
- Providers with image input: `--image-smoke`
- Providers with tool calling: `--tool-smoke`
- Azure: `--print-config`

`--capabilities` and `--require-capability` should become shared v1.1 after the common capability DTO is stable across non-OpenAI-family providers.

### Shared JSON Envelope

Provider reports should keep provider-specific fields when needed, but expose a common envelope shape:

```json
{
  "provider": "openai",
  "model": "gpt-5.4",
  "model_source": "--model",
  "catalog_model": "gpt-5.4",
  "catalog_model_source": "--catalog-model",
  "route": "responses_streaming",
  "route_reason": "catalog_model=gpt-5.4 supports Responses streaming",
  "checks": [],
  "request_preview": {},
  "smoke": {},
  "capabilities": {}
}
```

Provider-specific aliases:

- Azure may use `deployment` / `deployment_source` as the request target, but should also conceptually map it to the shared "request target" role.
- Bedrock uses AWS request IDs, not OpenAI response IDs.
- Ollama has no auth key and should report local endpoint / model availability instead.

### Shared Check Names

Use these names where applicable:

- `auth`: credential availability or explicit no-auth local provider status
- `endpoint`: endpoint URL / base URL reachability or syntax
- `provider_registration`: provider is registered in the runtime registry
- `model`: request model / deployment is resolved
- `catalog_model`: catalog identity is resolved when needed
- `route`: request route is selected
- `catalog_policy`: context window / max output / pricing metadata
- `function_calling`: tool payload setting and support
- `image_input`: image input setting and support
- `request_preview`: sanitized request shape was built without sending
- `smoke`: live smoke summary
- `<name>_smoke`: specialized live smoke request, for example `tool_smoke`, `image_smoke`, `retention_smoke`
- `capabilities`: local capability snapshot
- `required_capability`: CI gate result

Do not invent provider-specific names for a shared concept unless the provider has a real extra concept. For example, Azure can keep `deployment`, but a generic endpoint check should not be named `base_url` in one provider and `api_url` in another after v1 migration. Existing names can be preserved until the migration phase to avoid breaking users.

### Status Semantics

| Status | Meaning | Exit behavior |
| --- | --- | --- |
| `ok` | The check passed or the capability is available. | no failure |
| `warn` | The command can continue, but the result is incomplete, fallback-based, skipped, or not suitable for strict automation. | no failure |
| `fail` | The requested diagnostic, smoke, or required capability cannot be satisfied. | command exits non-zero |
| `unknown` | Use inside capability details when local metadata cannot determine availability. | `required_capability` check should fail |
| `skipped` | Use inside request-level smoke / preview objects when a requested specialized path cannot be sent by this route/provider. | check status depends on whether the skip violates the requested smoke contract |
| `unsupported` | Use for stable provider limitations. | usually warn unless the user explicitly required that capability |

### Live Smoke Contract

`--smoke` means "send the provider's minimal representative text request through the same runtime request path users use".

Each smoke result should include when available:

- request name
- route
- duration
- response content excerpt or normalized content
- provider request ID or response ID
- usage observed boolean
- input / output / cached / reasoning / cache creation token fields
- cost estimate and pricing-unavailable flag
- request-level error

Live smoke should fail when:

- prerequisites fail
- the live request errors
- a requested specialized smoke does not prove the requested behavior, such as missing tool call for `--tool-smoke`
- a required route-specific smoke cannot run, such as retention smoke on a non-Responses route

Live smoke should warn, not fail, when:

- the live request succeeds but usage is unavailable
- pricing metadata is unavailable
- provider request ID is unavailable
- a non-required observation is missing

### Request Preview Contract

`--print-request` means "build the same request shape as smoke would send, but do not send it".

Request preview should:

- be safe to run without credentials where possible
- redact auth headers
- show endpoint, method, headers, route, request name, and body
- support multi-request smoke previews
- record skipped request entries instead of silently omitting unsupported paths

Request preview must not:

- send network requests
- require live credentials unless the provider cannot build endpoint/auth-independent request shape
- mutate runtime session state

### Capability Contract

`--capabilities` should be local-only and should not require auth or endpoint unless the provider cannot determine capability locally.

Capability availability should use tri-state semantics:

- `ok` / available
- `missing` / known unavailable
- `unknown` / metadata unresolved

The owner for availability policy should be a provider-neutral diagnostic package where possible. Provider packages should own provider-specific resolution, such as Azure deployment to catalog model mapping or Ollama installed model discovery.

Initial shared capability names:

- `chat_completions`
- `responses_api`
- `responses_streaming`
- `function_calling`
- `image_input`
- `web_search`
- `thinking`
- `previous_response_id`
- `session_persistence`
- `server_compaction`
- `local_model_available`

Do not require every provider to implement every capability. Unsupported or inapplicable capabilities should be explicit.

## Owner Map

| Path | Current owner | Boundary notes |
| --- | --- | --- |
| `cmd/doctor*.go` | CLI flags, config loading, text / JSON rendering handoff | Flag names and user-visible command surface live here. Avoid provider policy here. |
| `cmd/doctor_render.go` | Shared text rendering helpers | Good owner for common smoke usage/cost rendering. Not a provider policy owner. |
| `internal/api/providers/<provider>/diagnostics*.go` | Provider-specific diagnostic orchestration and smoke execution | Correct owner for auth, endpoint, model resolution, request preview, and smoke request construction. Some providers do not have this package yet. |
| `internal/providerdiag` | Provider-neutral doctor policy DTOs and required capability evaluation | Correct owner for shared capability names, status semantics, catalog policy, and route decision DTOs. Do not put provider credentials or request I/O here. |
| `internal/api/providers/openai_compat*` | Shared OpenAI-compatible request/stream helpers | Good candidate for future DeepSeek/Groq/OpenRouter doctor request preview and smoke helpers, but should not own CLI contract. |
| `docs/commands.md` / `docs/providers.md` | User-facing command reference | Should link to this contract once provider doctor v1 starts migrating. |

## Impact Map for Future Implementation

When adding or migrating a provider doctor, check these surfaces:

- CLI: new `cmd/doctor_<provider>.go` or shared command builder
- Provider diagnostics: `internal/api/providers/<provider>/diagnostics*.go`
- Shared helpers: `cmd/doctor_render.go`, `internal/providerdiag`
- Tests: cmd invocation tests, provider diagnostics tests, request preview tests, smoke fake-server tests
- Docs: `docs/commands.md`, `docs/providers.md`, provider-specific docs if present
- Makefile: live smoke target only when useful and credentials are well-defined
- Runtime source of truth: model defaults, endpoint env vars, function calling toggles, image support, usage/cost calculation

## Recommended Phases

### Phase 1: OpenAI-compatible doctor template

Target: OpenAI-compatible Chat Completions providers. `groq` is the first implemented provider in this phase; `deepseek` followed in Phase 2.

Reason:

- Both are Chat Completions style providers.
- Both already use `openai_compat` helpers.
- Auth, endpoint, model, function calling, usage, cost, `--smoke`, `--tool-smoke`, and `--print-request` can validate the v1 contract without Responses retention complexity.

Completed for Groq:

- Add `doctor groq`
- Add provider diagnostics package files
- Add `--json`, `--model`, `--catalog-model`, `--smoke`, `--tool-smoke`, `--print-request`, `--timeout`
- Add fake-server tests for smoke and request preview
- Add docs matrix row updates

Remaining expected scope:

- Use Groq / DeepSeek implementation experience before extracting any shared OpenAI-compatible doctor helper

Do not include:

- `--capabilities` / `--require-capability` for every provider in this first migration
- `doctor all`
- OpenRouter route split

### Phase 2: DeepSeek doctor

Target: `deepseek`.

Status: completed.

Focus:

- OpenAI-compatible request shape
- thinking config visibility
- function calling toggle
- usage and cache token normalization

### Phase 3: OpenRouter doctor

Target: `openrouter`.

Status: completed.

Focus:

- OpenAI-compatible vs Anthropic-skin route explanation
- upstream model identity
- image support
- Claude compaction route warnings

### Phase 4: Native provider doctors

Targets: `gemini`, `claude` / `anthropic`, `ollama`.

Focus:

- Gemini: model URL, function calling, image, thinking, caching, web search
- Claude: Messages endpoint, tool use, image, thinking, context management, web search
- Ollama: local endpoint reachability, installed model check, local usage counts, no-auth semantics

Status:

- Gemini: completed.
- Claude / Anthropic: completed.
- Ollama: completed.

Preparatory boundary for native providers:

- Keep shared doctor status names, JSON projection rules, and generic catalog/pricing checks in `cmd` / `internal/providerdiag`.
- Keep native request payload construction, endpoint shape, tool/image/thinking/cache/web-search payload policy, and stream/response interpretation in each provider package.
- For Gemini doctor, reuse pure request builders in `internal/api/providers/gemini/request_builder.go` for request preview and smoke construction instead of duplicating runtime payload assembly.

## Refactor Decision

MUST before broad implementation:

- Define v1 contract in this document and use it as the source of truth for subsequent provider doctor work.
- For each new doctor, keep provider-specific diagnostics in that provider package and shared status/capability semantics in `internal/providerdiag`.

SHOULD during implementation:

- Extract a small OpenAI-compatible diagnostic helper only after `groq` and `deepseek` both show duplicated request preview / smoke code.
- Add shared cmd assertion helpers for new doctor JSON tests instead of copying provider-specific JSON parsing.

NO for Phase 0:

- Do not rename existing check names yet. That is a user-visible JSON contract migration and should be handled separately.
- Do not add `--capabilities` / `--require-capability` to all providers in one change.
- Do not introduce a generic `doctor` interface before two non-OpenAI-family providers have been migrated; it would likely hard-code the wrong abstraction.

## Verification Strategy

For each future provider doctor:

- Focused provider diagnostics tests with no live credentials
- Cmd invocation tests for flag parsing and JSON output
- Fake-server smoke tests where the provider uses HTTP
- `--print-request` tests proving no network call and redacted auth
- Failure tests for missing auth / invalid endpoint / missing model
- Live smoke Makefile target only when the provider has a stable credential story

For contract migration:

- Keep old provider-specific JSON fields unless explicitly versioning a breaking change
- Add common fields without removing existing fields
- Update docs before adding broad provider support
