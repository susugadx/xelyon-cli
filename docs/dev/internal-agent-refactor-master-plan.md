# internal/agent Refactor Master Plan

この文書は Codex Goal で `internal/agent` 周辺をまとめてリファクタリングするための内部実装仕様書である。公開 docs ではなく、実装前の設計・handoff・resume 後の source of truth として使う。

## 0. Purpose

`internal/agent` 周辺を、挙動変更ではなく責務境界改善を主目的として段階的にリファクタリングする。最小差分 patch ではなく、owner、source of truth、test boundary を明確にし、今後の変更が巨大な `agent` package に条件分岐を積み増す形にならないようにする。

Goal 完了条件は、少なくとも Phase 0 の owner map と baseline を固定し、低リスクな pure extraction を 1 tranche 実施し、focused tests と broader baseline を通し、Final-A / Final-B 相当の self-audit で残 debt を file/function 単位で報告することである。

commit / push / PR 作成はユーザーの明示指示があるまで行わない。

## 1. Current State / Implemented Preconditions

- `go list` で確認した現状は `internal/agent` が 180 production files、210 test files、`internal/agent/plan` が 18 production files / 14 test files、`internal/agent/viewfmt` が 1 production file / 1 test file。
- `find internal/agent -maxdepth 1 -type f -name '*.go' -print0 | xargs -0 wc -l` で確認した `internal/agent` 直下は合計約 69k 行。
- 既存 subpackage は `internal/agent/plan` と `internal/agent/viewfmt`。`plan` は parser / extractor / plan domain、`viewfmt` は表示整形の pure helper を担う。
- 外部 caller は主に `internal/app` と `internal/tuiagent`。`cmd` と `e2e` は headless surface を参照する。
- `internal/agent/package_boundaries_test.go` は `internal/agent` が `internal/tui`、Bubble Tea、Lip Gloss を import しない契約を固定している。この契約は弱めない。
- 変更前 baseline として `go test ./internal/agent ./internal/agent/plan ./internal/agent/viewfmt ./internal/app ./internal/tuiagent ./cmd` は通過済み。

## 2. Global Contracts

- headless JSON shape、session/history persistence、provider-facing history/projection、MCP exported names、tool execution context、config defaults、prompt content、TUI lifecycle boundary は変えない。
- `internal/agent` の exported API は、外部 caller 互換が必要な限り thin facade / type alias で保持する。
- `internal/tui`、Bubble Tea、Lip Gloss は `internal/agent` へ戻さない。
- MCP contract は `internal/mcp` / `internal/mcptool` / `internal/mcpnames` / provider schema surface の owner を越えて agent 側だけで直さない。
- provider-facing history/projection は `internal/providerhistory` を source of truth とし、`internal/agent` は request context / response-id-chain 判断の adapter として扱う。
- project-map runtime state は `agentProjectPromptState` と `toolExecutionContext` の両方を owner map に含める。prompt 表示だけの整理で runtime state を取り残さない。
- `utils` / `helpers` / `common` のような曖昧 package は作らない。
- import cycle 回避だけの interface / wrapper は作らない。
- package split は、独立 owner、小さい API、自然な依存方向、focused tests を持てる場合だけ行う。file 分割だけで十分な場合は package を増やさない。

## 3. Non-goals

- MCP runtime/config/schema/docs の contract 修正。
- provider request shape、Responses API、pricing/token/compression policy の仕様変更。
- headless JSON の field / status / error type 変更。
- TUI lifecycle の移動、または `internal/agent` から TUI component を直接扱う変更。
- config schema、YAML key、generated metadata、README/docs の公開 contract 変更。
- commit / push / PR 作成。

## 4. Source Findings

- Public surface は `go doc ./internal/agent` で、top-level run functions、headless result constructors、format helpers、provider history aliases、runtime types、`Agent` methods が外へ見えている。
- `go doc ./internal/agent.Agent` で確認した `Agent` surface は chat/run/review/session/config/provider/model/status/tool-result/TUI adapter 用 method が混在している。
- `internal/app/entrypoints.go` は `agent` を thin facade として re-export し、legacy interactive / headless / once の entrypoint を保持している。
- `internal/app/tui.go` は `agent.NewAgentRuntimeWithConfig`、`agent.NewInteractiveAgentWithRuntime`、`agent.BuildInteractiveHeader`、`tuiagent.NewTUIAdapter` に依存する。
- `internal/tuiagent/tui_adapter.go` は `Agent` の chat、command、session、status、review、config sync、provider/model switch、copy output を幅広く呼ぶ。
- `cmd/root_test.go` と `e2e/helpers_test.go` は `HeadlessResult`、headless status/error constants、`RunHeadlessWithConfig`、result constructors に依存する。
- `internal/agent/stats.go` には metrics state と表示整形 helper `FormatFileSize` / `FormatTokens` / `FormatNumber` が同居している。
- `internal/agent/viewfmt` はすでに `Number` / `USD` / `USDWithSuffix` / `FirstLine` / `Truncate` を持ち、`command_status_helpers.go` と `view_renderer.go` から使われている。
- `FormatNumber` 系の tests は `stats_test.go`、`stats_metrics_extra_test.go`、`viewfmt_facade_test.go` に残っている。production owner は `viewfmt`、agent package 側は compatibility facade tests を担う。

## 4.1 Implemented Progress In This Goal

