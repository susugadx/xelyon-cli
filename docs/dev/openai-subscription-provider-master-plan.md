# XELYON openai_subscription Provider Master Plan v2

この文書は Codex Goal で `openai_subscription` provider を実装するための内部実装仕様書である。
公開 docs ではなく、実装前の壁打ち、risk gate、handoff の source of truth として使う。

作成日: 2026-06-07
v2 更新日: 2026-06-07

## 0. Source Of Truth

この master plan は実装方針の source of truth である。

Phase 0 の live probe 詳細は以下に残す。

```text
docs/dev/openai-subscription-phase0-reality-check.md
```

Phase 0 doc の役割:

- `originator=xelyon` browser OAuth / endpoint smoke の evidence log。
- `store=true`、`previous_response_id`、`context_management`、tool smoke の probe 手順と結果の記録。
- probe command shape、redaction contract、wire-shape discovery の記録。

この master plan の役割:

- Phase 0 結果を踏まえた full provider 実装方針。
- 実装 owner、非目的、runtime contract、test / doctor / docs / final gates。
- Goal に渡す handoff prompt。

実装者は Phase 0 doc を evidence として参照してよいが、実装判断はこの master plan v2 を優先する。

## 0.1 Implementation Tooling Contract

Goal 実装者は manual file edits に `apply_patch` を使う。

Do not use:

```text
str_replace / stella replace style bulk edits
ad hoc shell redirection for source rewrites
```

理由:

- 大きい provider/runtime 変更では、意図しない置換や周辺差分を避ける。
- review 時に hunk 単位で owner / source of truth を追えるようにする。
- sensitive redaction / auth / request builder 変更で accidental rewrite を避ける。

## 0.2 Retired Phase 0 Probe Boundary

The original standalone Phase 0 probe was a live probe artifact.
It has been retired after the provider, auth CLI, and doctor smoke checks were implemented.
It is not kept in the production tree.

Historical role:

- reproduce OAuth / endpoint / retention / tool smoke evidence.
- document wire-shape discovery.
- support local diagnostics during plan validation.

Not allowed:

- Do not wire it into `xelyon` commands.
- Do not reuse it as provider runtime code.
- Do not put auth/token store production logic in `tools/`.
- Do not treat probe success as a substitute for provider tests.
- Do not keep production behavior that only works through the probe binary.

Production implementation owners must live under the normal XELYON code paths:

```text
cmd/
internal/api/providers/openai/
internal/api/providers/openai_responses/
internal/agent/
internal/config/
internal/llmcatalog/
internal/history/
docs/
```

If the probe is kept in the final branch, it must remain clearly documented as a dev/probe tool.
If it becomes stale or conflicts with production implementation, remove it or move the evidence into `docs/dev/openai-subscription-phase0-reality-check.md`.

## 1. 結論

`openai_subscription` は、OpenAI Platform API provider の subscription 版ではない。

v2 の実装方針は以下。

```text
OpenCode 型の ChatGPT/Codex OAuth subscription provider
  + honest originator=xelyon
  + Codex-backed Responses-shaped endpoint
  + store=false
  + full provider-facing payload
  + prompt_cache_key
  + streaming
  + XELYON tool loop
  + XELYON local history reduction / compression
```

使わないもの:

```text
responses.store=true
previous_response_id safe chain
invalid previous_response_id runtime recovery
context_management / server compaction
```

`previous_response_id` が使えないため、OpenAI API provider と同等の server-side response chain optimization は提供しない。
代わりに、XELYON 側の provider-facing history reduction、stable prefix、prompt cache design を主役にする。

## 2. Phase 0 Evidence Summary

Phase 0 probe で確認済み:

```text
originator=xelyon browser OAuth: OK
token exchange: OK
subscription endpoint text smoke: OK
prompt_cache_key: enabled
streaming: required and enabled
instructions: required
store=true: unsupported_required_false
previous_response_id: unsupported
invalid previous_response_id retry probe: possible as full payload retry
context_management: not_applicable_previous_response_id_unsupported
function_call: enabled
function_call_output continuation: enabled via full payload fallback
```

実装方針への反映:

- production runtime は常に `store=false`。
- production runtime は `previous_response_id` を送らない。
- production runtime は `context_management` を送らない。
- doctor は `store` / `previous_response_id` / `context_management` を enabled と表示しない。
- `prompt_cache_key`、streaming parser、tool/function call parser、usage parser は使う。
- tool continuation は full payload 形式で送る。

## 3. OpenCode / Codex Source Evidence

OpenCode / official Codex は、ChatGPT/Codex OAuth で Codex-backed backend を使う先例として参考にする。
ただし XELYON runtime が OpenCode / Codex の identity を名乗ってはいけない。

確認済みの要点:

- OpenCode built-in OpenAI/Codex plugin は `https://chatgpt.com/backend-api/codex/responses` を使う。
- OpenCode は `originator=opencode` を使う。
- official Codex は default `originator=codex_cli_rs` を使う。
- OpenCode は OpenAI Responses protocol を使い、OpenAI 系に `store=false` を default で設定する。
- OpenCode は `promptCacheKey = sessionID` を入れる。
- OpenCode の recorded tool-loop fixture は `store=false`、`previous_response_id=null`、`prompt_cache_key`、streaming、function call、usage を示す。
- OpenCode は `store=false` 時に reasoning / tool call / function_call_output を full payload 側へ戻す設計を持つ。

XELYON への反映:

- OpenCode 型の full payload + prompt cache strategy は参考にする。
- `originator=opencode` fallback は入れない。
- `originator=codex_cli_rs` fallback は入れない。
- OpenCode auth cache / Codex auth cache は読まない。

## 4. Gate Policy

### Gate A: OSS / Terms / Release Gate

技術的には `originator=xelyon` で入口は通った。
ただし third-party OSS CLI が direct subscription backend provider として公開してよいという明示許可は、この計画書では証明しない。

Gate A の扱い:

- private dogfood 実装として進めることは技術的に可能。
- OSS mainline に入れるかは project policy / Terms risk の判断が必要。
- public docs では「OpenAI API をサブスクで無料利用」と書かない。
- OpenAI Platform API provider の代替や料金回避として表現しない。

### Gate B: Identity / OAuth Gate

Phase 0 で `originator=xelyon` browser OAuth と endpoint smoke は OK。

runtime contract:

- OAuth authorize URL / request header は honest `originator=xelyon`。
- OpenCode / Codex identity fallback は禁止。
- `OPENAI_API_KEY` は使わない。

### Gate C: Endpoint Compatibility Gate

Phase 0 で以下の互換性分類が確定。

```text
supported:
  prompt_cache_key
  streaming
  function_call
  function_call_output continuation
  usage parsing shape

unsupported:
  store=true
  previous_response_id
  context_management via previous_response_id
```

Gate C の結果により、v2 provider は server-side chain optimized provider ではなく、full payload provider として実装する。

## 5. Purpose

XELYON に `openai_subscription` provider を追加する。

この provider は:

- ChatGPT/Codex OAuth token で subscription backend に送る experimental provider。
- OpenAI Platform API key provider ではない。
- ChatGPT subscription billing 表示を行う。
- API cost は `N/A` または `pricing unavailable` と表示する。
- 個人 dogfood 向き。
- CI / shared server / production automation 向け stable provider ではない。

この provider の価値:

- ChatGPT/Codex subscription entitlement を XELYON の harness から使える。
- XELYON の tool loop、history reduction、active context、provider-facing payload control を活かせる。
- `prompt_cache_key` と stable prefix で full payload 方式の性能劣化を抑える。
- OpenCode 型の subscription route を、XELYON の履歴最適化に載せる。

## 6. Non-goals

この実装では以下はやらない。

- OpenAI Platform API key を `openai_subscription` で受ける。
- OpenAI API provider と同等の `store=true` / `previous_response_id` chain を主張する。
- `previous_response_id` を subscription runtime で送る。
- `context_management` を subscription runtime で送る。
- Codex app-server 経由。
- Codex CLI / Codex MCP / Codex SDK を外部 agent として呼ぶ。
- OpenCode auth cache の読み取り。
- Codex auth cache の読み取り。
- `~/.codex/auth.json` の読み取り。
- `~/.config/opencode` の読み取り。
- `originator=opencode` fallback。
- `originator=codex_cli_rs` fallback。
- OpenAI Platform API 料金としての cost 表示。
- legacy model 対応。
- CI / shared server 向け stable provider 化。

## 7. Provider Identity

Canonical key:

```text
openai_subscription
```

Aliases:

```text
openai-subscription
chatgpt
codex-subscription
```

内部保存、session identity、provider config key は必ず `openai_subscription` に正規化する。

Display:

```text
OpenAI Subscription
```

Billing:

```text
ChatGPT subscription
```

API cost:

```text
N/A
```

## 8. Supported Models

`openai_subscription` の対応モデルはこの 4 つだけ。

```text
gpt-5.5
gpt-5.4
gpt-5.4-mini
gpt-5.3-codex-spark
```

Default:

```text
gpt-5.5
```

Utility / subagent default:

```text
gpt-5.4-mini
```

Model allowlist vs account entitlement:

- The allowlist means "XELYON knows how to build subscription Responses requests for this model".
- It does not guarantee that the logged-in ChatGPT/Codex account can use every listed model.
- `gpt-5.3-codex-spark` may be subscription-plan / account-entitlement gated.
- Do not hard-code Plus/Pro/Team/Enterprise plan names unless an official source or backend response gives a stable machine-readable entitlement.
- If backend rejects an allowed model for account entitlement, report `model access not available for this account` / `subscription entitlement required`, not `unsupported model`.
- Entitlement failures are per-account live capability results and should be surfaced by doctor/model smoke.

Unsupported model error:

```text
model gpt-5.2 is not supported by openai_subscription.
Supported models: gpt-5.5, gpt-5.4, gpt-5.4-mini, gpt-5.3-codex-spark.
Use provider openai if you need OpenAI Platform API / legacy models.
```

Allowed but unavailable model error:

```text
model gpt-5.3-codex-spark is supported by openai_subscription, but this account cannot access it.
Run: xelyon doctor openai-subscription --smoke --model gpt-5.3-codex-spark
Use another supported model or upgrade/change the ChatGPT/Codex subscription account if the backend indicates an entitlement requirement.
```

## 9. Runtime Contract

### 9.1 Request Shape

Subscription request body MUST include:

```text
instructions
stream: true
store: false
prompt_cache_key
full provider-facing input payload
```

Subscription request body MUST NOT include:

```text
previous_response_id
context_management
store: true
```

Headers:

```text
Content-Type: application/json
Authorization: Bearer <access_token>
ChatGPT-Account-Id: <account_id>  // only when available
originator: xelyon
User-Agent: xelyon/<version> (<os> <arch>)
```

Security requirements:

- Existing `Authorization` header must not be preserved.
- `OPENAI_API_KEY` must not be read or used.
- Authorization header and tokens must not appear in debug output.
- Error bodies must be redacted before logging/reporting.

### 9.2 Full Payload Policy

Because `previous_response_id` is unsupported, every turn sends a full provider-facing payload.

This is not a minimal fallback. For `openai_subscription`, full payload construction is the primary optimization surface.
The payload must be built in cache-friendly order.

Core contract:

```text
raw conversation history:
  durable user-visible conversation log
  not mutated for provider replay details

provider-facing projection:
  provider-specific replay view derived from raw history + provider output items
  owns Responses item continuity across turns

request builder:
  converts projection + current turn evidence into Responses JSON

runtime profile:
  selects full-payload replay vs chain-capable behavior
```

`openai_subscription` must not rebuild context by scraping display text only.
It must replay the provider-facing Responses items that are needed for continuation.

Adopt this layout:

```text
A. stable prefix
  1. instructions / developer policy
  2. deterministic tool definitions
  3. persistent project context
  4. reduced history summary

B. dynamic conversation body
  5. recent user / assistant messages
  6. reasoning items / encrypted_content
  7. function_call
  8. function_call_output

C. request-local tail
  9. active context / rehydrate context
  10. current user message
```

Goals:

- Keep stable prefix stable across turns.
- Move high-churn request-local evidence behind stable prefix.
- Preserve XELYON tool loop semantics.
- Avoid server-side chain dependency.

Hard requirements:

- Current user message must be last, except for transport-specific wrappers required by the existing Responses builder.
- Active context / rehydrate context must be request-local tail content.
- Active context must not implicitly update reduced summary.
- Reduced history summary is part of stable prefix.
- Reduced history summary changes only when provider-facing history reduction runs.
- Recent messages grow or rotate behind the stable prefix.
- Tool definitions must be deterministic for the same enabled tool set.
- Tool result / function_call_output continuation must be represented in the dynamic conversation body, not as `previous_response_id`.
- Do not emit `item_reference` values that require server-side stored items.
- Do not mutate raw `Agent.History` / session message history to insert provider-only replay state.
- Do not drop `call_id`, tool name, tool arguments, tool result status, assistant output item order, reasoning item identity, or `encrypted_content` when those fields are available from the parser.

Implementation notes:

- If existing OpenAI Responses builder naturally emits messages in a different order, add a profile-aware payload layout owner rather than patching call sites.
- If active context currently appears before tools or stable summaries in shared paths, adjust the subscription profile path so cache layout is preserved without changing OpenAI API provider behavior.
- If exact ordering cannot be achieved in the first implementation because of a shared builder constraint, report the file/function-level debt and keep `active context before stable prefix` out of production behavior.

### 9.2.1 Provider-Facing Projection Owner

