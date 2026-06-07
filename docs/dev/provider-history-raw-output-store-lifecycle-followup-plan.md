# Provider History Raw Output Store Lifecycle Follow-up Plan

この文書は Codex Goal でまとめて実装するための内部実装仕様書である。
公開 docs ではなく、`provider-history-raw-output-artifact-rehydrate-plan.md` と `provider-history-reduction-next-plan.md` の実装後監査で残った raw output artifact store / lifecycle / path security follow-up を相談しながら固める source of truth として使う。

## 0. Purpose

既存実装は provider-facing history reduction、data-bearing raw output artifact、active context rehydrate、review prompt raw output ledger の主要経路を実装済みである。
ただし、計画書の最高構成として見ると、以下がまだ完全ではない。

- `rawoutputs.Create` の true streaming / bounded-memory pipeline
- legacy raw history/session source からの artifact materialize runtime wiring
- caller-owned liveness と `rawoutputs.CollectGarbage` の runtime lifecycle wiring
- artifact root の親ディレクトリ symlink policy

この計画書では、これらを data loss なし、security regression なし、provider-facing 品質低下なしで最高構成へ寄せる。

Goal 完了条件:

- raw output artifact creation が full body memory 前提から streaming pipeline へ移行している。
- sensitive body detection、hash、quota、temp write、encryption、manifest append が同一 pipeline 上で fail-closed になっている。
- legacy materialize が exact source identity 付きで runtime から利用でき、history-backed source だけでは apply compact を許可しない。
- liveness owner が agent/session/review 側にあり、GC は caller-provided live refs だけで mark-and-sweep する。
- artifact root 親ディレクトリ symlink policy が実装と docs/internal contract で一致している。
- `/rawoutputs` は read-only diagnostics command として、store health / verify / refs / GC dry-run を表示できる。
- focused tests と `make ci-check` が通っている。
- 実装後に Final-A impact audit と Final-B comprehensive refactor を実施している。

## 1. Current State / Implemented Preconditions

実装済み:

- `internal/rawoutputs` に `Store.Create` / `Resolve` / `Verify` / `MaterializeLegacy` / `CollectGarbage` / `RebuildIndex` API がある。
- raw output artifact は session-local content hash dedupe、manifest/index、hash verification、quota、sensitive body gate、encrypted body/manifest/index、symlinked object parent rejection を持つ。
- `providerhistory` は command / MCP / XELYON `web_search` data-bearing outputs を artifact-backed candidate として扱える。
- apply compact は parent `provider_history_reduction.mode=apply`、child `raw_output_artifacts.mode=apply`、rehydrate context、artifact verify、threshold gate を通った場合だけ provider-facing payload を変更する。
- `/status` は aggregate summary、`/ledger` は raw output candidate/ref/gate/reason の bounded detail を表示する。
- review prompt 6-D は review probe result raw output artifact、rehydrate ledger、fail-closed saturation/revision rule を持つ。
- focused tests は commandoutputs/rawoutputs/providerhistory/agent/review/modelinput で通っている。

未到達:

- `CreateRequest.Body` は `io.Reader` だが、`Create` は現在 `readPlainObjectToMemory` で full body を `[]byte` 化してから sensitive scan / object write / encryption を行う。
- `CollectGarbage` は store API と tests までで、agent/session/review lifecycle から caller-provided live refs を渡す production path が未接続。
- `MaterializeLegacy` は exact source requirement を持つが、providerhistory/agent の legacy fallback resolver としては未接続。
- artifact root は最終 root symlink を拒否するが、root より上の既存親ディレクトリ symlink 拒否と explicit root override がまだ未実装。

## 2. Global Contracts

