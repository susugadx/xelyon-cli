# History / Active Context Contract

Phase 5a records the current contract before raw log and provider-facing active context are separated further.
This document is descriptive. It does not change retention, compression, provider request payloads, or Responses continuation behavior.

## Current Storage Paths

- Conversation messages are stored through `Agent.appendSessionMessage`, `appendSessionMessageFromAPI`, and `appendSessionMessageFromAPIWithStoredContent`; they mutate `history.Session.Messages` and persist through `history.Storage.Save`.
- In the normal session-persisted tool path, tool results are stored in three places today:
  - runtime `Agent.History`, via `appendToolResultToHistory`.
  - session conversation messages, via `appendSessionMessage` or `appendSessionMessageFromAPI`.
  - session tool execution audit entries, via `appendSessionToolExecution` and `history.Session.AddToolExecution`.
- Headless tool-loop history keeps provider-facing `Agent.History` continuity but does not use the session conversation/tool-execution persistence helpers.
- `history.Session.ToAPIMessages` skips `entry_type=tool_execution`, so audit entries stay in the raw session log but do not become provider input on restore.
- `history.Storage.Save` appends unsaved JSONL entries. `history.Storage.Rewrite` rewrites the whole session when messages, compacted state, or response context require it.
- Compact API state is stored on `Session.CompactedItems` / `IsCompactedMode` and persisted as compacted state entries. `Agent.RestoreCompactedState` restores it into runtime mirrors.
- File mutation records are separate from conversation history. `MutationTracker.RecordFileChangeForTurn` writes to `history.ChangeStorage.AppendChange`.
- Tool execution audit logging is separate from session storage. `tools.ExecutionContext.AuditLogger` comes from `AgentRuntime.AuditLogger`.
- Resume/load uses `history.Storage.Load`, `Session.ToAPIMessages`, `Agent.restoreSessionConversation`, `RestoreCompactedState`, and response ID restore for the current provider/model identity.

## Provider Input Assembly

- `Agent.requestContext` is the owner for provider request context assembly.
- Phase 5a introduces the internal no-op seam `modelInputAssemblyPlan`, which carries the same two context inputs as before:
  - `CompactedInput`, copied from current compacted runtime state.
  - `ActiveContextBlocks`, selected by `activeContextInputPolicy`.
- `activeContextInputPolicy` keeps the current behavior:
  - default is off.
  - enabled only when `RuntimeOptions.EnableCurrentTaskStateContext` is true.
  - sent to Azure Responses and OpenAI Responses models.
  - not sent to OpenAI Chat Completions, Gemini, Claude, DeepSeek, or other non-consuming providers.
- Active context is only injected into provider request context. It is not appended to `Agent.History` or `Session.Messages`.
- `requestContextWithoutActiveContext` remains the boundary for internal model calls that must not receive active context.

## Provider-Facing History Projection

Phase 5b-1 introduces a provider-facing `History` projection seam.

- `Agent.History` remains the raw runtime conversation history.
- User-facing provider requests use `Agent.providerFacingHistory()` instead of passing `Agent.History` directly.
- The default Phase 5b-1 projection is a no-op content clone: provider input contains the same messages, tool calls, tool results, and provider continuation metadata as raw `Agent.History`.
- The projection defensively clones message slices, function-calling tool calls, Gemini thought parts, and Anthropic provider state before handing history to providers.
- Image requests keep the existing payload shape: past history is sent as provider history, and the current image prompt is sent through the `userMessage` argument.
- No history reduction, replacement, compression, or pruning policy is implemented in Phase 5b-1.
- Raw storage is unchanged: `history.Session.Messages`, session tool execution audit entries, tool execution audit logs, and change records continue to store the raw conversation/audit data.

Phase 5b-2 adds a dry-run reduction candidate detector behind the same projection seam.

- Default projection policy remains disabled.
- Disabled projection still returns a no-op clone of raw `Agent.History` and produces an empty report.
- Dry-run policy produces an internal `ProviderHistoryProjectionReport` only; provider-facing message content is not omitted, replaced, compressed, or pruned.
- Candidate detection is limited to old `read_file`, `search_code`, and `gather_context` tool results where a later assistant message exists.
- The detector keeps the final contiguous trailing `tool` suffix, the latest tool result, write/command tools, non-allowlisted tools, and ambiguous or invalid tool-call linkage.
- Request call sites continue to pass only projected history to providers; the dry-run report is not sent to provider implementations.
- Raw storage remains unchanged: session conversation messages, tool execution audit entries, audit logs, and change records continue to store the raw conversation/audit data.
- Actual provider-facing reduction is deferred to Phase 5b-3 or later.

Phase 5b-3 adds gated actual replacement mode behind the internal projection policy.

