# internal/tui root boundary refactor master plan

## Goal

`internal/tui` root package を挙動維持で分割し、screen-specific state/input/render の owner を subpackage へ移す。`internal/app` と `internal/tuiagent` から見える TUI API、slash command semantics、key binding、prompt/config/project/review behavior、startup/resume lifecycle は変えない。

## Non-goals

- UX、表示文言、key binding、CLI flag、config schema の変更。
- root package からの薄い compatibility wrapper / alias の追加。
- import cycle 回避だけを目的にした interface / forwarding helper の追加。

## Boundary map

- `internal/tui` root: `Model` orchestration、agent invocation、modal lifecycle、chat forwarding、config/project/review save/run command、provider/session resume or switch execution。
- `internal/tui/configscreen`: `/config` の state/input/render/edit/save-result handling。root には config load/save command と chat forwarding だけを残す。
- `internal/tui/promptmodal`: prompt modal の choice/text state、key handling、overlay rendering、response construction。root には prompt open/cancel lifecycle、background forwarding、response channel send を残す。
- `internal/tui/slashsuggestions`: slash suggestion state、windowing、key decision、row rendering。root には composer mutation、footer budget、catalog sourcing、submit execution を残す。
- `internal/tui/attachments`: composer attachment の value type、kind/source、append limit/duplicate policy、dispatch prompt/display/context pure builders、dropped path parse DTO。root には attachment mutation orchestration、file/PDF/clipboard/path I/O、status mutation を残す。
- `internal/tui/projectscreen`: `/project` の state/input/render/save-result/template-result handling。root には project config load/save/template create command と chat forwarding だけを残す。
- `internal/tui/reviewscreen`: `/review` preset/custom の state/input/render と review report plain rendering。root には review timeline runner、agent activity、progress message handling を残す。
- `internal/tui/providerpickerscreen`: provider/model picker の UI state/input/render。root には candidates 取得、provider/model switch、Azure setup execution を残す。
- `internal/tui/sessionpickerscreen`: resume session picker の UI state/input/render。root には candidates 取得、session resume/start-new execution、startup orchestration を残す。

## Tranche log

### 2026-06-18 reviewscreen

- Moved: `review_screen_body.go`, `review_screen_input.go`, `review_screen_render.go`, `review_screen_types.go`, `review_report_render.go`, and report rendering tests to `internal/tui/reviewscreen`.
- New owner: `reviewscreen.Screen` owns preset/custom mode, custom input focus, notices, body viewport, key handling, full screen rendering, and report-to-plain-lines rendering.
- Root remains owner for: `openReviewScreen`, `updateReviewScreen`, `closeReviewScreen`, `startReviewTimeline`, busy rejection, agent activity, review run invocation, and usage-summary insertion into timeline text.
- Exported API: `Screen`, `New`, `Resize`, `View`, `HandleKey`, `SetNotice`, `ClearNotice`, `Snapshot`, `Mode`, `Command`, `PlainLines`, `TimelineMessage`.
- Kept private: preset catalog, preset action, body viewport state, custom input model, notice storage, key-specific mode handlers, report evidence formatting helpers.
- Tests moved/updated: report rendering tests now live in `reviewscreen`; root review tests assert public `Snapshot` instead of direct screen fields.
- Verification: `go test ./internal/tui ./internal/tui/reviewscreen ./internal/tuiagent -run 'TestReview|TestTUIAdapter' -count=1`.

### 2026-06-18 sessionpickerscreen

- Moved: `session_picker_state.go` and session picker panel rendering to `internal/tui/sessionpickerscreen`; root keeps only overlay composition and resume orchestration.
- New owner: `sessionpickerscreen.Screen` owns resume picker candidates, filter text, selection index, startup/all flags, key handling, panel rows, and row label formatting.
- Root remains owner for: `ResumeSessionCandidates`, cross-working-directory rejection, `ResumeSession` / `ResumeStartupSession`, transcript reset, status updates, and startup picker orchestration.
- Exported API: `Screen`, `Candidate`, `New`, `HandleKey`, `PanelLines`, `Snapshot`, `All`, `Startup`, `Command`, `KeyResult`.
- Kept private: filter matching, selected-row clamping, row-window calculation, label sanitization, and overlay row styling.
- Tests added/updated: `sessionpickerscreen` now has package-local key/filter/render tests; root startup picker test uses `Snapshot` instead of direct field access.
- Verification: `go test ./internal/tui ./internal/tui/sessionpickerscreen ./internal/app -run 'Test.*Picker|TestRunTUI|TestResume|TestStartup|TestHandleKey|TestPanelLines' -count=1`.

### 2026-06-18 providerpickerscreen