- raw `Agent.History` / `Session.Messages` / audit / persisted JSONL を軽量化・削除しない。
- provider-facing projection と request-local active context だけを軽量化対象にする。
- data-bearing output は raw artifact ref と rehydrate transport が成立しない限り apply compact しない。
- history-backed source だけでは apply compact を許可しない。apply compact には persisted artifact ref と hash/lifecycle verification が必要。
- sensitive/private/secret body は normal raw output artifact store に保存しない。
- artifact creation は ref を manifest に出す前に、hash、size、quota、sensitive、path safety、encryption policy を通す。
- GC は caller-provided live refs だけを source of truth にする。`rawoutputs` は providerhistory/review/session liveness を推測しない。
- GC は tombstone/quarantine/gc_collected event を append-only manifest に記録する。
- missing/ambiguous/unsafe source は fail-closed にし、placeholder apply へ進めない。
- symlink policy は実装、tests、internal docs で一致させる。
- token budget estimation、`/status` rendering、dry-run-only inspection は artifact materialization や GC を実行しない。
- `/rawoutputs` 初期実装は read-only diagnostics だけにする。削除、修復、GC apply は実行しない。

## 3. Non-goals

- provider-facing history reduction の対象 family を増やすことはこの計画の主目的ではない。
- raw history/session/audit/persisted JSONL の保存形式を削ることはしない。
- cross-session dedupe は実装しない。
- retention policy に `forever` / `days` / size-based eviction を追加しない。初期実装は `session` のまま。
- sensitive artifact 用 encrypted vault を追加しない。normal store では sensitive body を保存しない。
- provider-native built-in replay の apply compact 昇格はしない。
- public config surface は root override `provider_history_reduction.raw_output_artifacts.root` 以外に増やさない。
- `/rawoutputs gc --apply`、`/rawoutputs delete`、`/rawoutputs repair` は初期実装に含めない。
- commit / push / PR 作成はこの計画には含めない。ユーザーの明示指示に従う。

## 4. Source Findings

確認済み source:

- `internal/rawoutputs/store_create.go`
  - `Create` は body を `readPlainObjectToMemory` で full memory 化してから sensitive check と object commit を行う。
  - duplicate live ref / duplicate artifact quota no-charge は実装済み。
- `internal/rawoutputs/store_object.go`
  - `writeTempPlainObject` は chunk read + temp file + hash 計算を持つが、現行 `Create` の主経路では full memory 化後の再読み込み用途になっている。
  - encrypted commit は `crypto.EncryptSession([]byte, passphrase)` 前提。
- `internal/rawoutputs/manifest.go`
  - manifest/index は encryption enabled 時に encrypted line / encrypted payload になる。
  - manifest は append-only event log、index は rebuildable cache。
- `internal/rawoutputs/paths.go`
  - session ID / ref / sha256 path validation、store root containment、symlinked store parents rejection helper がある。
- `internal/rawoutputs/store_gc.go`
  - `CollectGarbage` は caller-provided `LiveRefs` を受け、unreachable refs を tombstone、no-live-ref artifacts を delete + `gc_collected` event にする。
- `internal/agent/provider_history_projection.go`
  - runtime store root は default `~/.xelyon/history/rawoutputs`。
  - request projection は side effects allowed 時だけ store を開く。
  - token budget projection は read-only policy で artifact materialization を避ける。
- `internal/agent/provider_history_request_context.go`
  - raw output active context が required refs を満たせない場合、apply disabled reason 付きで再 projection する。
- `internal/review/prompt_probe_raw_output*.go`
  - review prompt raw output refs、budget ledger、fail-closed saturation/revision rule がある。

Relevant tests:

- `internal/rawoutputs/store_test.go`
  - sensitive body forbidden、quota、idempotency、encryption plaintext leakage、symlinked object parent/root/tmp/rebuild/resolve/GC cases。
- `internal/providerhistory/*_projection_test.go`
  - command / MCP / XELYON web_search artifact-backed dry-run/apply/fail-closed。
- `internal/agent/provider_history_raw_output_request_test.go`
  - request-time active context injection、token budget read-only、web_search active context redaction。
- `internal/review/runner_prompt_probe_raw_output_test.go`
  - review raw output apply/dry-run/budget/fail-closed/stable command index hashing。

## 5. Responsibility Boundaries