- `internal/agent/viewfmt` now owns `Number` / `Tokens` / `FileSize` / USD / first-line / truncation formatting.
- `internal/agent/viewfmt_facade.go` keeps `agent.FormatNumber` / `agent.FormatTokens` / `agent.FormatFileSize` as compatibility wrappers.
- `internal/agent/viewfmt_facade.go` also keeps the package-private `formatNumber` compatibility facade used by existing status, compression, and provider-history display call sites.
- `internal/agent/stats.go` no longer owns display formatting policy; it remains focused on session stats, usage, cost, and session file size lookup.
- `internal/agent/command_status_helpers.go` no longer owns trivial USD / first-line / truncation wrappers; status-specific cost, pending, and error-message policy remains there, while pure formatting calls `viewfmt` directly.
- `internal/agent/view_renderer.go` now calls `viewfmt.USDWithSuffix` directly for pure cost suffix formatting and keeps table construction ownership in `agent`.
- `internal/agent/viewfmt/format_test.go` covers the pure formatting owner for token and file-size formatting; existing `agent` tests continue to cover facade compatibility.
- Command surface dispatch now lives in `internal/agent/command_surface_dispatch.go`; `agent_commands.go` was removed as a vague bucket.
- Slash-command confirmation, history/exit commands, and command header rendering now live in `command_confirmation.go`, `command_history.go`, and `command_render_header.go` respectively.
- `agent_commands_test.go` was removed. `splitCommand` compatibility tests now live in `command_surface_runtime_test.go`; `formatNumber` compatibility tests now live in `viewfmt_facade_test.go`.
- `internal/agent/plan` now owns approved-plan implementation handoff projection through `plan.ImplementationHandoff`.
- `internal/agent` now keeps Plan Mode orchestration, approval, session/state restoration, tool visibility, and normal-mode execution, but no longer owns approved-plan handoff string construction.
- `internal/agent/plan/handoff_test.go` owns pure handoff formatting, clone, file grouping, and verification-hint tests; broader `internal/agent` Plan Mode flow tests continue to cover caller/session behavior.
- Plan review display now belongs to `internal/uiplanview`; verification normalization still delegates to `plan.CompactVerificationHints` instead of duplicating trim/dedupe policy.
- `internal/providerhistory` remains the source of truth for applied evidence pointers, projection report cloning, and response-id-chain disable decisions.
- `internal/agent` now calls `providerhistory.AppliedEvidencePointers`, `providerhistory.CloneProjectionReport`, and `providerhistory.ProjectionDisablesResponseIDChain` directly from adapter/status paths instead of keeping package-local pass-through wrappers.
- Agent-side tests no longer duplicate providerhistory pure policy tests for applied evidence pointer filtering or projection report clone internals; those stay in `internal/providerhistory`, while agent tests keep rehydrate-plan and request/status caller coverage.

## 5. Responsibility Boundaries

- `internal/agent`: high-level orchestration、public compatibility facade、runtime/session/tool/provider integration。
- `internal/agent/plan`: plan parser / extractor / plan-domain pure logic / approved-plan implementation handoff projection。
- `internal/agent/viewfmt`: status / table / token / cost / file-size などの deterministic display formatting。
- `internal/agent/viewfmt_facade.go`: existing `agent` public formatting functions の compatibility facade。
- `internal/app`: CLI/TUI entrypoint wiring。`agent` を直接深く変形しない。
- `internal/tuiagent`: TUI adapter。Bubble Tea/TUI lifecycle はここか `internal/tui` に閉じる。
- `internal/providerhistory`: provider-facing projection/report mutation の source of truth。
- `internal/mcp` / `internal/mcptool` / `internal/mcpnames`: MCP config/connect/runtime wrapper/exported-name contract の owner。

## 6. Implementation Priority

1. Display formatting pure extraction into `internal/agent/viewfmt`.
2. Command/status rendering helper consolidation, without changing output text.
3. Plan mode and normal-mode plan parsing boundary review, using existing `internal/agent/plan`.
4. Provider-history adapter boundary review, without moving providerhistory policy into `agent`.
5. MCP surface adapter boundary review, without changing MCP contract.
6. Project-map state boundary review, including prompt and tool execution context paths.

## 7. Implementation Sections

## 7.1 Display Formatting Owner Cleanup

### Purpose

Move deterministic formatting policy for numbers, token counts, and file sizes from `stats.go` into `internal/agent/viewfmt`, while preserving `agent.FormatNumber`, `agent.FormatTokens`, and `agent.FormatFileSize` as compatibility wrappers.

### Non-goals

- No output format change.
- No cost/pricing/token accounting behavior change.
- No public API removal.

### Current source findings

- `stats.go` currently owns `FormatFileSize`, `FormatTokens`, and `FormatNumber`.
- `command_status_helpers.go` already delegates `formatNumber` to `viewfmt.Number`.
- `view_renderer.go` already imports `viewfmt` for USD formatting but still calls `FormatFileSize` for session file size.
- Existing tests cover agent-level wrapper behavior and viewfmt number/USD/string helpers.

### Design contract

- `viewfmt` owns pure display formatting.
- `agent` keeps exported wrappers for compatibility and existing callers.
- Tests should cover both the pure owner (`viewfmt`) and the agent facade where public compatibility matters.

### Safety gates

- Same input produces byte-for-byte same output.
- No import from `viewfmt` back into `agent`.
- No TUI/UI dependency added to `viewfmt`.

### Tests

