# Provider History Raw Output Artifact / Rehydrate Master Plan

この文書は Codex Goal でまとめて実装するための内部実装仕様書である。
公開 docs ではなく、provider-facing history reduction の raw output artifact / active context rehydrate 設計・壁打ち・handoff の source of truth として使う。

## 0. Purpose

provider-facing history reduction で、data-bearing command / tool result を品質劣化なしに削減対象へ広げる。

現在は `curl` / `sqlite3` / `psql` / MCP result / web search replay のような data-bearing output を provider prompt から削ると、同じ turn 以降の model が raw evidence を再取得できない。そのため Raw Output Artifact Store / raw-output ref と active context rehydrate を導入し、raw source を保持しながら provider-facing projection だけを軽量化できるようにする。

この計画の完了条件:

- artifact / raw-output rehydrate がない状態では data-bearing output を apply 削減しない。
- data-bearing output を apply 削減する場合は、artifact-backed raw output ref と active context rehydrate transport が両方成立している。
- `commandoutputs` は artifact I/O を持たず、分類・decision・replacement builder の owner に閉じる。
- `rawoutputs` は physical raw output artifact store / manifest / resolver / retention policy の owner になる。
- `providerhistory` は provider-facing projection と raw output ref metadata の owner になる。
- `agent` は request-local active context rehydrate の owner になる。
- dry-run / apply / report / status / tests / Final-A / Final-B の完了条件を明確にする。

## 1. Current State / Implemented Preconditions

確認済みの現在状態:

- `internal/providerhistory.Project` は raw messages を clone し、provider-facing projection と report を作る。
- raw `Agent.History` / session messages / persisted JSONL は provider-facing projection とは別に保持する契約がある。
- `internal/providerhistory` には evidence-backed replacement がある。
  - `read_file`
  - `search_code`
  - `gather_context`
- evidence-backed replacement は `taskstate.EvidencePointer` と active context transport availability によって apply 可否を決める。
- `internal/agent` には `EnableProviderHistoryRehydrateContext` と provider history rehydrate plan 系の wiring がある。
- `taskstate.EvidencePointer` は file path + line range の current file rehydrate 用である。
- `internal/commandoutputs.BuildReplacement` は command output の replacement / keep reason を返すが、raw artifact / ref metadata は持たない。
- `internal/providerhistory/command_edit_dry_run.go` は `commandoutputs.BuildReplacement` の結果だけで command output replacement を apply する。
- `internal/review/artifact` は `/review` debug artifact writer であり、provider history runtime raw output store ではない。

既存の問題:

- `internal/commandoutputs/commandoutputs.go` では `classifyFailure` が `commandFamilyNetwork` / `commandFamilyDatabase` の raw keep guard より前に走る。
- そのため `curl` / `sqlite3` の出力本文が `Error:` で始まるなど failure-like な場合、data-bearing raw body が failure compact placeholder に置換されうる。
- 現時点の command output replacement には artifact-backed raw output ref / active context rehydrate がないため、この置換は provider-facing evidence loss になる。

## 2. Global Contracts

すべての phase で守る契約:

- raw `Agent.History` を変更しない。
- `Session.Messages` を変更しない。
- audit / change records を変更しない。
- persisted JSONL の raw message payload を provider-facing projection のためだけに削らない。
- provider-facing projection / request-local active context だけを軽量化・補完対象にする。
- request-local active context は history / session / audit に保存しない。
- latest tool suffix は置換しない。
- no later assistant の直近 tool result は置換しない。
- dry-run mode では provider payload を変更しない。
- apply mode でも safety gate / threshold gate / rehydrate gate を満たすものだけ変更する。
- 実 replacement / compact が起きた場合だけ Responses `previous_response_id` chain を disable する。
- no replacement の場合は existing chain / state を保持する。
- placeholder / compact が元 payload より小さくない場合は置換しない。
- min saved tokens threshold 未満は replacement しない。
- savings report は apply eligibility と同じ threshold を使い、過大計上しない。

### 2.1 Provider-facing Data Preservation Priority

provider-facing output strategy の優先順位:

1. `sensitive` / credential / secret sink guard
2. `data-bearing` family guard
3. raw output artifact/ref availability
4. active context rehydrate transport availability
5. failure / success classifier
6. compact / omit strategy

意味:

- data-bearing output は、failure-like marker より前に data preservation policy で扱う。
- `Error:` / `fatal:` / `non-zero` などの marker は execution status の evidence ではあるが、data-bearing body を不可逆に落としてよい根拠にはしない。
- artifact-backed raw output ref がない data-bearing output は apply で削減しない。
- artifact-backed raw output ref だけでは不十分。provider request 前に raw output / bounded excerpt を active context として戻す transport が必要。

### 2.1.1 Adopted Raw Source Backing

最高設計として、raw source backing は artifact-store-backed hybrid を採用する。

契約:

- primary rehydrate backing は Raw Output Artifact Store。
- raw `Agent.History` / `Session.Messages` / persisted JSONL は、当面 audit / compatibility ledger として raw payload を保持する。
- history-backed resolver は legacy session / migration / missing artifact diagnostics の fallback として残す。
- provider-facing projection は raw body を直接持たず、`raw_output_ref` / content hash / bounded excerpt / source metadata を持つ。
- artifact store は content-addressed storage を基本にする。
- artifact manifest は source event / surface / tool call / redacted source preview / semantic role / hash / size / retention metadata を持つ。
- raw command / raw URL / raw output text は manifest source of truth にしない。
- command text は `command_hash` と redaction 済み `command_preview` に分ける。
- raw output artifact の path / permissions / atomic write / size limit / retention / cleanup / hash validation は `rawoutputs` が owner になる。

意味:

- `providerhistory` は raw body を保存する package ではない。
- `commandoutputs` は raw body を保存する package ではない。
- `agent` は request 前に `rawoutputs` resolver を使って active context を組み立てる。
- session reload 後も artifact store から deterministic に rehydrate できることを目標にする。

### 2.2 Raw Output Ref Is Not Decoration

provider-facing placeholder に `raw_output_ref` を入れる場合、その ref は agent runtime が解決できる必要がある。

禁止:

- 実体のない `raw_output_ref` を provider prompt に書く。
- model が自力で local file / artifact を読む前提にする。
- raw artifact があることだけを provider-facing data loss の修正根拠にする。

必須:

- raw output ref は stable key / content hash / source metadata を持つ。
- request path は ref から raw body または bounded excerpt を rehydrate できる。
- rehydrate できない場合、apply compact は keep raw に戻る。

### 2.3 Sensitive Output Handling

credential / secret / sensitive output は data-bearing optimization より優先して保護する。

- sensitive output を通常の Raw Output Artifact Store に複製しない。
- sensitive output の raw body を active context に自動再注入しない。
- 既存 raw history に存在する sensitive content を provider-facing projection / report / status / artifact path で新たに拡散しない。
- sensitive family は dedicated classifier / reason を持ち、data-bearing family と同じ artifact-backed compact strategy に混ぜない。

### 2.4 Security / Redaction / Sensitive Data Contract

raw output artifact / report / ledger / active context は fail-closed redaction contract に従う。

Core rule:

```text
sensitive or ambiguous
  -> do not create normal raw output artifact
  -> do not apply compact
  -> keep raw provider-facing payload
  -> report only sanitized reason metadata
```

Sensitive handling:

- Sensitive output is `no artifact`, `no artifact-backed apply`, `raw keep`.
- Ambiguous secret detection is treated as sensitive for artifact-backed optimization.
- Existing raw history/session/audit may already contain the raw body; this plan must not create an additional artifact copy for sensitive output.
- Sensitive refs are not body rehydrated automatically, even if referenced later.
- Dedicated future sensitive storage policy is out of scope and must not be implied by encrypted history mode alone.

Raw body allowed locations:

```text
Allowed:
  raw Agent.History / Session.Messages / audit / persisted JSONL existing surfaces
  encrypted raw output artifact object for non-sensitive data-bearing output
  request-local active context body/excerpt

Forbidden:
  ProjectionReport
  /status
  /ledger
  manifest metadata
  index cache
  provider-facing placeholder metadata beyond bounded safe excerpt
  config
  logs
```

`raw_output_ref` identity:

- `RawOutputRefID` is opaque.
- Ref IDs must not include command text, URL, path, DB name, table name, query text, raw output text, provider/tool label, user label, or secret-bearing metadata.
- Provider/tool/user-provided labels are untrusted input. They may be displayed only after sanitization and must not become ref IDs, artifact paths, manifest keys, or filesystem names.

Metadata redaction:

- URL metadata may include method, domain, and path shape only when sanitized. Query and fragment are redacted.
- DB metadata may include database family, query kind, and result shape. Full query text and literal values are not report/ledger metadata.
- Command metadata may include source kind, command family, semantic role, classifier, hash, and redacted preview. Full args are not report/ledger metadata.
- Hash, size, MIME/content kind, line count, approximate tokens, surface, semantic role, and lifecycle state are allowed metadata when they do not encode raw content or secrets.

Shared redaction owner contract:

- Surface-specific classifiers may live in `commandoutputs`, providerhistory tool-result packages, and review prompt surfaces.
- Placeholder, report, status, `/ledger`, manifest metadata, and active context metadata must use the same redaction contract.
- Redaction behavior must be testable from caller-visible surfaces, not only helper-level unit tests.

## 3. Non-goals

今回やらないこと:

- artifact / active context rehydrate なしで data-bearing command output apply compact を有効化すること。
- `commandoutputs` package に file I/O / session I/O / artifact writer を持たせること。
- provider model が local artifact file を直接読める前提で設計すること。
- raw `Agent.History` / session / persisted JSONL を provider-facing optimization のために削ること。
- `/review` artifact writer を provider history runtime raw output store として流用すること。
- history-backed resolver だけを最終 backing として扱うこと。
- command execution behavior / tool runtime behavior 自体を変えること。
- live provider smoke を必須にすること。
- MCP / web search / provider-native replay を Phase 0-5 の中で無理に同時実装すること。

## 4. Source Findings

確認済み:

- `internal/commandoutputs/commandoutputs.go`
  - `BuildReplacement` は `classifyFailure` を先に実行し、その後 `commandFamilyNetwork` / `commandFamilyDatabase` を raw keep している。
  - data-bearing failure-like body の keep contract が分類順で破れる。
- `internal/commandoutputs/success_test.go`
  - data-bearing success output raw keep は固定されている。
  - failure-like data-bearing output raw keep はまだ固定されていない。
- `internal/providerhistory/command_edit_dry_run.go`
  - command output replacement は `commandoutputs.BuildReplacement` の `Replacement` / keep reason に依存する。
  - `CommandEditDryRunCandidate` に raw output ref metadata はない。
  - apply mode では `SuggestedReplacementText` を tool result content に直接入れる。
- `internal/providerhistory/reduction_apply.go`
  - evidence-backed tool result replacement は `taskstate.EvidencePointer` と active context transport availability を見る。
  - command output replacement はこの evidence-backed path とは別の dry-run/apply path にある。
- `internal/providerhistory/reduction.go`
  - `Policy` は `EvidencePointers`, `EvidenceReductionRequiresActiveContext`, `ActiveContextTransportAvailable` を持つ。
  - command output raw ref 用 field はない。
- `internal/agent/provider_history_reduction_runtime.go`
  - runtime config から provider history reduction mode と rehydrate context availability を解決している。
- `internal/taskstate/evidence_rehydrate.go`
  - `EvidencePointer` は file path + line range rehydrate 用であり、command output raw body pointer ではない。
- `internal/review/artifact`
  - review run artifact writer は `/review` debug artifact 用で、provider request lifecycle の source of truth ではない。

Not checked yet:

- session save/load が raw output ref / manifest metadata を持つ場合の exact owner。
- active context prompt assembly の最終 injection point。
- status / UI 表示に raw output ref / artifact counts を出す最適な場所。
- Raw Output Artifact Store の directory / security / retention policy。

## 5. Responsibility Boundaries

### `internal/commandoutputs`

Owner:

- command family classification
- semantic role classification
- failure / success classifier
- compact / placeholder text builder
- decision model

禁止:

- file I/O
- session I/O
- artifact path generation
- agent runtime state access
- provider request injection

### `internal/rawoutputs`

Owner:

- raw output artifact ID / ref / manifest domain types
- content-addressed artifact storage
- artifact path generation under the XELYON data/session root
- atomic write / read / close / cleanup
- content hash calculation and verification
- raw artifact size limits
- manifest persistence / load / clone-safe metadata
- retention / garbage collection policy
- history-backed fallback resolver for legacy sessions

禁止:

- provider-facing projection policy
- command family / success / failure classification
- active context prompt formatting
- provider request assembly

### `internal/providerhistory`

Owner:

- provider-facing projection
- dry-run / apply policy
- raw output ref metadata in candidates / reports / placeholders
- artifact-backed replacement eligibility
- replacement report / kept reason / saved bytes / saved tokens
- Responses chain disable decision

禁止:

- raw artifact physical write implementation
- raw artifact path/security/retention policy
- active context prompt injection
- command execution

### `internal/agent`

Owner:

- runtime raw history source / event lifecycle
- provider request path
- active context rehydrate
- request-local injected context
- runtime config wiring
- session/load interaction if raw output ref metadata must round-trip
- raw output artifact capture wiring through `rawoutputs`

### `internal/taskstate`

Owner:

- file evidence pointer and current file rehydrate.

原則:

- command output raw ref を既存 `EvidencePointer` に無理に混ぜない。
- 必要なら `RawOutputRef` / `ProviderHistoryRawOutputRef` のような別 contract を作る。

### `internal/review/artifact`

Owner:

- `/review` run debug artifact.

原則:

- provider history raw output artifact store として流用しない。

## 6. Implementation Priority

1. Phase 0: Immediate P2 fix / data-bearing failure-like output raw keep
2. Phase 0.5: boundary refactor / owner split / test foundation
3. Phase 1: `commandoutputs` decision model
4. Phase 2: Raw Output Artifact Store + raw output ref contract
5. Phase 3: artifact-backed dry-run
6. Phase 4: active context rehydrate
7. Phase 5: artifact-backed apply for data-bearing command output
8. Phase 6: MCP / web search / provider-native replay extension
9. Phase Final-A: impact audit / review-hole sweep
10. Phase Final-B: mandatory comprehensive refactor including tests

### 6.1 Phase Gate / Continuation Policy

The implementation should keep moving by falling back to safe behavior.

Principle:

```text
unsafe / missing / ambiguous / not yet implemented
  -> raw keep
  -> candidate-only
  -> dry-run only
  -> sanitized reason report
  -> continue to the next safe phase
```

This plan does not use "stop" as the default response to uncertainty.
It uses fail-closed projection behavior and explicit reports so the Goal can continue implementing other safe phases.

Non-blocking fail-closed cases:

- artifact missing -> raw keep + reason report + continue
- hash mismatch -> quarantine/fail-closed reason + raw keep + continue
- decrypt failure -> fail-closed reason + raw keep + continue
- rehydrate unsupported -> raw keep + reason report + continue
- active context budget exhausted -> required/optional ref failure report + keep raw or metadata-only as allowed + continue
- redaction ambiguous -> no artifact + raw keep + sanitized reason + continue
- provider-native source owner unclear -> candidate-only / dry-run only + continue
- review prompt required ref weak -> reject saturation/apply absorption for that prompt, keep raw prompt context, continue other review optimization work
- candidate count too large -> bounded report/ledger + continue
- owner boundary mismatch -> do local refactor/package split/test split and continue

Hard stop / phase-blocking conditions:

- Any implementation path deletes or rewrites raw `Agent.History`, `Session.Messages`, audit records, or persisted JSONL for provider-facing optimization.
- Raw body is stored in `ProjectionReport`, `/status`, `/ledger`, manifest metadata, index cache, config, logs, or provider-facing placeholder metadata beyond bounded safe excerpt.
- Artifact path, ref ID, manifest key, or filesystem name is derived from user/provider/tool text.
- Secret/sensitive/ambiguous output would need to be stored in the normal raw output artifact store to continue.
- Data-bearing apply compact can happen without artifact-backed raw output ref and active context transport.
- Tests cannot prove whether provider-facing payload stayed raw or was compacted for the target path.