```text
commandoutputs:
  command output semantic classification only.
  No artifact persistence, no provider-facing apply decision.

rawoutputs:
  physical artifact storage, streaming write pipeline, manifest/index, resolver,
  hash verification, path safety, encryption policy, lifecycle events, GC eligibility.
  No providerhistory/review/session liveness discovery.

providerhistory:
  projection candidates, placeholders, report metadata, raw_output_ref linkage,
  data-bearing apply eligibility gates.
  No filesystem ownership and no GC deletion.

agent:
  runtime store root/session ID, request projection side-effect policy,
  active context rehydrate, token-budget read-only policy,
  session lifecycle live-ref collection and GC trigger orchestration.

review:
  review prompt raw output refs, review run liveness contribution,
  saturation/revision fail-closed rule, review prompt report/ledger.

config/docs/generated:
  existing public config/docs alignment plus raw output artifact root override,
  `/rawoutputs` command docs.
  No new provider history reduction mode unless explicitly decided.
```

## 6. Design Decisions

### D1. Artifact Root Parent Symlink Policy

Status: adopted.

問題:

- 現在は final artifact root が symlink でないことを確認している。
- root より上の既存 parent、例 `~/.xelyon` や `~/.xelyon/history` が symlink の場合、見た目の root と実保存先がずれる。
- raw output artifact は command / web / MCP / review の data-bearing raw body を持つため、保存境界を曖昧にしない。
- 将来 GC/delete を入れるため、delete 対象 root の path policy を先に硬くする必要がある。

採用方針:

- Default root の親ディレクトリ symlink は拒否する。
- raw output artifact を別 disk / directory に置きたい場合は、symlink ではなく explicit root override を使う。
- root override は stable/advanced config surface として追加する。
- config key は `provider_history_reduction.raw_output_artifacts.root` とする。
- env override は `XELYON_RAW_OUTPUT_ARTIFACT_ROOT` とする。
- root 解決優先順位は env override、config、default root の順にする。
- root override path も default root と同じ path safety policy で検査する。
- root override path は absolute path のみ許可し、relative path は拒否する。
- default root parent symlink reject は warning ではなく hard error にする。
- final root symlink、root 配下の symlinked parents、object/tmp/manifest/index parents は拒否し続ける。
- `/rawoutputs summary` には effective raw output artifact root と symlink policy を表示する。
- `/status` は aggregate に留め、root/path detail を出しすぎない。
- docs/config.md と config.yaml.example には、`~/.xelyon` symlink ではなく root override を使う方針を書く。

理由:

- Security posture と運用自由度を両立できる。
- symlink による暗黙移設より、config/env による明示移設の方が audit しやすい。
- 将来 GC/delete を入れる時に、削除対象 root を説明しやすい。
- `/rawoutputs summary` で effective root を見せれば、user も保存先を確認できる。

実装影響:

- config schema / default / validation / registry / docs / config.yaml.example を触るため、実装時は `xelyon-config-contract-change` 対象にする。
- root override path validation は security boundary なので、実装時は `security-boundary-change` 対象にする。
- existing `~/.xelyon` symlink user は raw output artifact store open が fail する可能性がある。docs と error message で root override を案内する。

確定した細部:

- config key は `provider_history_reduction.raw_output_artifacts.root`。
- env var 名は `XELYON_RAW_OUTPUT_ARTIFACT_ROOT`。
- root 解決優先順位は env override、config、default root。
- default root parent symlink reject は hard error。ただし provider-facing projection は data-bearing output を raw keep に戻す。
- root override path は absolute path のみ許可し、relative path は拒否する。

### D2. Streaming Sensitive Detection Design

Status: adopted.

問題:

- sensitive body を artifact に書く前に拒否する必要がある。
- 現在は全 body を memory に積むため実装は簡単だが、計画書の streaming contract と矛盾する。

採用方針:

- chunk scanner が body stream を読みながら以下を同時に行う。
  - byte/rune count
  - max artifact bytes check
  - rolling sensitive detector
  - SHA-256 hash
  - temp object write