- Add or extend `internal/agent/viewfmt` tests for file size and token formatting.
- Keep existing `agent` wrapper tests passing.
- Run `go test ./internal/agent/viewfmt ./internal/agent`.

### Implementation owner candidates

- `internal/agent/viewfmt/format.go`
- `internal/agent/viewfmt/format_test.go`
- `internal/agent/stats.go`
- Existing agent tests in `stats_test.go`, `stats_metrics_extra_test.go`, `viewfmt_facade_test.go`

## 7.2 Command / Status Rendering Boundary Review

### Purpose

After display formatting owner cleanup, review `command_status_helpers.go` and `view_renderer.go` for duplicate formatting wrappers and table-rendering responsibilities.

### Non-goals

- No `/status` output text change.
- No cost calculation change.
- No TUI rendering ownership change.

### Current source findings

- `command_status_helpers.go` mixes request usage policy, session info lookup, status-specific display policy, and command dispatch aliases.
- `view_renderer.go` owns table construction and display rows.
- Trivial USD / first-line / truncation wrappers have been removed from `command_status_helpers.go`; remaining helpers encode status-specific policy or orchestration.
- Before the command surface tranche, `agent_commands.go` mixed command dispatch/surface policy, confirmation prompting, history/exit commands, and common command header rendering.

### Design contract

Pure formatting belongs in `viewfmt`; table construction remains in `agent` unless a stable smaller owner emerges without broad export growth.

### Implemented boundary

- `command_surface_dispatch.go` owns `handleSpecialCommand`, surface filtering, unsupported-surface warnings, catalog/runtime parsing facade, and `splitCommand` compatibility.
- `command_confirmation.go` owns slash-command confirmation through `uiruntime.Runtime` and command-confirm bypass behavior.
- `command_history.go` owns `/history` rendering and `/exit` process termination.
- `command_render_header.go` owns shared command header rendering.
- `command_surface_runtime_test.go` owns command runtime splitting compatibility; `viewfmt_facade_test.go` owns the agent-local `formatNumber` compatibility facade.
- No command output text, dispatch matrix, catalog ownership, or command alias behavior was changed.

### Tests

- Existing status tests must remain green.
- If helper movement changes test setup readability, use `test-boundary-refactor`.

## 7.3 Plan Mode Boundary Review

### Purpose

Use existing `internal/agent/plan` as the source of truth for parser/domain behavior and keep runtime orchestration in `agent`.

### Non-goals

- No plan JSON shape change.
- No Plan Mode user-visible flow change.

### Current source findings

- `internal/agent/plan` is already a subpackage for plan parser/domain.
- `internal/agent` still contains plan orchestration files such as `plan_mode.go`, `plan_request.go`, and `turn_runner_normal_planning.go`.
- Approved-plan implementation handoff string construction moved from `internal/agent/plan_handoff.go` to `internal/agent/plan/handoff.go`.
- `internal/agent/plan_verification_followthrough.go` now delegates plan-derived verification hint normalization to `plan.CompactVerificationHints`.

### Design contract

Parser/domain/projection logic may move toward `internal/agent/plan`; request lifecycle, provider calls, tool visibility, session persistence, and state updates stay in `agent`.

### Implemented boundary

- `plan.ImplementationHandoff` owns `NormalModeInput` and `VerificationHints`.
- `plan.CompactVerificationHints` owns plan-derived verification trim/dedupe policy for both implementation handoff and plan review display.
- `agent` receives a handoff object from Plan Mode and uses it only to seed the normal implementation turn.
- The old `internal/agent/plan_handoff.go` unit tests moved to `internal/agent/plan/handoff_test.go`, while caller-path tests remain in `internal/agent`.

## 7.4 Provider-History Adapter Boundary Review

### Purpose

Keep provider-facing projection/report mutation in `internal/providerhistory`; `internal/agent` should only consume reports and decide request-local runtime consequences.

### Non-goals

- No provider-facing payload behavior change.
- No response-id chain behavior change unless a separate bug task is opened.

### Current source findings

- Memory from prior work identifies `internal/providerhistory` as projection/report owner.
- `internal/agent/provider_history_request_context.go` consumes reduction reports for response-id-chain behavior.
- Agent previously kept pass-through wrappers around providerhistory `AppliedEvidencePointers`, `CloneProjectionReport`, and `ProjectionDisablesResponseIDChain`, plus duplicate tests for providerhistory-owned policy.

### Safety gates

- Provider-facing data loss is the primary risk.
- Run providerhistory-focused tests if this area is touched.

### Implemented boundary

- `internal/providerhistory` owns report clone policy, applied evidence pointer filtering/dedupe/copying, and projection response-id-chain disable reporting.
- `internal/agent` owns request-local side effects: recording the last report, clearing provider/session response context, appending active context blocks, and rendering status counts.
- `internal/agent/provider_history_rehydrate_plan_test.go` now covers agent rehydrate-plan adapter behavior only. Pure pointer policy remains covered by `internal/providerhistory` tests.

## 7.5 MCP Surface Adapter Boundary Review

### Purpose

Review `agent_mcp.go`, `mcp_tool_surface.go`, and related status/prompt integration as adapter code only.

### Non-goals

- No MCP config/load/connect/schema/runtime wrapper contract change.
- No exported-name source of truth change.

### Current source findings