Hard stop should be scoped to the unsafe phase or surface whenever possible.
Other surfaces and earlier safe phases should continue as raw keep, dry-run, or candidate-only.

Phase verification rule:

- Each phase must have focused tests for its owner boundary before moving to the next phase.
- Phase 5 apply is not enabled until Phases 1-4 have passing focused tests and report/status evidence proves all gates.
- Phase 6 apply expansion is surface-specific and must not inherit command-output apply eligibility automatically.

## 7. Implementation Sections

## 7.1 Phase 0: Immediate P2 Fix

### Purpose

現在の reviewer finding を閉じる。
artifact / rehydrate が未実装の状態では、network/database data-bearing output を failure compact より前に raw keep する。

### Non-goals

- artifact-backed compact は実装しない。
- command output decision model への全面移行は行わない。
- database subfamily の最適化を深追いしない。

### Current source findings

`BuildReplacement` は failure classifier が data-bearing keep guard より前にある。

### Design contract

- `commandFamilyNetwork` / `commandFamilyDatabase` は、artifact-backed compact がない限り failure-like output でも keep raw。
- immediate fix は safety-first とし、削減量より evidence preservation を優先する。
- sensitive family はこの keep guard と混ぜない。
- database subfamily classification は Phase 1+ の decision model で扱う。Phase 0 では database family 全体を raw keep する。

### Safety gates

- `curl` output starting with `Error:` and large JSON-like body is kept raw.
- `sqlite3` output starting with `Error:` and large tabular/JSON-like body is kept raw.
- package / deploy / sensitive textual failures remain failure compact candidates.
- validation failure remains failure compact candidate.

### Behavior / output format

No provider-facing replacement for data-bearing network/database command output.

Keep reasons:

- `data_bearing_network_command_output_keep`
- `data_bearing_database_command_output_keep`

### Report / status

Existing kept reason counts should record the data-bearing keep reason.

### Tests

- `internal/commandoutputs`
  - `TestBuildReplacementDataBearingFailureLikeOutputKeepsRaw`
- `internal/providerhistory`
  - apply projection keeps raw `curl` output starting with `Error:`
  - apply projection keeps raw `sqlite3` output starting with `Error:`
  - Responses chain is not disabled when no command replacement applies.

### Implementation owner candidates

- `internal/commandoutputs/commandoutputs.go`
- `internal/commandoutputs/success_test.go` or focused `data_bearing_test.go`
- `internal/providerhistory/projection_test.go`

### Decision status

Phase 0 has no open decisions. Safety-first raw keep is required before artifact-backed compact exists.

## 7.1.5 Phase 0.5: Boundary Refactor / Owner Split / Test Foundation

### Purpose

Phase 1+ の実装を巨大 file への追記で進めないため、behavior-preserving な owner split と test boundary を先に作る。

### Non-goals

- data-bearing artifact-backed apply は有効化しない。
- raw artifact store behavior は追加しない。
- provider-facing payload behavior は Phase 0 修正以外で変えない。

### Design contract

Owner split:

```text
commandoutputs
  semantic decision / classifier / replacement display strategy

rawoutputs
  artifact storage / manifest / resolver / lifecycle ledger

providerhistory
  projection / candidate gate / report / placeholder finalization

agent
  runtime config wiring / request-local active context transport / command surface rendering
```

Refactor expectations:

- If `commandoutputs` is still accumulating classifier, report, artifact, or projection policy in one file, split before Phase 1 logic grows.
- If providerhistory projection, report counters, and command/edit dry-run policy are coupled in one helper, extract owner-focused helpers before Phase 3.
- If tests add unrelated cases to already large files, create focused test files by owner.
- Do not add generic helpers that hide semantic differences between data-bearing, sensitive, side-effect, validation, and review surfaces.

### Safety gates

- Behavior remains Phase 0-equivalent after refactor.
- Public/provider-facing payload is unchanged except approved Phase 0 raw keep.
- Tests still prove data-bearing failure-like command output is raw keep.
- No new exported API is added solely to make tests easier.

### Tests

- package boundary / owner-focused tests where useful
- existing Phase 0 regression tests still pass
- focused test files exist for:
  - classifier / decision
  - projection / report
  - raw artifact store once introduced
  - active context transport once introduced
  - config resolution once introduced

### Implementation owner candidates

- `internal/commandoutputs`
- `internal/providerhistory`
- `internal/agent`
- `internal/config`
- test files adjacent to the owner under test

### Continuation rule

If a split becomes larger than expected, keep moving by preserving behavior and reporting the remaining owner debt by package/file.
Only block later apply phases if the owner debt would make apply safety untestable.

## 7.2 Phase 1: `commandoutputs` Decision Model

### Purpose

`commandoutputs` を pure decision engine に寄せる。
`BuildReplacement` の `Replacement, reason, ok` contract を本体にせず、classification / semantic role / strategy / artifact eligibility / replacement plan を構造化した `Decision` を source of truth にする。

### Non-goals

- providerhistory artifact I/O は入れない。
- data-bearing apply compact は有効化しない。

### Design contract

最高構成:

- Main API is `Decide(...) Decision`.
- `BuildReplacement(...)` remains as a compatibility wrapper during migration.
- `commandoutputs` never decides provider-facing apply eligibility.
- `commandoutputs` never writes artifacts and never reads artifact/store/session/agent state.
- `providerhistory` consumes `Decision.Action`, `SemanticRole`, `ArtifactPolicy`, and `ReplacementPlan` to decide dry-run/apply/report behavior.
- Replacement text is a display strategy, not the decision source of truth.

Decision model direction:

```go
type DecisionAction string

const (
    DecisionKeepRaw                 DecisionAction = "keep_raw"
    DecisionInlineCompact           DecisionAction = "inline_compact"
    DecisionArtifactBackedCandidate DecisionAction = "artifact_backed_candidate"
)

type SemanticRole string

const (
    SemanticRoleValidationLog SemanticRole = "validation_log"
    SemanticRoleOperationLog  SemanticRole = "operation_log"
    SemanticRoleDataBearing   SemanticRole = "data_bearing"
    SemanticRoleSensitive     SemanticRole = "sensitive"
    SemanticRoleSideEffect    SemanticRole = "side_effect"
    SemanticRoleUnknown       SemanticRole = "unknown"
)

type DatabaseSubfamily string

const (
    DatabaseSubfamilyQueryResult     DatabaseSubfamily = "database_query_result"
    DatabaseSubfamilySchemaResult    DatabaseSubfamily = "database_schema_result"
    DatabaseSubfamilyOperationLog    DatabaseSubfamily = "database_operation_log"
    DatabaseSubfamilyMigrationLog    DatabaseSubfamily = "database_migration_log"
    DatabaseSubfamilyConnectionError DatabaseSubfamily = "database_connection_error"
    DatabaseSubfamilyUnknown         DatabaseSubfamily = "database_unknown"
)

type Decision struct {
    Action          DecisionAction
    SemanticRole    SemanticRole
    Family          string
    Subfamily       string
    Classifier      string
    KeepReason      string
    FailureSignal   FailureSignal
    SuccessSignal   SuccessSignal
    ReplacementPlan ReplacementPlan
    ArtifactPolicy  ArtifactPolicy
    Preconditions   []DecisionPrecondition
    Evidence        DecisionEvidence
}
```

Implementation may adapt names to local style.

Suggested supporting contracts:

```go
type ArtifactPolicy struct {
    Eligible         bool
    RequiredForApply bool
    Reason           string
    ExcerptPolicy    string
}

type ReplacementPlan struct {
    Kind       ReplacementKind
    Header     string
    Excerpt    string
    Reason     string
    Classifier string
}
```

Artifact policy meaning:

- `commandoutputs` may say "this needs an artifact before apply compact can be safe".
- `commandoutputs` may not say "artifact exists" or "apply is allowed".
- `ArtifactPolicy.RequiredForApply=true` is a precondition for providerhistory, not a complete apply decision.
- Data-bearing provider-facing placeholders are finalized by `providerhistory` after raw ref / artifact / rehydrate / threshold gates pass.
- `commandoutputs` must not generate a final data-bearing apply placeholder that lacks a real `raw_output_ref`.

### Safety gates

- Data-bearing guard happens before failure/success compact strategy.
- Sensitive guard happens before data-bearing artifact-backed strategy.
- Strong success/failure evidence precedence remains explicit.
- Unknown family does not become artifact-backed compact by default.
- Database query/schema results are data-bearing.
- Database operation/migration/connection errors are operation/failure evidence, not data-bearing artifact-backed apply candidates.
- Database unknown remains raw keep or dry-run only; no artifact-backed apply.

Global command output precedence:

```text
1. sensitive / secret guard
2. data-bearing preservation
3. side-effect / mutation / deploy / migration
4. explicit fatal / non-zero / command failed
5. strong validation success
6. weak textual failure marker
7. success / validation compact
8. unknown raw keep
```

Meaning:

- A failure-like substring cannot outrank sensitive or data-bearing preservation.
- A success-like substring cannot erase explicit non-zero/fatal command failure.
- Expected exception/error text inside successful validation logs does not become failure by substring alone.
- Side-effect/deploy/migration/write-operation logs are not treated as data-bearing response bodies.
- Unknown command output with error words does not become a failed-command compact unless fatal/non-zero context is present.

### Database subfamily policy

Adopt these database subfamilies:

```text
database_query_result
database_schema_result
database_operation_log
database_migration_log
database_connection_error
database_unknown
```

Treatment:

- `database_query_result`
  - semantic role: `data_bearing`
  - artifact-backed data-bearing compact eligible after artifact/rehydrate/threshold gates
  - excerpt policy: columns/header, first rows, matched rows, row count, last rows
- `database_schema_result`
  - semantic role: `data_bearing`
  - artifact-backed data-bearing compact eligible after artifact/rehydrate/threshold gates
  - excerpt policy: schema/table/columns/indexes first
- `database_operation_log`
  - semantic role: `operation_log`
  - use success/failure command output compaction path, not data-bearing artifact-backed apply
- `database_migration_log`
  - semantic role: `side_effect` / `operation_log`
  - use validation/failure evidence path; failed migrations remain failure evidence
  - not data-bearing artifact-backed apply eligible
- `database_connection_error`
  - semantic role: `operation_log` / failure
  - use failure compaction path
  - not data-bearing artifact-backed apply eligible
- `database_unknown`
  - raw keep or dry-run only
  - no artifact-backed apply

Classification precedence:

```text
1. sensitive / secret guard
2. explicit migration / side-effect command
3. explicit connection / auth failure
4. explicit query / schema result command
5. output shape evidence
6. database_unknown
```

### Behavior / output format

Phase 1 should be behavior-preserving except for already-approved Phase 0 raw keep fix.

### Report / status

No public report field required in Phase 1 unless existing tests need classifier / keep reason updates.

### Tests

- Existing `internal/commandoutputs` tests continue to pass.
- Add a precedence matrix test:
  - sensitive vs data-bearing
  - data-bearing vs failure marker
  - validation success vs weak failure marker
  - textual failure vs success marker
  - side-effect textual failure
  - unknown command with error words
  - network/database query result vs failure-like body
  - database query/schema result vs failure-like marker
  - database migration/operation/connection error vs data-bearing artifact eligibility
  - database unknown raw keep / dry-run only
  - git diff/status/list-dir structured compact paths are not misclassified by content-only failure words
- Add wrapper compatibility tests proving `BuildReplacement` is derived from `Decide`.
- Add providerhistory-facing tests proving artifact-backed candidates require providerhistory raw ref / rehydrate gates before apply.

### Implementation owner candidates

- `internal/commandoutputs/commandoutputs.go`
- `internal/commandoutputs/failure.go`
- `internal/commandoutputs/family.go`
- `internal/commandoutputs/*_test.go`

### Decision status

Phase 1 API direction is decided. Exact enum/type names may adapt to local Go style as long as `Decide` is the source of truth and `BuildReplacement` remains a compatibility wrapper during migration.

## 7.3 Phase 2: Raw Output Artifact Store + Ref Contract

### Purpose

Raw Output Artifact Store を導入し、command output raw body を stable ref で解決できる contract を作る。
この phase では provider-facing payload をまだ変更しないか、dry-run report に限定する。

### Non-goals

- Active context injection is not implemented in this phase.
- Apply compact for data-bearing output remains disabled.
- Sensitive output artifact storage is not enabled.
- Raw history/session payload elision is not implemented.

### Design contract

`RawOutputRef` は既存 `taskstate.EvidencePointer` と別 contract にする。

最高設計の backing:

- primary backing: Raw Output Artifact Store
- fallback backing: raw history/session resolver for legacy or missing-artifact diagnostics
- source ledger: raw history/session/audit remains unmodified in this plan

Package direction:

```text
commandoutputs -> classification / semantic role only
rawoutputs     -> artifact storage / manifest / resolver / hash / retention
providerhistory -> projection candidates / placeholder / report metadata
agent          -> capture wiring / active context rehydrate request path
```

Rawoutputs package contract:

- `rawoutputs` is a storage engine + resolver + lifecycle ledger.
- `rawoutputs` owns raw body persistence, manifest append, index rebuild, hash verification, encryption policy, quota checks, path safety, quarantine, tombstone, GC collection, and legacy artifact materialization.
- `rawoutputs` does not classify command output.
- `rawoutputs` does not decide provider-facing apply eligibility.
- `rawoutputs` does not format provider prompts or active context.
- `rawoutputs` does not decide review saturation/revision behavior.

Recommended package API direction:

```go
type Store struct { ... }

func OpenStore(root Root, opts StoreOptions) (*Store, error)

func (s *Store) Create(ctx context.Context, req CreateRequest) (CreateResult, error)
func (s *Store) Resolve(ctx context.Context, ref RawOutputRef) (ResolvedArtifact, error)
func (s *Store) Verify(ctx context.Context, ref RawOutputRef) (VerifyResult, error)
func (s *Store) MaterializeLegacy(ctx context.Context, req LegacyMaterializeRequest) (CreateResult, error)
func (s *Store) CollectGarbage(ctx context.Context, req GCRequest) (GCResult, error)
func (s *Store) RebuildIndex(ctx context.Context) (IndexResult, error)
```

API owner rules:

- `Create` receives a raw body stream and returns `RawOutputRef` / artifact metadata after a successful write + manifest append.
- `Resolve` takes `RawOutputRef`, not only `RefID`, so it can validate session, surface, artifact ID, and content hash before returning body.
- `Verify` performs lifecycle + hash checks without exposing body.
- `MaterializeLegacy` creates an artifact from an exact raw history/session source and is not a permission to apply compact by itself.
- `CollectGarbage` receives caller-provided live refs; `rawoutputs` calculates delete eligibility but does not discover providerhistory/review liveness on its own.
- `RebuildIndex` rebuilds cache from manifest and never treats index as source of truth.

Create request direction:

```go
type CreateRequest struct {
    Surface        Surface
    SessionID      string
    Source         SourceMetadata
    Classification ClassificationMetadata
    Body           io.Reader
    SizeHintBytes  int64
    Retention      RetentionPolicy
}
```

Create contract:

- Body input is streaming (`io.Reader`) so large artifacts do not require full body memory.
- Size and session quota are checked before committing a ref.
- Content hash is calculated while streaming the body.
- Body is written to a temp file and atomically renamed into the content-addressed object path.
- Directory permissions are `0700`; object file permissions are `0600`.
- History encryption mode is applied before body/manifest/index become durable.
- `raw_output_artifact_created` is appended only after object write, hash verification, and metadata validation pass.
- Cap/quota/sensitive/security failures do not create refs and do not allow provider-facing compact.

Resolve result direction:

```go
type ResolvedArtifact struct {
    Ref         RawOutputRef
    Body        io.ReadCloser
    SizeBytes   int64
    ContentHash string
}
```

Resolve contract:

- Resolve validates session ID, surface, artifact ID, manifest/index lifecycle state, object path, and content hash.
- Resolve never follows symlinks and never resolves paths derived from raw command/URL/output.
- Resolve returns no body for missing, tombstoned, quarantined, GC-collected, hash-mismatched, decrypt-failed, or path-invalid artifacts.
- Resolve may use the index for lookup, but manifest lifecycle + object hash verification are the final gates.
- Any mismatch fails closed and records/returns a structured reason.

方向性:

```go
type RawOutputRef struct {
    RefID          string
    Surface        string
    SessionID      string
    EventID        string
    HistoryIndex   int
    ToolName       string
    ToolCallID     string
    CommandHash    string
    CommandPreview string
    ArtifactID     string
    Family         string
    SemanticRole   string
    Classifier     string
    ContentHash    string
    ByteSize       int
    RuneSize       int
    ApproxTokens   int
}

type RawOutputArtifact struct {
    ArtifactID   string
    ContentHash  string
    ByteSize     int
    StorageKind  string
    RelativePath string
}

type RawOutputManifest struct {
    SchemaVersion int
    RecordType    string
    Ref           RawOutputRef
    Source        RawOutputSource
    Artifact      RawOutputArtifact
    Retention     RawOutputRetention
    CreatedAt     time.Time
}

type RawOutputSource struct {
    Provider       string
    Model          string
    CommandHash    string
    CommandPreview string
}

type RawOutputRetention struct {
    Policy    string
    CreatedAt time.Time
}
```

`RefID` example:

```text
rawout_c3f8a19b2d
```

Ref ID contract:

- `RefID` is an opaque short stable ID.
- `RefID` must not include raw command text, URL, path, args, provider output text, or user/provider-provided labels.
- `RefID` should be short enough for provider-facing placeholders.
- `RefID` uniqueness is scoped by session/report plus content/source metadata; exact implementation may use a deterministic hash-derived short ID with collision handling.
- Human-readable source details live in redacted metadata, not in the ID.

`ArtifactID` should be content-addressed:

```text
sha256:<hash>
```

Raw output metadata placement:

```text
provider-facing placeholder:
  minimal readable metadata + raw_output_ref

ProjectionReport.RawOutputRefs:
  full projection metadata for resolver/report/status

rawoutputs manifest:
  storage lifecycle source of truth

artifact object:
  raw body only
```

Provider-facing placeholder contract:

- Keep placeholder short.
- Include `raw_output_ref`, `surface`, `semantic_role`, family/subfamily/classifier when useful, byte size, content hash prefix or hash ID, and bounded excerpt.
- Do not include raw command, raw URL, raw args, raw output body beyond bounded excerpt, object path, manifest path, or local filesystem path.
- Placeholder `raw_output_ref` must resolve through `ProjectionReport.RawOutputRefs` and `rawoutputs`; otherwise apply compact is forbidden.

Placeholder direction:

```text
[compacted old data-bearing command output;
 raw_output_ref=rawout_c3f8a19b2d;
 surface=command_output;
 semantic_role=data_bearing;
 family=network;
 bytes=48231;
 sha256=4fa2...]
excerpt:
  <bounded excerpt>
```

`ProjectionReport.RawOutputRefs` full metadata direction:

```text
id
surface
semantic_role
family
subfamily
classifier
artifact_id
content_hash
size_bytes
line_count
approx_tokens
source_event_id
history_index
tool_name
tool_call_id
command_hash
command_preview
created_at
artifact_status
rehydrate_status
gate_status
```

Report metadata contract:

- `ProjectionReport.RawOutputRefs` is the source of truth for full projection metadata.
- Candidate structs carry `RawOutputRefID` plus gate/status/reason fields only.
- Candidate structs do not duplicate full raw ref metadata.
- Every candidate `RawOutputRefID` must have exactly one matching report-level ref.
- Report-level refs may exist without replacement candidates when discovery happened but apply/dry-run gates kept raw.
- Missing report-level ref for a candidate fails closed.

Manifest relation:

- Manifest is the storage lifecycle source of truth.
- Report raw ref metadata is projection/report source, not artifact lifecycle source.
- Manifest and report may duplicate stable identifiers, hash, size, surface, semantic role, and redacted preview fields for verification/reporting.
- Manifest owns object path, storage encoding, encryption status, retention, quarantine/tombstone/GC lifecycle events.
- Report owns provider-facing gate status, dry-run/apply status, placeholder linkage, and rehydrate status.
- Raw body exists only in artifact object.

Store requirements:

- Artifact root is session-scoped under history storage:

  ```text
  ~/.xelyon/history/rawoutputs/sessions/<session-id>/
  ```

- Directory layout:

  ```text
  ~/.xelyon/history/rawoutputs/sessions/<session-id>/
    manifests/
      raw_outputs.jsonl
    indexes/
      raw_outputs.index.json
    objects/
      sha256/
        <first2>/
          <next2>/
            <full-sha256>.raw
  ```

- Writes are atomic.
- Paths are derived from constrained session/artifact IDs, not raw command text.
- Artifact files are not named from user/provider/tool output.
- Directories are created with `0700`.
- Artifact files are created with `0600`.
- Raw output artifact storage follows history encryption policy: if `XELYON_ENCRYPT_HISTORY=1`, artifact bodies and manifest/index payloads must not be left as plaintext.
- Artifact body is not included in reports/status.
- Manifest metadata is clone-safe and does not carry raw body.
- Artifact resolution verifies hash before returning raw body.
- Retention/GC can remove unreferenced artifacts without corrupting live refs.
- Store supports deterministic session reload.
- Store can deduplicate identical raw bodies by content hash within the same session.
- Cross-session dedupe is not enabled.

Manifest format:

- `manifests/raw_outputs.jsonl` is the source of truth.
- `indexes/raw_outputs.index.json` is a rebuildable cache, not the source of truth.
- Manifest is a versioned append-only JSONL event log.
- Every record has `schema_version` and `record_type`.
- Raw body is never stored in manifest or index.
- Raw command is never stored in manifest/index as an unbounded trusted string.
- Manifest stores `command_hash` plus redacted `command_preview`.
- Manifest/index payloads follow history encryption policy.
- Existing records are not edited for lifecycle state changes; append a new event.

Record types:

```text
raw_output_artifact_created
raw_output_artifact_quarantined
raw_output_artifact_tombstoned
raw_output_artifact_gc_collected
```

Created record shape direction:

```json
{
  "schema_version": 1,
  "record_type": "raw_output_artifact_created",
  "ref": {
    "ref_id": "rawout_c3f8a19b2d",
    "surface": "command_output",
    "session_id": "1780000000000000000-1",
    "event_id": "history:42",
    "history_index": 42,
    "tool_call_id": "call_abc",
    "tool_name": "bash",
    "artifact_id": "sha256:4fa2...",
    "content_hash": "sha256:4fa2..."
  },
  "source": {
    "provider": "openai",
    "model": "gpt-5.3-codex",
    "command_hash": "sha256:...",
    "command_preview": "curl https://api.example.test/items"
  },
  "classification": {
    "semantic_role": "data_bearing",
    "family": "network",
    "classifier": "network_response",
    "sensitive": false
  },
  "artifact": {
    "artifact_id": "sha256:4fa2...",
    "hash_algorithm": "sha256",
    "content_hash": "sha256:4fa2...",
    "relative_path": "objects/sha256/4f/a2/4fa2...9c.raw",
    "byte_size": 48231,
    "storage_encoding": "raw",
    "encrypted": false
  },
  "retention": {
    "policy": "session",
    "created_at": "2026-06-06T00:00:00Z"
  }
}
```

Retention / GC policy:

- Default retention policy is `session`.
- GC algorithm is live-ref mark-and-sweep.
- `rawoutputs` does not infer live refs by itself; live refs are supplied by providerhistory / agent-session / review owners.
- Session-local content hash dedupe is allowed.
- Cross-session dedupe is not enabled.
- Session delete can delete the session-scoped raw output root after manifest-aware cleanup.
- Session rewrite / truncate / reset appends tombstone events for refs no longer reachable from active session history or compacted provider-facing refs.
- Startup may run opportunistic cleanup, but must not delete artifacts that are still live.
- Future explicit cleanup commands must use the same GC owner and manifest contract.

Live definition:

```text
artifact/ref is live iff:
  a raw_output_artifact_created event exists
  no later raw_output_artifact_tombstoned event applies to the ref
  no later raw_output_artifact_gc_collected event applies to the ref
  no later raw_output_artifact_quarantined event makes the ref unusable
  the ref is still reachable from active session history or compacted provider-facing refs
```

Delete eligibility:

```text
object can be deleted iff:
  no live ref points to its artifact_id within the session
```

GC request direction:

```go
type GCRequest struct {
    SessionID string
    LiveRefs  []RawOutputRef
    DryRun    bool
}
```

GC owner contract:

- `providerhistory` provides live refs from provider-facing compact refs and projection reports.
- `agent` / session owner provides live refs from active session history and session lifecycle.
- `review` provides live refs from review prompt artifacts/ledgers once Phase 6-D participates.
- `rawoutputs` reads manifest lifecycle state, compares against caller-provided live refs, and calculates tombstone/collect candidates.
- If multiple live refs point to the same artifact ID, the object remains.
- Dry-run GC returns candidates without deleting objects or appending collection events.
- Real GC appends lifecycle events and deletes only session-local unreferenced objects.
- GC failure fails closed and leaves refs unresolved rather than allowing apply compact.

Quarantine policy:

- Hash mismatch, decrypt failure, path validation failure, manifest/object mismatch, or unsafe metadata appends `raw_output_artifact_quarantined`.
- Quarantined refs are not eligible for provider-facing apply compact.
- Quarantined refs are not rehydrated into active context.
- Quarantined objects may be deleted by GC after the quarantine event is recorded.
- Missing artifact and intentional GC must be distinguishable in reports/status.

GC collection policy:

- Deleting an object appends `raw_output_artifact_gc_collected`.
- Existing created records are not edited during GC.
- Index rebuild must treat tombstoned/quarantined/gc_collected refs as non-live.
- GC failure must fail closed and keep refs unresolved rather than allowing placeholder apply.

Artifact size / quota policy:

- Per-artifact max is `64 MiB`.
- Per-session artifact quota is `1 GiB`.
- Storage I/O should be chunked/streaming.
- Initial chunk size target is `1 MiB`.
- Object layout remains a single content-addressed object file.
- Full-body content hash is calculated over the entire raw body.
- Active context token budget is separate from artifact storage size.
- Cap/quota failure does not create a ref and does not allow apply compact.

Cap/quota kept reasons:

```text
raw_output_artifact_too_large
raw_output_artifact_session_quota_exceeded
```

Active context budget:

- Default active context budget for raw output rehydrate is `4096` tokens.
- Upper active context budget for raw output rehydrate is `8192` tokens.
- Storing a `64 MiB` artifact does not imply injecting the full body into provider requests.
- Rehydrate should select bounded excerpts or relevant excerpts under the active context budget.

Capture timing:

- Preferred: capture non-sensitive data-bearing raw output when the runtime accepts the tool result into history.
- Legacy command output materialization is included in Phase 2.
- Legacy materialization is lazy: materialize artifacts from raw history only when providerhistory reduction finds a candidate old command output without an artifact-backed ref.
- History-backed resolver is used only as artifact materialization input and diagnostics.
- History-backed source alone does not allow provider-facing apply compact.
- Apply compact is allowed only after artifact creation and hash verification succeed.
- MCP / web search / provider-native replay legacy materialization is deferred to Phase 6.
- Provider-facing projection should consume `RawOutputRef`; it should not parse filesystem paths or write raw files directly.

Legacy materialization contract:

- `MaterializeLegacy` requires an exact history/session source identity, not a fuzzy output search.
- Missing or ambiguous legacy source fails closed.
- Legacy materialization starts with command output only.
- The resulting artifact follows the same manifest, hash, quota, encryption, and path-safety rules as normal `Create`.
- Apply eligibility begins only after `MaterializeLegacy` returns a resolvable artifact-backed `RawOutputRef` and providerhistory/agent gates pass.
- History-backed source alone is never a provider-facing placeholder backing.

Legacy fallback fail-closed reasons:

```text
raw_output_legacy_source_missing
raw_output_legacy_source_ambiguous
raw_output_artifact_materialization_failed
```

Rawoutputs error / reason contract:

Use structured reason kinds, not only free-form errors:

```text
raw_output_artifact_too_large
raw_output_artifact_session_quota_exceeded
raw_output_artifact_missing
raw_output_artifact_hash_mismatch
raw_output_artifact_quarantined
raw_output_artifact_tombstoned
raw_output_artifact_gc_collected
raw_output_manifest_corrupt
raw_output_index_corrupt
raw_output_encryption_required
raw_output_decrypt_failed
raw_output_path_invalid
raw_output_ref_invalid
raw_output_legacy_source_missing
raw_output_legacy_source_ambiguous
raw_output_artifact_materialization_failed
```

Reason contract:

- These reasons are suitable for report/status/keep-reason surfaces.
- Reasons must distinguish missing artifact, intentional GC, quarantine, tombstone, hash mismatch, and encryption/decrypt failure.
- A reason that means "body not safely resolvable" forbids apply compact and active context rehydrate.

Security contract:

- Object paths are derived only from validated session root and content hash.
- Raw command, URL, args, provider output, and user-provided labels never influence object paths.
- Session IDs and artifact IDs are parsed through constrained validators.
- Symlinks are not followed.
- Path traversal is rejected.
- Temp files are created under the target session/object directory and committed by atomic rename.
- Encrypted history mode must not leave artifact body, manifest, or index plaintext.
- Sensitive output is not written to the normal raw output artifact store.

### Safety gates

- Ref must include content hash.
- Ref must include source tool call identity.
- Ref must include artifact ID when artifact-backed.
- Ref must be defensive-copied in report clone.
- Ref / manifest must not carry raw body in report.
- Artifact is not created for sensitive output unless a future dedicated sensitive policy explicitly permits encrypted non-emitting storage.
- Missing artifact does not produce a provider-facing placeholder in apply mode.
- History-backed source alone does not produce a provider-facing placeholder in apply mode.
- Hash mismatch makes the ref unusable for apply/re-hydrate.

### Behavior / output format

No provider-facing compact yet.

Dry-run report may show:

- raw output artifact/ref candidate count
- estimated data-bearing savings
- keep/apply blocker:
  - `raw_output_artifact_missing`
  - `raw_output_artifact_too_large`
  - `raw_output_artifact_session_quota_exceeded`
  - `raw_output_legacy_source_missing`
  - `raw_output_legacy_source_ambiguous`
  - `raw_output_artifact_materialization_failed`
  - `raw_output_rehydrate_not_available`

### Report / status

Candidate/report fields to consider:

- `RawOutputRefs []RawOutputRef`
- `RawOutputRefCount`
- `RawOutputArtifactCount`
- `DataBearingCandidateCount`
- `ArtifactBackedEstimatedSavedBytes`
- `ApproxArtifactBackedEstimatedSavedTokens`
- `ArtifactBackedKeptReasonCounts`

Raw ref metadata placement:

- `ProjectionReport.RawOutputRefs` is the source of truth for full `RawOutputRef` metadata.
- `CommandEditDryRunCandidate` stores only `RawOutputRefID`.
- `CommandEditDryRunCandidate` may store gate booleans / reasons such as `RawOutputArtifactRequired`, `ArtifactBackedEligible`, or kept reason fields.
- Candidate-level structs must not duplicate full raw ref metadata.
- Report-level raw refs are clone-safe and never include raw body.

Invariant:

```text
if candidate.RawOutputRefID != "":
  ProjectionReport.RawOutputRefs contains exactly one matching ref_id
```

Allowed asymmetry:

