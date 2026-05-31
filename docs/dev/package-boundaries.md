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

- `internal/providerhistory`: provider-facing history projection / reduction / report / rehydrate plan helper の pure logic owner。`internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss を直接知らない。raw history の永続化や request context の副作用は持たず、`api.Message` と `taskstate` の入力から provider payload 用 projection を返す。
- `internal/token`: model token limit と token count estimation の owner。agent だけでなく provider adapter や repomap から直接参照する shared helper として扱い、`internal/agent` には置かない。
- `internal/agent`: raw history の clone、runtime option 解決、TaskLedger snapshot からの evidence pointer 抽出、request context / response id chain / active context append の owner。provider history の pure policy 判断は `internal/providerhistory` に委譲する。

`internal/providerhistory/package_boundaries_test.go` は、`internal/providerhistory` 配下から `internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss への import を禁止する。Agent の runtime state や UI 表示に触る必要が出た場合は、providerhistory へ wrapper を増やさず `internal/agent` 側の caller で policy input に変換する。

## Phase 2-B: task state boundary

Phase 2-B 後の runtime task state は次の owner に分ける。

- `internal/taskstate`: `RuntimeTaskState`、snapshot / reset、recorder、tool/test observation、evidence pointer、rehydrate plan / execution、edit readiness、current task state snapshot rendering の provider-neutral な state owner。`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss を直接知らない。
- `internal/agent`: TaskLedger store の runtime 初期化と lifecycle、turn loop / mutation tracker / final check / edit readiness から state を更新・消費・reset するタイミング、provider-facing active context へ渡すかどうかの policy、`api.ActiveContextBlock` への wrapping、`/ledger` command surface の owner。
- `internal/providerhistory`: provider history projection / reduction から `taskstate.EvidencePointer` と `taskstate.RehydratePlan` を扱う pure helper の owner。TaskLedger store の所有や request context の副作用は持たない。

`internal/taskstate/package_boundaries_test.go` は、`internal/taskstate` 配下から `internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import を禁止する。Provider payload、TUI 表示、turn orchestration に触る必要が出た場合は、`internal/taskstate` へ wrapper を増やさず caller 側で `taskstate` の provider-neutral な型へ変換する。

## Phase 2-C: turn support / final check boundary

Phase 2-C 後の normal turn support と final check policy は次の owner に分ける。

- `internal/turnsupport`: normal turn の retry / stalled 判定、error fingerprint 正規化、turn-local mutation tracking、FileChange snapshot / progress fingerprint の pure state・policy owner。`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss を直接知らない。
- `internal/finalcheck`: final check の結果 DTO、failure feedback、failure fingerprint、no-progress retry gate、target file content fingerprint の owner。shell command 実行、taskstate 記録、colored output、taskTestResult 更新は持たない。
- `internal/agent`: final check command の読み取りと実行、timeout/env/process group、git diff / untracked context、taskstate observation 記録、history append、normal turn の continue / break / done 制御の owner。normal turn support と final check の deterministic policy は `internal/turnsupport` と `internal/finalcheck` に委譲する。

`internal/turnsupport/package_boundaries_test.go` と `internal/finalcheck/package_boundaries_test.go` は、それぞれ `internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import を禁止する。Agent runtime state、provider payload、TUI 表示に触る必要が出た場合は、新 package 側へ wrapper を増やさず `internal/agent` 側で policy input に変換する。

`internal/token/package_boundaries_test.go` は、shared token helper が `internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss に依存しないことを固定する。
