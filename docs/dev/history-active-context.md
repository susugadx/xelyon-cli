# Provider History Reduction / Active Context

このページは、provider history reduction と rehydrated evidence active context の開発者向け dogfood メモです。
まだ stable public config ではありません。`/config`、`docs/config.md`、generated config metadata には出しません。

## 早見表

### これは何か

XELYON の raw history を消さずに、provider へ送る履歴だけを request ごとに軽くする runtime-owned history optimization です。

- `Agent.History`、`Session.Messages`、audit、change records、persisted JSONL は元の内容を保持します。
- provider-facing history projection だけで、古い `read_file` / `search_code` / `gather_context` 結果を evidence pointer placeholder にできます。
- 成功した test / build / lint command output は、安全条件を満たす場合だけ summary 化できます。
- 置き換えた read/search/gather evidence は、必要に応じて現在ファイルから rehydrate し、request-local active context として provider 入力へ戻せます。

### 何が嬉しいか

- 長い作業履歴を保持したまま、provider に再送する古い巨大 tool output を減らせます。
- raw log を失わないので、resume、audit、ledger、debug の根拠は残ります。
- 古い evidence を完全に捨てるのではなく、必要な範囲を現在ファイルから読み直して active context に戻せます。
- provider adapter ごとの既存 active-context transport を使うため、request assembly の責務を provider 側に閉じ込められます。

### dogfood 設定例

`xelyon.yaml` に project-local experimental 設定として書きます。default は `off` / `false` です。

```yaml
experimental:
  provider_history_reduction:
    mode: apply
    rehydrate_context: true
```

`mode` は `off` / `dry_run` / `apply` / `auto` を受け付けます。`auto` は現時点では safe な `dry_run` として動きます。
`rehydrate_context: true` は mode とは別の gate です。`mode: off` や `dry_run` と同時に指定しても設定としては有効ですが、実際に rehydrated block が出るのは read/search/gather evidence replacement が apply され、現在の provider route に active context transport がある場合だけです。

### env override

一時的に dogfood する場合は env で project config を上書きできます。

```sh
XELYON_PROVIDER_HISTORY_REDUCTION=dry_run xelyon
XELYON_PROVIDER_HISTORY_REDUCTION=apply xelyon
XELYON_PROVIDER_HISTORY_REDUCTION=off xelyon
XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT=1 xelyon
XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT=false xelyon
```

`XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT` は `1` / `true` / `0` / `false` を受け付けます。無効な値は config error にします。

### 見る場所

- `/status`: provider history reduction の mode、candidate / replacement summary、`rehydrate_context=on|off`、`active_context_transport=...` を確認します。
- `/tokens`: 通常の token 見積もりを見る場所です。provider history reduction diagnostics は混ぜません。
- `/ledger`: read/search/gather replacement から作れる rehydrate candidates を確認します。rehydrated file content 自体は表示しません。

### 大事な安全契約

- raw history は消えません。`Agent.History`、`Session.Messages`、audit entries、change records、persisted JSONL は元の内容を保持します。
- provider-facing projection で古い成功済み `write_file.content` を省略しても、raw `write_file` arguments、tool result、audit、persisted JSONL は保持します。
- rehydrated evidence は request-local model input です。history、session、audit、ledger actual content、persisted JSONL には保存しません。
- `write_file.content` replacement は evidence pointer ではないため、rehydrate 対象には入りません。
- active context transport がない unsupported provider では、read/search/gather evidence replacement を skip し、`active_context_transport_unsupported` として保持します。
- internal calls / compact / compression / Gemini repair / review model には active context を入れません。
- これは stable config ではありません。README には experimental 概要だけを置き、通常設定表、`/config`、`docs/config.md`、generated config metadata にはまだ出しません。
- default ON ではありません。コスト削減率も固定値としては約束しません。