- sensitive detector の source of truth は既存 `LooksSensitiveContent` 相当の coverage とし、streaming 化で coverage を落とさない。
- sensitive hit 時は temp file を削除し、manifest/ref/index を作らない。
- detector は chunk boundary をまたぐ secret pattern を見落とさないため、前 chunk tail を bounded window として保持する。
- full regex が必要な pattern は bounded rolling buffer で判定する。
- streaming 判定へ安全に移せない pattern がある場合、artifact 化せず fail-closed する。
- detector が `unknown` / scanner error になった場合は sensitive と同等に fail-closed する。

理由:

- raw output artifact は provider-facing compact の根拠になるため、secret detection coverage を下げると security regression になる。
- chunk boundary をまたぐ token / header / key-value を見落とすと、body 全体 scan より弱い実装になる。
- 検出できないものは保存しない方が、artifact-backed compaction の fail-closed contract と一致する。

### D3. Encryption Streaming Strategy

Status: adopted.

問題:

- 現在の `crypto.EncryptSession` は `[]byte` 入力で、encrypted artifact commit も full memory 前提。

採用方針:

- envelope streaming encryption を今回 follow-up Goal に含める。
- artifact body は chunk encrypt し、full body `[]byte` を作らない。
- encrypted body format は version を持つ。
- manifest/index は既存 line encryption policy を維持する。
- old encrypted body format resolve test、新 envelope streaming format write/read test、format mismatch failure test を追加する。
- encrypted mode で durable plaintext temp file を作らない。
- decrypt / verify / format parse failure は fail-closed にする。

理由:

- plain mode だけ streaming にすると、encrypted mode が bounded memory の例外 path として残り続ける。
- encrypted mode でも raw output artifact は data-bearing evidence なので、保存・検証・rehydrate contract を同じ品質にする。
- format version を持たせ、既存 encrypted artifact を読み続けることで、将来の crypto migration と compatibility test を明確にできる。

実装影響:

- crypto format 変更は `security-boundary-change` 対象にする。
- old encrypted artifacts を読める互換性を維持し、新規 write は versioned envelope streaming format に寄せる。

### D4. Legacy Materialize Runtime Scope

Status: adopted.

問題:

- `MaterializeLegacy` API はあるが、runtime fallback として未接続。
- legacy materialize は便利だが、source identity を間違えると別 output を artifact 化し、provider-facing data loss につながる。

採用方針:

- exact source identity は以下を含む。
  - session ID
  - history index
  - tool call ID
  - tool name
  - command hash / query hash / source hash
  - content hash if already known
- exact source が 1 件に定まる場合だけ materialize する。
- command output legacy、XELYON `web_search` legacy、MCP tool result legacy は exact source identity が取れる場合に今回対象にする。
- provider-native built-in replay は candidate-only / dry-run までにし、apply compact 用 materialize は今回対象外にする。
- ambiguous / missing / stale / mismatched source は fail-closed。
- materialize 成功だけでは apply compact を許可しない。
  - apply compact は created artifact ref + verify + rehydrate transport + threshold gate が必要。
- token estimate / `/status` / dry-run read-only path では materialize しない。

理由:

- command だけに限定すると、既に data-bearing として扱っている XELYON `web_search` / MCP の legacy path が不完全なまま残る。
- 一方で provider-native built-in replay は provider-runtime source-owner と round-trip contract が別 surface なので、apply 用 materialize へ昇格しない。
- source identity が曖昧なものは materialize しないことで、別 output を artifact 化する provider-facing data loss を避ける。

### D5. GC Runtime Trigger and Liveness Owner

問題:

- `CollectGarbage` は削除を伴うため、runtime trigger を雑に入れると raw evidence loss になる。

最高構成:

- liveness owner は `agent` / `session` / `review`。
- live refs は少なくとも以下から集める。
  - current raw historyに残る raw output refs
  - latest provider projection report refs
  - review prompt reduction report ledgers
  - active session task ledger / persisted session metadata に必要なら保存された refs
