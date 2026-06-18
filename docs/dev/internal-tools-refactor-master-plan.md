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
  - `direct_query_route.go` を direct route planning entrypoint に絞り、direct route plan / fallback mode / strict scoped error / execution adapter を `direct_query_route_plan.go`、`direct_query_route_fallback.go`、`direct_query_route_scoped_error.go`、`direct_query_route_execution.go` へ分離した。
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

2026-06-17 misc-1 next tranche:

- Phase 0 source map refresh:
  - `go list ./internal/tools/...` を取り直し、root / applypatch / file / search / gathercontext / skills / subagent の package owner は前回 tranche から維持されていることを確認した。
  - large file inventory では `internal/tools/applypatch/apply_test.go`、`internal/tools/search/web_test.go`、`internal/tools/skills/run_skill_script_test.go`、`internal/tools/file/direct_query_resolve_test.go`、`internal/tools/execute_test.go`、`internal/tools/parallel_test.go` が優先 test boundary 対象だった。
  - public surface / caller refs は `ParseToolCalls`、`Execute*`、`RuntimeObservation`、`FileChange`、direct route planning、`ExecuteWebSearch`、`RunSkillScriptTool`、`ApplyPatch` の既存 caller-visible semantics を維持する前提で確認した。
- Phase 1 test boundary refactor:
  - `internal/tools/applypatch/apply_test.go` は basic apply operation と parser/apply boundary を残し、chunk matching / preview consistency を `apply_matching_test.go`、shared fixture / assertion を `apply_test_helpers_test.go` に分けた。
  - `internal/tools/search/web_test.go` は削除し、provider runtime identity、Kimi native request / usage、incomplete cache rejection、Claude alias owner cache を `web_provider_identity_test.go`、`web_kimi_native_test.go`、`web_cache_incomplete_test.go`、`web_alias_owner_test.go` に分けた。
  - `internal/tools/skills/run_skill_script_test.go` は削除し、schema、script path guard、command execution / argv quoting、args validation、bash confirmation policy、shared fixture を owner file に分けた。
  - `internal/tools/file/direct_query_resolve_test.go` は削除し、cwd / parent path resolution、multi classification / missing batch、implicit direct file route、scoped gather_context resolution、ignored-tree bypass を owner file に分けた。
  - `internal/tools/execute_test.go` は preview / write classification、display publish、quiet / context isolation、fake tool support へ分けた。
  - `internal/tools/parallel_test.go` は classification、parallel scheduling / semaphore / ordering、context cancel、quiet smoke に分けた。
- Phase 2 production preparatory refactor:
  - `internal/tools/applypatch/apply.go` を public entrypoint、hunk planning、filesystem apply / rollback、chunk replacement owner に分けた。`ApplyPatch` / `ApplyResult` の external API は維持した。
  - `internal/tools/skills/run_skill_script_tool.go` を orchestration owner に絞り、schema を `run_skill_script_schema.go`、request validation を `run_skill_script_request.go` に分けた。script path security、command construction、bash execution adapter の contract は変えていない。
- Phase 3 package boundary map:
  - `applypatch` は export 追加なしで transaction / filesystem / chunk owner を file split できたため、package split はしない。
  - root `tools` execution / parallel は public surface が caller contract そのものなので、parser / execution subpackage 化は export 増加が大きく今回 scope ではしない。
  - `search` web tests は owner split で十分。production `web.go` は前回 tranche の provider/cache/context/result split を source of truth とし、今回 production split は不要。
  - `file` direct query は production owner が既に route / resolve / execution / scoped policy に分かれているため、今回 package split はしない。
- Verification so far:
  - `go test ./internal/tools/applypatch` を通過済み。
  - `go test ./internal/tools/search` を通過済み。
  - `go test ./internal/tools/skills` を通過済み。
  - `go test ./internal/tools/file` を通過済み。
  - `go test ./internal/tools` を通過済み。
  - `go test ./internal/tools/...` を通過済み。
  - `go test ./internal/agent ./internal/toolruntime ./internal/taskstate ./internal/mcptool ./internal/reviewadapter` を通過済み。
  - `go test ./internal/api/providers/...` を通過済み。
 - `git diff --check` を通過済み。
 - `make ci-check` を通過済み。coverage は 83.4%。

2026-06-18 search structured-impact tranche:

- Phase 0 source map refresh:
  - `go list ./internal/tools/...` を取り直し、package list は `tools` / `applypatch` / `common` / `dev` / `file` / `gathercontext` / `lsp` / `planning` / `search` / `skills` / `subagent` のまま維持されていることを確認した。
  - large file inventory では `internal/tools/search/impact_javascript_test.go`、`search_code_symbol_go_path_test.go`、`impact_js_family_alias_fallback_test.go`、`search_code_symbol_impact_method_probe_helper_dispatch_test.go`、`search_code_symbol_multi_cache_test.go`、`search_code_symbol_impact_ranking_test.go`、`impact_structured_pipeline_test.go` が search test boundary の優先対象だった。
  - production 側は `semantic_evidence_bridge.go`、`impact_go_receiver_probe.go`、`symbol_bundle_builder.go`、`locator_helpers.go` が Go / JS family、cache / local AST probe、generic bundle / key、locator construction / primary-file-ref extraction を混在させていた。
- Phase 1 test boundary refactor:
  - JavaScript structured impact tests は shape / fallback と class caller、refs / CommonJS evidence / recommended reads、filters / cache / ambiguous、risk classification、shared helper に分けた。
  - JS family alias fallback tests は TSX / JSX positive aliases、JavaScript / CommonJS aliases、TypeScript alias / type-only refs、negative shadow / type-only / unmatched-source cases に分けた。
  - Go symbol path tests は repo-relative / invocation-CWD / project-root affected files、snapshot-backed LSP path resolution、AST fallback affected files に分けた。
  - Go method test-probe helper dispatch tests は methodized / function-value helper、imported helper chain、returned / tuple helper、function / interface adapter に分けた。
  - multi-pattern cache tests は affected-files collection / repair と symbol bundle dedupe / warm cache / unrelated invalidation に分けた。
  - impact ranking tests は pure ranking policy と cache-hit integration に分けた。
  - structured impact pipeline tests は language spec policy、route / scope / symbol query、ambiguous affected-file cache、malformed bundle fallback、shared pipeline fixture に分けた。
- Phase 2 production preparatory refactor:
  - `semantic_evidence_bridge.go` は Go semantic evidence owner に絞り、JS family semantic evidence を `semantic_evidence_js_family.go` に分離した。
  - `impact_go_receiver_probe.go` は cache / context / role orchestration owner に絞り、local Go AST type / method probe を `impact_go_receiver_probe_local_ast.go` に分離した。cache lifecycle hook と key semantics は変えていない。
  - `symbol_bundle_builder.go` は Go inspect bundle builder owner に絞り、generic bundle builder と canonical key / route helpers を `symbol_bundle_generic.go`、`symbol_bundle_keys.go` に分離した。
  - `locator_helpers.go` は locator construction owner に絞り、rendered output から primary file refs を抽出 / resolve する owner を `locator_primary_file_refs.go` に分離した。
- Phase 3 package boundary map:
  - `search` structured impact は `SearchOptions`、`SymbolBundle`、route trace、cache sidecar、navigation result を広く共有しているため、subpackage 化すると export が大きく増える。今回 tranche では package split せず、同一 package 内の file split に限定した。
  - web search owner は前回 tranche の provider / cache / context / result split を source of truth として維持し、今回触らない。
  - `file` / `gathercontext` / root execution は caller verification 対象だが、今回の production split owner ではない。
- Phase 3b package boundary refactor:
  - file split だけでは同一 package 内 private symbol 共有を止められないため、親 `search` package を import しない純粋 owner を `internal` subpackage として切り出した。
  - Go symbol bundle key source of truth を `internal/tools/search/internal/bundlekeys` へ分離した。export は stable Go key、canonical Go key、generic canonical key の 3 関数に限定し、`SymbolBundle` / route / artifact は外へ出していない。
  - Go receiver local AST probe を `internal/tools/search/internal/goreceiverlocal` へ分離した。export は `Role` enum、`RoleFromDir`、`HasDirectMethod` に限定し、cache lifecycle / `SearchOptions` owner は親 `search` に残した。
  - rendered output から primary file refs を抽出 / dedupe する parser / source enum を `internal/tools/search/internal/primaryrefs` へ分離した。path resolution policy は親 `search` の adapter callback に残し、subpackage は `SearchOptions` を知らない形にした。
  - `search/impact` の丸ごと split は引き続き見送るが、中心型に直接触れない pure owner は package boundary で固定する方針へ変更した。
- Verification so far:
  - `go test ./internal/tools/search/internal/...` を通過済み。
  - `go test ./internal/tools/search` を通過済み。
  - `go test ./internal/tools/gathercontext` を通過済み。
  - `go test ./internal/tools/...` を通過済み。
  - `go test ./internal/agent ./internal/toolruntime ./internal/taskstate ./internal/mcptool ./internal/reviewadapter` を通過済み。
 - `go test ./internal/api/providers/...` を通過済み。
 - `git diff --check` を通過済み。
 - `make ci-check` を通過済み。coverage は 83.4%。