- Report-level refs may exist without a replacement candidate when a ref was discovered but threshold / age / latest-suffix / safety gates kept the payload raw.
- Candidate-level `RawOutputRefID` without report-level metadata is invalid and must fail closed.

Apply lookup:

```text
candidate.RawOutputRefID
  -> ProjectionReport.RawOutputRefs[ref_id]
  -> rawoutputs.Resolve(ref)
  -> hash verify
  -> placeholder apply allowed
```

Active context lookup:

```text
applied placeholder raw_output_ref
  -> ProjectionReport.RawOutputRefs[ref_id]
  -> rawoutputs.Resolve(ref)
  -> bounded excerpt / selected excerpt injection
```

### Tests

- `rawoutputs` writes artifact atomically and resolves by ref.
- `rawoutputs` verifies content hash on resolve.
- `rawoutputs.Resolve` takes full `RawOutputRef` and rejects wrong session/surface/artifact/hash combinations.
- `rawoutputs.Verify` validates lifecycle/hash without exposing body.
- `rawoutputs` rejects unsafe artifact paths / invalid IDs.
- `rawoutputs` does not store sensitive command output under normal policy.
- `rawoutputs` appends versioned manifest records without raw body.
- `rawoutputs` stores command hash and redacted command preview, not raw command text.
- `rawoutputs` rebuilds index from manifest.
- encrypted history mode does not leave manifest/index payloads as plaintext.
- tombstone/quarantine/GC lifecycle records append new events instead of mutating created records.
- live-ref mark-and-sweep keeps live artifacts and deletes only unreachable session-local objects.
- GC uses caller-provided live refs and does not infer providerhistory/review liveness internally.
- GC keeps a shared object when any live ref points to the same artifact ID.
- dry-run GC reports candidates without deleting objects or appending collection events.
- cross-session dedupe is not performed.
- session rewrite/truncate/reset tombstones unreachable refs.
- quarantined refs are not resolved for apply compact or active context.
- GC appends `raw_output_artifact_gc_collected` after deleting an object.
- startup opportunistic cleanup does not delete live refs.
- per-artifact `64 MiB` cap rejects oversized output without creating a ref.
- per-session `1 GiB` quota rejects new artifacts without compacting provider-facing output.
- chunked/streaming write avoids requiring the full raw body in memory when possible.
- active context budget defaults to `4096` tokens and never exceeds `8192` tokens for raw output rehydrate.
- legacy command output without artifact ref lazily materializes from raw history.
- history-backed source alone does not allow apply compact.
- lazy materialization failure keeps raw and reports `raw_output_artifact_materialization_failed`.
- missing legacy source keeps raw and reports `raw_output_legacy_source_missing`.
- ambiguous legacy source keeps raw and reports `raw_output_legacy_source_ambiguous`.
- MCP / web search / provider-native replay legacy materialization is not included before Phase 6.
- `CloneProjectionReport` copies raw output ref metadata defensively.
- candidate with `RawOutputRefID` but missing report-level ref fails closed.
- report-level raw refs may exist without applied replacement.
- candidate does not duplicate full raw ref metadata.
- apply lookup resolves candidate ref through report-level refs and rawoutputs resolver.
- dry-run detects `curl` / `sqlite3` raw output ref candidates without changing payload.
- raw ref / manifest metadata does not include raw body.
- structured rawoutputs reasons distinguish too-large, quota, missing, hash mismatch, quarantine, tombstone, GC-collected, manifest corrupt, index corrupt, encryption required, decrypt failed, invalid path/ref, and legacy missing/ambiguous.
- encrypted-history mode does not leave artifact body, manifest, or index plaintext.
- sensitive command output does not expose raw ref body or raw content in report/status.
- session reload can resolve a stored artifact ref if metadata persistence is in scope.

### Implementation owner candidates

- `internal/rawoutputs`
- `internal/providerhistory/reduction.go`
- `internal/providerhistory/command_edit_dry_run.go`
- `internal/providerhistory/projection.go`
- `internal/providerhistory/projection_test.go`
- `internal/agent` capture wiring if artifact materialization happens at tool-result acceptance

### Decision status

Phase 2 artifact store / raw ref placement is decided. Policy and rollout decisions are captured in `Adopted Decisions`.

## 7.4 Phase 3: Artifact-backed Dry-run

### Purpose

data-bearing command output を artifact-backed compact した場合の provider-facing replacement と savings を dry-run で可視化する。
apply mode ではまだ keep raw にする。

### Non-goals

- Apply compact is not enabled for data-bearing output.
- Active context injection is not required.

### Design contract

Providerhistory role:

- `providerhistory` is projection policy + gate engine + report owner.
- `providerhistory` consumes `commandoutputs.Decide` and `rawoutputs` refs.
- `providerhistory` owns artifact-backed candidate creation, raw ref/report linkage, placeholder finalization, dry-run/apply savings accounting, and Responses chain disable decisions.
- `providerhistory` does not write artifacts.
- `providerhistory` does not assemble request-local active context body.

Dry-run may generate suggested replacement text:

```text
[compacted old data-bearing command output;
 command_preview="curl https://api.example.test/items";
 family=network;
 classifier=network_response;
 raw_output_ref=rawout_c3f8a19b2d;
 bytes=48231;
 lines=812]
excerpt:
  <first lines>
  ...
  <last lines>
```

The replacement is a candidate only.

Artifact-backed candidate shape direction:

```go
type ArtifactBackedCandidate struct {
    RawOutputRefID       string
    DecisionAction       string
    SemanticRole         string
    Family               string
    Subfamily            string
    Classifier           string
    ArtifactGateStatus   string
    RehydrateGateStatus  string
    ThresholdStatus      string
    FreshnessStatus      string
    SafetyStatus         string
    ApplyEligible        bool
    FailClosedReason     string
    EstimatedSavedBytes  int
    EstimatedSavedTokens int
    ActualSavedBytes     int
    ActualSavedTokens    int
}
```

Candidate metadata contract:

- Candidate stores `RawOutputRefID` plus gate/status/reason fields only.
- Full ref metadata lives in `ProjectionReport.RawOutputRefs`.
- Candidate with missing or duplicate report-level raw ref metadata fails closed.
- Candidate may be dry-run visible even when not apply eligible.
- Actual savings are zero unless provider-facing payload changed.

Dry-run gate ledger:

```text
1. decision gate
2. raw ref gate
3. artifact verify gate
4. rehydrate transport gate
5. freshness gate
6. safety gate
7. threshold gate
8. placeholder size gate
9. mode gate
```

Dry-run meaning:

- Dry-run evaluates all gates and reports whether the candidate would be apply-eligible.
- Dry-run never mutates provider-facing payload.
- Dry-run never disables Responses `previous_response_id` chain.
- Dry-run estimated savings must be separated from actual apply savings.

### Safety gates

- Suggested replacement requires a resolvable raw output ref and artifact.
- If ref/artifact cannot be created, dry-run reports keep reason.
- Dry-run must not mutate provider-facing history.
- Suggested replacement must include bounded excerpt, not full raw output.
- Excerpt must not leak secrets beyond existing provider-facing raw output policy. If sensitive markers are detected, do not build artifact-backed candidate.
- Final data-bearing placeholder is built by `providerhistory`, not `commandoutputs`, and only after a report-level raw ref exists.
- Placeholder must not contain local artifact path, manifest path, raw URL, or unredacted raw command.

### Report / status

Dry-run should distinguish:

- inline command replacement candidates
- artifact-backed command replacement candidates
- artifact-backed apply-eligible candidates
- threshold-skipped candidates
- data-bearing kept because rehydrate/apply is unavailable
- fail-closed candidates by reason
- estimated savings and actual savings as separate counters
- Responses chain unchanged because dry-run did not mutate payload

Threshold policy:

- Minimum saved tokens for artifact-backed data-bearing apply is `2048`.
- Maximum replacement ratio is `0.75`.
- Replacement ratio is `replacement_tokens / original_tokens`.
- Both absolute savings and ratio gates must pass for apply.
- Dry-run may report candidates that fail threshold, but must label them as not apply-eligible.

Threshold kept reasons:

```text
raw_output_artifact_saved_tokens_below_threshold
raw_output_artifact_replacement_ratio_too_high
```

### Tests

- dry-run for large `curl` output reports artifact-backed suggested replacement and estimated savings.
- dry-run reports threshold-skipped candidates separately from apply-eligible candidates.
- apply for same input still keeps raw until Phase 5.
- status/report counts do not count dry-run estimate as actual apply savings.
- dry-run candidate includes `RawOutputRefID` but does not duplicate full raw ref metadata.
- dry-run candidate with missing report-level ref fails closed.
- dry-run fail-closed does not disable Responses chain.
- dry-run placeholder contains `raw_output_ref` and bounded excerpt but no local path/raw URL/unredacted raw command.

### Implementation owner candidates

- `internal/commandoutputs`
- `internal/providerhistory`
- `internal/agent/provider_history_reduction_status.go`

### Decision status

Excerpt policy and minimum saved token threshold are decided. Apply rollout is defined in Phase 5.

## 7.5 Phase 4: Active Context Rehydrate

### Purpose

artifact-backed provider-facing placeholder が出た後、agent が request-local active context として raw output / selected excerpt を provider request 前に戻せるようにする。

### Non-goals

- Raw output is not written back into history.
- Raw output is not persisted as active context.
- Model does not fetch raw artifacts directly.

### Design contract

Agent owns request-local rehydrate execution.

Agent rehydrate role:

- `agent` is the request-local rehydrate transport owner.
- `agent` builds raw output active context immediately before provider request assembly.
- `agent` reads applied raw output refs from providerhistory projection/report state.
- `agent` resolves raw bodies through `rawoutputs`.
- `agent` selects excerpts using current user request / latest context relevance.
- `agent` never mutates raw history, session messages, audit records, or saved active context.
- `agent` does not decide command classification or providerhistory apply eligibility.

Rehydrate plan sources:

```text
providerhistory applied replacements
  -> applied placeholder raw_output_ref values
  -> ProjectionReport.RawOutputRefs full metadata
  -> current user request / latest context
  -> relevance scoring
  -> rawoutputs.Resolve(ref)
  -> excerpt selection
  -> request-local Provider History Raw Output Context
```

Source contract:

- Prefer refs that were actually applied in provider-facing history.
- Do not rehydrate unrelated discovered refs just because report metadata exists.
- Current user request and latest conversation context decide relevance at request time.
- `ProjectionReport.RawOutputRefs` supplies metadata; `rawoutputs` supplies verified body streams.
- If the applied placeholder ref cannot be matched to report metadata, the ref is unresolved and must be reported as fail-closed.

Possible active context section:

```text
Provider History Raw Output Context
- ref: rawout_c3f8a19b2d
  command_preview: curl https://api.example.test/items
  family: network
  bytes: 48231
  content_hash: sha256:<hash>
  body:
    <bounded raw body or selected excerpt>
```

Rehydrate relevance rule:

- Use deterministic priority scoring.
- Start with bounded excerpt rehydrate under the raw output active context budget.
- Full raw body rehydrate only when under budget or explicitly selected by deterministic relevance rules.
- Do not rehydrate sensitive output automatically.
- File evidence active context and raw output active context remain separate sections.

Priority order:

```text
0. explicit raw_output_ref / user explicit reference
1. latest/current-turn adjacency
2. lexical match with current user request
3. same command family / tool name / task surface
4. recent applied artifact-backed refs
5. metadata-only fallback
```

Priority details:

- Explicit reference includes a literal `raw_output_ref`, direct mention of the command result, or user wording like "さっきの curl 結果" / "sqlite3 の結果".
- Latest/current-turn adjacency covers compacted refs that are adjacent to the latest unresolved user/assistant/tool context even when lexical terms are weak.
- Lexical match compares the current user request against ref metadata, command preview, tool name, family, classifier, and already-available safe excerpts.
- Same family/tool/task surface is a tie-breaker, not enough by itself to pull large body excerpts when budget is tight.
- Recent applied refs outrank older applied refs.
- Low relevance refs are represented as metadata-only entries rather than body excerpts.

Required refs:

- Applied data-bearing refs in the current provider-facing projection.
- Refs explicitly named by the user or by literal `raw_output_ref`.
- Latest/current-turn adjacent applied refs.

Optional refs:

- Older applied refs with weak relevance.
- Same-family-only refs without direct current request match.
- Background refs included only for orientation.

Required/optional contract:

- Required refs must resolve and receive a body excerpt unless the original body is smaller than the metadata-only representation.
- Required refs must not degrade to metadata-only solely because of budget exhaustion.
- Optional refs may degrade to metadata-only with a structured skipped reason.
- Sensitive refs are never body rehydrated automatically, even if referenced.
- A required ref failure is a correctness signal and must be visible in report/status.

Budget allocation:

```text
default total budget: 4096 tokens
upper total budget:   8192 tokens
metadata reserve:     512 tokens or 15%, whichever is larger within budget
body excerpt budget:  remaining budget allocated by priority order
reserve:              keep a small remainder for section framing / skipped reasons
```

Metadata-only fallback includes:

```text
ref_id
command_preview
family
classifier
byte_size
content_hash
skipped_reason
```

Fail-closed relevance behavior:

- No hash verification means no body excerpt and required ref failure when the ref is required.
- Sensitive refs get no automatic body excerpt.
- Budget exhaustion yields metadata-only plus skipped reason only for optional refs.
- Required ref budget exhaustion is reported as `raw_output_active_context_required_ref_failed`.
- Relevance miss yields metadata-only when the ref is optional and was applied in provider-facing history, otherwise no active context entry is required.
- Request-time artifact missing/hash mismatch/decrypt failure for a required ref is recorded as fail-closed and no body is injected.

Excerpt policy:

- Use classifier-specific structured excerpt first.
- Use query/keyword matched excerpt second.
- Use deterministic first/last fallback third.
- Parser failure falls back to deterministic first/last excerpt.
- Universal metadata is always included before body excerpts.
- Dry-run placeholder excerpts and active-context excerpts must use the same excerpt policy owner.

Universal metadata:

```text
ref_id
surface
tool_name
command_preview
family
classifier
byte_size
content_hash
excerpt_policy
skipped_or_truncated_reason
```

Network / HTTP excerpts:

- include status / headers if present and safe
- include JSON top-level keys / shape summary when parseable
- include `error`, `message`, `status`, and similar fields when present
- include first objects / first array items
- include matched fields when current user request terms match
- include tail excerpt for log/error-like responses

Database excerpts:

- include column headers / schema-like header when present
- include first rows
- include matched rows when current user request terms match
- include row count if available or cheaply inferable
- include last rows when budget remains

Log excerpts:

- include failure/success summary when classified as log-like
- include `error` / `fatal` / `non-zero` nearby context
- include tail excerpt
- validation logs remain governed by validation classifier precedence, not data-bearing artifact policy alone

Web search / provider-native replay excerpts:

- Phase 6 only.
- include title / redacted URL / snippet / top-ranked results / matched excerpts
- provider-native replay and XELYON web search evidence remain separate surfaces

Budget use:

```text
metadata:           about 512 tokens
structured summary: 512-1024 tokens when available
matched excerpts:   1024-2048 tokens when relevant
first/last fallback: remaining body excerpt budget
```

### Safety gates

- Rehydrate result must validate content hash if possible.
- Rehydrate must fail closed: if raw body not found or hash mismatch, keep raw in provider-facing projection or omit artifact-backed apply.
- Rehydrate must be request-local.
- Rehydrate budget must be bounded.
- Rehydrate must not mutate history/session.
- Rehydrate ordering must be deterministic for the same request/report/ref set.
- Excerpt extraction must not leak sensitive values beyond the existing provider-facing raw output policy.
- Disabled `provider_history_reduction.rehydrate_context` makes artifact-backed apply ineligible.
- Apply projection must not rely on a rehydrate transport that is absent at request assembly time.
- Request-time required ref failures must be recorded so the next projection can keep raw or fail closed instead of repeatedly emitting unresolved placeholders.