- Moved: `provider_picker_state.go` and provider picker panel rendering to `internal/tui/providerpickerscreen`; root keeps overlay composition and provider/model execution.
- New owner: `providerpickerscreen.Screen` owns provider/model/custom/Azure picker mode, filter text, selection index, custom input, key handling, panel rows, and provider/model row labels.
- Root remains owner for: provider/model candidate sourcing, status line refresh, `SwitchProviderModel`, `SwitchModelForCurrentProvider`, `ConfigureAndSwitchAzureDeployment`, and transient status messages.
- Exported API: `Screen`, `NewProvider`, `NewModel`, `HandleKey`, `PanelLines`, `Snapshot`, `ShowModels`, `BeginAzureCatalogModelSelection`, `ReturnToAzureDeploymentPicker`, `Provider`, `Mode`, `Step`, `Command`, `KeyResult`.
- Kept private: filter matching, custom-input placeholder policy, Azure step transitions, selected-row clamping, row-window calculation, and label formatting helpers.
- Tests added/updated: `providerpickerscreen` now has package-local action/render tests; root provider picker tests assert public `Snapshot` instead of direct fields.
- Verification: `go test ./internal/tui ./internal/tui/providerpickerscreen ./internal/app -run 'TestProviderPicker|Test.*Picker|TestRunTUI|TestHandleKey|TestPanelLines' -count=1`.

### 2026-06-19 projectscreen

- Moved: `project_screen_edit_state.go`, `project_screen_final_checks.go`, `project_screen_input*.go`, `project_screen_lists.go`, `project_screen_messages.go`, `project_screen_render*.go`, `project_screen_save_state.go`, and `project_screen_types.go` to `internal/tui/projectscreen`.
- New owner: `projectscreen.Screen` owns `/project` state, missing-config template state, browse/edit/confirm key handling, list/final-check mutation, save-result state transitions, and full screen rendering.
- Root remains owner for: project config load/save/template I/O, modal lifecycle, chat forwarding, `screenProject` activation/deactivation, and screenID allocation.
- Exported API: `Screen`, `New`, `View`, `HandleKey`, `HandleSaveResult`, `BeginSave`, `InstallTemplateResult`, `NormalizeSize`, `Snapshot`, `Command`, `SaveResult`, `SaveAction`, `PendingSave`, `TemplateResult`.
- Kept private: project panes/sections/edit modes/save status enums, item list mutation, final-check preservation policy, template-created status text, list windowing, status/header/detail rendering helpers.
- Tests added/updated: `projectscreen` now has package-local save-result/windowing/render sanitation tests; root `/project` tests use public `Snapshot` and key paths instead of direct field reads/mutation.
- Verification: `go test ./internal/tui ./internal/tui/projectscreen -run 'TestProjectScreen|TestProjectCommand|TestScreen|TestProjectList' -count=1`.

### 2026-06-19 config test-boundary

- Test boundary only: production `configScreen` files remain in `internal/tui` for this tranche.
- New test owner split: setup/key/save helpers stay in `model_config_helpers_test.go`; field selection/input helpers live in `model_config_selection_helpers_test.go`; struct-map key/entry helpers live in `model_config_structmap_helpers_test.go`.
- Root `/config` behavior tests now route category/field/entry selection and raw dirty/confirm setup through package-local helpers instead of mutating `configScreen` indices/input state inline.
- Guard result: direct `catIndex` / `fieldIndex` / `fieldScroll` / `activePane` / editor input/index assignments are confined to the new helper files.
- Verification: `go test ./internal/tui -run 'TestConfigScreen|TestModel_Config|TestProviderDefaultModel|TestProviderModel|TestGeminiFunctionCalling|TestTUIIntegration_Config' -count=1`; `go test ./internal/tui ./internal/tui/configscreen -run 'TestConfigScreen|TestProviderDefaultModel|TestProviderModel|TestGeminiFunctionCalling|TestPaneWidths|TestLayout' -count=1`.

### 2026-06-19 configscreen production

- Moved: `/config` state/input/render/editor/save-result files and pure render helper tests to `internal/tui/configscreen`.
- New owner: `configscreen.Screen` owns config screen state, pane navigation, scalar/slice/struct-map editors, validation, dirty/save-result state, provider default model sync, and full screen rendering.
- Root remains owner for: config load/save command creation, modal lifecycle, chat forwarding, `screenConfig` activation/deactivation, and agent config sync invocation after save.
- Exported API: `Screen`, `New`, `View`, `HandleKey`, `HandleSaveResult`, `BeginSave`, `ConfigSnapshot`, `Snapshot`, `NormalizePaneState`, `SyncEditedProviderDefaultModel`, `DefaultModelSyncProvider`, `AddStructMapKey`, `ValidateSaveSnapshot`, `Command`, `KeyResult`.
- Kept private: pane/category/field selection state, editor mode internals, render row helpers, struct-map mutation helpers, save-status enums, and config entry mutation details.
- Tests moved/updated: pure render helper tests live in `configscreen`; root `/config` tests use public snapshots and caller key/save paths instead of direct screen field construction.
- Verification: `go test ./internal/tui ./internal/tui/configscreen -run 'TestConfigScreen|TestModel_Config|TestProviderDefaultModel|TestProviderModel|TestGeminiFunctionCalling|TestPaneWidths|TestLayout' -count=1`.

### 2026-06-19 promptmodal