- GC trigger は destructive operation なので段階導入する。
  - Phase 1: `/rawoutputs gc --dry-run` read-only diagnostics only
  - Phase 2: session rewrite/truncate/compress 後の dry-run report
  - Phase 3: explicit cleanup command or session close hook with tests
- future GC real apply は `dry-run result` と `live refs collection` の focused tests が通ってから別 Goal で有効化する。

確定方針:

- 今回 Goal では `/rawoutputs gc --dry-run` までを実装する。
- GC real apply / delete は初期実装に含めない。
- real delete は dry-run observability、live refs collection、lifecycle tests が安定してから別 Goal で扱う。

## 7. Proposed Implementation Priority

推奨順:

1. 採用済みの artifact root parent symlink policy を実装する。
2. `rawoutputs.Create` を streaming write pipeline に寄せる。
3. 既存 sensitive coverage を落とさない chunk-boundary aware streaming detector を実装する。
4. envelope streaming encryption を実装し、encrypted mode でも full body memory / durable plaintext temp を残さない。
5. legacy materialize runtime fallback を command / XELYON `web_search` / MCP から exact identity 付きで接続する。
6. live refs collector を agent/session/review owner に作る。
7. read-only `/rawoutputs` diagnostics を追加し、GC dry-run observability を `/rawoutputs gc --dry-run` に置く。
8. GC real apply は実装せず、別 Goal 用の未実装条件として残す。
9. Final-A impact audit。
10. Final-B comprehensive refactor including tests。

この順番の理由:

- symlink policy は path safety の土台なので先に決める。
- streaming write は `Create` の中心 contract なので legacy/GC より先に安定させる。
- sensitive detector と encryption は artifact write safety の一部なので、legacy materialize より先に安定させる。
- legacy materialize は artifact write path が最高構成になってから接続する。
- GC は delete operation なので最後に回す。

## 8. Implementation Sections

## 8.1 Root Parent Symlink Policy

### Purpose

raw output artifact root の containment policy を明確にする。
Default root は symlink による暗黙移設を拒否し、移設は explicit root override に寄せる。

### Design contract

- final store root 以下の symlink は拒否する。
- default root `~/.xelyon/history/rawoutputs` は root parent chain の symlink も拒否する。
- explicit root override を指定した場合も、root parent chain と final root symlink を検査する。
- root override は `provider_history_reduction.raw_output_artifacts.root` として config contract に追加する。
- env override は `XELYON_RAW_OUTPUT_ARTIFACT_ROOT` として追加する。
- root 解決優先順位は env override、config、default root の順にする。
- root override path は path traversal、unsafe relative path、empty path、filesystem root path を拒否する。
- root override path は absolute path のみ許可する。
- default root parent symlink reject は warning ではなく hard error にする。
- root override が invalid な場合は raw output artifact materialization を fail-closed にし、provider-facing data-bearing apply compact は raw keep する。
- error message / docs は symlink 移設ではなく root override を案内する。

### Tests

- default root parent symlink を拒否する focused test。
- default root parent symlink reject が warning ではなく hard error になる test。
- explicit root override が valid absolute path の場合に store を開ける test。
- `XELYON_RAW_OUTPUT_ARTIFACT_ROOT` が config より優先される test。
- explicit root override の parent symlink を拒否する test。
- root override が empty / filesystem root / traversal / unsafe relative path の場合に拒否する test。
- final root symlink は拒否し続ける。
- object parent/tmp/index/manifest parent symlink は拒否し続ける。
- invalid root override 時に provider-facing projection が data-bearing output を raw keep する test。

## 8.2 Streaming Create Pipeline

### Purpose

`CreateRequest.Body io.Reader` の contract 通り、large artifacts を full body memory に積まずに作成する。

### Design contract