- Default projection policy remains disabled.
- `ProviderHistoryReductionApply` is an explicit internal mode; normal, headless, image, and plan provider requests still call the default disabled projection and continue to send raw-clone content.
- Apply mode uses the same dry-run detector candidates, then replaces candidate `Content` on the projection clone when matching task-ledger evidence pointers exist and the placeholder is smaller than the original content.
- Evidence pointers are matched by `ToolCallID` and `Source == ToolName`. If the runtime task ledger is missing, a matching pointer is absent, or another candidate/kept tool result shares the same `(ToolCallID, ToolName)`, the candidate is kept in the projection.
- Replacement text is a single-line placeholder such as `[omitted old read_file result; evidence: README.md:L1-L80 source=read_file; +2 more]`.
- Message shape is preserved: role, tool call id, assistant tool calls, reasoning content, provider state, and continuation metadata are not changed. If a replaced tool result is missing `ToolName`, apply mode copies the tool name inferred from the matching assistant tool call onto the projection clone so provider adapters can keep function-response continuity.
- Raw storage remains unchanged: `Agent.History`, `history.Session.Messages`, session tool execution audit entries, audit logs, and change records continue to store the raw conversation/audit data.
- The projection report records detected candidates, kept candidates, replacement count, original/projected content bytes, estimated saved bytes, approximate saved tokens, kept reason counts, and whether a replacement disabled Responses continuation.
- Phase 5b-4 can enable this policy on a limited request path without adding a new storage migration.

Phase 5b-4 enables replacement on user-facing provider request paths behind an internal runtime option.

- Default behavior remains disabled. At this phase, `RuntimeOptions.EnableProviderHistoryReduction` defaults to false and there is no config key, environment variable, CLI flag, `/config` entry, generated config field, or session migration for this gate.
- When the runtime option is false, normal, headless, image, and plan investigation requests still send the raw-clone projection and overwrite `AgentRuntime.LastProviderHistoryProjectionReport` with an empty report.
- When the runtime option is true, normal, headless, image, and plan investigation requests use `ProviderHistoryReductionApply` through the provider-facing projection seam.
- Image requests project only the past history that is actually sent through `ChatWithImage(..., history, userMessage, image, ...)`; the current image prompt stays in the `userMessage` argument.
- `AgentRuntime.LastProviderHistoryProjectionReport` is runtime-only diagnostic state. It is not appended to `Agent.History`, `history.Session.Messages`, session tool execution audit entries, tool execution audit logs, change records, compacted state, or persisted session JSONL.
- Apply mode changes candidate `Content` on the provider projection clone, and may fill a missing projected tool-result `ToolName` from the matched assistant tool call. Tool call continuity, provider metadata, raw runtime history, raw session storage, and audit storage remain unchanged.
- Internal model calls remain excluded: `CompressHistory`, Compact API compression, Gemini apply-patch repair, review model calls, and other isolated calls that use explicit single-user prompts or compact inputs do not call `providerFacingHistory()` and do not update the last projection report.

Phase 5c exposes the last provider history reduction projection as a `/status` runtime diagnostic only.

- `/status` shows a `Provider history reduction` section only when the runtime mode is not `off` without a report, or when `AgentRuntime.LastProviderHistoryProjectionReport` contains a non-empty report.
- With a non-`off` runtime mode and no report yet, `/status` prints the configured mode, for example `mode=apply; no report yet`.
- With a report, `/status` prints a deterministic count/byte summary such as `mode=apply; candidates=3; replaced=2; kept=1; original=1,000 B; projected=250 B; saved=750 B; approx_saved_tokens=42; kept_reasons=dry_run:1, missing_evidence_pointer:2; responses_chain_disabled=true`.
- The diagnostic reports counts, bytes, approximate saved tokens, kept reason counts, and whether the Responses continuation chain was disabled. It does not add cost estimates, config, CLI flags, generated config, `/config`, or `/tokens` output.
- The diagnostic remains runtime-only: it is not appended to `Agent.History`, `history.Session.Messages`, tool execution audit entries, audit logs, change records, compacted state, model input, or persisted session JSONL.

Phase 5d adds an experimental project-local mode selector for controlled rollout.

- The default remains `off`. `xelyon.yaml` can opt in with `experimental.provider_history_reduction.mode: off|dry_run|apply|auto`; `XELYON_PROVIDER_HISTORY_REDUCTION` overrides the project file with the same values.
- `/config`, generated `config.yaml` metadata, README, and `docs/config.md` remain unchanged because this is not a stable global config surface.
- `dry_run` updates `AgentRuntime.LastProviderHistoryProjectionReport` but sends raw provider payload and keeps any OpenAI/Azure Responses `previous_response_id` chain unchanged.
- `apply` uses the existing safe replacement path and disables the Responses continuation chain only when a replacement is actually applied.
- `auto` is accepted as an enum but currently resolves to the safe `dry_run` effective mode. `/status` reports the configured mode separately, for example `mode=auto; effective=dry_run; report: mode=dry_run; ...`.
- Raw storage remains unchanged in every mode: `Agent.History`, `history.Session.Messages`, session tool execution audit entries, audit logs, change records, compacted state, and persisted session JSONL keep the original conversation/audit data.

## Provider History Reduction Dogfood