2026-06-18 file package boundary split tranche:

- Phase 0 source map refresh:
  - `go list ./internal/tools/...` を取り直し、`internal/tools/file/{directquery,listtool,mutation,pathpolicy,readtool,schema}` が新しい package owner として追加されたことを確認した。
  - caller refs は `internal/agent` の read batch が `readtool`、list_dir cache key が `listtool`、`internal/tools/gathercontext` の direct route が `directquery`、prefetch/locator read が `readtool` に移った。root `internal/tools/file` への non-test named import は残していない。
  - provider tests の root `file` import は built-in tool 登録用 blank import として維持し、具体 tool 型への依存は `internal/api/providers/gemini` の `mutation.StrReplaceTool` へ移した。
  - file package inventory は production 6,469 行 / tests 9,999 行。largest production は `directquery/direct_query_resolve.go` 256 行、largest test は `readtool/read_locator_resolution_test.go` 534 行で、今回差分で 800 行超の touched file は作っていない。
- Phase 1 shared owner split:
  - file-domain の path/root validation owner を `internal/tools/file/pathpolicy` に分離した。`ResolveValidatedPath*`、`NormalizeWorkspaceRoot`、`AppendUniqueString`、`IsPathWithinRoot` だけを caller-facing にし、ignore policy や tool policy は持たせていない。
  - provider-facing parameter schema owner を `internal/tools/file/schema` に分離した。export は `ReadFileParameters`、`WriteFileParameters`、`DeleteFileParameters`、`StrReplaceParameters`、`ListDirParameters` の tool-specific builder に限定し、generic schema helper は private のまま。
- Phase 2 readtool split:
  - `read_file` / `read_files` / locator / detail / render / observation / read section owner を `internal/tools/file/readtool` に移した。
  - agent read batching と gather_context prefetch/locator read は `readtool.ReadExecutionSection`、`ExecuteReadPathsWithDetailSections`、`ExecuteReadTargetsWithDetailSections`、`RenderReadExecutionSections`、`MergeReadExecutionSectionObservations` を直接使う。
  - directquery から解決済み path/root を渡すため、`readtool.ResolvedRequest` と `ExecuteResolvedRequestsWithDetailSections` を追加した。read rendering と observation merge の owner は readtool のまま。
- Phase 3 listtool split:
  - `list_dir` execution、ignore / project-map / cache key / render / summary owner を `internal/tools/file/listtool` に移した。
  - agent dir cache は `listtool.NormalizeCacheKey` / `CachePhysicalPath` に依存する。
  - directquery から解決済み directory target を渡すため、`listtool.ResolvedTarget` と `ExecuteResolvedTarget` を追加した。ignore bypass の実装 owner は listtool に閉じた。
- Phase 4 directquery split:
  - gather_context direct route の parse/classify/resolve/execute owner を `internal/tools/file/directquery` に移した。
  - caller-facing API は `directquery.Policy` / `Outcome` / `Route` / `Plan` / `ExecuteWithObservation` に絞った。
  - `gathercontext` は root `file` ではなく `directquery` + `readtool` を import する。
  - directquery は read/list 実行の private 実装に触れず、readtool/listtool の caller-facing API 経由で実行する。
- Phase 5 mutation split:
  - `write_file` / `delete_file` / `str_replace` の confirmation、preview、apply、diagnostics、`FileChange` gate を `internal/tools/file/mutation` に移した。
  - str_replace は tool ごとの further split をせず、既存 workflow owner を package boundary で固定した。
- Phase 6 root file facade:
  - root `internal/tools/file` は `register.go` と provider schema drift guard test だけに縮めた。tool 型の re-export / wrapper / alias は残していない。
  - `internal/tools/package_boundaries_test.go` を拡張し、root file が registration facade に留まること、`pathpolicy` / `schema` / `readtool` / `listtool` / `mutation` / `directquery` の依存方向を固定した。
- Verification so far:
  - `go test ./internal/tools/file/...` を通過済み。
  - `go test ./internal/tools ./internal/tools/file/... ./internal/tools/gathercontext ./internal/agent ./internal/api/providers/gemini` を通過済み。
  - `go test ./internal/tools/...` を通過済み。
  - `go test ./internal/agent ./internal/toolruntime ./internal/taskstate ./internal/mcptool ./internal/reviewadapter` を通過済み。
 - `go test ./internal/api/providers/...` を通過済み。
  - `git diff --check` を通過済み。
  - `make ci-check` を通過済み。coverage は 83.2%。

2026-06-18 str_replace pure engine split tranche:

- Phase 0 source map refresh:
  - `internal/tools/file/mutation` の `str_replace` は、path validation、file I/O、preview、confirm、apply、syntax warning、failure/status 文言、`FileChange` gate を同一 workflow owner として維持する前提で確認した。
  - pure planning / failure data / diff stats は `write_file` / `delete_file` と共有しないため、`mutation` package 内 file split だけでは private symbol 共有を止められないと判断した。
- Phase 1 package boundary refactor:
  - `internal/tools/file/mutation/replaceengine` を追加し、exact / normalized string replacement、line-range replacement、batch sequential planning、batch execution line stats、line range parsing、nearby duplicate detection を移した。
  - `replaceengine` の dependency は stdlib と `internal/tools/common.FindWithNormalizedWhitespace` に限定した。file I/O、prompt/UI、config、schema、pathpolicy、LSP、`tools.FileChange`、mutation workflow は持たせていない。
  - exported API は `Edit`、`BuildStringExecution` / `StringExecution` / `StringPlan` / `StringFailure`、`BuildLineRangeExecution` / `LineRangeExecution` / `LineRangePlan` / `LineRangeFailure` / `ParseLineRange`、`BuildBatchOutcome` / `BatchOutcome` / `BatchPlan` / `BatchFailure`、`ResolveBatchExecutionLineStats` / `BatchEditLineStats`、`FindAllOccurrencesLineRanges`、`HasNearbyStringDuplicate`、`HasNearbyLineRangeDuplicate` に限定した。
  - failure reason enum や tuning policy は export せず、mutation は builder から返った plan / failure data を accessor 経由で消費する形にした。
- Phase 2 mutation adapter:
  - `str_replace_string.go`、`str_replace_single.go`、`str_replace_batch.go` は `replaceengine` の plan / failure を消費する orchestration に寄せた。
  - batch JSON parse / validation、preview/noop decision、failure message wording、normalized-match warning、large-change warning、syntax warning、confirm handler、apply、`FileChange` recording は `mutation` に残した。
  - `mutation.EditEntry` の alias / re-export は作らず、batch parse は `[]replaceengine.Edit` を返すようにした。
- Phase 3 test boundary:
  - exact replacement、normalized fallback、line range parse / planning、batch sequential planning、diff stats、env/tuning policy は `replaceengine` tests に移した。
  - failure message wording、batch preview terminal result、tool execution、cancel/comment、syntax warning、`FileChange` recording は `mutation` tests に残した。
- Boundary guard:
  - `internal/tools/package_boundaries_test.go` を更新し、`replaceengine` が root `file`、`mutation`、`readtool`、`listtool`、`directquery`、`schema`、`pathpolicy`、root `internal/tools`、`internal/ui`、`internal/config`、LSP 系を import しないことを固定した。
  - `mutation` 以外の file subpackage が `replaceengine` を直接 import しないことを固定した。
- Verification so far:
  - `go test ./internal/tools/file/mutation/replaceengine` を通過済み。
  - `go test ./internal/tools/file/mutation` を通過済み。
  - `go test ./internal/tools` を通過済み。
  - `go test ./internal/tools/file/mutation ./internal/tools/file/...` を通過済み。
  - `go test ./internal/tools/...` を通過済み。
  - `go test ./internal/agent ./internal/toolruntime ./internal/taskstate ./internal/mcptool ./internal/reviewadapter` を通過済み。
  - `go test ./internal/api/providers/...` を通過済み。
  - `git diff --check` を通過済み。
  - `make ci-check` を通過済み。coverage は 83.2%。

残 scope:

- Phase 4 は read request / locator access、direct route、mutation gate、locator test boundary、read/list/direct/mutation/schema/path policy の package boundary を整理済み。`list_dir` の further split は naming drift や次の list-specific finding が出た場合に限定する。
- Phase 5 は web search owner、symbol resolver owner、structured impact production owner、gather_context 側の search route integration test boundary を整理済み。残る追加候補は `search_code_options_test.go` の option / path-basis / ignore-policy split と、`impact_go.go` の metadata / read-item builder split。どちらも今回の changed owner graph 外なので別 tranche 候補とする。
- Phase 6 は request pattern owner、prefetch policy / read / merge owner、direct / search orchestration owner、search integration test boundary を整理済み。残る追加候補は caller-visible observation merge contract の追加 regression を、次に observation semantics を変える場合に置くこと。
- Root は execution core / publish-display / preview owner を整理済み。残る追加候補は `execute_cache_invalidation.go` を mutation `FileChange.Details` contract と一緒に扱う必要が出た場合に限る。
- Subagent は manager test boundary と `manager.go` / `spawn_runtime.go` production owner split を整理済み。残る追加候補は `register.go` の tool parameter / execution adapter 境界を、schema owner と runtime adapter owner に分ける必要が出た場合に限る。
