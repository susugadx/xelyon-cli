# Classic terminal refactor master plan

## Scope

この plan は、TUI primary 化後に残っている classic terminal surface と、TUI/provider でも使う console UI runtime の責務境界を整理するための source of truth です。

この branch の完了条件:

- `uiruntime.Runtime.StartSpinner` を spinner 起動の source of truth にする。
- legacy classic REPL の startup / image startup / resume startup / loop を `agent_classic_repl_*` file に隔離する。
- command handler からも使う context-size 表示は shared interactive owner に置く。
- TUI からも使う interactive startup / signal cleanup は shared owner として classic-only file へ閉じない。
- root command と resume subcommand の `--no-tui` routing policy を cmd helper に集約する。
- 既存の classic behavior と `--no-tui` flag は維持し、focused tests と `make ci-check` で閉じる。

対象:

- `--no-tui` / classic REPL の起動経路
- classic REPL の loop / startup 表示
- interactive surface 共通の signal cleanup
- `internal/uiruntime.Runtime` の spinner ownership
- provider streaming と tool execution から見た spinner lifecycle
- command catalog の classic surface 表示

対象外:

- MCP runtime / MCP tool policy
- TUI の Bubble Tea model / viewport / composer の機能変更
- `--no-tui` の即時削除
- `--no-tui` flag / `CommandSurfaceClassic` / classic REPL の削除判断
- provider request / prompt / history contract の変更

## Current owner map

### `cmd`

Owner: CLI flag と execution mode routing。

現状:

- `cmd/legacy_no_tui.go` が `--interactive --no-tui`、`--resume --no-tui`、`--interactive --image --no-tui`、resume subcommand の `--no-tui` policy を集約する。
- `cmd/root.go` と `cmd/resume.go` は classic 分岐時に helper を呼び、個別に warning / unsupported operation / runtime selection を判断しない。

整理方針:

- `--no-tui` routing policy を cmd 内の helper に寄せる。
- behavior change までは行わず、まず runtime selection / warning / unsupported resume picker / direct session ID の判断を一箇所で読めるようにする。

### `internal/agent`

Owner: agent runtime、normal mode、tool execution、classic REPL compatibility。

現状:

- legacy classic REPL の startup、reader setup、bracketed paste cleanup、header/status 表示、loop は `agent_classic_repl_*` file に分かれている。
- context-size 表示は command handler からも使うため `agent_interactive_context_size.go` が owner する。
- shared interactive startup と signal cleanup は `agent_interactive_startup.go` に残し、TUI と classic の共通 owner として扱う。
- command surface は `CommandSurfaceClassic` として agent command dispatcher に残っている。
- tool execution は `uiruntime.Runtime.StartSpinner` へ委譲して current spinner を登録する。

整理方針:

- classic REPL は同一 package 内で `agent_classic_repl_*` へ分け、TUI/normal/tool execution と混ざらない名前にする。
- package split は初期段階では行わない。private method 依存が多く、export を増やす危険が大きい。
- tool execution の spinner 起動は `uiruntime.Runtime.StartSpinner` を source of truth にする。
- `initInteractiveAgentWithRuntime` と signal cleanup は TUI からも使うため、classic-only には閉じない。

### UI owner packages

Owner: process / injected runtime、prompt contract、console renderer、file/tool/config/plan display を package owner ごとに分ける。

現状:

- `internal/uiruntime.Runtime` が current spinner を持つ。
- spinner 起動は `uiruntime.Runtime.StartSpinner` が current spinner 登録まで owner する。
- `Runtime.NewSpinner` / `Runtime.SetSpinner` / `Spinner.Start` は低レベル primitive と test / stream-local restart 用に残る。
- prompt contract は `internal/uiprompt`、tool display は `internal/uitoolview`、file/patch display は `internal/uifileview`、classic config editor は `internal/uiconfig`、pure config edit helper は `internal/configedit` が owner する。

整理方針:

- `Runtime.StartSpinner(message)` を spinner start owner にする。
- `Runtime.StopSpinner()` を current spinner の stop owner にする。
- `Spinner.Start/Stop` は low-level primitive として残し、provider stream parser の局所再開には当面使う。
- UI owner package を classic 専用として扱わない。TUI adapter、provider diagnostics、tool confirmation、config editor から使う共有 contract は owner package で維持する。

