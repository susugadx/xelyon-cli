# internal/tools Comprehensive Refactor Master Plan

この文書は `internal/tools` 全体の長期リファクタリング用内部計画書である。
公開 docs ではなく、Goal / 長時間実装の source of truth として扱う。

context compact / resume 後は、実装を再開する前にこの文書を再読する。

## 0. Purpose

`internal/tools` の tool 実行基盤、組み込み tool 群、調査 / 編集 / shell / sub-agent / skill tool の責務境界を整理する。

完了条件:

- `internal/tools` の owner / source of truth / caller contract が本文で説明できる。
- `tools` core の `Tool` / `Registry` / `ExecutionContext` / `ToolCall` / `RuntimeObservation` / `ToolCacheInterface` の責務が狭くなる。
- `toolmeta` と `tools/common` の metadata / safety / confirmation 役割が混ざらない。
- `file` / `search` / `gathercontext` の巨大 package に、新しい policy 判断を安易に積まない構造へ寄せる。
- provider-facing tool definitions、tool result history、task ledger observation、TUI/headless visibility、cache invalidation、confirmation/security behavior を壊さない。
- focused tests と caller tests を通し、最後に `make ci-check` を通す。

## 1. Non-goals

- commit / push / PR 作成は含めない。ユーザーの明示指示があるまで行わない。
- tool 名、provider-facing JSON schema、config schema、永続化形式、CLI UX を主目的として変えない。
- 新機能追加や behavior change をリファクタリングに混ぜない。
- `utils` / `helpers` / `common` に domain policy を逃がさない。
- テストを通すためだけの production hook、global reset、sleep、過剰 mock を追加しない。

## 2. Current Source Findings

現状の `internal/tools` は複数 owner を含む。

- `internal/tools`: tool core。`Tool`、`Registry`、`ExecutionContext`、`ToolCall`、parser、parallel execution、result publish、cache invalidation、runtime observation、compat alias を持つ。
- `internal/toolmeta`: built-in tool の description / safety / help order の source of truth。
- `internal/tools/common`: confirmation、safety lookup、path validation、diff / preview / output helper、quiet mode を持つ。
- `internal/tools/file`: read / read_files / list_dir / direct gather_context route / write / delete / str_replace / mutation result / schema helper を持つ。
- `internal/tools/search`: web_search、search_code、text backend、symbol resolution、structured impact、artifact / observation / cache を持つ。
- `internal/tools/gathercontext`: direct read/list/search orchestration。`file` と `search` の caller。
- `internal/tools/dev`: bash execution / safety / streaming / truncation。
- `internal/tools/applypatch`: apply_patch parser / preview / execution / registration。
- `internal/tools/skills`: activate_skill / run_skill_script。
- `internal/tools/subagent`: spawn_agent / wait_agent orchestration。

主な caller:

- `internal/agent`: registry setup、tool visibility、tool execution、history append、TUI callback、runtime context。
- `internal/toolruntime`: tool result history message construction。
- `internal/taskstate`: `RuntimeObservation` と rendered fallback から ledger fact を作る。
- `internal/mcptool`: dynamic tool wrapper が `tools.Tool` と `ExecutionContext` に乗る。
- `internal/api/providers/*`: provider-facing tool definitions と tool call conversion を扱う。
- `internal/reviewadapter`: review runtime 用 tool surface を組み立てる。

## 3. Global Contracts

全 phase で守る契約:

- tool name は変えない。
- built-in tool の provider-facing schema は、意図的な shared contract change として扱わない限り変えない。
- `Tool.Run`, `StructuredResultTool.RunResult`, `ExecutionContext`, `ToolCall`, `FileChange`, `RuntimeObservation`, `ToolCacheInterface` の caller-visible semantics を変えない。
- `DefaultRegistry` と subpackage registration の互換挙動は、置換 phase まで維持する。
- `RuntimeObservation` は rendered output ではなく tool owner が出す構造化 fact である。
- `taskstate` / provider history / TUI callback が消費する result / observation / FileChange を欠落させない。
- cache invalidation は実変更結果 `FileChange.Details` を優先する。
- confirmation / safety / security boundary を弱めない。
- external input が file / process / network / cwd / env に到達する path は `security-boundary-change` の対象として扱う。
- goroutine / context / parallel execution / cancellation を触る場合は `go-concurrency-lifecycle` の対象として扱う。

