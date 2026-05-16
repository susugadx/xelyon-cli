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
- Apply mode uses the same dry-run detector candidates, then replaces only candidate `Content` on the projection clone when matching task-ledger evidence pointers exist and the placeholder is smaller than the original content.
- Evidence pointers are matched by `ToolCallID` and `Source == ToolName`. If the runtime task ledger is missing, a matching pointer is absent, or another candidate/kept tool result shares the same `(ToolCallID, ToolName)`, the candidate is kept in the projection.
- Replacement text is a single-line placeholder such as `[omitted old read_file result; evidence: README.md:L1-L80 source=read_file; +2 more]`.
- Message shape is preserved: role, tool call id, tool name, assistant tool calls, reasoning content, provider state, and continuation metadata are not changed.
- Raw storage remains unchanged: `Agent.History`, `history.Session.Messages`, session tool execution audit entries, audit logs, and change records continue to store the raw conversation/audit data.
- The projection report records detected candidates, kept candidates, replacement count, original/projected content bytes, and estimated saved bytes.
- Phase 5b-4 can enable this policy on a limited request path without adding a new storage migration.

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

Enable `ProviderHistoryReductionApply` on a limited request path while keeping the raw log intact.
Request-path enablement should preserve Responses tool-output continuity, keep default startup disabled until explicitly selected, and stay covered by tests that distinguish raw session storage from model-facing history.
