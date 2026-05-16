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

Introduce a provider-facing `History` projection while keeping the raw log intact.
Reduction should be selected by policy, applied only to provider input, and covered by tests that distinguish raw session storage from model-facing history.