- MCP contract owners are `internal/mcp`, `internal/mcptool`, and `internal/mcpnames`.
- Prior MCP work fixed cancellation, structured args, and exported-name behavior outside `agent`.
- Read-only boundary map:
  - `internal/agent/agent_mcp.go` is an adapter from `mcp.MCPTool` to prompt/provider/registry surfaces. It depends on `internal/mcp`, `internal/mcpnames`, `internal/mcptool`, `internal/prompt`, and provider `api`.
  - Before this tranche, `internal/agent/mcp_tool_surface.go` mixed pure selected/omitted tool-surface budgeting with runtime adapter methods (`refreshMCPToolSurface`, `currentMCPToolSurface`, `configureCurrentProviderMCPTools`).
  - `internal/mcpsurface` owns sanitized analysis/report DTOs and formatting only; it does not import `internal/mcp` and therefore cannot directly own selection of raw `mcp.MCPTool` values without creating an ownership/API change.
  - `internal/mcptool` owns registry wrapper execution, approval enforcement, argument validation, and timeout/cancel behavior.
  - `internal/mcpnames` is the source of truth for exported MCP tool names across prompt/provider/registry tests.
  - `internal/agent/mcp_output_guard.go` is runtime result guarding for MCP tool output and raw output artifact handoff; it is provider-facing/security-sensitive and should not be mixed into simple surface-selection cleanup.

### Safety gates

- If runtime/config/schema/docs behavior needs to change, stop and split into a dedicated MCP task.
- Do not move selection into `internal/mcpsurface` unless the new API and dependency direction are designed explicitly. A quick move would either create an import cycle or broaden exported surface across MCP packages.
- Do not change `mcpnames.ExportedToolName`, `mcptool.RegisterToRegistry`, MCP approval behavior, schema conversion, timeout/cancel behavior, or raw-output artifact behavior in this refactor tranche.

### Implemented boundary

- `internal/agent/mcp_tool_surface_selection.go` owns agent-local selected/omitted surface budgeting, round-robin ordering, omission reasons, exported-name list derivation, and conversion to `mcpsurface.Tool` analysis inputs.
- `internal/agent/mcp_tool_surface.go` now owns runtime warning emission and `Agent` methods that refresh/current/configure the active MCP tool surface.
- `internal/agent/agent_mcp.go` remains the adapter from `mcp.MCPTool` to prompt/provider/registry definitions and continues to use `mcpnames.ExportedToolName` as the exported-name source of truth.
- `internal/agent/mcp_tool_surface_selection_test.go` owns pure surface selection/budget/analysis policy tests.
- `internal/agent/agent_mcp_test.go` retains provider/prompt/registry cross-surface contract tests, duplicate exported-name behavior, timeout preservation, and provider debug logging tests.
- `internal/agent/agent_mcp_test_support_test.go` owns shared MCP test fixtures used by agent MCP surface tests and MCP output guard tests.
- No package-level move was performed. A move into `internal/mcpsurface` or MCP owner packages remains a separate MCP-owner refactor because it would broaden API surface and dependency direction.

### Verification evidence

- `go test ./internal/agent -run 'TestSelectMCPToolSurface|TestMCPToolSurface|TestDeniedMCPToolsDoNotReachPromptProviderOrRegistrySurface|TestConfigureMCPTools|TestMCPExportedName|TestMCPToolDefinitionsPreserveCallTimeoutForRegistry|TestArchitectureBoundaries'`
- `go test ./internal/mcp ./internal/mcptool ./internal/mcpnames ./internal/providerhistory`
- `go test ./internal/agent ./internal/agent/plan ./internal/agent/viewfmt ./internal/app ./internal/tuiagent ./cmd`

## 7.6 Project-Map State Boundary Review

### Purpose

Review project-map prompt and runtime state ownership together so prompt output and tool execution context cannot drift.

### Non-goals

- No project-map prompt content change.
- No root-resolution behavior change.

### Current source findings

- `agentProjectPromptState` owns project-map state.
- Prior regression risk was prompt path clearing without clearing `toolExecutionContext` fields.
- Memory from prior project-map lifecycle work identifies unavailable-source cleanup as a joint prompt/runtime-state contract. Project map disabled, ripgrep unavailable, cwd unavailable, and project root unavailable must clear both prompt-visible project map sections and runtime fields exposed through `toolExecutionContext`.
- `internal/agent/agent_project_map_state.go` owns build/reuse/state-key decisions for the cached `repomap.ProjectMap`.
- `internal/agent/agent_project_map_injection_prepare.go` owns source resolution and unavailable-source cleanup.
- `internal/agent/prompt_manager_refresh.go` owns dirty/freshness decisions and triggers rebuilds through `PromptManager`.
- `internal/agent/agent_tool_executor.go` projects `agentProjectPromptState` into `tools.ExecutionContext` via `ProjectMap`, `ProjectMapRootPath`, and `ProjectMapStateKey`.
- Before this tranche, `agentProjectPromptState` and its reset/clear/has-state methods lived in `agent.go`, away from the project prompt/project map owner files.

### Tests

- If touched, include `go test -tags grammar_set_core ./internal/agent`.

### Implemented boundary