### Behavior / output format

Active context should be a distinct provider prompt section, not hidden in the placeholder.
It must not be merged into file evidence active context because file evidence means current filesystem state, while raw output context means past provider-facing tool result body.
The section is injected only into the current provider request and is not persisted.

### Report / status

Status should distinguish:

- raw output ref / artifact candidates
- applied artifact-backed replacements
- active context raw output refs injected
- active context budget skipped refs
- active context metadata-only refs
- active context relevance skipped refs
- missing / stale / hash mismatch refs
- required refs failed
- sensitive refs skipped
- request-local injection count
- active context not persisted

Status / reason names:

```text
raw_output_active_context_injected
raw_output_active_context_metadata_only
raw_output_active_context_relevance_skipped
raw_output_active_context_budget_exhausted
raw_output_active_context_hash_mismatch
raw_output_active_context_missing_artifact
raw_output_active_context_decrypt_failed
raw_output_active_context_sensitive_skipped
raw_output_active_context_required_ref_failed
raw_output_active_context_not_persisted
```

### Tests

- active context injection includes bounded raw output for an applied ref.
- injection does not mutate raw history/session.
- injection does not persist request-local context to session/audit/history.
- hash mismatch skips or reports structured failure.
- budget cap prevents oversized raw output from flooding provider prompt.
- disabled `provider_history_reduction.rehydrate_context` keeps data-bearing raw in apply mode.
- explicit raw_output_ref gets highest priority.
- latest/current-turn adjacency beats older lexical-light refs.
- lexical match beats same-family-only refs.
- budget exhaustion emits metadata-only skipped refs for optional refs.
- required ref missing/hash mismatch/budget exhaustion is reported as required ref failure.
- applied data-bearing ref is treated as required for the request-local rehydrate plan.
- sensitive ref is not body rehydrated automatically.
- relevance miss does not inject body excerpts.
- network excerpt includes safe HTTP/status metadata, JSON shape, first items, matched fields, and tail fallback as available.
- database excerpt includes columns/header, first rows, matched rows, row count when available, and last rows as budget allows.
- parser failure falls back to deterministic first/last excerpt.
- dry-run placeholder and active context use the same excerpt policy owner.
- ordering is deterministic for stable inputs.

### Implementation owner candidates

- `internal/agent/provider_history_projection.go`
- `internal/agent/provider_history_rehydrate_plan.go`
- provider request assembly path in `internal/agent`
- `internal/providerhistory/rehydrate_plan.go` if pure planning helpers fit there

### Decision status

Relevance rule, active context budget, and excerpt extraction policy are decided.

## 7.6 Phase 5: Artifact-backed Apply

### Purpose

data-bearing command output を apply mode で compact 可能にする。
ただし artifact-backed raw output ref と active context rehydrate transport が成立する場合だけ許可する。

### Non-goals

- Sensitive output compact is not enabled.
- Unknown data-bearing output compact is not enabled unless classified and artifact-ref-backed.

### Design contract

Providerhistory apply role:

- `providerhistory` is the artifact-backed apply gate owner.
- `providerhistory` builds the final provider-facing placeholder only after all gates pass.
- `providerhistory` records the gate ledger and actual savings.
- `providerhistory` disables Responses `previous_response_id` chain only when provider-facing payload is actually replaced.

Apply eligibility:

```text
data-bearing command output apply compact is allowed iff:
  provider_history_reduction effective mode is apply
  provider_history_reduction.raw_output_artifacts.mode is apply
  provider_history_reduction.rehydrate_context is true
  raw output ref can be created
  raw output artifact store can resolve and hash-verify the ref
  active context transport is enabled and available
  replacement is smaller than original
  saved tokens >= 2048
  replacement ratio <= 0.75
  output is not sensitive
  candidate is old enough and not latest/no-later-assistant protected
```

If any gate fails, keep raw.

Exact apply gate order:

```text
1. Decision gate
2. Raw ref gate
3. Artifact gate
4. Rehydrate transport gate
5. Freshness gate
6. Safety gate
7. Threshold gate
8. Placeholder size gate
9. Apply mode gate
```

Gate contract:

- Decision gate: `commandoutputs.Decide` must return `artifact_backed_candidate` with `data_bearing` semantic role.
- Raw ref gate: `RawOutputRefID` must have exactly one matching `ProjectionReport.RawOutputRefs` entry.
- Artifact gate: `rawoutputs.Resolve` or `rawoutputs.Verify` must pass lifecycle and hash verification.
- Rehydrate transport gate: request-local active context transport must be enabled and available before apply.
- Freshness gate: latest suffix and no-later-assistant protected tool results are raw keep.
- Safety gate: sensitive/private/secret and disallowed side-effect surfaces are raw keep.
- Threshold gate: saved tokens must be at least `2048` and replacement ratio must be `<= 0.75`.
- Placeholder size gate: final placeholder must be smaller than original provider-facing payload.
- Apply mode gate: parent `provider_history_reduction.mode` and child `provider_history_reduction.raw_output_artifacts.mode` must both resolve to apply.

Fail-closed reasons:

```text
raw_output_decision_not_artifact_backed
raw_output_ref_missing
raw_output_ref_report_metadata_missing
raw_output_ref_report_metadata_duplicate
raw_output_artifact_missing
raw_output_hash_mismatch
raw_output_artifact_quarantined
raw_output_artifact_tombstoned
raw_output_artifact_gc_collected
raw_output_decrypt_failed
raw_output_rehydrate_unsupported
raw_output_latest_or_no_later_assistant_protected
raw_output_sensitive_or_private
raw_output_artifact_saved_tokens_below_threshold
raw_output_artifact_replacement_ratio_too_high
raw_output_placeholder_not_smaller
raw_output_parent_apply_mode_disabled
raw_output_artifacts_apply_mode_disabled
```

### Safety gates

- Apply replacement includes `raw_output_ref`.
- Report records raw ref metadata.
- Responses chain disabled only if replacement actually applied.
- Apply and dry-run estimated savings do not drift.
- Missing rehydrate transport keeps raw, not placeholder.
- Parent `provider_history_reduction.mode=dry_run` reports candidates only and never mutates provider-facing payload.
- Parent `provider_history_reduction.mode=apply` without `raw_output_artifacts.mode=apply` still keeps data-bearing output raw and may report dry-run candidate savings only.
- `raw_output_artifacts.mode=apply` still requires parent apply mode plus artifact/rehydrate/threshold/freshness/safety gates.
- Small or low-ratio savings keep raw even in apply mode.
- Candidate discovered but not applied never disables Responses chain.
- Fail-closed raw keep never disables Responses chain.
- Placeholder is finalized by `providerhistory` and must be linked to report-level raw ref metadata.

### Behavior / output format

Provider-facing tool result content becomes artifact-backed compact text.
Active context may include bounded raw context in the same request.

### Report / status

Add or update:

- `ArtifactBackedCommandCandidates`
- `ArtifactBackedCommandReplacedCount`
- `ArtifactBackedCommandReplacementSavedBytes`
- `ApproxArtifactBackedCommandReplacementSavedTokens`
- `RawOutputRefCount`
- `RawOutputArtifactCount`
- `RawOutputRehydrateInjectedCount`
- `ArtifactBackedCommandDryRunEstimatedSavedBytes`
- `ApproxArtifactBackedCommandDryRunEstimatedSavedTokens`
- kept reasons:
  - `raw_output_ref_missing`
  - `raw_output_ref_report_metadata_missing`
  - `raw_output_ref_report_metadata_duplicate`
  - `raw_output_artifact_missing`
  - `raw_output_rehydrate_unsupported`
  - `raw_output_hash_mismatch`
  - `raw_output_active_context_budget_exhausted`
  - `raw_output_artifact_saved_tokens_below_threshold`
  - `raw_output_artifact_replacement_ratio_too_high`

### Tests

- parent apply + `raw_output_artifacts.mode=apply` + rehydrate enabled compacts `curl` output and injects raw output context.
- apply with rehydrate disabled keeps raw.
- parent reduction dry-run keeps raw and reports candidate.
- parent reduction off does not produce artifact-backed replacement candidates.
- parent apply + `raw_output_artifacts.mode=dry_run` keeps raw and reports candidate only.
- parent dry-run + `raw_output_artifacts.mode=apply` keeps raw and reports dry-run candidate only.
- `raw_output_artifacts.mode=off` does not produce raw output artifact-backed replacement candidates.
- apply with missing ref/artifact keeps raw.
- apply with duplicate report-level raw ref metadata keeps raw.
- apply with saved tokens below `2048` keeps raw.
- apply with replacement ratio above `0.75` keeps raw.
- apply with placeholder not smaller keeps raw.
- apply with latest/no-later-assistant protected output keeps raw.
- apply with missing rehydrate transport keeps raw.
- apply with hash mismatch/quarantine/tombstone/GC-collected/decrypt failure keeps raw.
- apply replacement disables Responses chain.
- dry-run and fail-closed raw keep do not disable Responses chain.
- dry-run reports candidate but does not change payload.
- status/report saved bytes/tokens match actual projected content.
- report separates dry-run estimated savings from actual apply savings.
- final placeholder contains `raw_output_ref` and no local artifact path/raw URL/unredacted raw command.

### Implementation owner candidates

- `internal/providerhistory`
- `internal/agent`
- `internal/config`

### Decision status

Apply rollout and thresholds are decided.

## 7.7 Phase 6: MCP / Web Search / Provider-native Replay Extension

### Purpose

command output で確立した raw ref + artifact store + rehydrate contract を、他の data-bearing provider-facing surfaces へ横展開する。

Adopted order:

```text
Phase 6-A: MCP tool results
Phase 6-B: XELYON web_search tool results / evidence
Phase 6-C: provider-native built-in replay
Phase 6-D: review probe results / review prompt surfaces
```

Phase 6-A is the first target after command output.

### Non-goals

- それぞれの source semantics を無視して generic helper に混ぜない。
- `/review` debug artifact を providerhistory runtime artifact store として流用しない。
- provider-native built-in replay を XELYON `web_search` tool result と同じ surface として扱わない。
- MCP server output を trusted data として扱わない。

### Design contract

各 surface は semantic role と raw source owner を明確化してから参加させる。

Generic contract:

```text
data-bearing surface can be artifact-backed compacted iff:
  raw source is preserved in artifact-backed storage or a documented fallback
  raw ref is stable
  rehydrate transport can inject request-local context
  sensitive sink guard passes
  replacement and active context are budgeted
```

Surface separation:

```text
MCP tool result != XELYON web_search tool result
XELYON web_search tool result != provider-native built-in replay
providerhistory runtime raw artifact != /review debug artifact
review probe result != provider-facing runtime tool result
```

Each surface must define:

- source owner
- stable identity
- raw preservation point
- raw output ref metadata
- sensitive / private guard
- artifact materialization trigger
- dry-run candidate format
- apply eligibility
- active context excerpt policy
- missing artifact / missing transport behavior
- status/report fields
- tests

### Phase 6-A: MCP Tool Results

Source findings:

- MCP provider-facing tool results are currently identifiable by tool names prefixed with `mcp_`.
- `internal/providerhistory/candidate_only.go` currently treats MCP results as candidate-only and keeps unknown/sensitive MCP content.
- MCP result shape varies by server/tool and must not be treated as trusted structured JSON by default.

Owner candidates:

- `internal/providerhistory/candidate_only.go`
- `internal/providerhistory/toolresults`
- `internal/rawoutputs`
- `internal/agent/provider_history_rehydrate_plan.go`
- `internal/mcp` only for identity/source metadata if needed

Contract:

- Surface: `mcp_tool_result`.
- Stable identity includes tool name, tool call ID, MCP server name if available, MCP method/tool name, session ID, history index, and content hash.
- Default semantic role is `unknown`.
- Known non-sensitive structured MCP results may become `data_bearing`.
- Sensitive/private-looking MCP results are raw keep and no normal artifact-backed apply.
- Unknown MCP schema is dry-run only or raw keep; no artifact-backed apply.
- Apply is allowed only after a known MCP result classifier marks the result as data-bearing and all artifact/rehydrate/threshold gates pass.
- Excerpt policy starts with universal metadata + JSON shape/keys + query matched fields + first/last fallback.

MCP classifier strategy:

```text
1. sensitive / private guard
2. known result schema classifier
3. JSON shape / text shape evidence
4. unknown MCP schema
```

MCP kept reasons:

```text
mcp_sensitive_or_private_result_keep
mcp_unknown_schema_keep
mcp_raw_output_artifact_missing
mcp_raw_output_rehydrate_not_available
```

MCP tests:

- known safe data-bearing MCP result dry-runs artifact-backed candidate.
- known safe data-bearing MCP result apply compacts only with artifact/ref/rehydrate/threshold gates.
- sensitive/private-looking MCP result is raw keep and creates no normal artifact-backed apply.
- unknown MCP schema is raw keep or dry-run only.
- missing artifact keeps raw.
- active context injects bounded excerpt for applied MCP ref.
- report/status distinguishes MCP artifact candidates from command output candidates.

### Phase 6-B: XELYON Web Search Tool Results / Evidence

Source findings:

- `internal/providerhistory/toolresults/web_search.go` already has a web_search replacement path.
- Existing providerhistory web_search compaction is not the same surface as provider-native built-in replay.
- Review web search evidence exists under review/evidence surfaces and is not the same as runtime providerhistory `web_search` tool result.

Owner candidates:

- `internal/providerhistory/toolresults/web_search.go`
- `internal/providerhistory/generic_tool_results_test.go`
- `internal/rawoutputs`
- `internal/agent/provider_history_rehydrate_plan.go`
- `internal/review/evidence` only for review-specific Phase 6-D, not for runtime web_search owner

Contract:

- Surface: `xelyon_web_search_tool_result`.
- Stable identity includes tool call ID, query hash, redacted query preview, selected URL hashes, result count, session ID, history index, and content hash.
- URL values in metadata/excerpts must be redacted for secret-bearing query/fragment values.
- Temporal/current web search results remain conservative keep unless the existing currentness/citation gates allow compact.
- Referenced/cited search results must not be artifact-compacted away if the citation is the active evidence.
- Apply requires artifact-backed ref and active context excerpt plan.

Web search excerpt policy:

- universal metadata
- query preview
- top result titles
- redacted URLs
- snippets
- selected matched excerpts
- result count/source count

Web search kept reasons:

```text
web_search_temporal_or_current_keep
web_search_unknown_credibility_keep
web_search_citation_or_referenced_result_keep
web_search_raw_output_artifact_missing
web_search_raw_output_rehydrate_not_available
```

Web search tests:

- duplicate/old safe web_search result dry-runs artifact-backed candidate.
- apply compacts old safe web_search result only when artifact/ref/rehydrate/threshold gates pass.
- current/temporal/cited web_search result remains raw keep.
- URL query/fragment secret redaction is applied in metadata/excerpts/status.
- active context injects title/redacted URL/snippet/matched excerpt under budget.
- provider-native replay fixtures are not classified as XELYON web_search tool results.

Security preflight:

- Use `security-boundary-change` before implementing URL/external-source redaction changes.
- Redaction owner must be shared by placeholder, active context, report/status, and tests.

### Phase 6-C: Provider-native Built-in Replay

Source findings:

- Kimi `$web_search` can appear as provider-native built-in replay in provider-facing history.
- `internal/providerhistory/candidate_only.go` already identifies `$web_search`, `builtin_web_search`, `provider_native_web_search`, and `provider_native_builtin_replay` as provider-native replay candidates.
- Provider-native replay is not a normal XELYON tool result.

Owner candidates:

- `internal/providerhistory/candidate_only.go`
- provider-specific request/replay handling after source-owner audit:
  - `internal/api/providers/kimi`
  - `internal/api/providers/openai`
  - `internal/api/providers/claude`
- `internal/rawoutputs`
- `internal/agent/provider_history_rehydrate_plan.go`

Contract:

- Surface: `provider_native_builtin_replay`.
- Stable identity must preserve provider, model, provider route, built-in tool name, call ID if available, session ID, history index, and content hash.
- Provider-native replay keeps provider-specific request/replay semantics; do not normalize it into XELYON `web_search`.
- Apply is disabled until source-owner audit proves raw replay can round-trip without losing provider-specific fields.
- Dry-run/candidate-only can report savings and blockers before apply.
- Provider-native usage/cost metadata is not treated as data-bearing result body.

Provider-native replay gates:

```text
provider_native_replay_source_owner_verified
provider_native_replay_round_trip_preserved
provider_native_replay_raw_artifact_available
provider_native_replay_rehydrate_available
```

Provider-native tests:

- Kimi `$web_search` replay is classified separately from XELYON `web_search`.
- provider-native replay dry-run candidate does not mutate payload.
- apply remains disabled until round-trip/source-owner tests exist.
- provider-specific fields are not dropped in save/load/request rebuild.
- usage/cost metadata is preserved and not double-counted as search evidence.

Provider-runtime preflight:

- Use `xelyon-provider-runtime-change` before enabling apply for provider-native replay.
- Confirm warning path, request path, saved session state, and provider-specific replay shape use the same source of truth.

### Phase 6-D: Review Probe Results / Review Prompt Surfaces

Source findings:

- Review probe result absorption has separate prompt-reduction logic.
- `/review` debug artifacts are not providerhistory runtime raw artifacts.
- Probe command-level absorption has already needed command-index granularity.
- `BuildProbeResultPromptContextsWithOptions` currently owns prompt-facing probe result DTO reduction.
- Saturation / revision prompts consume probe result context separately from the initial report prompt.

Owner candidates:

- `internal/review/prompt_probe_result_absorption.go`
- `internal/review/modelinput`
- `internal/review/probe`
- `internal/review/report`
- `internal/review/artifact` only for debug artifact output, not runtime raw artifact store
- `internal/rawoutputs` as the shared physical artifact store engine, with a review-specific namespace/surface contract

Contract:

- Surface: `review_probe_result`.
- Phase 6-D is a highest-quality dedicated review prompt optimization phase, not a weak optional cleanup.
- Implement after providerhistory runtime artifact/ref/rehydrate contract is stable enough to reuse the physical artifact store contract.
- Use the shared raw artifact store engine, but keep review-specific source ownership, IDs, reports, and prompt injection policy separate from providerhistory runtime.
- Do not use `/review` debug artifacts as the source of truth for runtime or prompt rehydrate.
- Probe-level and probe-command-level refs remain distinct and non-interchangeable.
- Probe-level absorption is allowed only when the whole probe result is safely reflected by checked/dismissed/covered report evidence and no command-level sibling is unsafe or unabsorbed.
- Command-level absorption is allowed only for the exact `{probe_id, command_index}` ref reflected by report or saturation evidence.
- Review prompt saturation/revision must not lose unabsorbed probe commands.
- Optimization must never turn an insufficient review context into a clean/saturated signal.
- Review artifacts remain debug outputs unless a dedicated review raw artifact store contract is introduced.

Highest-quality raw ref design:

```text
surface = review_probe_result
stable identity =
  session_id
  review_run_id
  review_phase
  probe_id
  command_index when command-level
  probe mode/status
  command hash / args hash when command-level
  content hash
```

Review raw artifact namespace:

```text
~/.xelyon/history/rawoutputs/sessions/<session-id>/review_runs/<review-run-id>/
```

This namespace uses the same rawoutputs file permission, manifest, hash verification, quarantine, tombstone, and GC policy as command/providerhistory raw output artifacts, but its source metadata and live refs are owned by review prompt reduction.

Review prompt rehydrate transport:

- Add review-specific active context assembly for saturation / revision / repair prompts.
- Inject a bounded `Review Probe Raw Output Context` section, not generic providerhistory active context.
- Rehydrate explicit refs first, then refs needed by saturation missing surfaces/risks, then latest unabsorbed or high-signal probe commands.
- Keep body budget separate from saved artifact size.
- If artifact resolution fails, keep raw prompt context or mark the reduction item as not applied; do not emit an absorbed placeholder that depends on missing raw evidence.

Review prompt budget:

```text
default total budget:        4096 tokens
upper total budget:          8192 tokens
metadata reserve:            512 tokens or 15%, whichever is larger within budget
required ref body minimum:   512 tokens
optional ref body minimum:   160 tokens
per-command body maximum:    50% of total budget
single explicit ref maximum: 80% of total budget
```

Budget contract:

- Review probe rehydrate budget is separate from providerhistory raw output active context budget, even if the defaults match.
- Metadata reserve is used for probe ID, command index, command/args preview, workdir, status, exit code, duration, truncation state, absorbed_by, raw ref ID, content hash, and artifact status.
- Required refs must receive at least the required body minimum unless the original body is smaller.
- Optional refs may be metadata-only when the budget is exhausted, but required refs may not be metadata-only.
- A single command cannot starve other required refs unless it is the only explicit ref requested by saturation/revision.
- Full raw output is allowed only when it fits inside budget and does not starve required sibling refs.
- Budget exhaustion is a correctness condition, not only an optimization statistic.

Rehydrate priority:

```text
0. refs explicitly named by saturation/revision feedback
1. probe command refs for missing surface IDs
2. probe command refs for missing risk IDs
3. probe command refs for additional finding candidates
4. absorbed probe command refs used by checked/dismissed/covered report evidence
5. latest high-signal unabsorbed command refs
6. metadata-only fallback for optional refs
```

Required ref definition:

- Any ref explicitly named by saturation/revision feedback.
- Any ref needed to explain a missing surface, missing risk, or additional finding candidate.
- Any absorbed ref that is the only evidence supporting a checked/dismissed/covered report item under saturation review.
- Any failed, blocked, timed-out, mutated-worktree, uncertain, or finding-producing probe ref if it participates in saturation/revision. These normally stay raw; if already absorbed by a previous step, they must be rehydrated or fail closed.

Optional ref definition:

- Latest high-signal unabsorbed commands not directly needed by saturation/revision.
- Background passed probe output already reflected by multiple non-absorbed evidence refs.
- Duplicate command output whose exact required sibling is already rehydrated.

Quality-preserving apply policy:

- Apply absorption only for passed, non-mutating, non-sensitive probe results whose evidence was actually reflected by the latest report/saturation state.
- Keep raw or rehydrate full/bounded evidence for failed, blocked, mutated-worktree, uncertain, or finding-producing probes.
- Preserve command metadata even when output is absorbed: command, args, workdir, status, exit code, duration, truncation state, absorbed_by, raw ref ID/hash.
- Saturation prompt must see enough evidence to decide `saturated`, `needs_revision`, or `blocked`; if optimization leaves uncertainty, it must bias toward `blocked`/`needs_revision`, not clean.
- Revision prompt must receive the exact evidence refs named by saturation feedback and must not depend on a report summary alone.

Runner-side fail-closed validator:

- Prompt instructions alone are not sufficient. The review runner must validate the rehydrate ledger before accepting a `saturated` saturation result.
- `saturated` is rejected when any required ref is missing, metadata-only, unresolved, hash-invalid, quarantine/tombstone-blocked, or below the required body minimum.
- `saturated` is rejected when a probe-level absorbed placeholder hides an unabsorbed command-level sibling that is relevant to saturation/revision.
- `saturated` is rejected when failed, blocked, timed-out, mutated-worktree, uncertain, or finding-producing probe context is absent or represented only by a clean absorbed summary.
- `saturated` is rejected when budget exhaustion hides evidence required to decide a missing surface, missing risk, or additional finding candidate.
- Rejected saturation result is converted to the existing repair/revision path or blocked handling according to the review runner contract; it is not silently accepted.
- Revision prompt construction fails closed when a named saturation ref cannot be rehydrated. Do not run a normal revision based only on absorbed summary text.

Required refs ledger:

Each saturation / revision / repair prompt build records a structured ledger:

```text
review_run_id
phase
prompt_kind
budget_tokens
metadata_reserve_tokens
body_budget_tokens
required_refs[]
optional_refs[]
rehydrated_refs[]
metadata_only_refs[]
missing_refs[]
budget_exhausted_refs[]
fail_closed_reason
can_accept_saturated
```

Ledger contract:

- The ledger is the source of truth for runner-side fail-closed validation.
- The ledger is recorded in review prompt reduction report/status.
- The ledger never stores raw output body; it stores ref IDs, hashes, sizes, token allocations, status, and reasons.
- Dry-run ledger is emitted without mutating prompt payload.
- Apply ledger must match the actual prompt payload sent to saturation/revision.
- A `saturated` result is acceptable only when `can_accept_saturated=true`.

Optimization strategy:

1. Dry-run calculates probe-level and command-level candidate savings separately.
2. Artifact-backed refs are created before any prompt-facing absorption can apply.
3. Report/saturation evidence refs decide absorption eligibility; raw output size alone never decides it.
4. Applied absorption replaces only the prompt payload copy, never the stored review probe result.
5. Rehydrate selection restores bounded raw output for explicit refs, missing-surface refs, missing-risk refs, additional-finding candidates, and latest high-signal unabsorbed commands.
6. Runner-side validator checks the required refs ledger before accepting `saturated`.
7. Report/status records prove which refs were absorbed, which refs were rehydrated, which refs stayed raw, and which refs caused fail-closed.

Review-specific kept reasons:

```text
review_probe_failed_or_blocked_keep
review_probe_mutated_worktree_keep
review_probe_unreflected_evidence_keep
review_probe_command_sibling_unabsorbed_keep
review_probe_sensitive_or_private_keep
review_probe_raw_output_artifact_missing
review_probe_raw_output_rehydrate_not_available
review_probe_required_ref_missing
review_probe_required_ref_metadata_only
review_probe_required_ref_body_budget_too_small
review_probe_required_ref_hash_invalid
review_probe_required_ref_quarantined
review_probe_budget_requires_blocked_or_needs_revision
review_probe_saturated_rejected_by_rehydrate_ledger
```

Review tests:

- probe-level ref does not absorb command-level siblings accidentally.
- command-level ref rehydrates only that command output.
- saturation/revision prompt keeps unabsorbed commands.
- debug artifact path is not used as providerhistory raw source.
- failed/blocked/mutated probe results are not absorbed away as clean context.
- passed non-mutating probe command can be absorbed only when exact evidence ref is reflected by report/saturation state.
- saturation prompt with missing artifact does not report saturated solely from absorbed summary.
- revision prompt receives raw/bounded context for evidence refs named by saturation feedback.
- review prompt reduction report distinguishes dry-run candidate, applied absorption, raw keep, rehydrated ref, and missing artifact.
- review raw artifact namespace participates in hash verification and GC without sharing `/review` debug artifact paths.
- required refs ledger rejects `saturated` when a required ref is missing, metadata-only, or below required body minimum.
- budget allocation gives required refs at least `512` tokens before optional refs receive body excerpts.
- single explicit ref can use up to `80%` of budget without starving required siblings.
- revision prompt construction fails closed when a named saturation ref cannot be rehydrated.
- golden prompt tests cover absorbed summary, `Review Probe Raw Output Context`, missing artifact, budget exhaustion, command-level sibling preservation, and rejected saturation.

### Phase 6 Scope Decision

Same Goal implementation scope:

```text
Mandatory if Phase 0-5 are complete:
  Phase 6-A MCP tool results

Include if owner audit stays clean:
  Phase 6-B XELYON web_search tool results / evidence

Candidate-only / dry-run first:
  Phase 6-C provider-native built-in replay

Dedicated highest-quality review phase after runtime contract stabilizes:
  Phase 6-D review probe results / review prompt surfaces
```

Reason:

- MCP is closest to command output and can validate generic surface extension.
- XELYON web_search has existing providerhistory owner and high token value, but needs URL/security redaction care.
- Provider-native replay is high value but provider-runtime-specific and must start candidate-only/dry-run.
- Review probe results are high value and quality-sensitive; they should be optimized with a dedicated review prompt raw-ref/rehydrate contract after the runtime artifact contract is stable, not mixed into providerhistory runtime ownership.

### Safety gates

- MCP result and provider-native replay are different surfaces.
- XELYON web search evidence and provider-native web search replay are different surfaces.
- Review artifact/debug artifact and provider history raw source are different surfaces.
- Phase 6-A/B must not weaken command output apply gates.
- Phase 6-C apply is disabled until provider-runtime preflight and round-trip tests pass.
- Phase 6-D is not allowed to use `/review` debug artifact as runtime or prompt rehydrate raw source.
- Phase 6-D apply must preserve review correctness: insufficient rehydrated context cannot be treated as saturated/clean.

### Tests

Each surface needs:

- raw preservation test
- dry-run candidate test
- apply + rehydrate test
- missing transport keep raw test
- report/status test
- sensitive/redaction negative test
- source-owner round-trip test when provider/runtime state is involved

### Implementation owner candidates

- Phase 6-A:
  - `internal/providerhistory/candidate_only.go`
  - `internal/providerhistory/toolresults`
  - `internal/rawoutputs`
  - `internal/agent/provider_history_rehydrate_plan.go`
- Phase 6-B:
  - `internal/providerhistory/toolresults/web_search.go`
  - `internal/providerhistory/generic_tool_results_test.go`
  - `internal/rawoutputs`
  - URL redaction owner identified during `security-boundary-change`
- Phase 6-C:
  - `internal/providerhistory/candidate_only.go`
  - provider-specific packages after `xelyon-provider-runtime-change`
  - `internal/api/providers/kimi`
  - `internal/api/providers/openai`
  - `internal/api/providers/claude`
- Phase 6-D:
  - `internal/review/prompt_probe_result_absorption.go`
  - `internal/review/modelinput`
  - `internal/review/probe`

### Decision status

Phase 6 ordering is decided. Per-surface source-owner audits may still add implementation-specific stop conditions.

## 8. Mode / Policy / Defaults

Existing modes:

- disabled
- dry-run
- apply
- auto resolves to dry-run currently

Recommended rollout:

1. Phase 0: no mode change.
2. Phase 1-2: introduce optional stable `provider_history_reduction.raw_output_artifacts` config and Raw Output Artifact Store defaults.
3. Phase 3: `raw_output_artifacts.mode=dry_run` reports artifact-backed data-bearing savings while provider-facing payload stays unchanged.
4. Phase 4: raw output active context rehydrate is gated by `provider_history_reduction.rehydrate_context`.
5. Phase 5: apply compact for data-bearing output only when parent `provider_history_reduction.mode=apply`, child `raw_output_artifacts.mode=apply`, `rehydrate_context=true`, and all artifact/rehydrate/safety/threshold gates pass.
6. Default remains conservative: existing users with parent apply mode do not get data-bearing artifact-backed apply compact unless they explicitly set `raw_output_artifacts.mode=apply`.

Mode contract:

```text
provider_history_reduction.mode:
  off     -> provider-facing history reduction disabled
  dry_run -> report candidates and estimated savings; provider-facing payload unchanged
  apply   -> allow provider-facing replacements only when all runtime/artifact gates pass

provider_history_reduction.raw_output_artifacts.mode:
  off     -> no raw output artifact-backed candidate/apply behavior
  dry_run -> report candidates and estimated savings; provider-facing payload unchanged
  apply   -> allow data-bearing artifact-backed apply only when all parent/runtime/artifact gates pass
```

Effective apply condition:

```text
artifact-backed data-bearing apply is allowed iff:
  provider_history_reduction.mode == apply
  provider_history_reduction.raw_output_artifacts.mode == apply
  provider_history_reduction.rehydrate_context == true
  active context transport is available
  raw output artifact/ref exists and hash-verifies
  size/quota/sensitive/latest/no-later-assistant/threshold gates pass
```

Dry-run behavior:

- `raw_output_artifacts.mode=dry_run` may estimate artifact-backed savings even when `rehydrate_context=false`.
- If rehydrate is disabled/unavailable, dry-run must report `raw_output_rehydrate_not_available` and must not count estimated savings as apply savings.
- Parent `provider_history_reduction.mode=off` disables raw output artifact projection work.
- Parent `provider_history_reduction.mode=dry_run` forces raw output artifact behavior to dry-run even if child mode is `apply`.
- `raw_output_artifacts.mode=off` disables raw output artifact-backed candidate and apply behavior while leaving other provider history reduction behavior untouched.

## 9. Config / Docs / Generated Metadata Surface

Adopted config contract:

- Existing `provider_history_reduction.mode` remains the parent provider-facing reduction gate.
- Existing `provider_history_reduction.rehydrate_context` remains the active context transport gate.
- Add optional stable `provider_history_reduction.raw_output_artifacts` config as the data-bearing raw output artifact-backed compact gate and artifact budget policy.
- Default `provider_history_reduction.raw_output_artifacts.mode` is `dry_run`.
- Existing global and project config files remain valid.
- Data-bearing artifact-backed apply compact requires parent `provider_history_reduction.mode=apply`, child `raw_output_artifacts.mode=apply`, `rehydrate_context=true`, and all runtime/artifact gates.

Responsibility separation rationale:

- Parent `provider_history_reduction.mode` controls whether provider-facing history reduction may change the request payload at all.
- Child `raw_output_artifacts.mode` controls only data-bearing raw output artifact-backed candidate/apply behavior.
- `provider_history_reduction.rehydrate_context` controls only request-local active context transport.
- Do not overload parent apply mode to mean data-bearing artifact-backed apply. That would make normal history reduction, evidence preservation, artifact lifecycle, and rehydrate transport share one ambiguous switch.
- This config split is part of the safety contract, not just user-facing ergonomics. It lets classification, projection, artifact storage, and active context transport fail closed independently.

Stable config shape:

```yaml
provider_history_reduction:
  mode: dry_run | apply | off
  rehydrate_context: true
  raw_output_artifacts:
    mode: dry_run | apply | off
    max_artifact_bytes: 67108864
    session_quota_bytes: 1073741824
    chunk_bytes: 1048576
    active_context_budget_tokens: 4096
    active_context_budget_max_tokens: 8192
    retention: session
```

Implementation requirements:

- use `xelyon-config-contract-change`
- update `internal/config`
- run `make gen-all`
- update `docs/config.md`
- update `config.yaml.example`
- add config validation and resolution tests for `raw_output_artifacts`
- confirm saved old config compatibility
- update `/config` registry/generated metadata if this surface is exposed there
- ensure stable project config and global config resolve the same contract
- ensure experimental provider history reduction compatibility does not silently enable artifact-backed apply

Validation contract:

- `provider_history_reduction.mode` accepts only `off`, `dry_run`, `apply`, plus existing documented aliases if the current config contract already supports them.
- `provider_history_reduction.raw_output_artifacts.mode` accepts only `off`, `dry_run`, `apply`.
- `max_artifact_bytes` must be `> 0`.
- `session_quota_bytes` must be `>= max_artifact_bytes`.
- `chunk_bytes` must be `> 0` and `<= max_artifact_bytes`.
- `active_context_budget_tokens` must be `> 0`.
- `active_context_budget_max_tokens` must be `>= active_context_budget_tokens`.
- `retention` accepts `session` only in the first implementation.
- Unknown enum values are config errors.
- Old configs without `raw_output_artifacts` load successfully and resolve to default `raw_output_artifacts.mode=dry_run`.

Dry-run artifact behavior:

- `raw_output_artifacts.mode=dry_run` never changes provider-facing payload.
- Highest-quality dry-run may still create raw output artifacts, verify hashes, resolve refs, and report candidate/apply gates.
- Dry-run savings are estimates only and must not be counted as actual provider-facing savings.
- Public docs must state that dry-run can create raw output artifacts under the same history encryption, permission, retention, and GC policy.
- Users who do not want raw output artifacts created must set `raw_output_artifacts.mode=off`.

Docs:

- This plan is internal docs.
- Public docs must document that raw artifact storage follows history encryption policy.
- Public docs must state that data-bearing artifact-backed apply requires parent `provider_history_reduction.mode=apply`, child `raw_output_artifacts.mode=apply`, `rehydrate_context=true`, and all safety/rehydrate gates.
- Public docs must distinguish provider-facing payload changes from artifact creation. Dry-run means no provider-facing payload mutation, not necessarily no artifact writes.

## 10. Report / Status / Observability

Adopted display/source contract:

```text
ProjectionReport
  -> canonical source of truth for provider history raw output artifact candidates, gate ledger, raw ref metadata, and aggregate counters

/status
  -> user-facing aggregate summary renderer

/ledger
  -> sanitized candidate-level ledger renderer

/tokens
  -> token estimate only; no provider history reduction diagnostics
```

Report must let a reviewer distinguish:

- raw keep because data-bearing and no artifact/rehydrate
- dry-run artifact-backed candidate
- applied artifact-backed replacement
- active context rehydrate injected
- active context rehydrate skipped
- sensitive output protected
- provider-facing savings actually applied
- dry-run savings only estimated

Status/report fields should be deterministic and clone-safe.

`ProjectionReport` candidate-level ledger should include:

```text
candidate_id
surface
source_kind
semantic_role
raw_output_ref
artifact_state
projection_mode
apply_state
kept_reason
estimated_saved_bytes
estimated_saved_tokens
actual_saved_bytes
actual_saved_tokens
rehydrate_state
safety_state
```

Candidate-level ledger rules:

- Candidate ledger is diagnostic metadata, not raw evidence storage.
- Candidate records may carry `RawOutputRefID`, but full raw ref metadata lives in `ProjectionReport.RawOutputRefs`.
- Candidate records must not duplicate raw body, local artifact path, raw URL with query secrets, or full command args.
- Candidate records are suitable for `/ledger` sanitized rendering.
- Candidate rendering is bounded: show deterministic top N candidates and `... +X more` when truncated.

Aggregate summary should include:

```text
artifact_backed_candidate_count
artifact_backed_applied_count
artifact_backed_kept_raw_count
artifact_backed_dry_run_eligible_count
artifact_backed_threshold_skipped_count
dry_run_estimated_saved_bytes
dry_run_estimated_saved_tokens
actual_saved_bytes
actual_saved_tokens
raw_output_ref_count
raw_output_artifact_created_count
raw_output_artifact_verified_count
raw_output_artifact_failed_count
rehydrate_available_count
rehydrate_unavailable_count
top_kept_reasons
```

Aggregate rules:

- Dry-run estimated savings and actual apply savings must never be mixed.
- In dry-run, `dry_run_estimated_*` may be non-zero, but `actual_saved_*` must be zero.
- In apply, actual savings count only provider-facing replacements that changed the payload.
- Candidate discovery, fail-closed raw keep, and dry-run estimated savings must not disable Responses continuation chain.

Command surface responsibilities:

- `/status` shows aggregate summary only: modes, counts, actual vs estimated savings, top kept reasons, warnings, active context availability, and Responses chain state.
- `/ledger` shows sanitized candidate-level ledger for raw output artifact candidates, after existing runtime task ledger sections.
- `/ledger` is read-only diagnostic output. It does not append to history/session/audit, does not show raw body, and does not resolve artifact body for display.
- `/tokens` remains token estimate only. It may include request-local active context in token estimation, but it must not show provider history reduction diagnostics or candidate ledgers.

Candidate kept reasons should be specific enough to diagnose:

- `data_bearing_network_command_output_keep`
- `data_bearing_database_command_output_keep`
- `raw_output_ref_missing`
- `raw_output_artifact_missing`
- `raw_output_artifact_too_large`
- `raw_output_artifact_session_quota_exceeded`
- `raw_output_legacy_source_missing`
- `raw_output_legacy_source_ambiguous`
- `raw_output_artifact_materialization_failed`
- `raw_output_rehydrate_unsupported`
- `raw_output_hash_mismatch`
- `raw_output_active_context_budget_exhausted`
- `sensitive_output_artifact_forbidden`

Sensitive/report redaction contract:

- Report/status/ledger must not contain raw output body.
- Report/status/ledger must not contain local artifact path.
- Report/status/ledger must not contain raw URL query/fragment secrets.
- Report/status/ledger must not contain full command args when they may include paths, tokens, or secrets.
- Report/status/ledger must not trust provider/tool/user-provided labels as already-safe display text.
- Manifest/index/report metadata must not store raw body, raw command, raw URL, full DB query, full command args, or provider/tool/user label as an identity key.
- `raw_output_ref` is an opaque ID and may be shown.
- Redaction owner must be shared by placeholder, active context metadata, report/status, `/ledger`, manifest metadata, and tests.
- Sensitive or ambiguous output reports only sanitized reason metadata such as `sensitive_output_artifact_forbidden`; it does not create a normal artifact-backed candidate.

## 11. Tests

### Commandoutputs

- data-bearing success raw keep
- data-bearing failure-like raw keep
- validation success with weak failure markers
- validation failure with strong failure evidence
- side-effect textual failure compact
- sensitive output never gets data-bearing artifact-backed strategy
- ambiguous secret-looking data-bearing output is sensitive/raw keep for artifact-backed optimization
- classifier precedence matrix

### Providerhistory

- dry-run data-bearing artifact-backed candidate
- apply keeps data-bearing raw without rehydrate transport
- apply compacts data-bearing with artifact-backed ref + rehydrate transport
- parent apply + raw output artifacts dry-run keeps raw and reports candidate only
- parent dry-run + raw output artifacts apply keeps raw and reports dry-run candidate only
- raw output artifacts off suppresses raw output artifact-backed candidates
- missing / ambiguous / stale ref keep raw
- latest suffix / no later assistant guard
- saved bytes / saved tokens
- Responses chain disable only when replacement applied
- report clone defensive copy
- candidate-level ledger contains sanitized metadata and no raw body/local artifact path/secret-bearing URL
- aggregate summary separates dry-run estimated savings from actual apply savings
- secret-bearing URL query/fragment does not appear in placeholder/report/status/ledger
- full DB query and full command args do not appear in placeholder/report/status/ledger
- provider/tool/user-provided labels are sanitized and never become raw output ref IDs or artifact paths

### Rawoutputs

- content-addressed artifact write / read
- atomic write failure cleanup
- path traversal / invalid artifact ID rejection
- hash mismatch rejection
- versioned append-only manifest save / load
- manifest does not include raw body or raw command text
- index rebuilds from manifest and is not source of truth
- encrypted history mode encrypts artifact body / manifest / index
- quarantine / tombstone / GC event append
- default retention policy is `session`
- live-ref mark-and-sweep keeps live refs
- session-local dedupe only; cross-session dedupe is disabled
- session rewrite/truncate/reset tombstones unreachable refs
- quarantine prevents apply compact and active context rehydrate
- GC appends collected events after deleting objects
- sensitive output is not stored under normal policy
- ambiguous secret-looking output is not stored under normal policy
- manifest/index do not include raw command, raw URL, full DB query, full command args, raw body, or provider/tool/user label identity
- history-backed fallback resolver only activates for legacy/migration paths

### Agent

- active context rehydrate injects bounded raw output
- active context does not mutate history/session
- disabled rehydrate context keeps raw
- request-local context is not persisted
- hash mismatch / missing raw source structured failure
- status shows injected/skipped refs
- `/status` renders raw output artifact aggregate summary only
- `/ledger` renders bounded sanitized raw output artifact candidate ledger
- `/tokens` does not include provider history reduction diagnostics
- sensitive refs are not body rehydrated automatically
- request-local active context metadata follows the same redaction contract as placeholder/report/status/ledger

### Config

- default
- validation
- project/global override
- generated docs/config metadata
- old config compatibility
- existing `provider_history_reduction.mode` values: `off`, `dry_run`, `apply`
- existing `provider_history_reduction.rehydrate_context` remains boolean
- `provider_history_reduction.raw_output_artifacts.mode` values: `off`, `dry_run`, `apply`
- default `provider_history_reduction.raw_output_artifacts.mode=dry_run`
- parent `provider_history_reduction.mode=apply` does not imply data-bearing raw output artifact-backed apply
- explicit `raw_output_artifacts.mode=apply` still requires parent mode apply, `rehydrate_context=true`, and artifact/ref/hash/safety/threshold gates
- config docs and example are generated/updated from the same source of truth

### Test Boundary

- Put command output classifier tests under `internal/commandoutputs`.
- Put projection/report tests under `internal/providerhistory`.
- Put request-local active context tests under `internal/agent`.
- Put raw artifact store tests under `internal/rawoutputs`.
- Put MCP artifact-backed providerhistory tests under `internal/providerhistory` / `internal/providerhistory/toolresults`; use `internal/mcp` only for identity/source metadata tests.
- Put XELYON web_search providerhistory tests under `internal/providerhistory/toolresults`.
- Put provider-native replay request/round-trip tests under provider-specific packages and providerhistory classification tests under `internal/providerhistory`.
- Put review probe prompt-surface tests under `internal/review` / `internal/review/modelinput` / `internal/review/probe`, not providerhistory.
- Do not add more unrelated cases to already large test files if a focused file can own the contract.

## 12. Verification Commands

Focused commands:

```sh
go test ./internal/commandoutputs
go test ./internal/rawoutputs
go test ./internal/providerhistory
go test ./internal/agent
go test ./internal/taskstate
go test ./internal/mcp ./internal/mcptool
go test ./internal/api/websearch ./internal/api/providers/kimi ./internal/api/providers/openai ./internal/api/providers/claude
go test ./internal/review ./internal/review/modelinput ./internal/review/probe
```

When config/generator changes:

```sh
make gen-all
go test ./internal/config ./scripts/internal/configgen
```

Broader verification:

```sh
go test ./...
git diff --check
make ci-check
```

## 13. Final-A Impact Audit

After implementation and focused tests pass, run an impact audit before final report.

Must check:

- data-bearing output cannot be compacted without artifact-backed raw output ref.
- raw output ref without active context transport cannot enable apply compact.
- sensitive output is not duplicated into the normal artifact store or active context.
- failure classifier cannot outrank data-bearing preservation.
- dry-run savings and apply savings do not drift.
- Responses chain disable is tied to actual replacement only.
- no-later-assistant / latest suffix guards still protect current tool outputs.
- session reload / raw artifact store remains usable for rehydrate.
- history-backed fallback remains usable for legacy or missing-artifact diagnostics, if included in scope.
- provider-facing prompt contains enough ref/excerpt metadata to understand compacted data.
- fail-closed cases continue as raw keep / dry-run / candidate-only with specific reason reports.

If a correctness gap is found, switch to `post-implementation-impact-recovery`.
If the gap affects only apply safety for a phase/surface, block that apply path and continue other raw-keep/dry-run/candidate-only work.

## 14. Final-B Mandatory Refactor

After implementation and impact audit pass, run a behavior-preserving structure pass.

Use:

- `post-implementation-refactor`
- `test-boundary-refactor` if tests / fixtures / fakes / table tests changed

Final-B gates:

- `commandoutputs` does not become a mixed I/O / classifier / artifact owner.
- `providerhistory` does not duplicate classifier policy from `commandoutputs`.
- `agent` owns active context injection, not command classification.
- raw output ref type is not forced into file `EvidencePointer` if semantics differ.
- generic helpers do not mix data-bearing / sensitive / side-effect / validation strategies.
- test files are split by owner:
  - classifier
  - projection/report
  - active context
  - config
- repeated review findings in one package trigger owner map and split/extraction audit.
- same-file finding concentration in `commandoutputs`, `providerhistory`, `agent`, or tests is a patch accumulation incident and must be handled before final review.
- remaining owner debt must be reported by package/file; do not hide it behind a generic helper.

## 15. Adopted Decisions

These decisions are adopted and should be treated as implementation constraints.

1. Raw source backing
   - Decision: artifact-store-backed hybrid.
   - Primary: Raw Output Artifact Store.
   - Fallback: history-backed resolver for legacy sessions / migration / missing-artifact diagnostics.
   - Not adopted: history-backed-only is not the target design.