Use a provider-facing projection as the source of truth for full-payload replay.
This is separate from raw conversation history.

Owner split:

```text
raw history owner:
  user-visible conversation/session persistence
  stores XELYON semantic messages and tool results
  does not store provider-only replay fields as display text

parser owner:
  reads Responses output items from streaming/non-streaming backend responses
  preserves known item fields needed by later turns
  reports unsupported/unknown item shapes without panic

provider-facing projection owner:
  stores the provider-facing Responses replay view
  owns append/replace/truncate during provider history reduction
  preserves item order and provider IDs needed for continuation
  treats provider-only opaque data as sensitive provider state

request layout owner:
  places projection items into stable prefix / dynamic body / request-local tail
  enforces no previous_response_id / no item_reference for subscription

runtime profile owner:
  openai_subscription => FullPayload
  openai => existing chain behavior unless profile explicitly enters FullPayload
```

Projection item requirements:

- assistant text output is preserved as provider-facing assistant output, not re-synthesized from display text when richer output exists.
- `function_call` preserves item type, call ID, function name, arguments, status if present, and output order.
- `function_call_output` preserves call ID, output payload, status if present, and relative order after the matching call.
- reasoning items preserve type, ID/status if present, summary/text fields if exposed, and `encrypted_content` when present.
- unknown Responses output items are either losslessly stored as redacted provider metadata for replay, or explicitly classified as unsupported with WARN/FAIL depending on whether continuation breaks.
- active context / rehydrate context is not persisted into the projection baseline.
- provider history reduction replaces the provider-facing projection segment it owns; it does not mutate raw history.

Persistence / reload:

- If XELYON persists sessions across process restarts, provider-facing projection must round-trip through save/load for known Responses replay items.
- Persisted provider-facing opaque fields such as `encrypted_content` are sensitive provider state and must not be printed in logs, status, doctor JSON, or debug previews.
- Do not store provider-facing projection in `config.yaml`.
- Do not store OAuth tokens or account IDs inside provider-facing projection.
- Corrupt or unsupported persisted projection records should fail with a clear recovery path, not silently drop tool continuation state.

Canonical flow:

```text
backend response stream
  -> Responses parser
  -> provider-facing output items
  -> XELYON tool loop / assistant display
  -> provider-facing projection append
  -> next turn request layout
  -> Responses JSON payload
```

Do not implement:

```text
subscription request builder:
  rebuild assistant/tool history from plain text only
```

Prefer:

```text
provider-facing projection:
  replay known Responses items losslessly enough for continuation
```

### 9.2.2 Tool Definition Canonicalization

`openai_subscription` should treat tool definition stability as cache-critical.

Required:

- Emit tools in deterministic order for the same available tool set.
- Keep JSON schema shape stable.
- Keep generated function names stable.
- Keep `strict` / `additionalProperties` handling stable.
- Avoid map iteration order leaking into request JSON.

Recommended implementation:

- Reuse existing XELYON tool definition builder if it already guarantees deterministic output.
- If not, add canonical ordering at the tool definition owner, not in the subscription transport.

Tests should assert that two equivalent tool sets produce byte-equivalent tool JSON under the subscription profile.

### 9.2.3 Active Context Placement

Active context / rehydrate context is request-local evidence.

For `openai_subscription`:

- place active context behind stable prefix;
- do not persist it into reduced summary;
- do not use it to update any server-side chain state;
- do not let active context change `prompt_cache_key` unless the system prompt itself changed.

This mirrors the old chain safety idea in a full-payload world:

```text
old OpenAI API chain mode:
  active context present => do not send previous_response_id

openai_subscription full payload mode:
  active context present => keep it in volatile tail and preserve stable prefix
```

### 9.2.4 Reduced Summary Update Cadence

Reduced history summary is part of stable prefix.

Rules:

- It changes when provider-facing history reduction actually replaces history.
- It should not be rewritten every turn just to include the latest exchange.
- Recent messages carry normal turn-to-turn growth after the summary.
- A successful reduction creates a new stable prefix baseline.

Goal:

```text
stable reduced summary + stable tool definitions + stable instructions
  => high prompt_cache_key value in full payload mode
```

### 9.2.5 Cache Layout Failure Modes

Avoid these:

- active context before tools or summary;
- nondeterministic tool ordering;
- rewriting reduced summary every turn;
- putting current user message before stable summaries;
- serializing equivalent JSON schemas with unstable map order;
- mixing request-local evidence into persistent project context;
- copying subscription-only layout logic into unrelated provider call sites.

### 9.3 Prompt Cache Policy

Treat prompt cache as a profile-level runtime contract, not as an incidental header added by transport code.

`prompt_cache_key` is not a replacement for `previous_response_id`.
It is a routing hint for similar stable prefixes. Actual cache reuse still depends on backend prefix matching.

Adopt the highest-configuration policy:

```text
cache identity owner:
  OpenAI-compatible prompt cache key builder

stable prefix owner:
  OpenAI Responses request layout / provider-facing history projection

retention owner:
  runtime profile capability policy

observability owner:
  doctor cache smoke + redacted request preview
```

Cache key strategy:

- Use existing `BuildPromptCacheKey(model, systemPrompt)` / `openaicompat.BuildPromptCacheKey` unless the profile refactor creates a clearer shared owner.
- Keep the key aligned with existing OpenAI provider behavior.
- Do not use Kimi's session-aware `PromptCacheScope` strategy for `openai_subscription`.
- Do not include OAuth account ID, access token, refresh token, session ID, task ID, current user message, active context, rehydrate context, recent history, tool outputs, or response IDs in the key.
- The key should change when model request name, cwd hash, normalized core system prompt, or normalized project config section changes.
- The key should not change when only volatile request-local tail content changes.

Reason:

```text
same project + same model + same stable system prompt
  => same prompt_cache_key

same key + stable payload prefix
  => best chance of backend prefix cache reuse
```

Stable prefix strategy:

- Keep instructions / developer policy stable.
- Keep deterministic tool definitions near the stable prefix.
- Keep persistent project context and reduced summary stable.
- Keep recent history, reasoning/tool continuation, active context, and current user message behind that prefix.
- Do not let provider-facing history reduction rewrite the reduced summary every turn.
- Do not let active context or rehydrate context modify the cache key.

Retention strategy:

- `openai` keeps existing `prompt_cache_retention: "24h"` behavior.
- `openai_subscription` does not assume `prompt_cache_retention` is accepted.
- Add an explicit subscription cache compatibility smoke for `prompt_cache_retention`.
- If accepted and intentionally enabled by profile/config, send it through the same request builder owner.
- If rejected, omit it for subscription only and keep `prompt_cache_key` enabled.
- Do not add a production request retry loop that sends `prompt_cache_retention`, fails, then retries without it on every normal request.
- Do not persist hidden capability state unless a later design explicitly defines endpoint/model/account scoping, expiry, and invalidation.

Implementation requirements:

- Send `prompt_cache_key` for subscription requests.
- Keep `prompt_cache_key` generation in a shared OpenAI-compatible owner.
- Keep tool definitions in deterministic order.
- Keep stable instructions deterministic.
- Do not include volatile active context before stable prefix.
- Omit `prompt_cache_retention` for subscription unless capability policy explicitly enables it.
- Verify `prompt_cache_key` appears in request tests.
- Verify `prompt_cache_retention` is omitted/enabled according to subscription profile policy.
- Verify cache layout diagnostics do not print raw prompt, tokens, account ID, or Authorization.

Doctor / diagnostics should report:

- `prompt_cache_key`: enabled / rejected / omitted by policy.
- `prompt_cache_retention`: enabled / unsupported / omitted by policy / unknown.
- stable prefix digest.
- volatile tail digest.
- key changed reason when available.
- cached input tokens if usage includes them.

These diagnostics must use hashes, counts, and status strings. They must not print provider-facing prompt bodies by default.

Cache hit observability:

```text
request-level evidence:
  prompt_cache_key was sent
  stable prefix digest stayed the same
  request was accepted

response-level evidence:
  usage.input_tokens / output_tokens / total tokens when returned
  usage cached input tokens when returned
  usage reasoning tokens when returned

not evidence:
  prompt_cache_key present by itself does not prove a cache hit
  repeated request by itself does not prove a cache hit
  cached tokens absent/zero does not prove cache is broken
```

Doctor should report cache hit status as:

```text
cache_hit: observed
  cached input tokens > 0 in usage

cache_hit: not_observed
  usage returned but cached input tokens are zero

cache_hit: unknown
  usage or cached-token detail is absent
```

`--cache-smoke` should run at least two equivalent stable-prefix requests when live credentials are available.
It should compare stable prefix digest / prompt_cache_key / usage fields, then report the above status.
Do not make Gate OSS-A depend on observing cached tokens, because backend cache warmup and usage detail reporting are not fully controlled by XELYON.

### 9.4 History Reduction Policy

Provider-facing history reduction becomes more important for `openai_subscription` than for chain-enabled OpenAI API.

Required:

- Reuse existing provider-facing history reduction owner.
- Do not create a subscription-only history builder.
- Reduction should produce stable prefix where possible.
- Active context / rehydrate context remains request-local and should not be treated as persistent conversation state.
- Since no `previous_response_id` is sent, chain-disabled context markers should be no-op for subscription runtime but still respected by shared code paths.
- Do not rewrite reduced summary on every turn.
- Keep recent messages behind the reduced summary.

### 9.4.1 Compact API Policy

Distinguish three compression mechanisms:

```text
server_compaction:
  context_management.compaction on previous_response_id chain requests
  not applicable to openai_subscription v2

Compact API:
  /responses/compact style endpoint
  converts full input into compacted input items
  may be usable for subscription only if the subscription backend exposes an honest OAuth endpoint

provider_history_reduction:
  local provider-facing projection reduction
  always available as the subscription fallback / baseline
```

For `openai_subscription`, do not use the OpenAI Platform Compact API:

```text
POST https://api.openai.com/v1/responses/compact
Authorization: Bearer <OPENAI_API_KEY>
```

This provider must not use `OPENAI_API_KEY`.

Highest-configuration target:

```text
openai_subscription compression priority:
  1. subscription Compact API, using the live-verified ChatGPT/Codex compact endpoint when available
  2. provider-facing projection reduction
  3. normal local summary fallback where existing XELYON compression flow requires it
```

Subscription Compact API gate:

- Probe an endpoint owned by the subscription profile, not the OpenAI Platform API profile.
- Default endpoint is `https://chatgpt.com/backend-api/codex/responses/compact`, verified with OAuth + `originator=xelyon` on 2026-06-08.
- This remains an experimental ChatGPT/Codex subscription backend endpoint, not a public OpenAI Platform API contract.
- `XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT` can override it for diagnostics and can disable compact smoke by being explicitly set to an empty string.
- Request must use ChatGPT/Codex OAuth bearer via the same subscription auth/transport/redaction owner.
- Request must set honest `originator=xelyon`.
- Request must not use `OPENAI_API_KEY`.
- Request must not read OpenCode/Codex auth caches.
- Request/response debug output must redact auth, account IDs, compacted encrypted data, and token-like strings.

If subscription Compact API passes:

- expose `SupportsCompact()` for `openai_subscription`.
- implement `CompactHistory` through a subscription compact profile/transport, not by reusing OpenAI API-key transport.
- preserve raw conversation history if the plan's provider-facing projection contract requires raw history immutability.
- store compacted output as provider-facing projection / compacted input state.
- keep compacted output out of `config.yaml`.
- ensure compacted output round-trips through session persistence if needed for resume.
- after successful compaction, clear/disable any stale response-id chain state. For subscription this should be a no-op because no chain exists, but tests should lock it down.

If subscription Compact API fails or is absent:

- doctor reports `compact_api: unsupported` / WARN, not FAIL for Gate OSS-A.
- runtime uses provider-facing projection reduction / local summary fallback.
- do not retry every normal request against a known-unsupported compact endpoint.
- do not route to OpenAI Platform Compact API as fallback.

Compatibility with existing compression UX:

- User-facing trigger should continue to feel like existing auto-compress: `compression.enabled`, `trigger_percent`, `keep_recent`, and provider/model thresholds remain the trigger surface.
- Implementation action differs by profile capability: OpenAI API may use Platform Compact API; subscription may use subscription Compact API if proven; otherwise local projection reduction.
- Users should not need to understand `previous_response_id` to get compression behavior.

Doctor / smoke:

- Add `xelyon doctor openai-subscription --compact-smoke`.
- `--compact-smoke` sends a minimal compact request only when logged in.
- It verifies endpoint reachability, OAuth transport, model allowlist, compact output shape, usage if returned, and redaction.
- It classifies compact support as enabled / unsupported / unknown.
- Compact support is an optimization capability, not a Gate OSS-A blocker.

Tests must cover:

- subscription never calls `https://api.openai.com/v1/responses/compact`.
- subscription compact request uses OAuth bearer and `originator=xelyon`.
- `OPENAI_API_KEY` does not affect subscription compact request.
- compact endpoint override/probe can target httptest.
- compact output is stored in provider-facing projection / compacted state without mutating raw history when raw history immutability is required.
- compacted state save/load preserves known input item fields.
- debug/doctor JSON redacts compacted encrypted data and auth metadata.
- compact failure falls back to provider-facing reduction without adding Platform API fallback.

### 9.4.2 Compact Runtime Implementation Addendum

This addendum is the source of truth for wiring the live-verified subscription Compact endpoint into the actual XELYON compression runtime.

Implemented state after this addendum:

```text
doctor openai-subscription --compact-smoke:
  implemented
  live verified with OAuth + originator=xelyon
  sends to https://chatgpt.com/backend-api/codex/responses/compact

openai_subscription runtime:
  implements api.CompactCapable
  /compress --compact uses subscription Compact API
  auto-compress can prefer subscription Compact API when compression.prefer_compact_api=true
```

Runtime contract:

```text
openai_subscription implements api.CompactCapable:
  SupportsCompact()
  CompactHistory(ctx, input, model, instructions)

Existing XELYON compression orchestration:
  Agent.compressWithCompactAPI()
  /compress --compact
  auto-compress prefer_compact_api

should use subscription Compact API when the current provider is openai_subscription.

Live verification on 2026-06-08:
  doctor openai-subscription --compact-smoke: OK
  compact route: subscription_compact
  compact endpoint: https://chatgpt.com/backend-api/codex/responses/compact
```

Implementation contract:

- `SubscriptionProvider.CompactHistory` must use the subscription OAuth transport.
- It must send `Authorization: Bearer <ChatGPT/Codex OAuth access token>`.
- It must set honest `originator=xelyon`.
- It must set the XELYON User-Agent.
- It may set `ChatGPT-Account-Id` only when account id is available.
- It must not use `OPENAI_API_KEY`.
- It must not call `https://api.openai.com/v1/responses/compact`.
- It must not fall back to OpenAI Platform Compact API.
- It must not read OpenCode or Codex auth caches.
- It must use `XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT` only as the subscription compact endpoint override.
- If `XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT` is explicitly empty, `SupportsCompact()` should return false and manual `/compress --compact` should report unsupported.
- If the endpoint is configured to OpenAI Platform Compact API, `CompactHistory` must fail with a clear forbidden-endpoint error.
- `CompactHistory` should default the model through the existing subscription compression model policy (`gpt-5.4-mini`) when caller model is empty.
- `CompactHistory` should validate the selected model with the subscription allowlist before sending.
- Compact output must be returned as `api.CompactResponse` without lossy remapping beyond the existing `api.InputItem` representation.
- Runtime storage remains owned by existing `Agent.compressWithCompactAPI`; subscription provider must not write session history directly.
- Existing session compacted-state save/load remains the persistence owner.
- `store=false`, `previous_response_id` omitted, and `context_management` disabled remain unchanged for normal subscription Responses requests.
- A successful compact call must not create or update any response-id chain state. For subscription this is a no-op, but tests should lock the behavior down.

Refactor requirement:

- Split the current compact smoke helper into:
  - a reusable subscription compact request runner used by `CompactHistory`;
  - a small doctor/probe wrapper that supplies diagnostic input and classifies WARN/OK.
- Keep endpoint validation, OAuth request preparation, HTTP error redaction, and response decoding in the subscription provider package.
- Do not duplicate compact request construction in doctor code.

Doctor/report alignment:

- `doctor openai-subscription --compact-smoke` should keep working.
- Compact smoke request/result should identify the route as subscription compact, not normal Responses streaming.
- `doctor openai-subscription --print-request --json` should not imply that subscription runtime sends `max_output_tokens`; display omitted subscription-only fields as omitted/present, not raw zero values when that would be misleading.
- `--print-request` must continue to avoid raw prompt body, raw compacted output, token, account id, and OAuth secret disclosure.

Tests for this addendum:

- `SubscriptionProvider` satisfies `api.CompactCapable`.
- `SupportsCompact()` returns true for the default live-verified subscription compact endpoint.
- `SupportsCompact()` returns false when `XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT` is explicitly empty.
- `CompactHistory` sends to the configured subscription compact endpoint.
- `CompactHistory` uses OAuth bearer, `originator=xelyon`, XELYON User-Agent, and optional account header.
- `CompactHistory` ignores `OPENAI_API_KEY`.
- `CompactHistory` refuses OpenAI Platform Compact API endpoint.
- `CompactHistory` defaults empty model to the subscription compression model.
- `CompactHistory` rejects unsupported subscription models before sending.
- `CompactHistory` decodes compact output and usage.
- `CompactHistory` redacts token-like values in HTTP error bodies.
- `runSubscriptionCompactProbe` reuses the runtime runner rather than building a separate request path.
- `Agent.CompressWithCompactAPI` works with `openai_subscription` and stores compacted output through existing compacted-state owner.
- Session save/load preserves compacted rich `api.InputItem` fields used by subscription Compact output.
- Existing OpenAI API provider Compact API tests still pass.

Verification for this addendum:

```bash
go test ./internal/api/providers/openai
go test ./internal/agent ./internal/history ./cmd
go test ./...
go run . doctor openai-subscription --compact-smoke
go run . doctor openai-subscription --smoke --cache-smoke --thinking-smoke --tool-smoke
```

Manual runtime check:

```text
Start XELYON with provider openai_subscription, build enough history, then run:
  /compress --compact

Expected:
  compression uses subscription Compact API
  compacted state is stored locally
  subsequent request sends compacted input through normal full-payload subscription request
  no OpenAI Platform API key is used
```

### 9.5 Reasoning / Encrypted Content

OpenCode store=false path preserves reasoning items with encrypted state in the full payload.
XELYON should implement this as a shared OpenAI Responses full-payload mode, not as subscription-only logic.

Adopted policy:

```text
OpenAI Responses full-payload mode:
  parser reads provider output items as losslessly as practical
  providerhistory projection keeps provider-facing items needed by the next turn
  request builder replays full provider-facing items when store=false
  profile controls whether item_reference / previous_response_id may be used
```

Responsibility split:

```text
parser:
  reads response output items
  preserves reasoning, encrypted_content, function_call, assistant text, usage

providerhistory projection:
  owns what provider-facing items survive into the next turn
  stores projected provider-facing items, not raw user history mutation

request builder:
  in full-payload mode, replays full provider-facing items
  in chain mode, may use item_reference / previous_response_id where existing OpenAI API behavior allows it

runtime profile:
  openai can keep existing store/chain behavior
  openai_subscription is full-payload mode fixed
```

For `openai_subscription`:

- preserve reasoning items if parser exposes them;
- preserve `encrypted_content` if present;
- preserve `function_call`;
- preserve `function_call_output`;
- preserve assistant text;
- do not emit `item_reference`;
- do not emit `previous_response_id`;
- do not depend on server-side stored items.

For `openai`:

- existing chain behavior must not regress.
- `store=false` OpenAI API path may benefit from the same full-payload replay mode.
- invalid chain retry can use the shared full-payload path when it already has enough provider-facing payload.

Do not implement:

```text
if provider == openai_subscription {
  special-case reasoning replay
}
```

Prefer:

```text
if profile.PayloadMode == FullPayload {
  replay full provider-facing Responses items
}
```

Tests must cover:

- full-payload mode replays reasoning item with `encrypted_content`.
- full-payload mode replays `function_call`.
- full-payload mode replays `function_call_output`.
- full-payload mode does not emit `item_reference`.
- full-payload mode does not emit `previous_response_id`.
- chain-enabled `openai` behavior remains unchanged.

### 9.6 Thinking / Reasoning Request Policy

`/thinking` is the user-facing control surface for request-time reasoning effort.
It is separate from model selection and separate from the provider-facing replay of `reasoning` items described above.

Supported UI values remain the existing XELYON values:

```text
/thinking on
/thinking off
/thinking low
/thinking medium
/thinking high
/thinking xhigh
```

For `openai_subscription`, use the shared OpenAI Responses reasoning request builder.
Do not add a subscription-only request builder for thinking.

Policy:

| Model | `/thinking off` | `/thinking low/medium/high/xhigh` |
| --- | --- | --- |
| `gpt-5.5` | omit `reasoning` | send `reasoning.effort` when enabled and smoke-gated |
| `gpt-5.4` | omit `reasoning` | send `reasoning.effort` when enabled and smoke-gated |
| `gpt-5.4-mini` | omit `reasoning` | send `reasoning.effort` when enabled and smoke-gated |
| `gpt-5.3-codex-spark` | use Codex-required fallback only if the shared Codex model policy requires it | send `reasoning.effort` when enabled and smoke-gated |

`gpt-5.3-codex-spark` must not get an ad hoc subscription-only fallback.
If the existing OpenAI Responses path treats Codex-family models as requiring low reasoning when thinking is off, expose that through the runtime profile/model policy and test it there.

Non-blocking capability classification:

- Do not assume the subscription endpoint accepts every `reasoning.effort` level.
- `--thinking-smoke` classifies whether the selected model and configured thinking level are accepted.
- Unsupported enabled thinking levels are not an OSS release blocker as long as `/thinking off` works for basic text/tool use.
- If `reasoning` is rejected for a non-Codex model, `/thinking off` should still work if the basic text smoke works.
- If a requested thinking level is rejected, runtime must report an unsupported thinking-level error for that request and doctor should WARN; do not silently downgrade `xhigh` to `high`, `medium`, or `low`.
- If a Codex-required fallback is needed, it must be an explicit shared model policy, not a hidden subscription fallback.

Status / docs:

- `/thinking` remains selectable from the existing thinking UI/command.
- `openai_subscription` docs must say thinking is supported only to the extent the subscription endpoint accepts the corresponding Responses `reasoning.effort` payload.
- Usage may display reasoning tokens if usage includes them.
- Cost remains `N/A (ChatGPT subscription)`.

Tests must cover:

- `/thinking off` omits `reasoning` for `gpt-5.5`, `gpt-5.4`, and `gpt-5.4-mini`.
- `/thinking low/medium/high/xhigh` emits the expected `reasoning.effort`.
- `gpt-5.3-codex-spark` uses only the shared Codex-family fallback policy when thinking is off.
- unsupported thinking level or endpoint rejection is reported without silent downgrade.
- `--print-request` shows redacted request shape and may show `reasoning.effort`, but never auth/token/account data.
- existing `openai` provider thinking behavior remains unchanged.

## 10. Runtime Profile

Do not fork the entire `openai` provider.

Create or extend a Responses runtime profile that can vary:

- provider key
- display name
- endpoint
- auth strategy
- model allowlist
- store/previous/context policy
- cost family
- login/credential status

Example profile shape:

```go
type responsesRuntimeProfile struct {
    ProviderKey  string
    DisplayName  string
    DebugName    string

    ResponsesURL func() string
    PrepareRequest func(ctx context.Context, url string, payload []byte) (*http.Request, error)
    CompactURL func() string
    PrepareCompactRequest func(ctx context.Context, url string, payload []byte) (*http.Request, error)

    ModelCatalogProviderKey string
    ConfigProviderKey       string
    CostFamily              string

    SupportsCompletions bool
    SupportsResponses   bool

    PayloadMode                responsesPayloadMode
    PromptCacheKeyPolicy       responsesPromptCacheKeyPolicy
    PromptCacheRetentionPolicy responsesPromptCacheRetentionPolicy
    StablePrefixPolicy         responsesStablePrefixPolicy
    CompactPolicy              responsesCompactPolicy
    ThinkingPolicy             responsesThinkingPolicy
    StorePolicy                responsesStorePolicy
    PreviousResponseIDPolicy   responsesPreviousResponseIDPolicy
    ContextManagementPolicy    responsesContextManagementPolicy
}
```

`openai` profile:

```text
ProviderKey: openai
ResponsesURL: https://api.openai.com/v1/responses
CompactURL: https://api.openai.com/v1/responses/compact
auth: OPENAI_API_KEY bearer
SupportsCompletions: true
SupportsResponses: true
PayloadMode: auto / existing behavior
PromptCacheKeyPolicy: existing OpenAI-compatible key
PromptCacheRetentionPolicy: existing 24h behavior
StablePrefixPolicy: existing OpenAI request builder behavior
CompactPolicy: existing OpenAI Platform Compact API behavior
ThinkingPolicy: existing OpenAI Responses reasoning behavior
StorePolicy: existing config
PreviousResponseIDPolicy: existing safe chain
ContextManagementPolicy: existing behavior
CostFamily: openai
```

`openai_subscription` profile:

```text
ProviderKey: openai_subscription
ResponsesURL: https://chatgpt.com/backend-api/codex/responses
CompactURL: https://chatgpt.com/backend-api/codex/responses/compact
auth: ChatGPT/Codex OAuth bearer
SupportsCompletions: false
SupportsResponses: true
PayloadMode: full payload
PromptCacheKeyPolicy: existing OpenAI-compatible key
PromptCacheRetentionPolicy: omit by default / enable only by explicit capability policy
StablePrefixPolicy: full payload stable prefix
CompactPolicy: live-verified subscription compact endpoint; smoke-gated diagnostics; never OpenAI Platform API fallback
ThinkingPolicy: OpenAI-compatible reasoning request builder, subscription endpoint smoke-gated
StorePolicy: force false
PreviousResponseIDPolicy: disabled
ContextManagementPolicy: disabled
CostFamily: openai_subscription
```

### 10.1 Request Transport / PrepareRequest / Debug Redaction

Request transport is the boundary between provider-facing JSON payload and credential-bearing HTTP request.

Do not mix these responsibilities:

```text
OpenAI Responses builder:
  builds JSON payload
  does not know OAuth tokens
  does not know OPENAI_API_KEY

Responses runner:
  serializes payload
  delegates request construction to profile PrepareRequest
  applies shared retry/error handling

Compact runner:
  serializes compact payload
  delegates request construction to profile PrepareCompactRequest
  never falls back from subscription compact to OpenAI Platform compact

profile PrepareRequest:
  constructs http.Request
  sets endpoint-specific headers
  owns auth header injection

profile PrepareCompactRequest:
  constructs compact http.Request
  uses the same auth/redaction owner as the profile
  is separate from Responses request only because the endpoint and request shape differ

subscriptionauth:
  loads / refreshes OAuth token
  returns access token + optional account ID metadata

redaction owner:
  redacts headers, token-like strings, account IDs, OAuth callback/device secrets
```