- body stream を 1 pass で読み、temp file、hash、size、sensitive detection を同時に行う。
- sensitive detection は既存 `LooksSensitiveContent` 相当の coverage を落とさない。
- chunk boundary をまたぐ sensitive marker を検出する。
- streaming 判定できない pattern がある場合は artifact 化せず fail-closed する。
- sensitive / too large / ctx cancel / read error は temp file cleanup + no manifest append。
- ref ID と artifact metadata は hash 計算後に確定する。
- duplicate live ref / duplicate artifact quota no-charge は維持する。
- object commit 前後の idempotency を維持する。

### Tests

- custom reader で large body を chunked read しても成功する。
- sensitive marker が chunk boundary をまたいでも拒否する。
- existing sensitive coverage に含まれる代表 pattern を streaming path でも拒否する。
- streaming unsupported pattern は fail-closed で no artifact。
- too-large body は temp cleanup + no manifest。
- duplicate same ref は manifest created record を増やさない。
- duplicate same body different ref は quota additional bytes を charge しない。

## 8.3 Encryption Path

### Purpose

encrypted history mode でも streaming contract と plaintext non-durability contract を守る。

### Design contract

- encryption enabled で artifact body、manifest、index が plaintext を残さない。
- artifact body は envelope streaming encryption で chunk encrypt する。
- encrypted body format は version を持つ。
- temp files も plaintext durability risk として扱い、encrypted mode で durable plaintext temp を作らない。
- manifest/index は既存 line encryption policy を維持する。
- decrypt / verify / format parse failure は fail-closed。
- old encrypted artifact compatibility を維持する。

### Tests

- encrypted artifact/manifest/index に plaintext body/event が残らない。
- encrypted artifact body が chunk streaming で書かれ、full body memory path を使わない。
- encrypted create idempotency。
- old encrypted body format resolve compatibility。
- format mismatch is fail-closed。
- decrypt failure is fail-closed。
- encrypted path でも symlink parent rejection。

## 8.4 Legacy Materialize Wiring

### Purpose

既存 history/session source から exact identity 付きで artifact を materialize できるようにする。

### Design contract

- exact source identity がない場合は materialize しない。
- command output legacy、XELYON `web_search` legacy、MCP tool result legacy を exact source identity 付きで対象にする。
- provider-native built-in replay は candidate-only / dry-run までにし、apply compact 用 materialize は対象外にする。
- ambiguous source は fail-closed。
- materialize 成功だけで provider-facing apply compact を許可しない。
- read-only projection path では materialize しない。

### Tests

- exact command output legacy source を materialize する。
- exact XELYON `web_search` legacy source を materialize する。
- exact MCP tool result legacy source を materialize する。
- provider-native built-in replay は apply compact 用 materialize しない。
- missing/ambiguous/mismatched source は raw keep。
- token budget / `/status` path は materialize しない。
- materialized artifact ref は verify + active context gate が成立した場合だけ apply eligible。

## 8.5 Runtime Liveness and GC

### Purpose

session/review/providerhistory liveness を caller side で集め、`rawoutputs.CollectGarbage` に渡せるようにする。

### Design contract

- liveness collector は `rawoutputs` package に置かない。
- live refs source を report/status できる形にする。
- 今回 Goal は GC dry-run wiring までにする。
- real apply/delete は command surface に出さず、別 Goal の対象にする。
- future real delete は explicit trigger または safe lifecycle event だけで走る。

### Tests

- live refs がある artifact は dry-run で kept になる。
- unreachable ref は dry-run で tombstone candidate になる。
- no-live-ref artifact は dry-run で collectable になる。
- shared artifact に live ref が残る場合、dry-run で object deletion candidate にならない。
- review raw output refs が live refs に含まれる。
- GC dry-run は manifest/object を変更しない。

## 9. Report / Status / Observability

採用方針:

- `/status`
  - 普通の利用者向け aggregate summary。
  - provider history reduction mode、raw output artifact mode、概算削減量、主要 blocked reason counts に留める。
  - root override の詳細 path は出さないか、必要なら redacted / basename 程度に留める。