2. RawOutputRef exact contract
   - Decision: `RawOutputRefID` is an opaque short stable ID such as `rawout_c3f8a19b2d`.
   - Decision: `RawOutputRefID` must not include raw command text, URL, path, args, provider output text, or user/provider-provided labels.
   - Decision: provider-facing placeholders include only minimal readable metadata plus `raw_output_ref` and bounded excerpt.
   - Decision: full projection metadata lives in `ProjectionReport.RawOutputRefs`.
   - Decision: manifest is the storage lifecycle source of truth; report metadata is the projection/report source.
   - Decision: candidate structs carry `RawOutputRefID` plus gate/status/reason fields only.
   - Rule: every candidate `RawOutputRefID` must have exactly one matching report-level ref; missing report-level metadata fails closed.

3. Config gate
   - Decision: add optional stable `provider_history_reduction.raw_output_artifacts` config.
   - Decision: default `provider_history_reduction.raw_output_artifacts.mode=dry_run`.
   - Decision: existing `provider_history_reduction.mode` remains the parent provider-facing reduction mode gate.
   - Decision: `raw_output_artifacts.mode` is the data-bearing raw output artifact-backed compact gate.
   - Decision: existing `provider_history_reduction.rehydrate_context` boolean remains the active context transport gate.
   - Decision: this config split is a safety/responsibility boundary. Normal history reduction, data-bearing evidence compaction, artifact lifecycle, and active context transport must not share one ambiguous mode switch.
   - Decision: `raw_output_artifacts.mode=dry_run` may create artifacts and verify refs, but it must not mutate provider-facing payload.
   - Decision: `raw_output_artifacts.mode=off` is the no-artifact-backed-candidate/no-artifact-backed-apply escape hatch for this feature.
   - Decision: data-bearing apply compact requires parent `provider_history_reduction.mode=apply`, child `raw_output_artifacts.mode=apply`, `rehydrate_context=true`, active context transport, and artifact/hash/safety/threshold gates.
   - Existing configs remain valid and do not silently enable data-bearing artifact-backed apply.

4. Rehydrate amount
   - Decided default: bounded excerpt first, full raw only under strict budget or explicit deterministic trigger.
   - Decided budget: default `4096` tokens, upper `8192` tokens for raw output active context.
   - Decided relevance rule: deterministic priority scoring with explicit refs, latest/current-turn adjacency, lexical match, same family/tool/task surface, recent applied refs, and metadata-only fallback.
   - Decided excerpt policy: classifier-specific structured excerpt, then query/keyword matched excerpt, then deterministic first/last fallback.

5. Agent active context rehydrate transport
   - Decision: `agent` is request-local raw output rehydrate transport owner.
   - Decision: rehydrate plan is built from providerhistory applied replacements, `ProjectionReport.RawOutputRefs`, and current request/latest context relevance.
   - Decision: active context is injected as a distinct `Provider History Raw Output Context` section and is never persisted to history/session/audit.
   - Decision: file evidence active context and raw output active context remain separate sections.
   - Decision: applied data-bearing refs, explicit refs, and latest/current-turn adjacent applied refs are required refs.
   - Decision: required refs must resolve and receive body excerpt; optional refs may degrade to metadata-only.
   - Decision: sensitive refs are not body rehydrated automatically.
   - Decision: request-time required ref failure is recorded in report/status and must cause future projections to fail closed or keep raw.

6. Apply rollout
   - Decision: default `raw_output_artifacts.mode=dry_run`; provider-facing data-bearing artifact-backed apply requires explicit `raw_output_artifacts.mode=apply`.
   - Apply gates: parent provider history reduction mode apply, raw output artifacts mode apply, rehydrate context true, active context transport, artifact/ref/hash OK, relevance plan OK, excerpt plan OK, not sensitive, latest/no-later-assistant guards, and thresholds.
   - Thresholds: minimum saved tokens `2048`; maximum replacement ratio `0.75`.
   - Dry-run reports apply-eligible and threshold-skipped candidates separately.

7. Providerhistory candidate/apply gate
   - Decision: `providerhistory` is projection policy + gate engine + report owner.
   - Decision: `providerhistory` owns artifact-backed candidate creation, raw ref/report linkage, final placeholder building, dry-run/apply savings accounting, and Responses chain disable decisions.
   - Decision: candidate structs carry `RawOutputRefID` plus gate/status/reason fields, not full raw ref metadata.
   - Decision: exact gate order is decision, raw ref, artifact, rehydrate transport, freshness, safety, threshold, placeholder size, apply mode.
   - Decision: final data-bearing placeholder is built only after all gates pass and must link to exactly one report-level raw ref.
   - Decision: dry-run estimated savings and actual apply savings are separate counters.
   - Rule: Responses `previous_response_id` chain is disabled only when actual provider-facing replacement occurs.

8. Security / redaction contract
   - Decision: sensitive or ambiguous output is `no artifact`, `no artifact-backed apply`, `raw keep`, and sanitized reason metadata only.
   - Decision: raw body may exist only in existing raw history/session/audit surfaces, encrypted non-sensitive artifact object, and request-local active context body/excerpt.
   - Decision: `ProjectionReport`, `/status`, `/ledger`, manifest metadata, index cache, placeholder metadata, config, and logs must not store raw body.
   - Decision: `RawOutputRefID` is opaque and must not include command text, URL, path, DB name, table name, query text, raw output text, provider/tool label, user label, or secret-bearing metadata.
   - Decision: provider/tool/user-provided labels are untrusted and must not become ref IDs, artifact paths, manifest keys, or filesystem names.
   - Decision: URL query/fragment, full DB query text, literal DB values, and full command args are not report/ledger/status/placeholder metadata.
   - Rule: placeholder, active context metadata, report/status, `/ledger`, manifest metadata, and tests share one redaction contract.

9. Observability surfaces
   - Decision: `ProjectionReport` is the canonical source of truth for candidate-level raw output artifact ledger and aggregate counters.
   - Decision: `/status` renders aggregate summary only: modes, counts, estimated vs actual savings, top kept reasons, warnings, rehydrate availability, and Responses chain state.
   - Decision: `/ledger` renders bounded sanitized candidate-level ledger for raw output artifact candidates.
   - Decision: `/tokens` remains token estimate only and must not include provider history reduction diagnostics.
   - Rule: report/status/ledger must not show raw body, local artifact path, raw URL query/fragment secrets, or full command args.

10. Database command subfamilies
   - Decision: Phase 0 keeps entire database family raw before failure compact to close the current review risk.
   - Decision: Phase 1+ adds database subfamilies: query_result, schema_result, operation_log, migration_log, connection_error, unknown.
   - Query/schema results are data-bearing and artifact-backed eligible after gates.
   - Operation/migration/connection errors are operation/failure evidence and not data-bearing artifact-backed apply eligible.
   - Unknown remains raw keep or dry-run only.

11. Commandoutputs decision model
   - Decision: `commandoutputs` becomes a pure decision engine.
   - Decision: `Decide(...) Decision` is the source of truth.
   - Decision: `BuildReplacement(...)` remains as a compatibility wrapper during migration.
   - Decision: `commandoutputs` may return `ArtifactPolicy.RequiredForApply=true`, but providerhistory/agent own artifact availability, active context transport, threshold, dry-run/apply/report behavior.
   - Decision: global precedence is sensitive, data-bearing preservation, side-effect/mutation, explicit fatal/non-zero, strong validation success, weak textual failure marker, validation compact, unknown raw keep.
   - Rule: replacement text is a display strategy, not the decision source of truth.

12. Artifact store policy
   - Recommended default: implement secure content-addressed store now with internal defaults.
   - Decision: `rawoutputs` is storage engine + resolver + lifecycle ledger.
   - Decision: `rawoutputs` owns `Create`, `Resolve`, `Verify`, `MaterializeLegacy`, `CollectGarbage`, and `RebuildIndex` APIs.
   - Decision: `Resolve` takes full `RawOutputRef`, not only `RefID`, and returns no body unless lifecycle and hash verification pass.
   - Decision: `Create` streams body input, checks size/quota, writes atomically, applies encryption policy, and appends `raw_output_artifact_created` only after verification.
   - Decision: `CollectGarbage` uses caller-provided live refs; providerhistory/agent-session/review own liveness, rawoutputs owns delete eligibility.
   - Decision: `MaterializeLegacy` requires exact legacy history/session source identity; missing/ambiguous source fails closed.
   - Decision: rawoutputs exposes structured reason kinds for too-large, quota, missing, hash mismatch, quarantine, tombstone, GC-collected, manifest/index corrupt, encryption/decrypt, path/ref invalid, and legacy missing/ambiguous.
   - Decided artifact root: `~/.xelyon/history/rawoutputs/sessions/<session-id>/`.
   - Decided manifest format: versioned append-only `manifests/raw_outputs.jsonl` as source of truth; `indexes/raw_outputs.index.json` is rebuildable cache.
   - Decided retention/GC: default `session` retention, live-ref mark-and-sweep, tombstone/quarantine/gc_collected events, session-local dedupe only.
   - Decided size/quota: per-artifact `64 MiB`, per-session `1 GiB`, chunked I/O target `1 MiB`, active context budget is separate.
   - Decided legacy fallback scope: Phase 2 command output lazy materialization from raw history; history-backed source alone never enables apply compact; MCP / web search / provider-native replay deferred to Phase 6.
   - Decided raw ref metadata placement: full metadata lives in `ProjectionReport.RawOutputRefs`; candidates carry `RawOutputRefID` plus gate/reason fields only.
   - Not adopted: deferring the store is not acceptable if the goal is maximum safe cost optimization.

13. Implementation phasing / continuation
   - Decision: add Phase 0.5 boundary refactor / owner split / test foundation between the immediate safety fix and the decision model.
   - Decision: default response to unsafe/missing/ambiguous/not-yet-implemented behavior is raw keep, candidate-only, dry-run only, sanitized reason report, and continue.
   - Decision: hard stops are scoped to unsafe phase/surface whenever possible; other safe surfaces continue.
   - Decision: Phase 5 apply is blocked until Phases 1-4 have focused tests and report/status evidence proves all gates.
   - Decision: Phase 6 apply expansion is surface-specific and must not inherit command-output apply eligibility automatically.
   - Hard stop conditions: raw history/session/audit rewrite for provider-facing optimization, raw body in report/status/ledger/manifest/index/config/logs, user/provider/tool text-derived artifact paths or ref IDs, sensitive artifact creation under normal policy, data-bearing apply without artifact ref + active context transport, or tests unable to distinguish raw vs compact provider payload.
   - Rule: implementation should not stop merely because a surface is not apply-ready; degrade that surface to raw keep/dry-run/candidate-only and keep moving.

14. Phase 6 first target
   - Decision: Phase 6-A MCP tool results first.
   - Decision: Phase 6-B XELYON web_search tool results / evidence second if security owner audit stays clean.
   - Decision: Phase 6-C provider-native built-in replay candidate-only/dry-run first; apply disabled until provider-runtime source-owner and round-trip tests pass.
   - Decision: Phase 6-D review probe results / review prompt surfaces are a dedicated highest-quality review optimization phase after providerhistory runtime artifact contract stabilizes.
   - Decision: Phase 6-D uses review-specific raw refs and review prompt rehydrate, not `/review` debug artifacts and not providerhistory runtime active context.
   - Decision: Phase 6-D budget defaults to `4096` tokens, upper `8192`, required refs need at least `512` body tokens, optional refs need `160` when included with body.
   - Decision: Phase 6-D must implement runner-side fail-closed validation, required refs ledger, report/status visibility, and golden prompt tests before apply absorption is considered complete.
   - Rule: MCP, XELYON web_search, provider-native replay, and review probe result remain separate surfaces with separate source owners.

## 16. Goal Handoff Policy

### Goal objective

Implement provider history raw output artifact / active context rehydrate according to this plan.

### Same Goal Phase 6 scope

After Phase 0-5 are implemented and verified:

- Include Phase 6-A MCP tool results in the same Goal.
- Include Phase 6-B XELYON web_search tool results / evidence in the same Goal if the security/redaction owner audit stays clean.
- Include Phase 6-C provider-native built-in replay as candidate-only / dry-run reporting only.
- Do not enable Phase 6-C apply in the same Goal unless provider-runtime source-owner and round-trip contracts have been implemented and tested.
- Include Phase 6-D review probe results / review prompt surfaces as a dedicated highest-quality review optimization phase after providerhistory runtime artifact contracts are stable.
- Phase 6-D completion requires review-specific raw refs, review prompt rehydrate, prompt budget enforcement, runner-side fail-closed validation, required refs ledger, report/status visibility, and golden prompt tests.
- If Phase 6-D cannot satisfy the review-specific raw ref, rehydrate, budget, saturation, revision, validator, ledger, and golden prompt contracts, keep the unsafe absorption/apply path disabled, preserve raw prompt context or candidate-only/dry-run reporting, and continue other safe surfaces rather than shipping a weak absorption-only optimization.

Phase 6-A/B/C/D are separate surfaces. Do not merge their classifiers, refs, reports, or rehydrate policy into a single generic helper that hides surface-specific semantics.

### Handoff prompt

```text
/goal Implement docs/dev/provider-history-raw-output-artifact-rehydrate-plan.md end to end.

Use docs/dev/provider-history-raw-output-artifact-rehydrate-plan.md as the source of truth. Re-read it after any context compaction/resume and adapt to the latest source structure without weakening safety contracts.

Implement in order:
1. Phase 0 immediate P2 data-bearing raw keep fix.
2. Phase 0.5 boundary refactor / owner split / test foundation.
3. Phases 1-5 for commandoutputs decision model, rawoutputs store, providerhistory dry-run/report, agent active context rehydrate, and artifact-backed apply.
4. Phase 6-A MCP, Phase 6-B XELYON web_search if the security/redaction owner audit stays clean, Phase 6-C provider-native replay candidate-only/dry-run, and Phase 6-D review prompt optimization after runtime artifact contracts are stable.

Preserve these contracts:
- raw history/session/audit/persisted JSONL are not modified for provider-facing optimization.
- commandoutputs owns semantic decisions only and must not own artifact I/O.
- rawoutputs owns physical artifact storage, manifest, resolver, lifecycle, retention, and GC.
- providerhistory owns projection, candidate/apply gates, raw ref metadata, reports, placeholders, and savings accounting.
- agent owns request-local active context rehydrate and command-surface rendering.
- data-bearing output is not apply-compacted without explicit `raw_output_artifacts.mode=apply`, a resolvable artifact-backed raw output ref, and active context rehydrate transport.
- `ProjectionReport` is the canonical report source; `/status` renders aggregate summary, `/ledger` renders bounded sanitized candidate-level ledger, and `/tokens` remains token estimate only.
- sensitive or ambiguous output creates no normal artifact, receives no artifact-backed apply, and appears in reports only as sanitized reason metadata.

Keep implementation moving by degrading unsafe/missing/ambiguous paths to raw keep, candidate-only, dry-run only, and sanitized reason reports. Block only the unsafe apply path or surface.

For Phase 6, keep MCP, XELYON web_search, provider-native replay, and review probe results as separate surfaces with separate source owners. Do not enable Phase 6-C apply until provider-runtime source-owner and round-trip contracts are implemented and tested. Phase 6-D must use review-specific raw refs and review prompt rehydrate, preserve probe-level vs command-level granularity, enforce prompt budget, implement a required refs ledger, reject saturated results through runner-side validation when required refs are missing/metadata-only/budget-starved, and include golden prompt tests.

Run focused tests for commandoutputs/rawoutputs/providerhistory/agent/config/review surfaces as each owner changes, then go test ./..., git diff --check, and make ci-check when the final scope is implemented. Final-A impact audit and Final-B comprehensive refactor are mandatory. After tests pass, run post-implementation-refactor; if tests, fixtures, fakes, table tests, or assertion helpers changed, also run test-boundary-refactor. Do not commit or push unless explicitly requested.
```