## 詳細メモ

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
  - the current task state block is built only when `RuntimeOptions.EnableCurrentTaskStateContext` is true.
  - provider-history rehydrated evidence is built only when `RuntimeOptions.EnableProviderHistoryRehydrateContext` is true. The runtime gate defaults to false and is dogfooded through the experimental project-local config/env described below.
  - sent only when `internal/api` reports a provider active-context transport for the runtime provider/model.
  - unsupported providers keep active context out of the request context.
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
- The projection report records detected candidates, kept candidates, replacement count, original/projected content bytes, estimated saved bytes, approximate saved tokens, kept reason counts, command/edit replacement diagnostics, and whether a replacement disabled Responses continuation.
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
- `experimental.provider_history_reduction.rehydrate_context: true` enables request-local rehydrated evidence active-context injection for dogfood. `XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT=1|true|0|false` overrides the project file. The default is false.
- `/config`, generated `config.yaml` metadata, and `docs/config.md` remain unchanged because this is not a stable global config surface. README may point developers to this dogfood note as an experimental feature overview.
- `dry_run` updates `AgentRuntime.LastProviderHistoryProjectionReport` but sends raw provider payload and keeps any OpenAI/Azure Responses `previous_response_id` chain unchanged.
- `apply` uses the existing safe replacement path and disables the Responses continuation chain only when a replacement is actually applied.
- `auto` is accepted as an enum but currently resolves to the safe `dry_run` effective mode. `/status` reports the configured mode separately, for example `mode=auto; effective=dry_run; report: mode=dry_run; ...`.
- Raw storage remains unchanged in every mode: `Agent.History`, `history.Session.Messages`, session tool execution audit entries, audit logs, change records, compacted state, and persisted session JSONL keep the original conversation/audit data.

## Provider History Reduction Dogfood

This mode is experimental runtime dogfood only. Do not expose it as a stable public config in `/config`, generated config metadata, or `docs/config.md` until the public config contract is decided. README may link here as an experimental developer feature overview.

Project-local dry-run:

```yaml
experimental:
  provider_history_reduction:
    mode: dry_run
    rehydrate_context: true
```

Environment override for a single run:

```sh
XELYON_PROVIDER_HISTORY_REDUCTION=dry_run xelyon
XELYON_PROVIDER_HISTORY_REDUCTION=apply xelyon
XELYON_PROVIDER_HISTORY_REDUCTION=off xelyon
XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT=1 xelyon
XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT=false xelyon
```

`rehydrate_context` is independent from the reduction mode selector. Setting it to true is allowed with `mode: off` or `mode: dry_run`, but a rehydrated evidence block is emitted only when the provider-facing projection has applied read/search/gather evidence replacements and the current provider route has an active-context transport.

In `dry_run`, inspect `/status` after a provider-facing request:

- `candidates`, `replaced`, and `kept` show detector and replacement counts.
- `saved` shows estimated content bytes saved; `approx_saved_tokens` is a diagnostic-only token estimate, not billing usage.
- `kept_reasons` shows sorted keep reason counts using the internal reason strings.
- `responses_chain_disabled` should stay `false` in `dry_run` and `auto` because provider payload remains raw.
- `command/edit` is a separate diagnostic line for old `bash`/`command` outputs and edit tool arguments. In `dry_run`, `replacement=not_implemented` means provider payload remains raw; the byte/token numbers are diagnostic estimates only and are not billing usage. In `apply`, `edit_arg_replaced` and `edit_arg_replacement_saved` cover only applied `write_file.content` argument replacement.

Switch to `apply` only after dry-run candidates and kept reasons look expected, and test on a limited task where repeated command/read/search calls do not increase and answer quality does not regress. `apply` replaces only safe old successful `bash`/`command` outputs for single actual test/build/lint commands with explicit success evidence, and safe old successful `write_file.content` arguments with repo-relative paths and successful write results. Both replacements apply only when the placeholder is smaller and saves at least the internal token threshold. Failure logs, ambiguous build summaries, mixed passing/failing test output, lint output with errors/issues/problems/warnings, interrupted or partial command output, compound shell or command-substitution output, `git diff` output, generic successful command output, unsupported edit tool arguments, unsafe write paths, invalid write arguments, and latest/trailing/incompletely linked tool results remain raw. When any provider-facing replacement is applied, `/status` reports `replacement=partial_apply` for command/edit replacement and `responses_chain_disabled=true`; OpenAI/Azure Responses requests may drop the `previous_response_id` continuation chain. `apply` with zero replacements keeps the chain.

Raw storage is unchanged in all modes: runtime `Agent.History`, `Session.Messages`, audit entries, change records, and persisted JSONL keep the original content and original tool arguments.

## Synthetic Measurement And Rehydrate Dry-Run

