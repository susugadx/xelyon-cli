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

## Pending tranche notes

- `/config`: test-boundary pass is done; production move can now start from centralized root test helpers. Do not add broad production getters/setters/test hooks just to preserve old root assertions. Next tranche should move `configScreen` state/input/render/editor/save-result ownership to `internal/tui/configscreen` and replace test helpers with package-local screen tests plus root caller tests.
- `/project`: done for state/input/render/save-result. Later cleanup can reduce root caller tests further if package-local coverage grows, but no root private field dependency remains.
- Provider picker: done for state/input/panel rendering. Later cleanup can consider sharing overlay row styling with session picker if it can be done without a generic UI bucket.