### `internal/api`

Owner: provider request の context-bound UI adapter。

現状:

- `api.StartSpinnerWithMessage(ctx, message)` が context runtime の `StartSpinner` へ委譲する。
- provider 側は受け取った `*uiruntime.Spinner` を直接 stop / restart する。
- stream parser は assistant text と tool JSON の表示切替に spinner を使う。

整理方針:

- context-bound 起動は `uiruntime.Runtime.StartSpinner` へ委譲する。
- provider stream 内の direct `spinner.Stop/Start` は behavior-preserving のため初期 tranche では残す。
- provider-facing output ordering が変わる整理は別 task として扱う。

## Refactor sequence

### Tranche 1: spinner runtime ownership

目的:

- current spinner の start owner を `uiruntime.Runtime` に寄せる。
- tool execution と provider request の起動 path を同じ source of truth にする。
- classic REPL の behavior は変えない。

検証:

- `go test ./internal/uiruntime -run 'TestRuntime_.*Spinner|TestSpinner' -count=1`
- `go test -race ./internal/uiruntime -run 'TestRuntime_.*Spinner|TestSpinner' -count=1`
- `go test ./internal/agent -run 'TestExecuteToolCallsWithParallel_.*Spinner|TestExecuteToolWithSpinner|TestRunInteractiveWithConfig' -count=1`
- `go test ./internal/api ./internal/api/providers/... -run 'Test.*Spinner|Test.*Streaming|Test.*Stream|Test.*NonStreaming' -count=1`

### Tranche 2: classic REPL file boundary

目的:

- legacy classic REPL の classic-only 責務を名前と file boundary で隔離する。
- TUI primary path と classic fallback path を読み分けやすくする。

想定作業:

- classic startup / image startup / resume startup を classic file に寄せる。
- REPL loop を classic-only として命名する。
- TUI と classic の両方から使う signal handler は shared interactive startup 側に残す。
- `RunInteractive*` 互換入口を残す場合は deprecated wrapper に限定する。

検証:

- `go test ./internal/agent -run 'TestRunInteractiveWith.*|TestCommand.*Classic|TestStatus.*Surface|TestHelp.*Classic' -count=1`
- `go test ./cmd -run 'TestRootCommand_.*NoTUI|TestRootCommand_Resume.*TUI|TestRootCommand_Interactive.*TUI' -count=1`

### Tranche 3: `--no-tui` routing policy

目的:

- cmd 側の classic fallback policy を一箇所に集約する。
- 将来 `--no-tui` を hidden / error / removed に変更するときの差分を小さくする。

想定作業:

- root command と resume command の `legacyNoTUI` 分岐を helper 化する。
- warning、unsupported operation、runtime selection の順序を固定する。
- docs と command tests の expectation を同期する。

検証:

- `go test ./cmd -count=1`
- `go test ./internal/commandcatalog ./internal/agent -run 'Test.*Classic|Test.*Surface|Test.*Help' -count=1`

### Tranche 4: deletion decision (separate PR)

目的:

- classic fallback を残すか、TUI-only として削除するか決める。
- この branch では実施しない。`--no-tui` 完全削除は user-visible CLI contract を変えるため、別 PR の `shared-contract-change` として扱う。

残す場合:

- `--no-tui` は deprecated compatibility として残す。
- 新規 command は classic に追加しない。
- smoke/focused tests だけで壊れない状態を維持する。

削除する場合:

- `dead-code-cleanup` と `shared-contract-change` を使う。
- `--no-tui` flag、classic command surface、classic docs、classic-only tests を同時に閉じる。
- user-visible behavior change として別 PR にする。

## Safety notes

- UI owner package 群は classic 専用ではない。TUI adapter、provider diagnostics、tool confirmation、config editor も使う。
- spinner は provider stream の output ordering に影響するため、起動 owner の整理と stream parser の挙動変更を混ぜない。
- classic REPL 退役は docs / help / command catalog / status 表示まで同期しないと drift する。
- この branch で classic surface の削除や help 表示の contract change は行わない。
