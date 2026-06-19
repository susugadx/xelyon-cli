# internal/review boundary refactor master plan

この文書は `internal/review` 周辺を挙動維持で分割するための source of truth である。review prompt、report schema、JSON shape、probe 実行方針、external doc / web evidence semantics、provider/TUI/API behavior は変えない。

commit / push / PR 作成はユーザーの明示指示まで行わない。

## Goal

`internal/review` root と主要 subpackage の巨大 file を、既存 owner に沿って file boundary で分割する。package split や facade 削除は今回の目的に含めず、export 増加や import cycle が必要になった場合は別 task に切る。

## Non-goals

- review prompt wording、report validation semantics、external doc query semantics、probe command policy の変更。
- `review_report.v2` schema、JSON field、public facade type/function 名の変更。
- web search provider/config/cache の新設。
- `utils` / `helpers` / `common` package の新設。
- thin forwarding export、import cycle 回避だけの interface、test hook/fallback の追加。

## Boundary map

- `internal/review` root: `ReviewRunner` orchestration、probe execution sequencing、prompt reduction/redaction、web evidence compaction、public facade compatibility。
- `internal/review/evidence`: review evidence bundle construction、generic impact candidate construction、diff/repo/reference scan、rendering、web-search discovery-only evidence collection。
- `internal/review/report`: report domain validation、evidence ref validation、verdict contract、coverage audit、computed summary。
- `internal/review/externaldoc`: external doc query planning、focus token catalog、search result classification、bounded fetch request construction、credibility/focus/content helpers。
- `internal/review/probe`: probe plan/command/path/sandbox policy。今回 production policy は大きく動かさない。
- `internal/review/modelinput` / `modeloutput`: provider-facing prompt/report finalization contracts。今回の split では behavior を変えない。

## Implementation order

1. Test boundary: `internal/review/runner_test.go` を runner lifecycle、validation/repair、prompt/redaction、fake/support に分ける。
2. Evidence generic impact: `evidence_generic_impact.go` を builder orchestration、token extraction、path candidate roles、repo/reference scan、path filters、constants/stopwords に分ける。
3. Report validation: `report_validation.go` を entry/basic fields、evidence refs、verdict contract、enum/id/path helpers、blocked reason helpers に分ける。
4. External doc query: `externaldoc/query.go` を query candidate builder、plan rules、focus token catalog、classification/normalization、fetch request builder に分ける。
5. Root runner/prompt local cleanup: `ReviewRunner` orchestration、saturation、probe result absorption、web evidence compaction、prompt reduction/redaction を file boundary で薄くする。caller-facing API は維持する。
6. Final-A / Final-B: correctness gate と behavior-preserving refactor gate を実施し、残 scope を file/function 単位で記録する。

## Tranche log

### 2026-06-19 baseline

- Focused baseline passed:
  - `go test ./internal/review -run 'TestReviewRunner|TestReviewPrompt|TestReviewRun|TestNewReviewRunner' -count=1`
  - `go test ./internal/review/evidence -run 'Test.*GenericImpact|Test.*Evidence|Test.*Related|Test.*Render|Test.*Context' -count=1`
  - `go test ./internal/review/report -run 'TestValidateReviewReport|Test.*Validation|TestCoverage|TestSaturation|TestComputedSummary' -count=1`
  - `go test ./internal/review/externaldoc -run 'Test.*Search|Test.*Fetch|Test.*Focus|Test.*Support|Test.*Credibility' -count=1`
- Package-boundary decision: existing subpackages already own evidence/report/externaldoc/probe/modelinput/modeloutput. This tranche uses same-package file split only; no new package boundary is introduced.

### 2026-06-19 runner test boundary

- Split `internal/review/runner_test.go` into:
  - `runner_lifecycle_test.go`: happy path, no-probe path, dependency validation, stop-before-next-phase behavior, probe order.
  - `runner_validation_repair_test.go`: probe-plan/report JSON repair and validation repair contracts.
  - `runner_prompt_reduction_contract_test.go`: prompt compaction mode behavior and reduction report counters.
  - `runner_trusted_probe_summary_test.go`: trusted probe-summary injection, finalization, and revalidation.
  - `runner_prompt_path_redaction_contract_test.go`: prompt/final-report absolute path redaction.
  - `runner_test_support_test.go`: fake evidence builder/probe runner/model and runner assertion/marshal helpers.
- Behavior and fixture policy are unchanged; all moved tests still execute through `ReviewRunner.Run`.
- Verification: `go test ./internal/review -run 'TestReviewRunner|TestReviewPrompt|TestReviewRun|TestNewReviewRunner' -count=1`.

### 2026-06-19 evidence generic impact split

- Split `internal/review/evidence/evidence_generic_impact.go` into:
  - `evidence_generic_impact.go`: public entrypoint and builder orchestration.
  - `evidence_generic_impact_constants.go`: roles, caps, search policy constants, excluded parts, stopwords.
  - `evidence_generic_impact_tokens.go`: token extraction, diff-line parsing, identifier/token matching.
  - `evidence_generic_impact_candidates.go`: same-stem, nearby-test, nearby-config candidate roles.
  - `evidence_generic_impact_repo.go`: repo path collection and changed path/dir/stem derivation.
  - `evidence_generic_impact_search.go`: token reference scan and bounded search-file reads.
  - `evidence_generic_impact_paths.go`: path filters, sensitive-path checks, path/set helpers.