- Moved: prompt modal choice/text state, option construction, and overlay rendering to `internal/tui/promptmodal`.
- New owner: `promptmodal.Screen` owns prompt request state, choice/text mode key handling, overlay view construction, response action/value construction, and public snapshots.
- Root remains owner for: `OpenPromptMsg` / `CancelPromptMsg` lifecycle, background chat forwarding, prompt response channel send, and chrome rebuild.
- Exported API: `Screen`, `New`, `HandleKey`, `ViewOverlay`, `Snapshot`, `ID`, `Respond`, `KeyResult`, `ModeChoice`, `ModeText`.
- Kept private: choice row normalization, text input model, overlay layout helpers, and prompt response conversion details.
- Tests updated: root prompt tests assert public `Snapshot` and lifecycle behavior; overlay helpers for provider/session picker stay root-local to avoid depending on prompt modal internals.
- Verification: `go test ./internal/tui ./internal/tui/promptmodal -run 'TestPrompt|TestModel_.*Prompt|Test.*Overlay' -count=1`.

### 2026-06-19 slashsuggestions

- Moved: slash suggestion state, refresh/clear/windowing, key decision, selected detail, visible row DTO, and row rendering to `internal/tui/slashsuggestions`.
- New owner: `slashsuggestions.State` owns suggestion list selection, visible rows, chrome row count, key command classification, and render-row formatting.
- Root remains owner for: composer text/cursor, slash prefix parsing, footer row budget, provider/model/skill catalog sourcing, composer mutation, and command submission execution.
- Exported API: `State`, `Snapshot`, `Refresh`, `Clear`, `HandleKey`, `ActivateSelection`, `Visible`, `VisibleRows`, `VisibleRenderRows`, `ChromeRowCount`, `SelectedSuggestion`, `SelectedDetail`, `NewRenderRow`, `RenderRowString`, `KeyResult`.
- Kept private: selected index clamping, visible window calculation, render row truncation and detail formatting.
- Tests updated: root slash tests use public state methods; `internal/tui/slash` remains command/suggestion domain owner.
- Verification: `go test ./internal/tui ./internal/tui/slash ./internal/tui/slashsuggestions -run 'TestSlash|TestModel_Slash|TestSuggestions|TestProviderModel|TestSkill' -count=1`.

### 2026-06-19 attachments pure policy

- Moved: attachment value type, kind/source enums, append duplicate/limit decision, dispatch prompt/input/display builders, context block specs/builders, and dropped path parse DTO to `internal/tui/attachments`.
- New owner: `attachments` owns pure attachment policy and DTOs. Root no longer defines `composerAttachment` aliases or attachment kind/source constants.
- Root remains owner for: `Model.attachments` mutation orchestration, `os.Stat`, clipboard temp cleanup, file/PDF preview reads, path normalization / WSL conversion, dropped path token resolution, and transient status mutation.
- Exported API: `Attachment`, `Kind`, `Source`, `MaxComposerAttachments`, `AppendResult`, `PrepareAppend`, `SelectPrimaryImagePath`, `ResolveDispatchBasePrompt`, `BuildDispatchInput`, `BuildDispatchDisplay`, `ContextBlockSpec`, `ContextBlockSpecs`, `BuildAttachedImagePathContext`, `BuildAttachedFileContextBlock`, `BuildAttachedPDFContextBlock`, `DroppedPathParseKind`, `DroppedPathParseResult`.
- Kept private/root-local: attachment command handling, paste/drop attachability, image/file/PDF detection and reads, clipboard file lifecycle, path display/path candidate normalization.
- Tests added/updated: `attachments` has package-local pure policy tests; root attachment tests continue to cover caller lifecycle and I/O paths using `attachments.Attachment`.
- Verification: `go test ./internal/tui ./internal/tui/attachments -run 'TestAttachment|TestDropped|TestPaste|TestDispatch' -count=1`.

### 2026-06-19 agent activity file boundary

- Split: `model_agent_activity.go` into `model_agent_activity_state.go`, `model_agent_activity_update.go`, `model_agent_activity_tool.go`, and `model_agent_activity_render.go`.
- Owner boundary: state/options and constructor, activity lifecycle/update, tool upsert/error timing, and render formatting are now separate root files without adding exported API or package boundary.
- Tests split: `model_agent_activity_lifecycle_test.go`, `model_agent_activity_tool_test.go`, `model_agent_activity_render_test.go`, and shared `model_agent_activity_test_helpers_test.go`.
- Kept private/root-local: agent activity remains tied to root `Model`, transcript tracked blocks, viewport follow state, spinner/status snapshot, and tool-block fallback.
- Verification: `go test ./internal/tui -run 'TestAgentActivity|TestModel_AgentActivity' -count=1`.

## Pending tranche notes

- `/config`: production move is done. Later cleanup can reduce root caller tests further if package-local coverage grows, but root no longer owns screen state/input/render/editor/save-result.
- `/project`: done for state/input/render/save-result. Later cleanup can reduce root caller tests further if package-local coverage grows, but no root private field dependency remains.
- Provider picker: done for state/input/panel rendering. Later cleanup can consider sharing overlay row styling with session picker if it can be done without a generic UI bucket.