## 4. Owner Map And Target Boundaries

### Phase 0: source map and safety brief

- `go list ./internal/tools/...`、large file / test inventory、caller refs を確認する。
- shared contract change は `shared-contract-change`、package owner 調査は `package-boundary-map`、実装前整理は `preparatory-refactor` として扱う。
- この文書を更新し、次に実装する bounded tranche を明示する。

### Phase 1: metadata / safety boundary

Owner:

- `internal/toolmeta`: built-in tool metadata source of truth。
- `internal/tools/descriptions.go`: `tools` package から metadata を読む thin facade。
- `internal/tools/common/safety.go`: runtime confirmation policy 用の safety lookup。dynamic tool fallback と legacy alias 補完はここ。

実装方針:

- built-in tool descriptions は direct map indexing より `tools.ToolDescription` 経由へ寄せる。
- `tools.ToolDescriptions` は互換 surface として残すが、新規 caller の source of truth にはしない。
- `common.ToolSafetyLevels` は direct mutable source of truth として扱わず、lookup owner を `GetToolSafety` に寄せる。
- tests は metadata source of truth と unknown / dynamic tool fallback を固定する。

### Phase 2: registry / registration boundary

Owner:

- `internal/tools.Registry`: registered tool map、excluded tool surface、provider-facing definitions。
- subpackages: 自 package の tools を `RegisterTools(*tools.Registry)` で登録する。
- `internal/agent`: runtime registry clone / visibility / dynamic tool registration owner。

実装方針:

- `DefaultRegistry` と `init()` registration は互換維持しつつ、明示 registration path を読みやすくする。
- registry exclusion は provider surface と execution stale-call defense の両方を担う契約として test で固定する。
- provider-facing definition sync は `DefaultRegistry` だけの side effect として閉じる。

### Phase 3: execution context / runtime observation boundary

Owner:

- `ExecutionContext`: tool runtime dependency bundle。process-global state の代替。
- `RuntimeObservation`: machine-readable runtime fact。
- `taskstate`: observation consumption / rendered fallback。

実装方針:

- `ExecutionContext` accessor の責務を増やしすぎない。
- observation merge / clone / group semantics を tool owner と caller tests で固定する。
- rendered fallback は `taskstate` 側の compatibility path として残し、new structured owner を増やす場合は tool package 側で持つ。

### Phase 4: file package boundary

Owner candidates:

- read owner: path / locator / range / detail / render / observation。
- list owner: depth / ignore / project-map / cache key / compact summary。
- direct route owner: gather_context direct target resolution。
- mutation owner: write / delete / str_replace result, confirmation, FileChange。
- schema owner: tool parameter schema helpers。

実装方針:

- package split は export 増加より owner clarity が勝つ場合だけ行う。
- read/list/direct-route/mutation の helpers を混ぜない。
- mutation tests は confirmation / FileChange / LSP / path validation の contract ごとに分ける。

### Phase 5: search package boundary

Owner candidates:

- web search owner: provider/model resolution、cache、usage attribution、native search fallback。
- text backend owner: rg/grep args、parse、format。
- symbol owner: generic language symbol resolution。
- structured impact owner: Go / TypeScript / TSX / JavaScript impact pipeline。
- artifact owner: rendered output + observation + pattern groups。

実装方針:

- `search_code` public behavior は変えない。
- `SearchExecutionArtifact` を caller-facing source of truth として保つ。
- web search provider resolution/cache behavior は `internal/tools/search/web.go` owner のまま、review-local duplicate config を作らない。
- 巨大 test file は owner ごとに support helper / focused test file を検討する。

### Phase 6: gather_context boundary

Owner:

- query parse / route selection / direct vs search orchestration / prefetch merge。

実装方針:

- `gathercontext` は `file` / `search` の低レベル実装詳細を持たない。
- direct route と search route の result / observation merge contract を caller tests で固定する。

### Phase Final-A: impact recovery

必ず確認する:

- provider-facing tool definitions drift。
- `RuntimeObservation` / `ObservationGroups` 欠落。
- `FileChange` と cache invalidation の片側漏れ。
- TUI callback / headless visibility / excluded tool stale-call defense。
- safety / confirmation / path / process boundary regression。
- parallel/cancel behavior regression。

### Phase Final-B: comprehensive refactor including tests

必ず確認する:

- production diff の wrong owner / duplicate source of truth / generic helper 化。
- test diff の helper boundary / fixture boundary / huge test file 追加。
- touched file が 800 行超、または touched test file が巨大な場合の split / support helper 抽出要否。
- 残 debt は file / function / package 単位で報告する。

## 5. Verification Plan

Focused:

```sh
go test ./internal/tools/...
```

Caller:

```sh
go test ./internal/agent ./internal/toolruntime ./internal/taskstate ./internal/mcptool ./internal/reviewadapter
```

Provider surface risk がある場合:

```sh
go test ./internal/api/providers/...
```

Final gate:

```sh
make ci-check
```

## 6. Handoff Policy

- 最新ソースがこの文書と衝突したら、最新ソースを確認し、この文書を更新してから進む。
- 変える contract と変えない contract を phase ごとに明示する。
- owner が閉じない場合は、局所 patch で押し切らず別 skill / 別 tranche に分ける。
- commit / push / PR はユーザーの明示指示なしに行わない。

## 7. Implementation Status

2026-06-16 tranche:

- Phase 1 metadata / safety boundary:
  - subpackage `Description()` は `tools.ToolDescription()` 経由へ寄せた。
  - `ToolDescriptions` は互換 snapshot として残し、source of truth は `internal/toolmeta` に固定した。
  - `common.GetToolSafety` の lookup owner を明確化し、builtin / legacy alias / dynamic MCP / unknown fallback を test で固定した。
- Phase 2 registry / registration boundary:
  - `GetExcludedTools()` の返却順を deterministic にした。
  - provider-facing API tool definitions に exclusion と name sort が適用される contract を test で固定した。
  - `internal/tools` package boundary test を追加し、high-level caller / MCP runtime / TUI への逆依存を禁止した。
- Phase 3 execution context / runtime observation boundary:
  - registry が structured tool result の `RuntimeObservation` / `ObservationGroups` を clone してから caller へ渡すようにした。
  - observation group clone contract を test で固定した。
- Dead code cleanup:
  - repo 内 caller を `tools/common` owner へ寄せ、未使用になった `internal/tools/common_compat.go` を削除した。
- Phase 4 file package boundary:
  - `read_request.go` から locator resolved-path / allowed-root policy を `read_locator_access.go` へ分離した。
  - `read_file` の targets / paths decode を `read_target_resolve.go` へ分離し、request 型 / path request owner と混ぜない形にした。
- Phase 5 search package boundary:
  - `web.go` を thin orchestration / request-response owner に寄せた。
  - web search cache、provider/model resolution、request context / usage attribution、result URL parse をそれぞれ `web_cache.go`、`web_provider.go`、`web_context.go`、`web_results.go` に分離した。
- Phase 6 gather_context boundary:
  - `request.go` から quoted pattern / literal search pattern parsing を `request_pattern.go` へ分離した。
- Test boundary refactor:
  - 1000 行超だった `internal/tools/file/read_locator_test.go` を basic locator ID surface、locator detail / compact / whole-file detail、locator resolved-path / cwd / project-root policy の 3 owner file へ分割した。
  - 1000 行超だった `internal/tools/gathercontext/search_integration_test.go` を削除し、structured impact / prefetch、search file filter、search route guard、natural language inline scope、scoped direct filename の owner file へ分割した。
  - scoped direct filename tests は direct scope owner に寄せ、shared setup / assertion helper を使う形へ整理した。
  - 1200 行超だった `internal/tools/subagent/manager_test.go` を削除し、manager lifecycle、model core、Azure model selection、default / alias model selection、provider switching、prompt、summary、test provider fixture の owner file へ分割した。