- `agentProjectPromptState` and its project-map lifecycle methods now live in `internal/agent/agent_project_prompt_state.go`.
- `agent.go` no longer imports `internal/repomap` just to define project prompt state; it remains the high-level `Agent` state container.
- Project-map policy tests formerly in the vague `agent_helpers_test.go` were renamed to `agent_project_map_policy_test.go` so budget, focus-path, base/focus key, and refresh-decision tests have an explicit test owner.
- Project-map prompt/runtime tests formerly in `agent_repl_test.go` were split by contract:
  - `agent_project_map_prompt_injection_test.go` owns injection, focus overlay, manifest budget, and full-runtime-map retention tests.
  - `agent_project_map_prompt_runtime_test.go` owns refresh/rebuild/dirty-state/runtime-context and project-map section stripping tests.
  - `agent_project_map_test_support_test.go` owns shared project-map test helpers used by project-map and project-config tests.
  - `agent_ripgrep_test.go` owns ripgrep availability warnings outside the project-map prompt/runtime owner.
- No project-map prompt content, state-key algorithm, root resolution, cache invalidation behavior, or tool execution context shape was changed.

### Verification evidence

- `go test -tags grammar_set_core ./internal/agent -run 'TestCalcProjectMapBudget|TestEffectiveProjectMapContextRatio|TestExtractProjectMapFocusPaths|TestBuildProjectMap|TestShouldRefreshProjectPrompt|TestProjectPromptRefreshDecision|TestCurrentProjectMapStateKey|TestRefreshProjectPrompt_ClearsProjectMapStateWhenProjectRootDisappears|TestRefreshProjectPromptIfDirty_ClearsProjectMapStateWhenProjectMapDisabled|TestRefreshProjectPromptIfDirty_RebuildsProjectMapAfterToolMutation|TestArchitectureBoundaries'`
- `go test ./internal/agent -run 'TestCalcProjectMapBudget|TestEffectiveProjectMapContextRatio|TestExtractProjectMapFocusPaths|TestBuildProjectMap|TestShouldRefreshProjectPrompt|TestProjectPromptRefreshDecision|TestCurrentProjectMapStateKey|TestRefreshProjectPrompt_ClearsProjectMapStateWhenProjectRootDisappears|TestRefreshProjectPromptIfDirty_ClearsProjectMapStateWhenProjectMapDisabled|TestRefreshProjectPromptIfDirty_RebuildsProjectMapAfterToolMutation|TestArchitectureBoundaries'`
- `go test -tags grammar_set_core ./internal/agent -run 'TestInjectProjectMap|TestRefreshProjectPrompt|TestRefreshProjectPromptIfDirty|TestCurrentProjectMapStateKey|TestNoteProjectMapMutation|TestExtractProjectMapSection|TestStripProjectMapSection|TestCheckRipgrepAvailability|TestSaveAndSyncProjectConfigRefreshesProjectMapIgnorePatterns|TestHeadless_SearchCodeUsesFreshProjectMapRuntimeAfterEdit'`
- `go test ./internal/agent -run 'TestInjectProjectMap|TestRefreshProjectPrompt|TestRefreshProjectPromptIfDirty|TestCurrentProjectMapStateKey|TestNoteProjectMapMutation|TestExtractProjectMapSection|TestStripProjectMapSection|TestCheckRipgrepAvailability|TestSaveAndSyncProjectConfigRefreshesProjectMapIgnorePatterns|TestArchitectureBoundaries'`
- `go test -tags grammar_set_core ./internal/agent`

## 8. Mode / Policy / Defaults

This refactor is behavior-preserving. There is no new public mode, config key, env override, or default behavior.

## 9. Config / Docs / Generated Metadata Surface

No config schema, docs/config.md, README, generated metadata, or example config changes are intended. If a required change appears, stop and reclassify with the relevant contract Skill.

## 10. Report / Status / Observability

No user-visible report/status output changes are intended. Refactor reporting is limited to final implementation notes: changed owner/source of truth, kept public contract, tests run, Final-A/Final-B findings, and remaining debt.

## 11. Tests

- Baseline: `go test ./internal/agent ./internal/agent/plan ./internal/agent/viewfmt ./internal/app ./internal/tuiagent ./cmd`
- Display formatting tranche: `go test ./internal/agent/viewfmt ./internal/agent`
- Project-map/LSP/grammar touched: `go test -tags grammar_set_core ./internal/agent`
- MCP touched: `go test ./internal/mcp ./internal/mcptool ./internal/mcpnames ./internal/providerhistory` plus provider schema builder packages as needed.
- Final gate: `make ci-check`

## 12. Verification Commands

```sh
gofmt -w <changed go files>
go test ./internal/agent/viewfmt ./internal/agent
go test ./internal/agent ./internal/agent/plan ./internal/agent/viewfmt ./internal/app ./internal/tuiagent ./cmd
make ci-check
```

## 13. Goal Handoff Policy

Use this file as the source of truth. Re-read it after resume or context compaction. If the latest source structure conflicts with this plan, preserve the safety contracts and adapt to existing owner boundaries.

Do not weaken `internal/agent` import boundary tests. Do not commit or push unless explicitly requested.

## 14. Pre/Post Implementation Refactor Policy

### Phase 0: owner map / baseline / test foundation

Create and update this plan, collect current public surface and caller facts, and verify baseline tests before editing.

### Phase 1-N: implementation phases

Proceed from low-risk pure extraction to broader owner reviews. If a package split becomes necessary, confirm it has a small API and natural dependency direction; otherwise keep the change as file/helper extraction.

### Phase Final-A: impact audit / review-hole sweep

Before final response, inspect the diff for accidental public contract changes, output text changes, provider-facing data loss, caller breakage, config/docs drift, and test gaps. Fix behavior-preserving issues found in the current owner graph.

