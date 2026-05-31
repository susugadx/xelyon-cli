# Package Boundaries

## Phase 1: agent / TUI boundary

Phase 1 後の対話実行と TUI の依存方向は次の owner に分ける。

- `internal/agent`: conversation / turn / tool orchestration の core。`internal/tui`、Bubble Tea、Lip Gloss を直接知らない。`internal/ui` は interactive surface / runtime output contract として許可する。
- `internal/app`: CLI mode wiring と TUI startup の owner。TUI 起動時の package 接続をここで扱う。
- `internal/tuiagent`: `internal/agent` と `tui.AgentInterface` の adapter owner。agent の状態やイベントを TUI が扱う interface へ変換する。
- `internal/tui`: UI lifecycle、message、rendering の owner。Bubble Tea / Lip Gloss への直接依存はここに閉じる。

`internal/agent/package_boundaries_test.go` は、`internal/agent` 配下から `internal/tui`、Bubble Tea、Lip Gloss への import を禁止する。subpackage は禁止対象に含めるが、`internal/tuiagent` のように import path が似ている別 package は禁止しない。