Runner hook:

```go
type responsesRequestRunOptions struct {
    URL          string
    BuildRequest func() ResponsesRequest

    PrepareRequest func(ctx context.Context, url string, payload []byte) (*http.Request, error)

    HasPreviousResponseID   func() bool
    ClearPreviousResponseID func()

    Debug       bool
    DebugWriter io.Writer
    DebugName   string
}
```

Required runner behavior:

```go
if options.PrepareRequest != nil {
    req, err = options.PrepareRequest(ctx, options.URL, payload)
} else {
    req, err = openaicompat.NewBearerJSONBytesRequest(ctx, options.URL, p.APIKey, payload)
}
```

Rules:

- `PrepareRequest == nil` preserves existing `openai` behavior.
- `openai` profile may keep API-key bearer request behavior.
- `openai_subscription` profile must always provide `PrepareRequest`.
- `PrepareRequest` must not mutate payload bytes.
- `PrepareRequest` must not decide prompt/history/store policy.
- `PrepareRequest` must not read `OPENAI_API_KEY`.
- `PrepareRequest` must not preserve any caller-provided `Authorization`.
- token refresh is delegated to `subscriptionauth`; transport should not duplicate refresh policy.

Subscription headers:

```text
Content-Type: application/json
Authorization: Bearer <access_token>
ChatGPT-Account-Id: <account_id>  // only when account ID exists
originator: xelyon
User-Agent: xelyon/<version> (<os> <arch>)
```

Header requirements:

- `Authorization` is always rebuilt from OAuth access token.
- `ChatGPT-Account-Id` is omitted if account ID is missing.
- `originator` is `xelyon` in production runtime.
- User-Agent must not identify as OpenCode or official Codex.
- endpoint override may be used for tests/probes, but production default is subscription endpoint.

Runtime error redaction:

- HTTP error body is redacted before returning/logging.
- token exchange / refresh errors are redacted before being wrapped into provider errors.
- invalid JSON / malformed SSE errors must not include raw token-bearing headers.
- debug output must redact before write, not rely on caller discipline.

Doctor / `--print-request` redaction:

- request preview uses redacted headers.
- `Authorization` displays as `Bearer <redacted>`.
- `ChatGPT-Account-Id` displays as masked or `<redacted>`.
- request body preview may show structural fields, but must redact token-like strings.
- cache layout diagnostics show hashes/counts/status, not raw prompt body.
- preview must indicate `store=false`, omitted `previous_response_id`, omitted `context_management`, and subscription endpoint.

Do not implement:

```text
if provider == openai_subscription {
  build body and auth headers in the same helper
}
```

Prefer:

```text
profile builds body policy through shared Responses builder
profile PrepareRequest wraps serialized payload into authenticated HTTP request
```

## 11. Auth

Commands:

```text
xelyon auth openai-subscription login
xelyon auth openai-subscription login --device
xelyon auth openai-subscription status
xelyon auth openai-subscription logout
```

Browser OAuth:

- localhost callback.
- default port: `1455`.
- callback path: `/auth/callback`.
- PKCE S256.
- state verification.
- timeout: 5 minutes.
- success/failure HTML.
- display URL if browser cannot be opened.
- tokens never printed.
- `originator=xelyon`.

Constants:

```go
const (
    subscriptionDefaultIssuer     = "https://auth.openai.com"
    subscriptionDefaultEndpoint   = "https://chatgpt.com/backend-api/codex/responses"
    subscriptionDefaultCompactEndpoint = "https://chatgpt.com/backend-api/codex/responses/compact"
    subscriptionDefaultOAuthPort  = 1455
    subscriptionDefaultClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
    subscriptionDefaultOriginator = "xelyon"
)
```

Env overrides for testing:

```text
XELYON_OPENAI_SUBSCRIPTION_ISSUER
XELYON_OPENAI_SUBSCRIPTION_ENDPOINT
XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT
XELYON_OPENAI_SUBSCRIPTION_CLIENT_ID
XELYON_OPENAI_SUBSCRIPTION_ORIGINATOR
XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR
```

Env override rules:

- endpoint / issuer / client ID / auth dir overrides are for tests, probes, and local diagnostics.
- compact endpoint defaults to the live-verified ChatGPT/Codex subscription backend endpoint, can be overridden for diagnostics, and can be disabled by explicitly setting `XELYON_OPENAI_SUBSCRIPTION_COMPACT_ENDPOINT` to an empty string.
- originator override is allowed only for explicit tests/probes.
- production runtime and Gate OSS-A require `originator=xelyon`.
- doctor must WARN/FAIL if runtime config would use a non-`xelyon` originator.

### 11.0 Credential Boundary Contract

Auth is a credential boundary, not a provider helper.
The implementation must keep OAuth, token storage, request authentication, status rendering, and doctor smoke behavior separated.

Boundary rules:

```text
OAuth login:
  obtains token material
  validates PKCE/state/device flow
  never prints tokens/codes beyond user_code where required

token store:
  persists OAuth token material under ~/.xelyon/auth
  never writes config.yaml
  owns permissions, atomic write, refresh metadata

subscriptionauth:
  loads token store
  refreshes before expiry
  extracts account ID
  returns credential DTO to transport

subscriptiontransport:
  injects Authorization / account / originator / User-Agent headers
  never decides prompt/history/body policy

Responses/Compact request builders:
  build JSON body only
  never know token store internals

status:
  local/read-only auth state
  no surprise network call

doctor live smoke:
  explicit network verification
  may refresh token
  reports redacted capability evidence
```

User-facing command semantics:

| Command | Network by default | Token refresh | Purpose |
| --- | --- | --- | --- |
| `auth openai-subscription status` | no | no | local auth/account/expiry/permission status |
| `auth openai-subscription login` | yes | N/A | browser OAuth login |
| `auth openai-subscription login --device` | yes | N/A | headless device login |
| `auth openai-subscription logout` | no | no | remove only XELYON subscription auth file |
| `doctor openai-subscription` | no unless explicit live flag | no | local diagnostic summary |
| `doctor openai-subscription --smoke` and other smoke flags | yes | yes if needed | live capability proof |

Credential ownership invariants:

- API key credentials and OAuth credentials are different types.
- `openai_subscription` must not accept or fallback to `OPENAI_API_KEY`.
- `openai` provider behavior must remain API-key based.
- OpenCode/Codex auth caches are outside XELYON ownership and must not be read.
- `originator=xelyon` is part of the credential identity. If it stops working, stop rather than impersonating another client.
- All credential-bearing error paths must pass through one redaction owner before display, JSON, logs, debug, or HTML.

Auth error vocabulary:

```text
login required:
  no auth file / logged out

token expired:
  auth file exists but access token is expired and status is local-only

refresh failed:
  runtime or doctor attempted refresh and failed; include login suggestion

permission unsafe:
  auth file or auth directory is too open / symlink / unsafe owner

originator rejected:
  honest originator=xelyon failed; stop condition
```

Do not collapse these into "API key missing".

### 11.1 Auth Threat Model

`openai_subscription` stores ChatGPT/Codex OAuth credentials.
Treat this as a secret boundary first and a provider convenience second.

Primary risks:

- access token / refresh token / id token disclosure.
- OAuth callback query logging.
- device auth `device_code` disclosure.
- account ID disclosure beyond masked display.
- loose auth file permissions.
- accidental use of `OPENAI_API_KEY`.
- accidental read of OpenCode/Codex auth caches.
- originator impersonation fallback.
- debug / doctor / JSON / HTML error output leaking secrets.

Required posture:

- login flow is honest `originator=xelyon`.
- token material is never printed.
- token material is never stored in `config.yaml`.
- auth status reports masked/account-safe metadata only.
- request transport always constructs fresh OAuth Authorization header and never preserves caller-provided Authorization.
- every user-visible or debug error path uses the same redaction contract.

### 11.2 Auth Owner Split

Do not put OAuth or token-store policy in `cmd`.

Recommended owner split:

```text
cmd/auth_openai_subscription.go:
  CLI wiring, flag parsing, user-facing command dispatch

cmd/doctor_openai_subscription.go:
  doctor wiring, render text/JSON, command flags

internal/api/providers/openai/subscriptionauth:
  OAuth browser flow
  OAuth device flow
  token store
  refresh lifecycle
  account ID extraction
  auth status DTO

internal/api/providers/openai/subscriptiontransport:
  subscription request header construction
  OAuth bearer injection
  originator/User-Agent headers

internal/providerdiag:
  shared doctor DTO / smoke / preview helpers where reusable

internal/secretredaction or equivalent:
  shared token/account redaction if existing redaction owners are insufficient
```

`cmd` may format messages, but it must not know token JSON internals beyond stable DTOs.

### 11.3 Browser Login

Browser login requirements:

- bind localhost callback only.
- default port `1455`.
- callback path `/auth/callback`.
- PKCE S256.
- cryptographically random state.
- one-time state validation.
- timeout 5 minutes.
- close callback server after success, failure, or timeout.
- success/failure HTML uses redacted messages only.
- authorization code, code verifier, state, access token, refresh token, id token are never printed.
- if browser cannot be opened, print only the authorization URL needed by the user.
- do not log full callback URL.

Failure handling:

- state mismatch => fail, do not exchange code.
- missing code => fail.
- OAuth error => fail with redacted description.
- token exchange HTTP error => fail with redacted body.
- no access token => fail.
- no refresh token => fail unless OpenAI flow explicitly proves refresh is unavailable and runtime can still operate safely; otherwise stop.

### 11.4 Device Login

Device login is required for headless environments.

Flow:

```text
1. request device authorization
2. print verification URL and user_code
3. do not print device_code
4. poll according to server interval
5. treat 403 / 404 as pending
6. exchange authorization_code + code_verifier for tokens
7. save token store
```

Display rules:

- `verification_uri` / `verification_uri_complete` may be shown.
- `user_code` may be shown because the user needs it.
- `device_code`, `code_verifier`, access token, refresh token, id token must not be shown.

Polling rules:

- respect server interval.
- apply overall timeout.
- handle cancellation.
- 403 / 404 are pending.
- any other status is failure unless the OAuth server's documented response proves otherwise.
- redact all response bodies before reporting.

### 11.5 Auth CLI Semantics

`xelyon auth openai-subscription status` should be local/read-only by default.
It should not refresh token or call the endpoint unless an explicit future flag is added.

Status output:

- logged in / not logged in.
- masked account ID if known.
- token expiry state: valid / expiring soon / expired / unknown.
- auth file path.
- permission status.
- endpoint.
- originator.
- login suggestion when missing or expired.

`logout`:

- deletes only XELYON's `openai_subscription` auth file.
- does not touch OpenCode/Codex auth caches.
- does not edit `config.yaml`.
- should be idempotent.

Stop condition:

- If `originator=xelyon` stops working, do not add `originator=opencode` or `originator=codex_cli_rs` fallback.

## 12. Token Store

Do not store tokens in `config.yaml`.

Default path:

```text
~/.xelyon/auth/openai_subscription.json
```

Permissions:

```text
~/.xelyon                 0700
~/.xelyon/auth            0700
openai_subscription.json  0600
```

Format:

```json
{
  "type": "oauth",
  "access_token": "...",
  "refresh_token": "...",
  "expires_at_unix_ms": 1780000000000,
  "account_id": "...",
  "created_at_unix_ms": 1779996400000,
  "updated_at_unix_ms": 1779996400000
}
```

Requirements:

- access token / refresh token / id token never logged.
- status masks account ID.
- refresh 60 seconds before expiry.
- refresh success updates store.
- refresh failure returns login suggestion.
- concurrent refresh is collapsed by mutex.
- doctor warns/fails on loose file permissions.

### 12.1 Token Store Write Contract

Writes must be atomic and permission-safe.

Required:

- create `~/.xelyon` with `0700` if missing.
- create `~/.xelyon/auth` with `0700` if missing.
- write temp file in same directory.
- temp file permission is `0600`.
- write complete JSON before rename.
- rename atomically over `openai_subscription.json`.
- after write, verify final file permission is `0600`.
- do not follow symlinks for auth file writes.
- reject auth path if parent path is unsafe or not owned as expected by platform constraints.

Preferred:

- fsync temp file before rename where practical.
- fsync parent directory after rename where practical.
- preserve only known JSON fields; do not round-trip unknown secret-bearing fields into logs.

### 12.2 Token Store Read Contract

Reads must be fail-closed for unsafe permissions but user-friendly in messages.

Required:

- missing file => not logged in.
- malformed JSON => FAIL with login suggestion; do not print raw file body.
- loose file permission => doctor WARN/FAIL according to exposure.
- loose directory permission => doctor WARN/FAIL according to exposure.
- symlink auth file => FAIL unless a later explicit design allows it.
- `config.yaml` token fields are ignored and should be reported as unsupported if detected by future migration checks.

### 12.3 Refresh Lifecycle

Refresh owner is `subscriptionauth`, not request transport.

Runtime request flow:

```text
load token store
if access token valid for >60s:
  use access token
else:
  acquire refresh coordination
  re-read token store
  if another process/thread already refreshed:
    use updated access token
  else:
    refresh with refresh_token
    update token store atomically
    use updated access token
```