- Subagent production owner split:
  - `manager.go` を constructor / Events / Spawn orchestration の入口に絞り、type contract、lifecycle / shared state mutation、wait timeout / snapshot、summary aggregation を `manager_types.go`、`manager_lifecycle.go`、`manager_wait.go`、`manager_summary.go` へ分離した。
  - goroutine start owner は `spawnWithRuntimeContext`、`done` close owner は `runAgent` のまま維持し、新しい goroutine / channel / timer は追加していない。
  - `spawn_runtime.go` を runtime context から確定 `spawnConfig` を作る入口に絞り、model selection、provider factory / runtime identity、reasoning effort application を `spawn_model_selection.go`、`spawn_provider.go`、`spawn_reasoning.go` へ分離した。
- Final-A / Final-B:
  - provider-facing tool definitions、excluded tool stale-call defense、structured observation clone、safety lookup、subagent lifecycle / summary / model selection は focused tests と caller tests で確認済み。
  - 1000 行超の touched test file と 300 行超の subagent production owner は分割済み。現時点の touched file は 800 行超なし。
  - `go test ./internal/tools/...`、direct caller packages、provider packages、`git diff --check`、`make ci-check` を通過済み。

2026-06-17 tranche:

- Phase 4 file package boundary:
  - `direct_query_route.go` を `PlanGatherContextDirectRoute` の public entrypoint に絞り、direct route plan / fallback mode / strict scoped error / execution adapter を `direct_query_route_plan.go`、`direct_query_route_fallback.go`、`direct_query_route_scoped_error.go`、`direct_query_route_execution.go` へ分離した。
  - `resolveImplicitDirectFileQuery` を path resolution owner の `direct_query_resolve.go` から direct route policy 側へ移し、`direct_query_resolve.go` は existing path / target resolution owner に絞った。
  - write / delete / str_replace の `FileChange` 生成条件を `fileMutationResult.ShouldRecordChange()` と `fileChangeForAppliedMutation` に寄せ、preview / confirm / apply / diagnostics の workflow と provider-facing `FileChange` gate を分けた。
- Phase 5 search package boundary:
  - `search_code_symbol.go` を symbol resolver contract の owner に絞り、Go resolver、generic resolver、resolver registry、multi-result affected files、single-symbol cache write を別 file に分離した。
  - structured impact pipeline を language spec、search context / cache key、cache load、resolution / cache store / artifact assembly に分け、`impact_structured_pipeline.go` は intent entrypoint に絞った。
- Phase 6 gather_context boundary:
  - `prefetch.go` を orchestration 入口に絞り、diagnostics policy、locator registration、read execution、observation merge、discovery note assembly を owner file に分離した。
  - `search_flow.go` を route execution entrypoint に絞り、search artifact execution、route hint、result assembly を別 file に分離した。
- Root execution boundary:
  - `execute.go` の core execution / publish-display / preview responsibility を `execute_core.go`、`execute_publish.go`、`execute_preview.go` に分けた。
  - direct process output guard の allowlist を publish / preview owner file に更新した。
- Verification:
  - `go test ./internal/tools/file ./internal/tools/search ./internal/tools/gathercontext ./internal/tools` を通過済み。
  - `go test ./internal/tools/...` を通過済み。
  - caller / provider / `git diff --check` / `make ci-check` はこの tranche の final gate で実行する。

残 scope:

- Phase 4 は read request / locator access、direct route、mutation gate、locator test boundary を整理済み。`list_dir` は現状の request / cache / render / runtime split が 800 行未満に閉じており、追加整理は naming drift や次の list-specific finding が出た場合に限定する。
- Phase 5 は web search owner、symbol resolver owner、structured impact production owner、gather_context 側の search route integration test boundary を整理済み。残る追加候補は巨大な Go / JS structured impact tests の owner split だが、今回 diff では新規 case を追加していないため別 task 候補とする。
- Phase 6 は request pattern owner、prefetch policy / read / merge owner、direct / search orchestration owner、search integration test boundary を整理済み。残る追加候補は caller-visible observation merge contract の追加 regression を、次に observation semantics を変える場合に置くこと。
- Root は execution core / publish-display / preview owner を整理済み。残る追加候補は `execute_cache_invalidation.go` を mutation `FileChange.Details` contract と一緒に扱う必要が出た場合に限る。
- Subagent は manager test boundary と `manager.go` / `spawn_runtime.go` production owner split を整理済み。残る追加候補は `register.go` の tool parameter / execution adapter 境界を、schema owner と runtime adapter owner に分ける必要が出た場合に限る。