- `/ledger`
  - 直近 provider request / projection の transparency surface。
  - raw output candidate、raw_output_ref、artifact gate、rehydrate gate、threshold gate、fail-closed reason、estimated/actual saved を bounded 表示する。
  - store lifecycle、GC、manifest/index health の詳細は持たせない。
- `/rawoutputs`
  - developer / agent diagnostics 用の read-only command。
  - raw output artifact store の保存境界、健全性、refs、GC dry-run をコードを読まずに確認する surface。
  - 本文 excerpt は表示しない。ref ID / hash は bounded / prefix / redacted 表示にする。
  - 初期実装で削除、修復、GC apply はしない。

初期 `/rawoutputs` subcommands:

```text
/rawoutputs
/rawoutputs summary
/rawoutputs verify
/rawoutputs refs
/rawoutputs gc --dry-run
```

`/rawoutputs summary`:

- effective root
- root source: env / config / default
- symlink policy
- mode: off / dry_run / apply
- encryption status
- retention policy
- current session artifact/ref counts
- quota usage
- top-level verdict: healthy / degraded / unavailable

`/rawoutputs verify`:

- manifest/index/object hash consistency
- missing objects
- hash mismatch counts
- stale index entries
- duplicate live refs
- encrypted/plaintext policy mismatch

`/rawoutputs refs`:

- bounded ref list。
- ref ID prefix、surface、semantic role、family、byte size、approx tokens、created_at、live status、source breakdown を表示する。
- body excerpt は表示しない。
- liveness が判断できない ref は `unknown_live_state` として表示し、collectable とは扱わない。

`/rawoutputs gc --dry-run`:

- caller-owned live refs collector がある場合だけ実行する。
- kept / collectable / unknown_live_state / tombstone candidate / reclaimable bytes を表示する。
- liveness source が不足している場合は fail-closed で collectable にしない。
- manifest/object/index は変更しない。

raw output report fields:

- materialized legacy count
- materialize kept reasons
- GC dry-run kept/tombstone/collect counts
- streaming create failure reason counts

非目的:

- `/rawoutputs gc --apply`
- `/rawoutputs delete`
- `/rawoutputs repair`

## 10. Tests / Verification

Focused tests:

```sh
go test ./internal/rawoutputs
go test ./internal/providerhistory -run 'RawOutput|Command|WebSearch|MCP|Projection'
go test ./internal/agent -run 'ProviderHistory|RawOutput|RawOutputs|Ledger|TokenBudget'
go test ./internal/config -run 'ProviderHistory|RawOutput|Registry|Default|Validation'
go test ./scripts/internal/configgen
go test ./internal/review -run 'RawOutput|Probe|PromptReduction|Saturation'
```

Broad verification:

```sh
make gen-all
go test ./...
make ci-check
```

Security / lifecycle review checklist:

- path traversal / symlink parent / symlink final path
- temp cleanup on failure
- no manifest ref before object commit
- no plaintext body in encrypted mode
- sensitive body never stored
- existing sensitive coverage is preserved in streaming detector
- token estimate never materializes artifact
- GC dry-run no mutation
- GC apply/delete command is unavailable in initial `/rawoutputs`

## 11. Final-A Impact Audit

実装後、strict review 前に以下を self-audit する。

- provider-facing data loss がないか。
- active context required refs missing 時に apply が fail-closed raw keep になるか。
- streaming sensitive detector に chunk-boundary false negative がないか。
- streaming sensitive detector が既存 sensitive coverage を落としていないか。
- encryption enabled mode で plaintext durable temp がないか。
- envelope streaming encryption の format version / old format resolve compatibility / fail-closed tests があるか。
- legacy materialize が ambiguous source を artifact 化しないか。
- command / XELYON `web_search` / MCP legacy materialize scope と provider-native dry-run-only scope が一致しているか。
- GC live refs に providerhistory/review/session の必要 refs が入っているか。
- token budget / `/status` / dry-run が side-effect-free か。
- `/rawoutputs` が read-only diagnostics に留まり、delete/repair/gc apply を実行しないか。
- `/ledger` と `/rawoutputs` の責務が混ざっていないか。
- root symlink policy と tests/docs が一致しているか。
- root override config / generated metadata / docs/config.md / config.yaml.example が一致しているか。
- `/rawoutputs` command behavior / docs/commands.md / tests が一致しているか。