This mode is experimental runtime dogfood only. Do not expose it in `/config`, generated config metadata, README, or `docs/config.md` until the public config contract is decided.

Project-local dry-run:

```yaml
experimental:
  provider_history_reduction:
    mode: dry_run
```

Environment override for a single run:

```sh
XELYON_PROVIDER_HISTORY_REDUCTION=dry_run xelyon
XELYON_PROVIDER_HISTORY_REDUCTION=apply xelyon
XELYON_PROVIDER_HISTORY_REDUCTION=off xelyon
```

In `dry_run`, inspect `/status` after a provider-facing request:

- `candidates`, `replaced`, and `kept` show detector and replacement counts.
- `saved` shows estimated content bytes saved; `approx_saved_tokens` is a diagnostic-only token estimate, not billing usage.
- `kept_reasons` shows sorted keep reason counts using the internal reason strings.
- `responses_chain_disabled` should stay `false` in `dry_run` and `auto` because provider payload remains raw.
- `command/edit` is a separate diagnostic line for old `bash`/`command` outputs and edit tool arguments. In `dry_run`, `replacement=not_implemented` means provider payload remains raw; the byte/token numbers are diagnostic estimates only and are not billing usage.

Switch to `apply` only after dry-run candidates and kept reasons look expected, and test on a limited task where repeated command/read/search calls do not increase and answer quality does not regress. `apply` replaces only safe old successful `bash`/`command` outputs for single actual test/build/lint commands with explicit success evidence, and only when the placeholder is smaller and saves at least the internal token threshold. Failure logs, ambiguous build summaries, mixed passing/failing test output, lint output with errors/issues/problems/warnings, interrupted or partial command output, compound shell or command-substitution output, `git diff` output, generic successful command output, and edit tool arguments remain raw. When any provider-facing replacement is applied, `/status` reports `replacement=partial_apply` for command/edit replacement and `responses_chain_disabled=true`; OpenAI/Azure Responses requests may drop the `previous_response_id` continuation chain. `apply` with zero replacements keeps the chain.

Raw storage is unchanged in all modes: runtime `Agent.History`, `Session.Messages`, audit entries, change records, and persisted JSONL keep the original content.

## Synthetic Measurement And Rehydrate Dry-Run

The provider history reduction synthetic harness exercises `off`, `dry_run`, and `apply` against fixed read/search/gather, command, edit, latest-tool, trailing-tool, and invalid-linkage fixtures. This lets safety and savings regressions be checked before live dogfood depends on a real provider transcript.

`internal/ledger` owns the rehydrate planner dry-run. It plans which omitted old read/search/gather evidence ranges should be refreshed from current files, but it does not read files, inject provider input, append `Agent.History`, append `Session.Messages`, or change persisted JSONL. The intended design is that runtime state refreshes needed evidence instead of asking the model to rediscover raw history by itself.

Apply-mode projection reports keep matched evidence pointers only on read/search/gather candidates whose provider-facing placeholder was actually applied. Command output replacements do not attach evidence pointers.

Automatic rehydrate execution and provider-input injection are a later phase. The current planner only returns bounded path/range items for tests and future integration.

## Responses Continuation

- OpenAI and Azure Responses builders read active context from request context.
- If active context has nonblank content, `previous_response_id` is not used for that request.
- Active-context requests fall back to full input: developer message, compacted input if present, active context developer message, and full local history.
- Without active context, Responses continuation keeps the existing behavior:
  - `previous_response_id` can be reused when `responses.store` allows it.
  - trailing tool messages are sent as `function_call_output`.
  - non-tool turns can send only the latest message when continuing the response chain.
- `prepareChatRequest` clears saved/provider response context before an active-context request for providers that consume it. Providers that do not consume active context keep their response context.

## Compression And Internal Calls

- `CompressHistory` calls the provider through `requestContextWithoutActiveContext`; summary prompts do not receive active context.
- Compact API compression also uses `requestContextWithoutActiveContext`; compact requests do not receive active context.
- Gemini apply-patch repair uses an isolated internal request context and clears active context.
- Review model calls use isolated context, disable tools, omit compacted input, omit active context, and suspend response continuation.
- Phase 5a does not change existing local compression pruning. `CompressHistory`, Compact API input preprocessing, and history compaction helpers keep their current behavior.

## Phase 5b Classification

Safe provider-facing reduction candidates:

- Old `read_file` full outputs after newer evidence exists.
- Old `search_code` full outputs after the model has moved past those results.
- Old `gather_context` prefetch/full outputs that are no longer the latest evidence for the turn.

Keep in provider-facing projection:

- Latest tool result for a still-relevant tool call.
- User instructions and user-authored task constraints.
- Current turn tool results.
- Tool outputs required for `function_call_output` continuity in Responses API.

Never delete from raw storage:

- Audit log entries.
- Persisted session raw conversation messages.
- Session tool execution entries.
- Change records.

## Phase 5b TODO

Expose provider history reduction only after a separate config/user-facing contract decision.
Any future public gate must update config docs, generated config, migration/backward-compatibility notes, and request-path tests in the same change.
