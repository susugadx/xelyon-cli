# Internal UI refactor master plan

この文書は旧 `internal/ui` を owner package へ分割する挙動維持 refactor の source of truth です。CLI/TUI 表示、prompt semantics、config YAML/schema/default、provider streaming、`--no-tui` 退役判断は変更しない。

## Final owner map

- `internal/uiprompt`: `PromptRequest` / `PromptResponse`、prompt kind/action、confirm option helper、`Prompter` interface。
- `internal/uiruntime`: `Runtime`、`PromptIO`、`MultilineReader`、selector、questionnaire、spinner/progress/pager/log runtime。
- `internal/termtext`: table、display width、pad/truncate helper。
- `internal/uistyle`: color constants、file-op palette、box/divider の低レベル描画。
- `internal/uifileview`: diff、patch display/preview、file-op header/stats/path display、`CountDiffLines`。
- `internal/uitoolview`: tool execution line、tool target、parallel group display、`SpinnerMessageForTool`、tool output collapse/format。
- `internal/uiplanview`: plan display、plan review display、plan approval prompt request。
- `internal/uisummary`: task summary rendering。
- `internal/uiconfig`: classic `/config` menu/editor UI。
- `internal/configedit`: classic `/config` と TUI config screen が共有する pure config edit/value helper。config schema/default/validation は変更しない。

## Boundary decisions

- 旧 `internal/ui` compatibility facade は残さない。caller は新 owner package を直接 import する。
- `uiruntime` は spinner lifecycle の source of truth を維持する。provider stream 側の local `Spinner.Start/Stop` behavior は変更しない。
- `uistyle` は低レベル描画 owner とし、`uiruntime` へ依存しない。runtime 側が style を消費する。
- `uiconfig` は classic UI I/O を持ち、pure parse/mutation policy は `configedit` へ委譲する。
- package boundary tests で UI owner package から `internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への逆依存を禁止する。

## Baseline

- 作業前 `go test ./internal/ui` は通過。
- 作業前 `go test ./internal/tui ./internal/tuiagent ./internal/agent ./internal/tools/... ./internal/api ./internal/api/providers/...` は通過。

## Implemented tranche

- 旧 `internal/ui` の production/test files を上記 owner package へ移動。
- caller import を新 owner package へ移行。
- `internal/testutil/importguard` を追加し、新 UI owner package の import direction を固定。
- `docs/dev/package-boundaries.md` と `docs/dev/classic-terminal-refactor-master-plan.md` を split 後 package 名へ更新。

## Verification plan

- Focused: `go test ./internal/uiprompt ./internal/uiruntime ./internal/termtext ./internal/uistyle ./internal/uifileview ./internal/uitoolview ./internal/uiplanview ./internal/uisummary ./internal/uiconfig ./internal/configedit`
- Runtime race: `go test -race ./internal/uiruntime -run 'TestRuntime_.*Spinner|TestSpinner' -count=1`
- Caller paths: `go test ./internal/tui ./internal/tuiagent ./internal/agent`、`go test ./internal/tools/... ./internal/mcp ./internal/mcptool`、`go test ./internal/api ./internal/api/providers/... ./internal/providerdiag`
- Config safety: `go test ./internal/config ./scripts/internal/configdocs ./scripts/internal/configexample ./scripts/internal/configmeta ./scripts/internal/configregistry`
- Final: `go list ./...`、旧 `internal/ui` import 残存確認、`git diff --check`、`go test ./...`、`make ci-check`