### Phase Final-B: mandatory comprehensive refactor including tests

After tests pass, inspect production and test diffs for duplicate helpers, wrong owner, wrapper accumulation, large-file growth, fixture drift, and assertion duplication. Behavior-preserving cleanup in the touched owner graph is mandatory; unrelated cleanup is reported as debt.

## 15. Implementer Freedom

Implementation may choose exact helper names, file names, and test placement as long as the owner/source of truth contracts above remain true and public compatibility is preserved.

## 16. Open Decisions

- Whether any later phase deserves package split rather than same-package file/helper extraction. Default: do not split unless export surface stays small and focused tests can live with the new owner.

## 17. Goal Prompt

```text
/goal Implement docs/dev/internal-agent-refactor-master-plan.md end to end.

Use docs/dev/internal-agent-refactor-master-plan.md as the source of truth. Start with Phase 0 owner map / baseline, then implement the planned behavior-preserving refactor sections, then run the impact audit and mandatory post-implementation refactor described in the plan.

Final-B is mandatory, not optional. Preserve all public contracts, non-goals, safety gates, and verification requirements. Re-read the plan after resume or context compaction. Do not commit or push unless explicitly requested.
```

## 18. Current Tranche Final-A / Final-B Notes

### Final-A impact recovery result

- Classification: `STRUCTURAL_ONLY` after audit. No provider-facing payload behavior, headless JSON shape, session/history persistence, config default, MCP exported name, prompt content, or TUI lifecycle contract was intentionally changed.
- Public surface: `go doc ./internal/agent` still exposes the existing agent-level run functions, headless result constructors, formatting facades, providerhistory aliases, runtime types, and `Agent` methods. The old plan handoff symbols were private to `agent`; moving the pure handoff projection to `internal/agent/plan` did not remove an agent exported API.
- Provider-facing gate: providerhistory projection/report behavior remains owned by `internal/providerhistory`. The agent diff only removed pass-through wrappers and now calls `providerhistory.AppliedEvidencePointers`, `providerhistory.CloneProjectionReport`, and `providerhistory.ProjectionDisablesResponseIDChain` directly from adapter/status paths.
- MCP surface gate: MCP config/load/connect/runtime wrapper behavior remains owned by `internal/mcp` and `internal/mcptool`; exported names remain owned by `internal/mcpnames`. The agent diff only split agent-local tool-surface selection/budgeting from runtime/provider wiring.
- Command surface gate: command catalog ownership and commandruntime parsing remain owned by `internal/commandcatalog` and `internal/commandruntime`; the agent diff only split agent-local dispatch/confirmation/history/rendering files.
- Project-map state gate: `agentProjectPromptState` remains embedded in `Agent` and still projects into `tools.ExecutionContext` through the same `ProjectMap`, `ProjectMapRootPath`, and `ProjectMapStateKey` fields. The tranche moved the state owner out of `agent.go`; it did not change prompt content, state-key calculation, root resolution, dirty marking, cache invalidation, or unavailable-source cleanup semantics.
- Counterexample/invariant coverage: providerhistory owner tests cover applied evidence pointer filtering/dedupe/copying, structured `list_dir` exclusion, projection report clone defensive copy, and response-chain disable reporting. Agent tests cover rehydrate-plan nil/runtime adapter paths, status output, and request/status caller paths.
- Project-map invariant coverage: focused grammar-tag tests cover project-root disappearance, project-map-disabled cleanup, dirty tool mutation rebuild, current state key behavior, budget/focus/base-key policy, refresh-decision policy, and architecture boundaries.
- Repair decision: `NO_CHANGE` for correctness. No impact recovery code change was needed after the providerhistory adapter owner cleanup.

### Final-B behavior-preserving refactor result

- Display formatting owner is now `internal/agent/viewfmt`; `internal/agent/viewfmt_facade.go` preserves `agent.FormatNumber`, `agent.FormatTokens`, and `agent.FormatFileSize` for existing callers.
- Plan implementation handoff owner is now `internal/agent/plan`; `internal/agent` keeps Plan Mode orchestration, approval flow, session/state restoration, and normal-mode execution.
- Providerhistory pure policy wrappers were removed from `internal/agent`; providerhistory policy tests were not duplicated in agent tests.
- MCP tool surface selection owner is now explicit in `internal/agent/mcp_tool_surface_selection.go`; `internal/agent/mcp_tool_surface.go` now contains only runtime warning and `Agent` current/refresh/configure wiring.
- Command surface dispatch owner is now explicit in `internal/agent/command_surface_dispatch.go`; confirmation, history/exit, and command header rendering are split from the former `agent_commands.go` bucket.
- Project-map prompt/runtime state owner is now explicit in `internal/agent/agent_project_prompt_state.go`; `agent.go` no longer owns the project-map state type or lifecycle helper methods.
- Test boundary cleanup: plan handoff tests moved to `internal/agent/plan/handoff_test.go`; view formatting owner tests live in `internal/agent/viewfmt/format_test.go`; agent providerhistory tests retain caller/adapter behavior only.
- Test boundary cleanup: MCP surface selection tests moved to `mcp_tool_surface_selection_test.go`; shared MCP test fixtures moved to `agent_mcp_test_support_test.go`; cross-surface provider/prompt/registry tests remain in `agent_mcp_test.go`.
- Test boundary cleanup: command runtime splitting tests moved to `command_surface_runtime_test.go`; `formatNumber` facade compatibility tests moved to `viewfmt_facade_test.go`; the generic `agent_commands_test.go` bucket was removed.
- Test boundary cleanup: project-map policy tests formerly in `agent_helpers_test.go` moved to `agent_project_map_policy_test.go`. This keeps budget/focus/key/refresh policy tests under an explicit test owner instead of a generic helper file.
- Test boundary cleanup: project-map prompt/runtime tests formerly in `agent_repl_test.go` moved to `agent_project_map_prompt_injection_test.go` and `agent_project_map_prompt_runtime_test.go`; shared project-map fixtures moved to `agent_project_map_test_support_test.go`; ripgrep availability tests moved to `agent_ripgrep_test.go`.
- No package named `utils`, `helpers`, or `common` was introduced. No import-cycle-only interface/wrapper was introduced.