Concurrency requirements:

- process-local mutex collapses same-process refresh.
- cross-process refresh should use a file lock or atomic re-read strategy.
- failed lock acquisition should not corrupt token file.
- refresh request timeout is bounded.
- cancellation is respected.

Refresh failure:

- do not delete token store automatically unless explicit logout.
- return login suggestion.
- redact response body.
- do not print refresh token.
- status should show expired/needs login without attempting surprise network refresh.

### 12.4 Token Store DTO

Production code should use a domain type that cannot be confused with config.

Required fields:

- `type == "oauth"`.
- `access_token` non-empty for logged-in runtime use.
- `refresh_token` non-empty unless a later explicit design allows access-token-only operation.
- `expires_at_unix_ms`.
- `created_at_unix_ms`.
- `updated_at_unix_ms`.

Optional fields:

- `account_id`.
- `issuer`.
- `client_id`.
- `originator`.

Do not store:

- `id_token` after account ID extraction unless a later explicit design needs it.
- OAuth authorization code.
- PKCE verifier.
- device code.
- user code.

## 13. Account ID Extraction

Prefer `id_token`, then `access_token`.
JWT signature is not verified; parsing is only for account id extraction.

Claims:

```text
chatgpt_account_id
https://api.openai.com/auth.chatgpt_account_id
https://api.openai.com/auth.chatgpt_account_id nested under https://api.openai.com/auth
organizations[0].id
```

Parse failure must not panic.
Missing account ID should warn in doctor but requests may proceed without `ChatGPT-Account-Id`.

Storage rule:

- `id_token` may be parsed in memory during login/refresh.
- store extracted masked-safe `account_id` metadata only.
- do not persist `id_token` unless a later explicit design proves it is needed.
- if a token refresh returns a new account ID, update the store metadata.
- if account ID changes unexpectedly, doctor should WARN and runtime should continue only if token refresh succeeded.

## 14. Provider / Model / Config UI

Integrate with existing provider/model selection.

Required:

- `/provider` picker shows `OpenAI Subscription`.
- `/provider openai_subscription` works.
- alias `/provider openai-subscription` works.
- `/model` shows only 4 supported models.
- `/config` default provider choices include `openai_subscription`.
- `/config` `provider_models.openai_subscription.default_model` can select one of 4 models.
- `/providers` displays auth status as `logged in` / `login required`.
- Missing auth must not be shown as API key missing.
- model picker may show all 4 supported models, but live smoke/status can mark account entitlement as unavailable.
- entitlement-gated model failure must not remove the model from the static allowlist.
- entitlement-gated model failure must not be reported as unsupported model.

Config example:

```yaml
default_provider: openai_subscription

provider_models:
  openai_subscription:
    default_model: gpt-5.5
    max_output_tokens: 0
```

`max_output_tokens: 0` means omit and let backend default.
Do not change existing `openai` provider behavior.

## 15. Usage / Cost

Usage may be shown when backend returns it:

- input tokens
- output tokens
- total tokens if returned
- cached input tokens
- reasoning tokens

Cost must not be OpenAI API pricing.

Usage observability policy:

- Parse and display usage only from backend response fields.
- Do not estimate subscription cost from token counts.
- Do not claim cache hit unless cached input token detail is returned and greater than zero.
- If usage is absent, show usage as unknown / unavailable, not zero.
- If cached input tokens are absent, show cache hit as unknown, not failed.
- If cached input tokens are zero, show cache hit as not observed for that run.

Display:

```text
N/A (ChatGPT subscription)
```

or:

```text
pricing unavailable
```

Do not route subscription usage to `openai` pricing family.

## 16. Doctor

Commands:

```text
xelyon doctor openai-subscription
xelyon doctor openai-subscription --smoke
xelyon doctor openai-subscription --cache-smoke
xelyon doctor openai-subscription --compact-smoke
xelyon doctor openai-subscription --retention-smoke
xelyon doctor openai-subscription --thinking-smoke
xelyon doctor openai-subscription --tool-smoke
xelyon doctor openai-subscription --print-request
xelyon doctor openai-subscription --json
```

### 16.0 Doctor Proof Contract

Doctor is the proof surface for `openai_subscription`.
It must classify capabilities rather than treating every unsupported feature as a broken install.

Classification:

```text
required:
  needed for OSS experimental provider usability
  missing => Gate OSS-A FAIL

optional:
  useful optimization or observability
  missing => WARN / unknown, Gate OSS-A can still PASS

expected_unsupported:
  intentionally unsupported in v2
  unsupported => OK or WARN with "expected" wording
  unexpectedly supported => report changed backend behavior, do not silently change runtime policy

forbidden:
  must never be implemented or emitted
  present => FAIL / stop condition
```

Gate OSS-A decision:

```text
PASS:
  all required capabilities OK
  no forbidden capability/path observed
  expected_unsupported capabilities are reported accurately

WARN:
  required OK, but optional capability missing/unknown/degraded

FAIL:
  any required capability missing/broken
  any forbidden path exists
  docs/report would misrepresent expected_unsupported as enabled
```

Default `doctor openai-subscription` is local-first.
It should not perform surprise network calls or token refresh.
Live capability proof belongs to explicit smoke flags.

Doctor status example:

```text
OpenAI Subscription doctor
Status: OK/WARN/FAIL
Auth: logged in
Account: ****abcd
Endpoint: https://chatgpt.com/backend-api/codex/responses
Billing: ChatGPT subscription
API cost: N/A
Default model: gpt-5.5
Runtime mode: full payload
Responses compatibility:
  prompt_cache_key: enabled
  prompt_cache_retention: omitted by policy / enabled / unsupported
  compact_api: enabled / unsupported / unknown
  stream: required/enabled
  store: unsupported
  previous_response_id: unsupported
  context_management: disabled
  thinking: off / low / medium / high / xhigh / unsupported
  tool_call: enabled
```

`--smoke`:

- token exists.
- refresh possible.
- selected model allowed.
- endpoint reachable.
- one streaming response returns text.
- request uses `store=false`.

`--cache-smoke`:

- verifies `prompt_cache_key` is included.
- verifies `prompt_cache_key` is accepted by the endpoint.
- verifies equivalent stable-prefix requests keep the same cache key.
- verifies active context / volatile tail does not change the key.
- classifies `prompt_cache_retention` as enabled / unsupported / omitted by policy / unknown.
- reports cached input tokens if usage includes them.
- does not fail solely because cached tokens are zero or absent.
- includes redacted layout diagnostics in JSON: stable prefix digest, volatile tail digest, key changed reason.

`--compact-smoke`:

- verifies login/token refresh.
- uses subscription OAuth transport.
- does not use OpenAI Platform API key.
- sends a minimal compact request to the subscription compact endpoint selected by profile/probe policy.
- verifies compact output shape if returned.
- verifies compacted output can be consumed as provider-facing compacted input state.
- reports usage if returned.
- reports unsupported as WARN, not Gate OSS-A failure.

`--retention-smoke`:

- verifies `store=true` remains unsupported or classifies if backend behavior changes.
- verifies `previous_response_id` remains unsupported or classifies if backend behavior changes.
- verifies `prompt_cache_key` is included.
- verifies runtime fallback mode is full payload.

If backend later supports `store=true` / `previous_response_id`, doctor may report it, but production runtime should not silently change to chain mode in the same change. That would be a new design task.

`--thinking-smoke`:

- uses the selected model.
- uses the currently configured thinking level unless an explicit doctor flag is added by the implementation.
- verifies the emitted `reasoning.effort` payload is accepted when thinking is enabled.
- verifies `reasoning` is omitted when thinking is off for non-Codex-family models.
- reports endpoint rejection as unsupported for that model/level.
- does not silently downgrade the requested level.
- reports reasoning tokens if usage includes them.

`--tool-smoke`:

- dummy function call appears.
- tool output is sent as full payload continuation.
- final response completes.

`--print-request`:

Must redact:

- `Authorization`
- `access_token`
- `refresh_token`
- `id_token`
- `ChatGPT-Account-Id`
- `Bearer ...`
- JWT-looking strings

`--json`:

May include:

- provider key / display name.
- auth state: `logged_in`, `login_required`, `token_expired`, `permission_unsafe`.
- masked account ID.
- endpoint and originator.
- billing: `ChatGPT subscription`.
- API cost: `N/A`.
- selected/default model.
- supported model list.
- runtime mode: `full_payload`.
- capability status, severity, and evidence category.
- smoke request result summaries.
- usage counts if returned.
- cache hit status: `observed`, `not_observed`, `unknown`.
- compact status: `enabled`, `unsupported`, `unknown`.
- request preview structural fields with redaction.

Must not include:

- access token.
- refresh token.
- id token.
- Authorization header value.
- full account ID.
- OAuth authorization code.
- PKCE verifier.
- device code.
- raw JWT.
- raw prompt body / active context body.
- compacted encrypted data.
- reasoning `encrypted_content`.
- provider opaque replay state.

`--print-request` body preview:

- may show structural fields such as `model`, `stream`, `store`, `prompt_cache_key` presence, `tools` count, `reasoning.effort`, and omitted `previous_response_id` / `context_management`.
- must not show raw prompt body by default.
- must not show active context contents.
- must not show compacted/opaque provider state.
- should include stable prefix digest / volatile tail digest where useful.

### 16.1 Capability Matrix / WARN-FAIL Policy

Goal:

```text
OSS で普通に使える experimental provider として出す。
ただし OpenAI official support / API-free subscription usage / OpenCode compatibility clone とは表現しない。
```

OpenCode being available is an ecosystem signal, not permission proof.
XELYON's release gate is its own honest `originator=xelyon` flow, its own OAuth/token store, and its own capability smoke results.

Doctor should distinguish three categories:

```text
required for basic usability:
  missing or broken => FAIL

expected unsupported capability:
  unsupported by design / endpoint => OK or WARN, not FAIL

optional optimization:
  missing => WARN or omitted by policy, not basic FAIL
```

Severity matrix:

| Capability | Expected v2 state | Doctor severity |
| --- | --- | --- |
| provider registration | registered | FAIL if missing |
| model allowlist | exactly 4 supported models | FAIL if unsupported selected model |
| model entitlement | live account can access selected allowed model | FAIL for selected model smoke if backend rejects entitlement; does not remove model from static allowlist |
| auth file | present for live use | FAIL with login suggestion if missing |
| token refresh | succeeds or token still valid | FAIL with login suggestion if refresh fails |
| auth file permissions | 0600 file / 0700 dirs | WARN or FAIL depending on exposure |
| `originator=xelyon` OAuth | works | FAIL / stop condition if rejected |
| endpoint text smoke | returns one response | FAIL if unreachable or malformed |
| streaming | required/enabled | FAIL if streaming parser cannot extract text |
| `store=false` | required | FAIL if backend rejects `store=false` request |
| `store=true` | expected unsupported | OK/WARN, never required |
| `previous_response_id` | expected unsupported | OK/WARN, never required |
| `context_management` | disabled / not applicable | OK if disabled |
| full payload mode | required | FAIL if request builder emits chain-only payload |
| `prompt_cache_key` | required for release-quality v2 | FAIL for release gate; WARN at runtime if a later backend regression still allows text smoke |
| `prompt_cache_retention` | omitted by policy unless proven | OK if omitted; WARN if unsupported when probed |
| cached input tokens | best-effort observation | never FAIL solely because zero/absent |
| Compact API | optional optimization, live-verified subscription endpoint | OK if accepted; WARN if later unsupported; FAIL if it uses OpenAI Platform API key/path |
| thinking / `reasoning.effort` | selectable through `/thinking`, non-blocking smoke-gated capability | OK if accepted; WARN if unavailable while off still works; FAIL only if runtime silently downgrades or misreports request behavior |
| tool call | required for coding-agent usefulness | FAIL for `--tool-smoke`; WARN/unknown if not run |
| `function_call_output` continuation | required for tool loop | FAIL for `--tool-smoke` |
| reasoning/encrypted content replay | required when backend emits it | WARN if absent in fixture; FAIL if emitted and dropped/breaks continuation |
| usage parser | best effort | WARN if missing; never show API cost |
| cost family | `openai_subscription` / N/A | FAIL if OpenAI API pricing is shown |
| secret redaction | required | FAIL if token/account/header leaks in debug, status, JSON, or error |
| OpenAI API key use | forbidden | FAIL if `OPENAI_API_KEY` is used |
| OpenCode/Codex auth cache read | forbidden | FAIL if implemented |
| originator fallback | forbidden | FAIL if `opencode` / `codex_cli_rs` fallback exists |

Doctor default command should show current configured state without live network calls where possible.
Live flags should make capability evidence explicit:

```text
doctor openai-subscription:
  local auth/config/model/security status

doctor openai-subscription --smoke:
  basic live usability

doctor openai-subscription --cache-smoke:
  prompt_cache_key / prompt_cache_retention / layout diagnostics

doctor openai-subscription --compact-smoke:
  subscription Compact API compatibility and compacted state shape

doctor openai-subscription --retention-smoke:
  confirms store/previous remain unsupported or reports changed backend behavior

doctor openai-subscription --thinking-smoke:
  reasoning.effort compatibility for the selected model and configured thinking level

doctor openai-subscription --tool-smoke:
  coding-agent tool loop usability
```

Runtime fallback policy:

- If live text request works but `store=true` / `previous_response_id` are unsupported, runtime remains OK because v2 is full-payload by design.
- If `prompt_cache_retention` is unsupported, runtime remains OK because it is optional and profile-gated.
- If subscription Compact API later becomes unsupported, runtime remains OK because provider-facing reduction / local summary fallback is available.
- If `prompt_cache_key` is rejected after release, runtime may continue only if the request succeeds without it, but doctor must WARN loudly that the provider is degraded and no longer satisfies release-quality v2 optimization.
- If a thinking level is unsupported, runtime may continue with `/thinking off` where the selected model allows omission, but it must not silently downgrade the requested level.
- Do not silently change production runtime from full payload to response chain if backend behavior changes.
- Do not silently add impersonation fallback if `originator=xelyon` fails.

OSS docs posture:

- Say "experimental, personal ChatGPT/Codex subscription provider".
- Say "uses ChatGPT/Codex login and subscription backend".
- Say "not OpenAI Platform API and not API pricing".
- Say "full payload mode; server-side response chain unsupported".
- Say "for personal dogfood / local CLI use".
- Avoid saying "officially supported by OpenAI".
- Avoid saying "OpenAI API for free".
- Avoid saying "OpenCode-compatible" as a permission claim.

Release gate for OSS mainline:

```text
Gate OSS-A:
  originator=xelyon OAuth OK
  text smoke OK
  streaming OK
  store=false OK
  prompt_cache_key OK
  compact_api classified as enabled or unsupported without Platform API fallback
  thinking off OK
  enabled thinking level compatibility reported as non-blocking capability
  tool smoke OK
  no secret leaks
  auth file permissions safe
  token store atomic write / refresh lifecycle tests pass
  no API key path
  no OpenCode/Codex cache read
  no impersonation fallback
  cost N/A
  docs accurately describe unsupported store/previous chain
```

If Gate OSS-A passes, the provider can be presented as OSS-usable experimental support.
If Gate OSS-A fails only on optional optimization such as `prompt_cache_retention`, OSS usability still passes with WARN.
If Gate OSS-A fails on originator, endpoint, streaming, prompt_cache_key, tool loop, or secret redaction, stop and report.

## 17. Security

Absolute requirements:

- Do not print access token.
- Do not print refresh token.
- Do not print id token.
- Do not print Authorization header.
- Mask account ID in status/report.
- Do not store tokens in config.
- Do not read `~/.codex/auth.json`.
- Do not read OpenCode auth cache.
- Redact token-like strings from error bodies.
- Redact compacted encrypted data / provider opaque replay state from debug, doctor JSON, status, and errors.
- Tests use dummy tokens only.
- Do not add fallback identities.

Redaction targets:

```text
Bearer <token>
access_token
refresh_token
id_token
authorization code
code_verifier
device_code
state
ChatGPT-Account-Id
JWT-like long strings
acct_* identifiers
compacted encrypted data
reasoning encrypted_content
provider opaque replay state
```

`user_code` from device login may be displayed only inside the interactive device-login prompt.
It should not be included in doctor reports, debug logs, or persistent JSON artifacts unless redacted or explicitly marked as non-secret by the auth owner.

## 18. Tests

### 18.1 Model Tests

- allowed: `gpt-5.5`
- allowed: `gpt-5.4`
- allowed: `gpt-5.4-mini`
- allowed: `gpt-5.3-codex-spark`
- rejected: `gpt-5.3-codex`
- rejected: `gpt-5.2`
- rejected: `gpt-4.1`
- default model is `gpt-5.5`
- utility default is `gpt-5.4-mini`
- known models are exactly the 4 supported models
- allowed model with account entitlement rejection is not reclassified as unsupported model.
- model entitlement diagnostics can mark `gpt-5.3-codex-spark` unavailable for the current account without changing static model allowlist.

### 18.2 Provider Registration Tests

- `openai_subscription` is registered.
- aliases normalize to `openai_subscription`.
- `openai_subscription` does not require `OPENAI_API_KEY`.
- `openai` still requires `OPENAI_API_KEY`.
- missing auth reports login required, not API key missing.
- OAuth credential and API-key credential paths cannot be confused by provider registration.

### 18.3 Auth / Token Tests

- PKCE verifier/challenge generation.
- state verification.
- state mismatch does not exchange code.
- browser callback does not log full callback URL.
- JWT claims account ID extraction.
- invalid JWT does not panic.
- id_token is not persisted after account ID extraction.
- account ID changes after refresh are recorded and doctor warns.
- token store write is 0600.
- auth dir is 0700.
- token store write is atomic.
- token store write rejects symlink auth file.
- malformed token store fails with login suggestion without printing raw file body.
- status is local/read-only by default.
- status does not refresh token or call endpoint by default.
- default doctor does not refresh token or call endpoint without live smoke flags.
- live doctor smoke may refresh token and reports that action without printing token material.
- runtime request refreshes token only through `subscriptionauth`.
- logout is idempotent and deletes only XELYON subscription auth file.
- refresh success updates store.
- refresh failure suggests login.
- refresh failure does not delete token store.
- refresh re-reads store after lock and uses another process/thread's refreshed token when present.
- same-process concurrent refresh collapses to one refresh call.
- cross-process refresh coordination does not corrupt token file.
- device login prints verification URL and user_code.
- device login does not print device_code or code_verifier.
- device login treats 403 / 404 as pending.
- device login treats other statuses as failure.
- logs/status/errors do not leak tokens.
- logs/status/errors do not leak authorization code, device_code, code_verifier, state, or JWT-like strings.
- `login required`, `token expired`, `refresh failed`, `permission unsafe`, and `originator rejected` are distinguishable from API key missing.
- originator override is limited to explicit tests/probes and cannot become production fallback.

### 18.4 Request Tests With httptest

Use endpoint env override or injected endpoint.

- request goes to subscription endpoint.
- subscription profile uses `PrepareRequest`.
- `PrepareRequest == nil` keeps existing OpenAI API-key bearer behavior.
- `PrepareRequest` does not mutate serialized payload bytes.
- Authorization uses OAuth bearer.
- existing Authorization header is not preserved.
- `ChatGPT-Account-Id` set when account ID exists.
- `ChatGPT-Account-Id` omitted when account ID is missing.
- `originator=xelyon`.
- User-Agent identifies XELYON and does not identify as OpenCode/Codex.
- `OPENAI_API_KEY` not used.
- `OPENAI_API_KEY` in environment does not affect subscription request.
- subscription transport delegates refresh to `subscriptionauth`.
- runtime debug output redacts Authorization, bearer tokens, account ID, and token-like body content.
- HTTP error body is redacted before provider error/report/debug output.
- doctor `--print-request` shows redacted headers and subscription endpoint.
- doctor `--print-request` does not include prompt body in cache layout diagnostics.
- subscription compact request, when enabled by smoke-gated policy, goes to subscription compact endpoint, not OpenAI Platform Compact API.
- subscription compact request uses OAuth bearer and `originator=xelyon`.
- subscription compact request ignores `OPENAI_API_KEY`.
- `instructions` present.
- `stream=true`.
- `store=false`.
- `previous_response_id` omitted.
- `context_management` omitted.
- `prompt_cache_key` present.
- `prompt_cache_retention` omitted by default for subscription.
- `prompt_cache_retention` included only when subscription retention policy enables it.
- `/thinking off` omits `reasoning` for non-Codex-family subscription models.
- `/thinking low/medium/high/xhigh` emits matching `reasoning.effort`.
- `gpt-5.3-codex-spark` thinking-off behavior comes from the shared Codex-family model policy, not a subscription-only branch.
- rejected thinking level is surfaced without silent downgrade.
- `max_output_tokens` omitted by default if config is 0.
- tools included.
- tools are emitted in deterministic order.
- function_call_output continuation uses full payload.
- raw conversation history is not mutated to insert provider-only replay state.
- provider-facing projection preserves item order for assistant output, function_call, function_call_output, and reasoning items.
- full-payload mode replays reasoning item with `encrypted_content` when available.
- full-payload mode replays `function_call` and `function_call_output` as provider-facing Responses items.
- full-payload mode does not emit `item_reference`.
- active context appears behind stable prefix.

### 18.5 History / Cache Tests

- provider-facing full payload is sent every turn.
- raw history and provider-facing projection remain separate.
- history reduction output is included in payload.
- provider history reduction replaces only the projection segment it owns.
- stable prefix order is preserved.
- active context is behind stable prefix and request-local.
- active context does not change reduced summary.
- active context does not change `prompt_cache_key` unless system prompt changes.
- rehydrate context does not change `prompt_cache_key`.
- current user message does not change `prompt_cache_key`.
- recent assistant/tool history does not change `prompt_cache_key`.
- session/task `PromptCacheScope` does not change subscription `prompt_cache_key`.
- tool definitions are deterministic.
- equivalent tool definitions serialize byte-equivalently under subscription profile.
- map iteration order does not affect request tool JSON.
- prompt_cache_key is stable for same model/system prompt.
- changing system prompt changes cache key.
- changing model request name changes cache key.
- changing cwd hash input changes cache key.
- changing normalized project config section changes cache key.
- reduced summary changes only when provider-facing history reduction runs.
- recent messages are behind reduced summary.
- current user message is last in the logical provider-facing payload.
- chain-disabled context does not cause `previous_response_id` send.
- chain-enabled `openai` fixture keeps existing `previous_response_id` behavior.
- store=false/full-payload fixture uses the same replay path for `openai` and `openai_subscription`.
- provider-facing projection round-trips known Responses replay items through session save/load when session persistence applies.
- Compact API output round-trips known compacted input items through session save/load when subscription Compact API is enabled.
- corrupt persisted projection records fail with clear recovery instead of silently dropping tool continuation state.
- persisted projection does not include OAuth token, refresh token, account ID, or API key.
- cache layout diagnostics contain hashes/counts/status only, not raw prompt bodies.

### 18.6 Streaming / Tool Tests

- minimal SSE extracts text.
- function_call item maps to XELYON tool loop.
- function_call arguments delta/done handled.
- function_call_output continuation completes.
- function_call_output keeps matching call ID and order after session reload when persistence applies.
- reasoning items with `encrypted_content` do not break streaming/tool parsing.
- reasoning `encrypted_content` is not printed in logs, debug preview, doctor JSON, or status output.
- existing OpenAI provider streaming tests still pass.

### 18.7 Doctor Tests

- no auth file => FAIL with login suggestion.
- unsupported model => FAIL with supported model list.
- allowed model entitlement rejection => FAIL for that live selected model with entitlement/access wording, not unsupported-model wording.
- entitlement rejection for `gpt-5.3-codex-spark` does not fail model allowlist tests.
- `--cache-smoke` classifies `prompt_cache_retention` without enabling `previous_response_id`.
- default doctor reports `store=true` / `previous_response_id` unsupported as expected v2 state, not basic failure.
- default doctor reports `context_management` disabled / not applicable as expected v2 state.
- `--smoke` FAILs if endpoint rejects `store=false`.
- `--smoke` FAILs if streaming text cannot be parsed.
- `--cache-smoke` FAILs release gate if `prompt_cache_key` is rejected.
- `--cache-smoke` does not FAIL solely because cached input tokens are zero/absent.
- `--compact-smoke` reports unsupported compact endpoint as WARN, not Gate OSS-A failure.
- `--compact-smoke` FAILs if subscription compact path uses OpenAI Platform API key/path.
- `--compact-smoke` redacts compacted encrypted data, auth, and account metadata.
- `--thinking-smoke` verifies current selected model and configured thinking level.
- `--thinking-smoke` reports endpoint rejection as unsupported model/level without changing runtime config.
- `--print-request` may show `reasoning.effort` but redacts auth/account/token-like data.
- `--tool-smoke` FAILs if `function_call_output` continuation cannot complete.
- `--print-request` redacts auth.
- `--print-request` shows structural request fields but not raw prompt body, active context body, compacted encrypted data, or provider opaque replay state.
- `--json` contains billing/subscription info.
- `--json` contains no token.
- `--json` cache diagnostics contain no prompt body or active context body.
- `--json` contains no compacted encrypted data, reasoning `encrypted_content`, or provider opaque replay state.
- `--json` represents capabilities with status, severity, and evidence category.
- `--json` cost family is `openai_subscription` / N/A, never OpenAI API pricing.
- retention smoke reports `store` / `previous_response_id` unsupported as WARN.
- tool smoke reports enabled on compatible fixture.
- doctor fails if `OPENAI_API_KEY` is used by subscription transport.
- doctor fails if OpenCode/Codex auth cache read path exists in subscription auth.
- doctor fails if originator fallback exists.
- doctor warns/fails if runtime would use non-`xelyon` originator.
- Gate OSS-A fixture FAILs when any required capability is missing.
- Gate OSS-A fixture WARNs but does not FAIL when optional Compact API, thinking enabled level, cached token observation, usage detail, or `prompt_cache_retention` is unavailable.
- Gate OSS-A fixture treats expected unsupported `store=true`, `previous_response_id`, and `context_management` as expected, not broken.
- Gate OSS-A fixture FAILs if any forbidden path is present even when basic smoke succeeds.
- Gate OSS-A fixture passes when required v2 capabilities are OK and optional `prompt_cache_retention` is omitted/unsupported.

## 19. Docs

Update:

- `README.md`
- `docs/config.md`
- `docs/providers.md` if present
- `config.yaml.example`

Docs must say:

- `openai_subscription` is experimental.
- It is OSS-usable when Gate OSS-A passes.
- Uses ChatGPT/Codex login.
- Not OpenAI Platform API.
- Does not use API key.
- API cost is N/A.
- Intended for personal dogfood.
- Not recommended for CI/shared server/production automation.
- Token is stored in `~/.xelyon/auth/openai_subscription.json`.
- Token file is password-equivalent.
- Supported models are 4 only.
- Supported model list is not an account entitlement guarantee; some models may require a higher subscription/account entitlement and should be verified with doctor smoke.
- Runtime mode is `store=false` / full payload.
- `prompt_cache_key`, streaming, and tool loop are used.
- Compact API is optional and smoke-gated; OpenAI Platform Compact API is not used by `openai_subscription`.
- `/thinking` uses the shared OpenAI Responses `reasoning.effort` payload and is smoke-gated per model/level.
- In the TUI/session command surface, `/thinking high` and other existing values remain selectable.
- Reasoning tokens may be displayed if the backend returns them.
- `prompt_cache_retention` is subscription-profile gated and not assumed.
- `previous_response_id` / server-side chain is not used.
- `store=true` / `previous_response_id` unsupported is expected v2 behavior, not a broken install.
- `doctor openai-subscription --smoke --cache-smoke --compact-smoke --thinking-smoke --tool-smoke` is the recommended local proof of usability.

Usage:

```bash
xelyon auth openai-subscription login
xelyon auth openai-subscription status
xelyon doctor openai-subscription --smoke
xelyon doctor openai-subscription --cache-smoke
xelyon doctor openai-subscription --compact-smoke
xelyon doctor openai-subscription --thinking-smoke
xelyon doctor openai-subscription --retention-smoke
xelyon doctor openai-subscription --tool-smoke
xelyon --provider openai_subscription --model gpt-5.5 "hello"
```

Do not write:

```text
OpenAI API をサブスクで無料利用できる
OpenAI API provider と同等の response chain optimization
OpenAI official support
OpenCode compatible / OpenCode authorized
```

## 20. Implementation Phases

### 20.0 Package Boundary / Owner Map

Do not implement the provider in `tools/`.
Do not put all subscription behavior into one giant `subscription_provider.go`.

Initial production placement should stay close to the existing OpenAI Responses implementation:

```text
internal/api/providers/openai/
  subscription_provider.go
    provider construction, registration entrypoint, profile wiring

  subscription_models.go
    supported model allowlist, defaults, unsupported/entitlement error text

  subscription_profile.go
    responsesRuntimeProfile values, payload/cache/compact/thinking/store policies

  subscription_transport.go
    PrepareRequest, PrepareCompactRequest, headers, OAuth bearer injection

  subscription_auth.go
    auth status DTO, credential facade, refresh entrypoint

  subscription_oauth_browser.go
    browser OAuth login flow

  subscription_oauth_device.go
    device login flow

  subscription_token_store.go
    auth file read/write, permissions, atomic write, locking/re-read

  subscription_compact.go
    subscription Compact API profile/transport/smoke support

  subscription_diagnostics.go
    doctor report assembly and smoke orchestration

  subscription_redaction.go
    only if existing redaction owners are insufficient
```

Shared Responses runtime owners:

```text
internal/api/providers/openai_responses/
  request DTOs, runner, server compaction, response-id chain policy
  profile-aware builder helpers when shared by openai/openai_subscription

internal/api/providers/openai/
  OpenAI API provider-specific profile values
  subscription provider-specific profile values
```

Agent/runtime owners:

```text
internal/agent/
  provider selection UI
  local auto-compress trigger
  Compact API orchestration
  provider-facing projection integration
  session restore integration

internal/providerhistory/ or existing provider-facing history owner:
  projection-only reduction
  raw history immutability
  raw output artifact-backed reduction
```

Config/catalog owners:

```text
internal/llmcatalog/
  provider/model catalog and aliases

internal/config/
  provider_models/defaults/validation/generated registry

docs/ and config.yaml.example:
  user-facing contract and warnings
```

Command owners:

```text
cmd/auth_openai_subscription.go:
  auth command wiring only

cmd/doctor_openai_subscription.go:
  doctor command wiring only

cmd/auth.go / cmd/doctor.go:
  routing integration only
```

Boundary rules:

- Request builders build JSON bodies only.
- Transports build HTTP requests and inject credentials only.
- Auth/token store owns OAuth credential lifecycle only.
- Doctor owns classification/rendering, not production request policy.
- Config/catalog own provider/model availability, not live entitlement.
- Provider-facing projection owns replay state, not raw history mutation.
- Redaction owner is shared by auth, transport, doctor, compact, and runtime errors.

Package split stop conditions:

- auth + transport + doctor + parser + config policy accumulates in one file.
- subscription auth needs tests that require reaching into unrelated OpenAI provider internals.
- redaction logic is duplicated between runtime, doctor, and auth.
- Compact API transport starts sharing API-key code paths with subscription OAuth.
- provider-facing projection requires exporting broad mutable state from `openai` package.
- import-cycle workaround wrappers appear.

If any stop condition appears, pause implementation and run `package-boundary-map` / `package-boundary-refactor` before continuing.

Do not split prematurely:

- Keep early profile wiring near existing OpenAI Responses implementation if it avoids exports and import cycles.
- Prefer file-level owner separation before package split.
- Split packages only when it reduces public surface or prevents wrong dependency direction.

Required generated/docs follow-up:

- config contract changes require `make gen-all`.
- provider/model catalog changes require corresponding model/provider tests.
- docs updates must not claim OpenAI Platform API usage, OpenAI official support, or subscription-free API usage.

### Phase 0: Plan / Foundation

Status: this master plan v2 and Phase 0 probe are the foundation.

Required before implementation:

- Re-read this master plan.
- Re-read Phase 0 doc only for evidence and probe details.
- Confirm current source owner map still matches.
- Do not reintroduce `store=true` / `previous_response_id` as runtime defaults.

### Phase 1: Provider Catalog / Model Allowlist / UI Candidates

- provider key and aliases.
- model allowlist.
- default / utility defaults.
- `/provider` / `/model` / `/config` candidates.
- provider registration stub with login-required auth state.

Run:

```bash
go test ./...
```

### Phase 2: Auth CLI / Token Store

- browser login.
- device login.
- status.
- logout.
- credential boundary DTOs.
- token store.
- refresh.
- atomic write / safe permission.
- same-process and cross-process refresh coordination.
- account ID extraction.
- redaction.
- status local/read-only semantics.
- doctor live-smoke vs local-status separation.
- logout idempotency.

Implementation-time live OAuth coordination:

- Phase 2 is the first phase where live login may be required.
- Before running a browser or device OAuth command, the implementer must clearly announce the exact command and the expected user action.
- Do not start browser/device login implicitly from tests, provider initialization, default doctor, status, or runtime smoke setup.
- Unit tests and `httptest` integration tests must use dummy tokens / local overrides and must not require a real OpenAI subscription login.
- During live verification, the user should confirm:
  - `login` does not print access token, refresh token, id token, authorization code, device code, or code verifier.
  - token file is written to `~/.xelyon/auth/openai_subscription.json`.
  - auth directory permissions are `0700` and token file permission is `0600`.
  - `status` is local/read-only and does not call the network or refresh tokens.
  - runtime requests refresh expired access tokens through `subscriptionauth` without asking for login again.
  - refresh failure returns a login suggestion and does not delete the token store automatically.
  - default doctor does not login or refresh; live smoke flags may refresh and must report that without printing token material.
  - `OPENAI_API_KEY` is not read or used by subscription auth, refresh, request transport, or doctor smoke.
- Phase 2 completion requires the "one explicit login, then refresh" behavior to be established at the auth boundary. Phase 3 and later phases consume that boundary; they must not add their own login prompts or refresh policy.

Run:

```bash
go test ./...
```

### Phase 3: Responses Runtime Profile

- profile abstraction.
- PrepareRequest hook.
- subscription transport.
- transport/request builder owner split.
- compact transport/profile owner split.
- redacted runtime debug output.
- doctor print-request redaction.
- endpoint override.
- `store=false` forced for subscription.
- `previous_response_id` disabled for subscription.
- `context_management` disabled for subscription.
- `ThinkingPolicy` added to runtime profile.
- subscription uses the shared OpenAI Responses reasoning request builder.
- cost family separated.

Run:

```bash
go test ./...
```

### Phase 4: Full Payload / Cache / History Integration

- full provider-facing payload every turn.
- stable prefix ordering.
- prompt_cache_key.
- shared prompt cache key owner.
- prompt_cache_retention profile policy.
- subscription `--cache-smoke`.
- subscription Compact API smoke-gated policy.
- subscription `--compact-smoke`.
- subscription `--thinking-smoke`.
- redacted cache layout diagnostics.
- prompt_cache_retention compatibility smoke or disable-only-subscription.
- provider-facing history reduction integration.
- provider-facing projection owner split.
- raw history / projection separation.
- compacted input state integration when subscription Compact API is enabled.
- projection save/load round-trip for known Responses replay items where session persistence applies.
- active context placement.
- tool result / function_call_output full payload continuation.
- shared OpenAI Responses full-payload mode.
- reasoning/encrypted_content preservation where parser exposes it.
- no subscription-only reasoning/function_call replay special case.

Run:

```bash
go test ./...
```

### Phase 5: Streaming / Tool / Usage / Doctor / Docs

- streaming parser compatibility.
- tool smoke.
- usage parser.
- thinking status / reasoning tokens display where usage includes them.
- cost N/A display.
- doctor commands.
- capability matrix / WARN-FAIL policy.
- doctor proof contract.
- doctor JSON / print-request redaction contract.
- Gate OSS-A report.
- Gate OSS-A required / optional / expected unsupported / forbidden classification tests.
- docs/config examples.

Run:

```bash
go test ./...
```

### Phase Final-A: Impact Audit

Mandatory.

Check:

- OpenAI API provider behavior unchanged.
- subscription runtime never sends API key.
- subscription transport uses OAuth bearer via `PrepareRequest`.
- existing Authorization header cannot leak through subscription transport.
- subscription compact transport never uses OpenAI Platform API key/path.
- runtime debug and doctor preview redact Authorization/account/token-like strings.
- subscription runtime never sends `previous_response_id`.
- subscription runtime never sends `context_management`.
- token redaction covers report/debug/error paths.
- token store does not write config or follow unsafe auth file paths.
- OAuth credential flow and API-key credential flow remain separate.
- status/default doctor do not perform surprise network refresh.
- refresh lifecycle cannot corrupt token store under concurrent access.
- auth status/logout do not touch OpenCode/Codex auth caches.
- raw history remains user-visible semantic history and is not mutated for provider-only replay state.
- provider-facing projection preserves function_call/function_call_output/reasoning item continuity.
- compacted input state preserves known item fields when subscription Compact API is enabled.
- provider-facing projection persistence does not leak OAuth/account/API key data.
- corrupt projection persistence cannot silently erase tool continuation state.
- config/generated/docs are in sync.
- provider/model UI status uses login required vocabulary.
- doctor classification separates required, optional, expected unsupported, and forbidden capabilities.
- Gate OSS-A does not fail on expected unsupported `store=true` / `previous_response_id`.
- Gate OSS-A does not fail solely on optional Compact API / thinking enabled level / cached token observation.
- doctor JSON and request preview contain no secret, prompt body, active context body, compacted encrypted data, or opaque provider replay state.
- thinking warning path and actual request path use the same profile/model policy.
- subscription runtime does not silently downgrade requested thinking levels.
- cost estimator never shows OpenAI API price for subscription.
- tool continuation semantics are caller-verified.
- Gate OSS-A is either passed or blocked with concrete failing capability.
- expected unsupported `store=true` / `previous_response_id` is not accidentally reported as broken install.

If correctness risk is found, use `post-implementation-impact-recovery`.

### Phase Final-B: Comprehensive Refactor

Mandatory.

Use `post-implementation-refactor`.
Use `test-boundary-refactor` if test fixtures/helpers grow mixed responsibilities.

Check:

- runtime profile owner is clear.
- auth/token store owner is clear.
- request transport owner is clear.
- history/cache policy owner is clear.
- provider-facing projection owner is clear.
- parser / projection / request layout responsibilities are not collapsed into one giant helper.
- doctor/report formatting owner is clear.
- doctor capability classifier owner is clear and not duplicated across text/JSON renderers.
- no giant file accumulates auth + transport + parser + doctor + config.
- test helpers do not copy production policy in a way that hides bugs.
- behavior-preserving extraction is completed before final report.

Report owner map, refactors done, and concrete remaining debt.

## 21. Stop Conditions

Stop and report if:

- `originator=xelyon` OAuth no longer works.
- endpoint no longer accepts basic streaming request.
- endpoint rejects `prompt_cache_key`.
- endpoint rejects all `reasoning.effort` payloads while docs/runtime claim enabled thinking support.
- endpoint event shape no longer matches existing parser enough for safe extension.
- tool call / function_call_output shape diverges substantially.
- subscription Compact API implementation would need OpenAI Platform API key/path fallback.
- subscription Compact API compacted output cannot be represented in provider-facing projection or compacted input state.
- provider-facing projection cannot preserve known Responses replay items without mutating raw history.
- session persistence cannot round-trip required tool continuation state and no safe in-memory-only scope is explicitly chosen.
- parser drops `call_id`, function arguments, function output, reasoning `encrypted_content`, or item order needed for continuation.
- token store/refresh needs a bigger design change.
- auth/status/doctor cannot distinguish login required / token expired / refresh failed / API key missing.
- status/default doctor would need surprise network refresh to produce safe output.
- credential-bearing errors cannot pass through a shared redaction owner.
- OpenAI API provider behavior is at risk.
- `PrepareRequest` hook would break existing OpenAI API-key request behavior.
- thinking warning/status path and actual request path cannot share the same profile/model policy.
- transport/debug redaction cannot be made shared and reliable.
- cost estimator would show OpenAI API price.
- docs would need to claim API-free subscription usage.
- implementing OSS release requires legal/Terms decision not available to Codex.
- OSS docs would need to imply OpenAI endorsement or official support.
- Gate OSS-A fails on required usability/security capability.