## 12. Final-B Comprehensive Refactor

この follow-up は store lifecycle と security boundary を触るため、Final-B は必須。

対象:

- `internal/rawoutputs` の writer/scanner/encryption/manifest/GC の責務分離
- sensitive detector の owner
- test helper の整理
- path safety helper の命名と責務
- agent-side liveness collector の owner
- review/providerhistory raw output ref collector の重複
- report/status/ledger aggregation の命名
- `/rawoutputs` command rendering と diagnostics DTO の責務分離

完了条件:

- streaming writer と resolver/lifecycle が混ざりすぎていない。
- sensitive detection の source of truth が重複していない。
- path safety helper が write/read/GC で一貫している。
- GC/liveness の owner が `rawoutputs` に漏れていない。
- tests が巨大化している場合は helper/fixture を整理している。

## 13. Goal Handoff Prompt

```text
docs/dev/provider-history-raw-output-store-lifecycle-followup-plan.md を source of truth として、provider history raw output artifact store の follow-up を実装してください。

主目的は、rawoutputs.Create の true streaming / bounded-memory pipeline、sensitive body detection、encryption path、legacy materialize runtime wiring、caller-owned liveness + GC wiring、artifact root parent symlink policy を最高構成へ寄せることです。

artifact root parent symlink policy は採用済みです。default root の parent symlink は warning ではなく hard error として拒否し、別 disk / directory に置きたい場合は symlink ではなく explicit root override を使います。root override config は `provider_history_reduction.raw_output_artifacts.root` として追加し、env override は `XELYON_RAW_OUTPUT_ARTIFACT_ROOT` として追加してください。root 解決優先順位は env override、config、default root です。root override path は absolute path のみ許可し、relative path、empty path、filesystem root path、path traversal は拒否してください。

raw Agent.History / Session.Messages / audit / persisted JSONL は変更しないでください。provider-facing apply compact は persisted raw_output_ref + artifact verify + active context rehydrate + threshold gate が成立する場合だけ許可してください。history-backed source だけでは apply compact へ進めないでください。

sensitive detector は既存 `LooksSensitiveContent` 相当の coverage を落とさず streaming 化してください。chunk boundary をまたぐ marker を検出し、streaming 判定できない pattern は artifact 化せず fail-closed にしてください。encryption は envelope streaming encryption を実装し、encrypted mode でも full body memory / durable plaintext temp を残さないでください。encrypted body format は version を持たせ、old encrypted body format の resolve compatibility と format mismatch fail-closed を test で固定してください。legacy materialize は command output / XELYON `web_search` / MCP tool result を exact source identity 付きで対象にし、provider-native built-in replay は candidate-only / dry-run に留めてください。

token budget、/status、dry-run-only inspection は artifact materialization や GC を実行しない side-effect-free path にしてください。GC は caller-provided live refs を source of truth にし、rawoutputs package に liveness discovery を入れないでください。observability は `/status` を aggregate、`/ledger` を直近 provider projection transparency、`/rawoutputs` を raw output artifact store の read-only diagnostics として分けてください。初期 `/rawoutputs` は summary / verify / refs / gc --dry-run を持ち、body excerpt は出さず、delete / repair / gc --apply は実装しないでください。

削除を伴う GC real apply は今回 Goal では実装せず、別 Goal 用の未実装条件として残してください。

config schema / generated metadata / docs/config.md / config.yaml.example / docs/commands.md を触るため、実装時は `xelyon-config-contract-change` と `security-boundary-change` の観点を満たしてください。focused tests、make gen-all、go test ./...、make ci-check を実行してください。実装後は Final-A impact audit と Final-B comprehensive refactor including tests を必ず行ってください。commit / push はユーザーの明示指示があるまで行わないでください。
```