### Verification evidence for current tranche

- `git diff --check`
- `go test ./internal/agent/plan ./internal/agent/viewfmt ./internal/providerhistory`
- `go test ./internal/agent -run 'TestProviderHistoryRehydratePlan|TestProviderHistoryReduction|TestPhase5DStatusDiagnosticUsesLastProviderFacingRequestReport|TestProviderHistory.*ResponseID|TestProviderHistory.*Status|TestArchitectureBoundaries|TestFormatFileSize|TestFormatTokens|TestFormatNumber'`
- `go test -tags grammar_set_core ./internal/agent -run 'TestCalcProjectMapBudget|TestEffectiveProjectMapContextRatio|TestExtractProjectMapFocusPaths|TestBuildProjectMap|TestShouldRefreshProjectPrompt|TestProjectPromptRefreshDecision|TestCurrentProjectMapStateKey|TestRefreshProjectPrompt_ClearsProjectMapStateWhenProjectRootDisappears|TestRefreshProjectPromptIfDirty_ClearsProjectMapStateWhenProjectMapDisabled|TestRefreshProjectPromptIfDirty_RebuildsProjectMapAfterToolMutation|TestArchitectureBoundaries'`
- `go test -tags grammar_set_core ./internal/agent -run 'TestInjectProjectMap|TestRefreshProjectPrompt|TestRefreshProjectPromptIfDirty|TestCurrentProjectMapStateKey|TestNoteProjectMapMutation|TestExtractProjectMapSection|TestStripProjectMapSection|TestCheckRipgrepAvailability|TestSaveAndSyncProjectConfigRefreshesProjectMapIgnorePatterns|TestHeadless_SearchCodeUsesFreshProjectMapRuntimeAfterEdit'`
- `go test ./internal/agent -run 'TestInjectProjectMap|TestRefreshProjectPrompt|TestRefreshProjectPromptIfDirty|TestCurrentProjectMapStateKey|TestNoteProjectMapMutation|TestExtractProjectMapSection|TestStripProjectMapSection|TestCheckRipgrepAvailability|TestSaveAndSyncProjectConfigRefreshesProjectMapIgnorePatterns|TestArchitectureBoundaries'`
- `go test ./internal/agent -run 'TestSelectMCPToolSurface|TestMCPToolSurface|TestDeniedMCPToolsDoNotReachPromptProviderOrRegistrySurface|TestConfigureMCPTools|TestMCPExportedName|TestMCPToolDefinitionsPreserveCallTimeoutForRegistry|TestArchitectureBoundaries'`
- `go test ./internal/agent -run 'TestSplitCommand|TestFormatNumber|TestSpecialCommandDispatch|TestHandleSpecialCommandForSurface|TestSpecialCommandRegistryDoesNotRegisterTUILocalOwners|TestSetupCommandProjectConfigInstructionMatchesSurface|TestResolveConfigCommandSurfacePolicy|TestPromptConfirm|TestHandleHistoryCommand|TestHandleExitCommand'`
- `go test ./internal/mcp ./internal/mcptool ./internal/mcpnames ./internal/providerhistory`
- `go test -tags grammar_set_core ./internal/agent`
- `go test ./internal/agent ./internal/agent/plan ./internal/agent/viewfmt ./internal/app ./internal/tuiagent ./cmd`
- `make ci-check`

### Remaining debt / next phases

- MCP package-owner refactor remains out of scope. Do not change MCP runtime/config/schema/docs from agent; split into a dedicated MCP task if those owners need changes.
- Larger `internal/agent` orchestration boundaries remain intentionally unsplit in this tranche because splitting them now would require wider public/facade design across `internal/app`, `internal/tuiagent`, `cmd`, and `e2e`.

## 19. Next Tranche Result: Legacy REPL / Command Config / Status Rendering / Raw Output Context Split

### Classification

- Classification: `STRUCTURAL_ONLY`. This tranche removed the remaining generic files `agent_repl.go`, `agent_commands_config.go`, `view_renderer.go`, and `provider_history_raw_output_context.go` by splitting them into owner-named files.
- Public API / config schema / persisted session/history format / headless JSON / MCP contract were not changed.
- No new package was introduced. Same-package file split was sufficient; adding exports only to move private helpers would have been the wrong owner tradeoff for this tranche.

### Implemented owner map

- Legacy classic REPL:
  - `agent_repl_entry.go` owns `RunLegacyInteractive*` and deprecated `RunInteractive*` compatibility wrappers.
  - `agent_repl_runtime.go` owns runtime/UI/reader/audit/project-instruction/LSP startup setup.
  - `agent_repl_loop.go` owns `runREPLLoop`.
  - `agent_repl_signal.go` owns `setupSignalHandler`.
  - `agent_repl_context_size.go` owns `printContextSize`, `buildContextSizeBlock`, and context-size token section helpers.
  - `agent_repl_environment_checks.go` owns the ripgrep availability notice.