- Public surface remains `BuildReviewGenericImpactCandidates` and existing role constants.
- Verification: `go test ./internal/review/evidence -run 'Test.*GenericImpact|Test.*Evidence|Test.*Related|Test.*Render|Test.*Context' -count=1`.

### 2026-06-19 report validation split

- Split `internal/review/report/report_validation.go` into:
  - `report_validation.go`: `ValidateReviewReport` orchestration.
  - `report_validation_basic.go`: schema/target/generated fields, probe summaries, root-cause groups, required content shape.
  - `report_validation_evidence_refs.go`: report-wide evidence ref traversal and individual ref contract.
  - `report_validation_helpers.go`: enum checks, ID/path canonical validation helpers.
  - `report_validation_verdict.go`: clean/has-findings/blocked verdict contract.
  - `report_validation_blocked_reason.go`: blocked-reason compatibility checks.
- Report schema, enum semantics, error text, and exported validation helpers are unchanged.
- Verification: `go test ./internal/review/report -run 'TestValidateReviewReport|Test.*Validation|TestCoverage|TestSaturation|TestComputedSummary' -count=1`.

### 2026-06-19 externaldoc query split

- Split `internal/review/externaldoc/query.go` into:
  - `query.go`: `BuildSearchQueryCandidates`, candidate accessors, candidate shape validation, dedupe key facade.
  - `query_subject_focus.go`: subject/focus candidate construction and full planning corpus assembly.
  - `query_plan_rules.go`: post-Pass1 plan signal rules and plan corpus extraction.
  - `query_focus_catalog.go`: known focus token catalog, focus-token selection, generic-token concreteness.
  - `query_classification.go`: query text, intent/source/confidence classification, normalization.
  - `query_fetch_request.go`: `BuildFetchRequest`.
- Search/fetch contract, candidate cap, metadata reason shape, and dedupe normalization are unchanged.
- Verification: `go test ./internal/review/externaldoc -run 'Test.*Search|Test.*Fetch|Test.*Focus|Test.*Support|Test.*Credibility' -count=1`.

### 2026-06-19 root runner local split

- Split `internal/review/runner.go` into:
  - `runner.go`: runner dependency surface, constructor, and top-level `Run` orchestration.
  - `runner_post_pass1_web_search.go`: post-Pass1 web search evidence merge and coverage-audit delta context setup.
  - `runner_probe_plan.go`: pass1 model call, repair, and evidence-aware probe-plan decode.
  - `runner_report.go`: pass2 report model call, repair, and report finalization.
  - `runner_prompt_reduction_state.go`: prompt reduction stats/state and state-summary prompt bridge.
  - `runner_validate.go`: runner dependency validation.
- Caller-facing `ReviewRunner` API and run order are unchanged.
- Verification: `go test ./internal/review -run 'TestReviewRunner|TestReviewPrompt|TestReviewRun|TestNewReviewRunner' -count=1`.

### 2026-06-19 Final-A / Final-B

- Final-A correctness result: no prompt wording, report schema, JSON shape, externaldoc query semantics, probe policy, provider-facing data, config, or public API changes were introduced. All changes are declaration/file-boundary moves plus the internal master plan.
- Final-B MUST result: `runner_prompt_contract_test.go` still mixed prompt compaction, trusted probe summary injection, and path redaction. It was split into `runner_prompt_reduction_contract_test.go`, `runner_trusted_probe_summary_test.go`, and `runner_prompt_path_redaction_contract_test.go`.
- Final-B SHOULD/NO: `runner_saturation_test.go` remains large but was not changed in this tranche; splitting it would be a separate test-boundary task around saturation/revision/raw-output coverage, not required to close this diff.
- Boundary guards passed:
  - `rg "package .*helpers|package .*utils|package .*common" internal/review`
  - alias/compatibility wording guard from the handoff plan against `internal/review` and this file.
- Final verification passed:
  - `go test ./internal/review -run 'TestReviewRunner|TestReviewPrompt|TestReviewRun|TestNewReviewRunner' -count=1`
  - `go test ./internal/review/evidence -run 'Test.*GenericImpact|Test.*Evidence|Test.*Related|Test.*Render|Test.*Context' -count=1`
  - `go test ./internal/review/report -run 'TestValidateReviewReport|Test.*Validation|TestCoverage|TestSaturation|TestComputedSummary' -count=1`
  - `go test ./internal/review/externaldoc -run 'Test.*Search|Test.*Fetch|Test.*Focus|Test.*Support|Test.*Credibility' -count=1`
  - `go list ./internal/review/...`
  - `go test ./internal/review/...`
  - `go test ./internal/agent ./internal/tui ./internal/tuiagent ./internal/app`
  - `go list ./...`
  - `go test ./...`
  - `git diff --check`
  - `make ci-check`

## Completion checks

- Focused tests for each touched package pass after its tranche.
- `go list ./internal/review/...` passes.
- `go test ./internal/review/...` passes.
- Boundary guard passes:
  - `rg "package .*helpers|package .*utils|package .*common" internal/review`
  - Run the alias/compatibility wording guard from the handoff plan against `internal/review` and this file.
- Caller/final gate passes:
  - `go test ./internal/agent ./internal/tui ./internal/tuiagent ./internal/app`
  - `go list ./...`
  - `go test ./...`
  - `git diff --check`
  - `make ci-check`

## Pending scope

- `probe/probe_plan_test.go` は必要になった場合だけ test boundary 整理する。sandbox/path/security policy は別 task。
- Package split、facade removal、public API cleanup は今回の stop condition 外。
