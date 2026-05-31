# Package Boundaries

## Phase 1: agent / TUI boundary

Phase 1 後の対話実行と TUI の依存方向は次の owner に分ける。

- `internal/agent`: conversation / turn / tool orchestration の core。`internal/tui`、Bubble Tea、Lip Gloss を直接知らない。`internal/ui` は interactive surface / runtime output contract として許可する。
- `internal/app`: CLI mode wiring と TUI startup の owner。TUI 起動時の package 接続をここで扱う。
- `internal/tuiagent`: `internal/agent` と `tui.AgentInterface` の adapter owner。agent の状態やイベントを TUI が扱う interface へ変換する。
- `internal/tui`: UI lifecycle、message、rendering の owner。Bubble Tea / Lip Gloss への直接依存はここに閉じる。

`internal/agent/package_boundaries_test.go` は、`internal/agent` 配下から `internal/tui`、Bubble Tea、Lip Gloss への import を禁止する。subpackage は禁止対象に含めるが、`internal/tuiagent` のように import path が似ている別 package は禁止しない。

## Phase 2-A: provider history / token boundary

Phase 2-A 後の provider-facing history projection と token helper は次の owner に分ける。

- `internal/providerhistory`: provider-facing history projection / reduction / report / rehydrate plan helper の pure logic owner。`internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss を直接知らない。raw history の永続化や request context の副作用は持たず、`api.Message` と `ledger` の入力から provider payload 用 projection を返す。
- `internal/token`: model token limit と token count estimation の owner。agent だけでなく provider adapter や repomap から直接参照する shared helper として扱い、`internal/agent` には置かない。
- `internal/agent`: raw history の clone、runtime option 解決、TaskLedger snapshot からの evidence pointer 抽出、request context / response id chain / active context append の owner。provider history の pure policy 判断は `internal/providerhistory` に委譲する。

`internal/providerhistory/package_boundaries_test.go` は、`internal/providerhistory` 配下から `internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss への import を禁止する。Agent の runtime state や UI 表示に触る必要が出た場合は、providerhistory へ wrapper を増やさず `internal/agent` 側の caller で policy input に変換する。