- Command config/provider surface:
  - `command_model.go` owns `/model` display and model switch output.
  - `command_config.go` owns `/config show`, `/config model`, and interactive delegation.
  - `command_config_interactive.go` owns interactive config menu dependencies, scalar/struct-map validation, save/restore, and menu refresh.
  - `command_provider_switch.go` owns `/provider` and `/use` provider/model switching.
  - `command_providers_status.go` owns `/providers` credential status rendering.
- Status/session rendering:
  - `view_request_usage.go` owns last request usage table and shared web-search usage rows.
  - `view_session_tables.go` owns session overview/token/cost/review rows.
  - `view_session_sections.go` owns session section orchestration.
  - `view_subagent_render.go` owns sub-agent rows and breakdown rendering.
  - `view_tool_observability.go` owns tool-observability and savings sections.
- Provider-history raw output active context:
  - `provider_history_raw_output_active_context.go` owns Agent/resolver/build path and fail-closed active-context result state.
  - `provider_history_raw_output_refs.go` owns applied raw-output ref selection.
  - `provider_history_raw_output_render.go` owns metadata/body rendering.
  - `provider_history_raw_output_excerpt.go` owns excerpt matching, trimming, and coverage-insufficient classification.
  - `provider_history_raw_output_hints.go` owns request hint extraction.
  - `provider_history_raw_output_budget.go` owns active-context budget resolution.

### Test boundary result

- REPL tests already had focused owner files and remained aligned with production owner: `agent_repl_entry_test.go`, `agent_repl_loop_test.go`, `agent_repl_context_size_test.go`, `agent_ripgrep_test.go`.
- `agent_commands_config*_test.go` files were removed and replaced by owner-focused tests:
  - `command_model_test.go`
  - `command_config_test.go`
  - `command_config_interactive_test.go`
  - `command_config_gemini_test.go`
  - `command_provider_switch_test.go`
  - `command_providers_status_test.go`
  - `command_config_test_support_test.go`
  - `command_provider_registration_test.go`
- Raw-output request tests keep provider-facing caller-path cases in `provider_history_raw_output_request_test.go`; fake stores/setup helpers moved to `provider_history_raw_output_request_support_test.go`.

### Final-A impact recovery result

- Public surface: `go doc ./internal/agent` still exposes the same exported run functions, headless result constructors, formatting facades, providerhistory aliases, runtime types, and `Agent` methods.
- Command output text: command-focused tests passed after the file split. The command catalog and `commandruntime` parsing owners were not changed.
- Config/defaults: config schema, defaults, generated metadata, docs/config.md, and migration behavior were not touched. `/config` changes were file-boundary only.
- Session/history/headless JSON: legacy REPL/session resume tests and broader `internal/app`, `internal/tuiagent`, and `cmd` tests passed. No JSON shape changes were made.
- MCP boundary: no MCP runtime/config/schema/exported-name code was touched in this tranche.
- Provider-facing raw output gate: caller-path tests passed for applied command/MCP/web-search artifact replacement, fail-closed coverage-insufficient fallback, missing resolver fallback, too-small budget fallback, token-budget no-materialization, and immutable raw `Agent.History` / session messages.

### Final-B behavior-preserving refactor result

- `MUST` completed:
  - Remove the remaining generic production buckets listed above.
  - Split command tests by model/config/interactive/provider/provider-status owner.
  - Extract raw-output request support helpers so provider-facing data retention tests read as contracts rather than fixture plumbing.
  - Keep raw-output active-context policy in `agent` local files without widening package API or moving provider-facing policy into an unrelated package.
- `SHOULD` left out of scope:
  - Package-level split for legacy REPL or status rendering. Current callers are package-private and a package split would require export/facade design without reducing provider-facing or config risk in this tranche.
  - Moving raw-output active-context policy into `internal/providerhistory`. The current code depends on `AgentRuntime`, active-context provider capability, and raw artifact store wiring, so that requires a separate adapter contract design.
- `NO` for config/docs/generated updates because no config key, default, generated file, public doc, or public behavior changed.

### Verification evidence for next tranche

- `go test ./internal/agent -run 'TestRunInteractiveWith|TestBuildContextSizeBlock|TestREPL|TestSignal|TestCheckRipgrep'`
- `go test ./internal/agent -run 'TestHandleModelCommand|TestHandleConfigCommand|TestRunInteractiveConfig|TestHandleProvider|TestHandleUse|TestHandleProviders|TestProviderCredentialStatusDisplay|TestIsNonInteractiveConfigSubcommand'`
- `go test ./internal/agent -run 'TestRender|TestStatus|TestPrintTaskUsage|TestProviderHistoryRawOutput|TestNormalModeRequestApply.*RawOutput|TestTokenBudgetHistoryDoesNot.*RawOutput|TestNormalModeRequestApplyCompactsMCPResult|TestNormalModeRequestApplyCompactsWebSearchResult'`
- `go test ./internal/providerhistory`
- `go test ./internal/agent ./internal/agent/plan ./internal/agent/viewfmt ./internal/app ./internal/tuiagent ./cmd`
- `go doc ./internal/agent`
- `git diff --check`
- `make ci-check`