Do not work around stop conditions by:

- impersonating OpenCode.
- impersonating official Codex.
- weakening redaction.
- relabeling expected unsupported chain features as enabled.
- reading external auth caches.
- using Codex app-server secretly.
- falling back to OpenAI API key.

## 22. Completion Criteria

Commands:

```bash
xelyon auth openai-subscription login
xelyon auth openai-subscription status
xelyon doctor openai-subscription --smoke
xelyon doctor openai-subscription --retention-smoke
xelyon doctor openai-subscription --tool-smoke
xelyon --provider openai_subscription --model gpt-5.5 "hello"
xelyon --provider openai_subscription --model gpt-5.4-mini "list files if needed"
go test ./...
```

Expected properties:

- works without `OPENAI_API_KEY`.
- `openai` provider still requires API key.
- request uses OAuth bearer.
- request uses honest `originator=xelyon`.
- request sends `store=false`.
- request omits `previous_response_id`.
- request omits `context_management`.
- request sends `prompt_cache_key`.
- request uses streaming.
- tool loop completes.
- cost is N/A.
- tokens do not leak.
- provider/model/config UI can select provider and model.
- supported models are exactly 4.

## 23. Git Operations

Do not commit / push unless explicitly requested by the user.

After implementation, report:

- changed files.
- implementation summary.
- tests.
- doctor results.
- remaining risks.
- subscription endpoint capabilities:
  - prompt_cache_key
  - store
  - previous_response_id
  - context_management
  - compact_api
  - streaming
  - tool call
  - usage

## 24. Open Decisions

These must not be hidden inside implementation.

1. External Terms/legal approval for OSS release remains outside Codex implementation; implementation target is OSS mainline if Gate OSS-A passes.
2. Whether `prompt_cache_retention` is accepted by subscription endpoint and should be enabled by production profile after `--cache-smoke`.
3. Whether the live-verified subscription Compact endpoint remains accepted over time; if it later fails, doctor should WARN and runtime should fall back to provider-facing reduction/local summary without OpenAI Platform API fallback.
4. Whether doctor should persist capability findings or probe each run.
5. Whether `gpt-5.5` should be live-smoked before making it default in user docs.
6. How strongly to warn users that full payload mode can consume context faster than chain mode.
7. Exact subscription-plan entitlement wording for `gpt-5.3-codex-spark`; do not hard-code plan names unless official docs or backend response provides stable wording.

## 25. Final Consistency / Goal Readiness Audit

This section is the pre-Goal audit result. It records source-confirmed constraints found in the current codebase so the implementation Goal does not accidentally reintroduce the old v1 assumptions or hide cross-surface contract work inside local patches.

Goal must treat this section as a mandatory first checkpoint before coding beyond Phase 1.

### 25.1 Source-confirmed constraints

Existing OpenAI Responses builder is not yet profile-neutral.

- `internal/api/providers/openai/responses_request_builder.go` still hard-codes OpenAI identity in these request-path decisions:
  - server compaction provider key: `ResolveServerCompactionDecision(ctx, "openai", ...)`
  - model catalog lookup: `cfg.ModelCatalogName("openai", model)`
  - max output token lookup: `api.GetMaxOutputTokens(ctx, "openai", ...)`
  - store policy from global Responses config
  - unconditional `prompt_cache_retention: "24h"`
  - Codex-model reasoning fallback through OpenAI catalog naming
- Subscription implementation must not patch around these call sites one by one. It needs one profile-aware request-policy owner.

Existing Responses runner ownership is split.

- `internal/api/providers/openai/responses_nonstream.go` runs the actual OpenAI Responses request path and builds API-key bearer requests directly.
- `internal/api/providers/openai_responses/runner.go` already has a reusable `JSONRequestFactory` / `RunStreaming` shape, but the current OpenAI provider is not fully routed through that as the single canonical runner.
- Goal must choose one canonical Responses runtime/runner owner before adding subscription transport. Acceptable outcomes:
  - migrate the actual OpenAI path to a profile-aware shared runner, preserving OpenAI behavior; or
  - keep the current OpenAI runner owner but explicitly add profile hooks there and avoid a second drift-prone runner.

Existing debug paths print raw request/SSE data.

- `responses_nonstream.go` prints raw request JSON when debug is enabled.
- `openai_responses/runner.go` prints raw request JSON when debug is enabled.
- Existing streaming debug code can print raw SSE lines / completed payloads.
- Subscription full-payload mode can contain user prompts, tool output, compacted data, OAuth-derived identifiers, provider replay state, or encrypted/reasoning items. Subscription debug/doctor `--print-request` must use a redacted structural preview owner, not raw payload printing.

Existing provider config key policy preserves explicit aliases.

- `llmcatalog.ProviderConfigKey` currently canonicalizes display names, but explicit aliases are returned as typed.
- `config.ActiveProviderConfigKey` documents that explicit provider aliases are not rewritten.
- That is incompatible with the subscription requirement that `openai-subscription`, `chatgpt`, and `codex-subscription` all persist and identify as `openai_subscription`.
- Goal must add a canonicalization contract for subscription aliases without breaking existing alias-preservation behavior for other providers.

Existing `chatgpt` typo handling conflicts with the new alias.

- `internal/config/validator_provider.go` currently suggests `chatgpt -> openai` as a typo.
- Once `chatgpt` is a real alias for `openai_subscription`, that suggestion must be removed or replaced by the new canonical alias behavior.

Existing provider picker / provider-model service only understands API-key availability.

- `providerCredentialStatus` currently returns `configured`, `missing key`, `local`, or `aws auth`.
- `provider_model_service.go` blocks provider creation with an API-key-missing error before provider initialization.
- A provider descriptor with no `APIKeyEnv` can appear configured even when the subscription token store is missing.
- Goal must add a subscription-aware auth status path (`login required`, `logged in`, and local/read-only status) and ensure runtime errors say login required, not API key missing.

Existing auth command surface does not exist.

- The root command currently registers doctor, but there is no `auth` command tree.
- `xelyon auth openai-subscription login/status/logout` is therefore a new command surface and must have root-command wiring/tests rather than being hidden in doctor or provider initialization.

Existing OpenAI doctor retention smoke assumes `store=true` and `previous_response_id` success.

- `internal/api/providers/openai/diagnostics_smoke.go` forces `responses.store=true` for retention payloads and fails if the follow-up does not send `previous_response_id`.
- Subscription Phase 0 proved these are expected-unsupported at the backend.
- Subscription doctor must not reuse this as a success criterion. It needs a capability matrix that distinguishes required, optional, expected_unsupported, and forbidden.

Existing cost routing can avoid OpenAI API pricing, but status text needs a subscription-specific reason.

- A distinct pricing family with no resolver can make pricing unavailable.
- Doctor/status should still explain `N/A (ChatGPT subscription)` or equivalent. It should not silently look like a missing pricing config for OpenAI API usage.

Existing full-payload replay state is not sufficient for the highest-quality subscription path.

- `api.Message` has basic text/tool fields and Anthropic-specific provider state.
- `api.InputItem` can represent message/function_call/function_call_output and Compact API items, but it is not a complete generic Responses replay item store.
- `history.MessageProviderMetadata` currently persists Anthropic-specific provider metadata only.
- Existing OpenAI streaming parsing accumulates text/tool calls, but does not preserve arbitrary Responses output items such as reasoning/encrypted replay state.
- For subscription full-payload mode, Goal must add or select a provider-facing Responses replay/projection owner that can preserve known required Responses replay items without mutating raw conversation history.

### 25.2 Contracts to resolve before broad implementation

Resolve these as explicit contracts, with tests, before wiring the full provider:

- profile contract:
  - `openai` and `openai_subscription` share request construction where behavior is truly common.
  - profile owns provider key, config key, model catalog key, billing family, endpoint, auth strategy, payload mode, prompt cache policies, store policy, previous-response policy, context-management policy, compact policy, and thinking policy.
- runner contract:
  - one actual Responses runner path owns marshal, request preparation, retry classification, streaming/non-streaming dispatch, usage emission, response-id capture, and redacted debug preview.
  - subscription does not add a parallel runner that diverges from OpenAI parser/tool behavior.
- alias/config contract:
  - canonical key is `openai_subscription`.
  - aliases are accepted at CLI/UI input boundaries.
  - internal storage, session identity, provider config key, model lookup, and provider state use `openai_subscription`.
  - `chatgpt` no longer suggests OpenAI API provider in validation.
- auth status contract:
  - missing API key and missing subscription login are separate states.
  - provider picker, `/providers`, `/provider`, `/model`, runtime provider creation, doctor, and JSON reports use the same status vocabulary.
- redaction contract:
  - tokens, auth codes, JWTs, bearer strings, account IDs, and opaque provider replay state pass through one redaction owner before logs/debug/doctor/errors/JSON/HTML.
  - existing OpenAI debug behavior is not accidentally weakened or coupled to subscription secrets.
- replay/projection contract:
  - raw history remains user-visible conversation history.
  - provider-facing projection is the source of truth for subscription full-payload replay.
  - projection can replay assistant text, function_call, function_call_output continuation, and known Responses replay items needed by the backend.
  - session persistence round-trips replay metadata where session resume can send it.
- doctor capability contract:
  - `prompt_cache_key`, streaming, usage parsing, basic text, tool call, and full-payload continuation are required capabilities.
  - `prompt_cache_retention`, Compact API, and thinking levels are optional capabilities.
  - `store=true`, `previous_response_id`, and `context_management` are expected unsupported for subscription and must not be reported as enabled.
  - sending `OPENAI_API_KEY`, reading OpenCode/Codex auth caches, or using another originator is forbidden.

### 25.3 Do not weaken during implementation

- Keep Phase 0 v2 result as the runtime baseline:
  - `originator=xelyon`
  - OAuth bearer auth
  - subscription endpoint
  - `store=false`
  - no `previous_response_id`
  - no `context_management`
  - full provider-facing payload
  - `prompt_cache_key`
  - streaming
  - XELYON tool loop
  - cost N/A
- Do not reintroduce v1 `store=true` / server-side response chain assumptions as required behavior.
- Do not make subscription stateless-only if provider-facing full-payload replay and prompt-cache layout can be preserved.
- Do not implement fallback originators.
- Do not use `OPENAI_API_KEY` for subscription auth, Compact API probes, or smoke tests.
- Keep Phase 0 probe evidence in docs only. Do not keep a separate probe binary in `tools/`; production code lives in normal XELYON package owners.
- If any item in this section cannot be satisfied without destabilizing the existing OpenAI provider, stop and report the exact owner conflict.

## 26. Goal Handoff Prompt

Use this prompt for implementation Goal:

```text
Implement openai_subscription provider using docs/dev/openai-subscription-provider-master-plan.md as the source of truth. Re-read the plan after resume or context compaction. Start by resolving Section 25 Final Consistency / Goal Readiness Audit before coding beyond Phase 1. Treat docs/dev/openai-subscription-phase0-reality-check.md as historical evidence only; do not recreate a separate probe binary in production commands, and do not copy old store=true / previous_response_id assumptions into runtime.

Build the provider as an experimental ChatGPT/Codex OAuth subscription provider with honest originator=xelyon, endpoint https://chatgpt.com/backend-api/codex/responses, Compact endpoint https://chatgpt.com/backend-api/codex/responses/compact, store=false, full provider-facing payload, prompt_cache_key, stable prefix layout, streaming, XELYON tool loop, and shared OpenAI Responses thinking/reasoning request policy. Implement auth/token store as a credential boundary: browser + device login, local/read-only status, live doctor smoke refresh, atomic 0600 token store under ~/.xelyon/auth, distinct auth error vocabulary, and shared redaction for token/code/JWT/account/opaque provider state. Use a provider-facing projection as the full-payload replay source of truth: do not mutate raw conversation history for provider-only replay state, preserve known Responses replay items needed for assistant text, reasoning/encrypted_content, function_call, and function_call_output continuation, and round-trip projection persistence where session persistence applies. Use subscription Compact API as an experimental optimization through OAuth + originator=xelyon only; never route openai_subscription compaction to OpenAI Platform Compact API or OPENAI_API_KEY. If subscription Compact API later becomes unsupported, report WARN and use provider-facing reduction/local summary fallback. Use a profile PrepareRequest hook for subscription OAuth transport. Do not use OPENAI_API_KEY, do not read OpenCode/Codex auth caches, do not add originator=opencode or originator=codex_cli_rs fallback, and do not send previous_response_id or context_management in subscription runtime. Keep prompt_cache_retention profile-gated for subscription. Treat thinking level acceptance as a non-blocking capability classification: do not assume every level is accepted, and do not silently downgrade requested thinking levels, but do not block Gate OSS-A solely because an enabled thinking level is unsupported while `/thinking off` works.

Proceed through the plan phases, run go test ./... after each major phase, implement the doctor proof contract and capability matrix with required / optional / expected_unsupported / forbidden classification, and report Gate OSS-A as passed, warned, or blocked. Complete mandatory Final-A impact audit plus Final-B post-implementation refactor before final report. Do not commit or push unless explicitly requested.
```