The provider history reduction synthetic harness exercises `off`, `dry_run`, and `apply` against fixed read/search/gather, command, edit, latest-tool, trailing-tool, and invalid-linkage fixtures. This lets safety and savings regressions be checked before live dogfood depends on a real provider transcript.

`internal/ledger` owns the rehydrate planner dry-run. It plans which omitted old read/search/gather evidence ranges should be refreshed from current files, but the planner itself does not read files, inject provider input, append `Agent.History`, append `Session.Messages`, or change persisted JSONL.

`internal/ledger` also owns the separate `ExecuteRehydratePlan` execution seam. Execution is not part of the dry-run planner contract: it reuses the existing evidence-pointer path safety policy, rejects unsafe plan paths before reading files, reads only current repo-root-relative file ranges, and returns a `RehydratedEvidenceBlock` plus diagnostic failures. Failed items are omitted from model input. The executor budget is bounded by item count, total lines, and rendered block bytes; if the next item would exceed the budget, it is omitted instead of partially rendered.

The intended design is that runtime state refreshes needed evidence from current files instead of asking the model to rediscover raw history by itself.

Apply-mode projection reports keep matched evidence pointers only on read/search/gather candidates whose provider-facing placeholder was actually applied. Command output and `write_file.content` replacements do not attach evidence pointers.

The runtime can pass those applied read/search/gather `EvidencePointers` into `ledger.BuildRehydratePlan` as old evidence. `/ledger` shows non-empty rehydrate candidates after the normal task-ledger snapshot so the dry-run can be inspected during dogfood. `/ledger` remains a candidate diagnostic and does not show rehydrated file content.

Command output replacement is backed by successful command summaries, and `write_file.content` replacement is backed by the successful write result plus retained path argument. Neither is an evidence pointer, so command/edit replacements are not rehydrate candidates.

Provider-input injection is available behind `RuntimeOptions.EnableProviderHistoryRehydrateContext`. The gate defaults to false and can be enabled for dogfood with `experimental.provider_history_reduction.rehydrate_context: true` in `xelyon.yaml` or overridden for one run with `XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT=1|true|0|false`. This remains an experimental project-local surface only: it is still not exposed as stable config in `/config`, `docs/config.md`, or generated config metadata.

When the gate is true, provider-facing request assembly uses the projection report from the same request, builds a rehydrate plan from applied read/search/gather replacements, executes it against current files, renders `<rehydrated_evidence>`, and appends it as a dynamic active context block named `provider_history_rehydrated_evidence`. This happens only for provider request paths that already use `providerFacingHistoryForRequest` and only when `internal/api.ProviderActiveContextTransportForRequest` reports a supported transport. The active context core contract is provider-independent and request-local; provider adapters own the transport-specific placement.

Supported transports:

- OpenAI Responses and Azure Responses use the native Responses developer input path.
- OpenAI Chat Completions, DeepSeek, Groq, Kimi, Ollama, and the OpenRouter OpenAI-compatible route add an ephemeral system message after the base system prompt and before history.
- Claude, the OpenRouter Claude route, and Bedrock Claude Messages append active context to the dynamic system suffix, separated from the static prompt by `SystemPromptCacheBoundary`.
- Gemini text, function-calling, and multimodal requests add active context as request-local user content immediately before the latest user request sent to Gemini. Cached `systemInstruction` is not modified.
- Bedrock Converse adds active context as a separate `System` content block.

If the rehydrate gate is true but a provider has no active-context transport, read/search/gather evidence replacement is skipped for that request and the projection report keeps those candidates with `active_context_transport_unsupported`. Successful command output and `write_file.content` replacement keep their safe replacement behavior because they do not require rehydrated evidence.

The same request-local rehydrated block is included in provider-facing token estimates, `/tokens`, token warnings, and local auto-compress decisions. Token estimation does not update `AgentRuntime.LastProviderHistoryProjectionReport`.

The rehydrated block is request-local model input only. It is not appended to `Agent.History`, `history.Session.Messages`, tool execution audit entries, audit logs, change records, compacted state, `/ledger` actual-content output, or persisted session JSONL. Unsupported providers keep the safety fallback: evidence replacement is skipped rather than injecting or persisting refreshed evidence.

## Provider Transports And Responses Continuation

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
