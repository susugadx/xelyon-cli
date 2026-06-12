# XELYON Skill Router / Skill Doctor 2.0 Master Plan

この文書は、XELYON の Agent Skills を「読める手順書」から「増えても運用できる手順書群」へ進めるための内部実装計画である。公開 docs ではなく、設計相談と Goal handoff の source of truth として使う。

現時点では v1 product 方針は確定済みの master plan とし、実装前には `Resolved Decisions And Implementation-Time Checks` を確認する。

## V1 Summary

v1 で作るもの。

- Optional sidecar `agents/xelyon.yaml` を読み、既存 `SKILL.md` frontmatter は portable なまま維持する。
- Skill Router は全 skill を score/rank し、runtime には bounded hint だけを渡す。
- `/skills suggest <text>` は debug / authoring 用に full ranked list を表示する。
- `/skills doctor --routing` は sidecar / routing metadata / description quality / prompt budget を診断する。
- `/skills usage` は local usage ledger の要約を表示し、`/skills usage clear` / `--all` で消せる。
- public config は `skills.router.enabled`、`skills.router.activation`、`skills.router.usage_ledger`、`skills.router.usage_retention_days` だけにする。
- product default は hint-only。auto activation の architecture は持つが、v1 では有効化しない。

v1 で作らないもの。

- Codex / Claude system skills の discovery。
- XELYON-specific fields の `SKILL.md` frontmatter 追加。
- runtime auto-load of full skill bodies。
- `/skills suggest` / `/skills usage` の `--json` public schema。
- full diff / file content を routing signal として読むこと。

## 0. Purpose

Skill が増えたときに、ユーザーやモデルが description だけで探す状態を脱し、XELYON runtime がタスクに合う skill 群を整理できるようにする。

目標は以下。

- 通常依頼のまま、runtime が primary / supporting / maybe / conflict skill 候補を作れる。
- skill の description だけでなく、optional `agents/xelyon.yaml`、task text、touched files、requested mode、read-only constraint などを使って選びやすくする。
- `/skills doctor` を parse / duplicate 診断だけでなく、routing 品質、metadata 不足、役割衝突、description の弱さを診断できるものへ拡張する。
- `/skills suggest` 相当は主導線ではなく、router の debug / authoring 用窓として扱う。
- runtime architecture は v1 から auto activation policy まで表現できる構造にする。ただし product default は dogfood で routing 品質が確認できるまで hint-only にする。
- user-facing docs も更新し、`/skills`、Skill Router、`/skills suggest`、Doctor 2.0、auto activation default の意味を説明する。
- Codex / Claude など他 runtime の system skills を読んだり vendor したりする方向には戻さない。

Goal で完了とみなす条件は、少なくとも deterministic な Skill Router と Doctor 2.0 が入り、既存 skill catalog / activate_skill / prompt injection と整合し、focused tests と `make ci-check` が通ること。

## 1. Current State / Implemented Preconditions

2026-06-08 時点の current source findings。

- Skill catalog は `internal/skills` が owner。
  - `Discover` は project `.agents/skills` と home `~/.agents/skills` を読む。
  - `Catalog` は parse と duplicate 解決を行い、XELYON built-in skills を追加する。
  - `SourceProject` / `SourceHome` / `SourceXelyon` がある。
- XELYON built-in `skill-creator` は `internal/skills/builtin.go` にある。
  - `xelyon://skills/skill-creator` として catalog に入る。
  - Codex system skills を読まない / copy しない / vendor しない方針が本文に入っている。
- prompt catalog は `internal/skills/prompt.go` が owner。
  - `BuildPromptCatalog` が name + description の metadata only block を作る。
  - `activate_skill(name)` で full `SKILL.md` を読むよう案内する。
  - `skill-creator` は capped catalog でも落ちないよう pinned されている。
- agent への prompt 注入は `internal/agent/agent_skills.go` が owner。
  - `injectSkillCatalogPrompt` が invocation cwd の catalog を system prompt に注入する。
- skill activation は read-only tool。
  - `internal/tools/skills/activate_skill_tool.go` の `activate_skill` が catalog から full payload を返す。
  - scripts / references / assets の file body は勝手に実行しない。
- `/skills` command は `internal/agent/command_skills.go` と render helpers が owner。
  - `/skills overview`
  - `/skills show <name>`
  - `/skills doctor`
  - `/skills list` alias
- 現在の `/skills doctor` は `internal/skills/format.go` の `FormatDoctorReport` が `catalog.Diagnostics` を列挙するだけ。
  - parse error
  - missing frontmatter field
  - duplicate skill name
  - discover failure
  - routing 品質、description 品質、role conflict はまだ見ない。
- docs は `docs/commands.md` に `/skills` の概要だけがある。

## 2. Problem Statement

今の skills は「機能する/しない」までは診断できるが、「選びやすい/運用しやすい」までは扱えていない。

現状の弱点。

- description が似た skill が増えると、モデル任せの選択が不安定になる。
- read-only skill と implementation skill のような conflict を runtime が明示できない。
- 複数 skill を使うべき依頼で、primary と supporting の役割が分からない。
- どの skill が候補になったか、なぜ候補になったかが見えない。
- prompt catalog の budget により、重要 skill が metadata から落ちるリスクがある。
- `/skills doctor` が skill authoring の品質改善につながらない。

この問題は skill 数が少ないうちは見えにくいが、project-local / home / built-in skills が増えるほど大きくなる。

## 3. Global Contracts

すべての実装で以下を守る。

- XELYON は Codex / Claude など他 runtime の system skills を読まない。
- XELYON built-in skill は XELYON-owned guidance として管理する。
- project `.agents/skills` と home `~/.agents/skills` の既存 discovery contract を壊さない。
- project / home skill が同名 built-in skill を上書きできる現在の override behavior を壊さない。
- 既存 `SKILL.md` の `name` / `description` frontmatter だけでも valid であり続ける。
- XELYON routing metadata は `SKILL.md` に混ぜず、optional sidecar `agents/xelyon.yaml` に置く。
- `agents/xelyon.yaml` は後方互換で optional にする。
- XELYON は `agents/xelyon.yaml` だけを XELYON routing metadata として読む。`agents/openai.yaml` など他 agent 用 sidecar は解釈しない。
- Router は候補を補助する runtime layer であり、user instruction / repo mandatory rules / safety policy を上書きしない。
- read-only skill が primary になった場合、implementation / mutation 系 skill を同一 step の primary として扱わない。
- Auto activation を入れる場合も context budget と confidence gate を持つ。大量の skill body を無条件に読む実装は禁止する。
- Architecture は auto activation に対応するが、v1 の product default は hint-only とする。
- hint-only default でも、Recommendation / Doctor / `/skills suggest` は activation policy を表示できるようにする。
- Router は内部的には全 skill を score/rank する。runtime hint はカテゴリ別 limit で絞るが、`/skills suggest` は関連度順に全件を確認できる debug surface とする。
- Skill routing usage ledger は v1 scope に含める。ただし local-only、raw prompt 保存なし、bounded retention、clear 可能な privacy-preserving ledger とする。
- `/skills suggest` 相当は通常ユーザーに使わせる主導線にしない。通常依頼の runtime 補助が本命。
- Doctor 2.0 の deterministic 診断は model call に依存しない。
- Router / Doctor の pure logic は I/O、git command、prompt mutation、tool execution と混ぜない。
- Git diff / touched files など重い signal は timeout / budget / failure-safe を持つ。

## 4. Non-goals

この計画でやらないこと。

- Codex `$skill-creator` や `CODEX_HOME/skills` を読む実装へ戻すこと。
- Claude / Codex の skill format 完全互換を目指すこと。
- `agents/openai.yaml` / Claude-specific frontmatter など他 agent 用 metadata を XELYON routing source として読むこと。
- 既存 skill を XELYON metadata 必須にして破壊的 migration すること。
- XELYON 都合の routing metadata を `SKILL.md` frontmatter に追加すること。
- ユーザーに毎回 `/skills suggest ...` を打たせる UX を主導線にすること。
- Router が user approval なしに file edit / tool execution を行うこと。
- skill body 内の scripts を自動実行すること。
- remote telemetry や外部 service に skill 使用履歴を送ること。
- raw user prompt / raw assistant response / full diff を usage ledger に保存すること。
- 最初から perfect ML ranking を作ること。

## 5. Terminology

- Skill: `SKILL.md` と resources で構成される手順書。
- XELYON routing metadata: runtime が選択補助に使う optional `agents/xelyon.yaml` sidecar。
- Signal: task text、command surface、git diff、touched files、language、requested mode など router input。
- Candidate: Router が評価した skill。
- Primary: この turn の主 workflow。
- Supporting: primary を補助する guardrail / checklist / domain note。
- Maybe: 関連はあるが confidence が低い候補。
- Conflict: 今回の primary / mode と同時使用すべきでない候補。
- Hint injection: Router 結果を model に渡す system/developer-level guidance。
- Auto activation: Router が高 confidence skill を model に hint するだけでなく、full skill body を事前に載せる policy。

## 6. Resolved XELYON Routing Metadata Contract

Resolved v1 contract。

`SKILL.md` remains portable and shared across skills-compatible agents. XELYON-specific routing metadata lives in a sidecar file:

```text
.agents/skills/strict-diff-review/
  SKILL.md
  agents/
    xelyon.yaml
```

Example `agents/xelyon.yaml`:

```yaml
version: 1
intents:
  - code-review
role: primary
read_only: true
modes:
  - review
triggers:
  - review
  - レビュー
  - 差分レビュー
conflicts:
  - implementation
  - file-edit
activation: hint
```

Fields。

- `version`: metadata schema version. v1 accepts only `1`.
- `intents`: router が task intent と照合する semantic tags。
- `role`: `primary` / `supporting` / `guardrail` / `authoring` の候補。
- `read_only`: file edit / command mutation と conflict するか。
- `modes`: `review` / `implementation` / `planning` / `investigation` / `authoring` など。
- `triggers`: description より明示的な trigger phrase。
- `conflicts`: conflict group names。
- `activation`: `manual` / `hint` / `auto` / `never`。

Out of v1 schema。

- `paths.include` / `paths.exclude`
- `languages`
- `priority`

v1 keeps these out because `agents/xelyon.yaml` is an OSS-facing XELYON sidecar, not a personal local-only file. These fields require a separate product decision after dogfood.

Authoring contract。

- New skills created by XELYON should include `agents/xelyon.yaml` when the authoring context is clear enough.
- Existing skills remain valid without `agents/xelyon.yaml`.
- When updating an existing skill, XELYON may add or update `agents/xelyon.yaml` after reading `SKILL.md` and explaining the routing intent.
- `skill-creator` guidance must be updated so XELYON-created skills use the sidecar and do not add XELYON-specific frontmatter to `SKILL.md`.
- Generated sidecars are proposals from the authoring assistant, not a requirement for every skill to function.

Compatibility。

- `agents/xelyon.yaml` がない skill は description-only skill として扱う。
- `agents/xelyon.yaml` の parse/type failure は warning にし、skill 自体は catalog に残す。
- invalid `agents/xelyon.yaml` の skill は sidecar-based scoring を使わず、description-only fallback として routing する。
- missing `version`、wrong type `version`、unsupported `version` は invalid sidecar warning にし、description-only fallback として routing する。
- v1 accepts only `version: 1`.
- unknown fields in `agents/xelyon.yaml` produce warnings, the unknown fields are ignored, and the skill remains valid.
- `agents/openai.yaml` や他 agent sidecar は XELYON Router では無視する。
- core frontmatter の `name` / `description` が invalid な場合だけ skill parse error として扱う。
- built-in `skill-creator` のような filesystem sidecar を持たない XELYON built-in skills は code-defined routing metadata を持てる。
- XELYON が skill を create/update する場合は、必要に応じて `agents/xelyon.yaml` を作成・更新する。`SKILL.md` frontmatter へ XELYON-specific fields を混ぜない。
- Skill catalog cache / fingerprint must include `agents/xelyon.yaml` content and existence. Editing only the sidecar must invalidate cached routing metadata without requiring `SKILL.md` changes.

### 6.1 Initial Routing Vocabulary

v1 uses a known vocabulary for routing metadata validation. Unknown values warn but do not invalidate the skill.

Known `intents`:

- `code-review`
- `bug-investigation`
- `risk-scan`
- `implementation`
- `refactor`
- `cleanup`
- `test-coverage`
- `test-boundary`
- `config`
- `provider-runtime`
- `state-lifecycle`
- `concurrency-lifecycle`
- `security-boundary`
- `package-boundary`
- `skill-authoring`
- `docs-authoring`
- `planning`

Known `modes`:

- `review`
- `implementation`
- `investigation`
- `planning`
- `authoring`
- `refactor`
- `cleanup`
- `test`
- `docs`
- `config`

Known `role` values:

- `primary`
- `supporting`
- `guardrail`
- `authoring`

Known `conflicts` groups:

- `read-only`
- `implementation`
- `file-edit`
- `review`
- `planning-only`
- `authoring`
- `runtime-execution`
- `security-boundary`
- `provider-runtime`
- `config`

Doctor should warn on unknown `intent`, `mode`, `role`, or `conflict` values, but the skill remains valid and can still route using description fallback and known metadata fields.

## 7. Skill Router Architecture

Router は以下の層に分ける。

```text
Signal Collector
  -> Router Input
  -> XELYON Routing Metadata Resolver
  -> Candidate Scorer
  -> Conflict Resolver
  -> Recommendation
  -> Prompt / UI Renderer
```

### 7.1 Signal Collector

Inputs。

- raw user task text
- slash command or regular chat
- command surface
- current cwd / project root
- git status presence
- staged / unstaged / untracked summary
- touched file paths
- language hints from file extensions
- explicit words: review, implement, fix, investigate, plan, refactor, test, config, provider, skill
- current mode constraints: read-only review, planning, implementation, approval state
- current catalog size and prompt budget pressure

Contracts。

- Collector は failure-safe。git がない / timeout / large repo なら signal を empty or partial にする。
- Collector は raw task text を mutate しない。
- Collector の expensive checks は cache / timeout / max file count を持つ。
- v1 does not read full diff content or file contents for routing signals.
- Use path/status/language-extension signals only.
- Total signal collection should stay within a small interactive budget, target max `750ms`.
- Git status / touched-file collection should use a shorter timeout, target max `500ms`.
- Touched file paths should be capped, target max `200` paths across staged / unstaged / untracked.
- If a cap or timeout is hit, keep partial signals and record a diagnostic reason for `/skills suggest` / Doctor, not a runtime error.

### 7.2 Candidate Scorer

Scoring inputs。

- description match
- trigger phrase match
- intent match
- path / language match
- mode match
- source ordering
- role fit
- explicit user mention of skill name
- conflict penalties
- prompt budget / pinned skill status

Output。

```go
type SkillRecommendation struct {
    Primary    []SkillCandidate
    Supporting []SkillCandidate
    Maybe      []SkillCandidate
    Conflicts  []SkillConflict
}
```

Candidate should include。

- skill name
- source
- role
- score / confidence band
- matched signals
- reason text
- activation recommendation
- conflict group if any

Ranking contract。

- Router scores every skill in the catalog, including low-confidence matches.
- Recommendation keeps a full ranked list for debug and diagnostics.
- Runtime hint rendering applies category limits so the model is not flooded with weak candidates.
- `/skills suggest` can render the full ranked list in relevance order, with category, score, confidence, activation policy, and reason.
- Category classification and global ranking must use the same scoring result. Do not maintain separate suggestion-only scoring logic.

Default v1 runtime hint limits:

- `primary`: max 2
- `supporting`: max 5
- `conflict`: max 5
- `maybe`: 0 by default; keep maybe candidates for `/skills suggest` and diagnostics, not normal runtime hint injection

These limits apply only to runtime hint rendering. Router still scores all catalog skills and `/skills suggest` still renders the full ranked list.

Default v1 score bands:

- `high`: `80..100`
- `medium`: `50..79`
- `low`: `25..49`
- `none`: `<25`

Runtime hint injection should use only high/medium primary, supporting, and conflict candidates after category limits. Low and none remain visible in `/skills suggest` and diagnostics.

Explicit user mention of a skill name should strongly boost that skill and explain the boost in the reason text, unless a conflict or safety constraint prevents activation guidance.

### 7.3 Conflict Resolver

Initial conflict groups。

- `read-only` vs `implementation`
- `review` vs `file-edit`
- `planning-only` vs `implementation`
- `authoring` vs `runtime-execution`
- `security-boundary` as supporting guardrail, not automatic primary unless task asks security review

Rules。

- If primary is read-only, implementation skills may be listed as conflicts or blocked follow-up, not supporting execution guidance.
- If task is implementation and a read-only review skill matches only because the word "review" appears in prior context, it should not become primary.
- Supporting skills must not weaken primary's safety contract.

## 8. Runtime Behavior

Target highest-level behavior。

```text
User says normal task
-> XELYON builds SkillRecommendation
-> Model receives concise recommendation hint
-> Model activates full skills when needed
-> Optional verbose/debug UI can show recommendation
-> Doctor evaluates why routing works or fails
```

### 8.1 Prompt Hint Injection

Router hint should be concise and bounded.

Example:

```text
Recommended skills for this turn:
Primary:
- post-implementation-impact-recovery: user is asking to fix review findings and close adjacent impact.

Supporting:
- test-coverage-improvement: tests are explicitly requested.

Conflict:
- strict-diff-review: read-only review skill, not compatible with implementation in this turn.

Use activate_skill(name) only for skills you need to follow.
```

Contracts。

- Router hint is not a replacement for mandatory project instructions.
- Hint must not include full skill body.
- Hint must not include too many candidates. Default cap should be small.
- Hint uses category limits instead of one global candidate limit.
- Maybe candidates are omitted from v1 runtime hint.
- Hint should be stripped/replaced on prompt refresh like the existing skills catalog block, not duplicated.

### 8.2 Auto Activation Policy

Auto activation is powerful but risky. Treat it as gated.

Internal policy shape:

- `manual`: never auto-load full body.
- `hint`: recommend; model decides whether to call `activate_skill`.
- `auto`: runtime may include full skill body only when:
  - confidence is high
  - candidate count is within cap
  - no conflict
  - skill body is within token budget
  - sidecar or code-defined routing policy permits auto
  - user did not disable skill automation
- `never`: do not recommend or activate except explicit user mention.

Default policy for v1 is hint-only. The runtime architecture must still carry activation policy from the first implementation so that auto activation can be enabled by a non-v1 product decision without redesigning Router / Doctor / prompt rendering.

Auto activation can be enabled only after dogfood proves routing quality, and only behind explicit config or another explicit product decision.

### 8.3 UI / Observability

Normal UX should not require the user to call `/skills suggest`.

V1 surfaces:

- hidden prompt hint only
- visible activated skill summary
- `/skills suggest <text>` shows detailed debug output

Out of v1:

- `/status` skill routing summary

Runtime hint and debug output must be separate surfaces.

- Runtime hint: limited primary / supporting / conflict candidates only.
- `/skills suggest`: full ranked list by relevance, including maybe and low-confidence matches.
- Normal UI must not show router candidates by default.
- Normal UI may show activated skills because those are the actual full `SKILL.md` guidance loaded for the turn.
- Candidate visibility and activated-skill visibility are separate contracts.

## 9. Skill Doctor 2.0

Doctor 2.0 should move from "can parse" to "can operate".

### 9.1 Existing Diagnostics To Keep

- discover failure
- parse failure
- missing `name`
- missing `description`
- duplicate skill name

### 9.2 New Static Diagnostics

Metadata diagnostics。

- `missing_xelyon_metadata`: skill has no `agents/xelyon.yaml`; warn only in routing-focused diagnostics, not in plain `/skills doctor`.
- `invalid_xelyon_metadata`: `agents/xelyon.yaml` parse/type error.
- `unknown_xelyon_metadata_field`: `agents/xelyon.yaml` contains a field unsupported by the current schema version.
- `unknown_intent`: intent not in known vocabulary.
- `unknown_mode`: mode not in known vocabulary.
- `unknown_role`: role not in known vocabulary.
- `unknown_conflict`: conflict group not in known vocabulary.
- `read_only_without_conflicts`: read-only skill has no conflicts.
- `auto_activation_without_budget_guard`: auto skill is too large or lacks safe role.
- `trigger_too_broad`: trigger like "code", "fix", "work" is too broad.
- `description_too_short`: description lacks clear trigger condition.
- `description_too_broad`: description matches too many generic tasks.
- `description_duplicates_trigger`: multiple skills have near-identical trigger text.

Routing diagnostics。

- `overlapping_primary_candidates`: same intent has multiple primary skills without differentiating paths/modes.
- `missing_primary_for_intent`: common intent has supporting skills but no primary.
- `conflict_cycle`: conflicts imply unusable combinations.
- `prompt_budget_pressure`: catalog exceeds prompt metadata budget and non-pinned important skills may fall out.
- `source_shadowing`: project/home skill shadows built-in skill; may be expected but should be visible.

Severity policy:

- Error:
  - discovery / read failure that prevents a skill from being loaded
  - `SKILL.md` parse failure
  - missing required `name`
  - missing required `description`
- Warning:
  - duplicate skill names where source precedence hides a lower-priority skill
  - invalid `agents/xelyon.yaml`
  - unknown `agents/xelyon.yaml` fields
  - unknown intent / mode / role / conflict values
  - read-only skill missing conflict metadata
  - overlapping primary candidates
  - prompt budget pressure
  - conflict cycle
- Info:
  - missing `agents/xelyon.yaml` in `/skills doctor --routing`
  - description quality suggestions
  - expected source shadowing that follows documented precedence

Plain `/skills doctor` must not warn just because a valid legacy skill lacks `agents/xelyon.yaml`.

### 9.3 Runtime / History Diagnostics

v1 includes a local-only usage ledger for routing quality diagnostics.

The ledger records routing outcomes, not raw conversation content.

Default enabled rationale:

- Doctor 2.0 needs local outcome data to detect "recommended but never activated" and similar routing quality problems.
- The ledger is local-only, bounded by retention, clearable, and stores no raw prompt / response / diff / file content.
- Users can disable persistence with `skills.router.usage_ledger: false`.

Allowed ledger fields:

- timestamp
- repo key derived from the normalized project root hash
- optional task fingerprint, never raw task text; v1 diagnostics should not require storing prompt text
- activation-recommendable skills: name, role/category, score/confidence band, activation recommendation
- conflict candidates may be rendered in runtime hints, but must not count as "recommended but never activated" usage.
- activated skills: names loaded through `activate_skill`, recorded by the Agent runtime after a successful tool result, not by the read-only tool implementation itself
- router policy snapshot: enabled, activation mode, category limits
- optional turn/session id if already local and non-sensitive

Forbidden ledger fields:

- raw user prompt
- raw assistant response
- full `SKILL.md` body
- raw diff or file contents
- secrets / environment values

Ledger diagnostics:

- skill often recommended but never activated, excluding conflict-only candidates
- skill often activated after not being recommended
- user frequently overrides recommendation
- skill description changed but routing tests were not updated

No remote telemetry. Data must stay local, be bounded by retention, and be clearable.

Ledger storage contract:

- Store usage ledger outside the repository under XELYON local state.
- Default path shape: `~/.xelyon/skills/router/usage/<repo-key>.jsonl`.
- `<repo-key>` is a hash of the cleaned project root path. Do not expose the raw project path in the file name.
- Resolve the project root from project-map root first, then invocation cwd / project instruction root resolution. Agent v1 current-repo usage paths require a resolved project root; when no project root is available, skip recommendation/activation writes and show root-unavailable output for `/skills usage` current-repo reads/clears instead of using a shared `no-repo` bucket.
- The ledger is append-only JSONL for normal writes. Pruning may rewrite the bounded file.
- Ledger write failure must be non-fatal and must not block the turn.

Ledger cleanup and retention:

- `skills.router.usage_retention_days` defaults to `30`.
- Validate `usage_retention_days` as a bounded positive value, recommended range `1..365`.
- Disabling persistence is controlled by `skills.router.usage_ledger: false`, not by setting retention to `0`.
- Prune the current repo ledger opportunistically on write.
- Also prune before `/skills usage` and `/skills doctor --routing` read ledger data.
- Do not run global ledger pruning during startup.
- Add `/skills usage` to inspect local routing usage summaries.
- Add `/skills usage clear` to clear the current repo ledger.
- Add `/skills usage clear --all` to clear all skill router usage ledgers.
- `/skills doctor --routing` reads ledger diagnostics, but destructive cleanup stays under `/skills usage clear`.

### 9.4 Doctor Output Contract

Doctor report should remain readable in classic REPL and TUI.

Default noise policy。

- Plain `/skills doctor` should stay quiet for legacy valid skills.
- Missing `agents/xelyon.yaml` is not a warning in plain `/skills doctor`.
- Routing-focused diagnostics should show missing sidecar warnings.
- Use `/skills doctor --routing` or equivalent routing mode to inspect sidecar completeness and routing quality.
- This keeps adoption gentle while still giving skill authors a path to improve Router quality.

Recommended shape:

```text
Skills Doctor

Catalog:
- 18 skills
- 1 parse error
- 3 warnings

Errors:
- [ERROR] parse_skill_failed ... 

Routing:
- [WARN] overlapping_primary_candidates: strict-diff-review and codebase-risk-scan both match "review" without mode/path distinction.
  Suggestion: add modes or narrower triggers in agents/xelyon.yaml.

Prompt budget:
- [WARN] prompt_budget_pressure: 31 skills, catalog cap 24.
  Suggestion: pin critical skills or improve router hints.
```

Diagnostics should be structured internally and formatted at the edge.

## 10. Commands / User Surface

### `/skills doctor`

Primary command for catalog and routing health.

- `/skills doctor` shows all deterministic diagnostics.
- `/skills doctor --routing` focuses on router metadata.
- Do not add `/skills doctor --task <text>` in v1. Use `/skills suggest <text>` for sample task routing inspection.
- Slash command parser implementation may adapt to current parser constraints, but the v1 user surface above should stay stable.

### `/skills suggest <text>`

This is a debug / authoring window, not the main UX.

Example output:

```text
Skill Routing Suggestion

Ranked skills:
1. post-implementation-impact-recovery (94, primary, hint)
   reason: task asks to fix review findings and close adjacent impact
2. test-coverage-improvement (82, supporting, hint)
   reason: tests are explicitly requested
3. test-boundary-refactor (64, maybe, hint)
   reason: test helper boundaries may be affected
4. strict-diff-review (52, conflict, never)
   reason: read-only review conflicts with implementation request

Primary:
- post-implementation-impact-recovery (high)
  reason: task asks to fix review findings and close adjacent impact

Supporting:
- test-coverage-improvement (high)
  reason: tests are explicitly requested

Conflicts:
- strict-diff-review
  reason: read-only review conflicts with implementation request
```

Use `/skills suggest <text>` as the user-facing debug command name. It is intentionally not the primary workflow; normal user requests should still be routed automatically behind the scenes.

By default, `/skills suggest` should show the full ranked list, not only the candidates that would be injected into the runtime hint.

v1 output is human-readable text only. Do not add `--json` for `/skills suggest` in v1; keep structured recommendation data internal and do not freeze a public machine-readable schema yet.

### `/skills usage`

Shows local skill routing usage summaries from the usage ledger.

v1 output is human-readable text only. Do not add `--json` for `/skills usage` in v1; ledger records remain internal and no public machine-readable schema is exposed.

### `/skills overview`

Keep overview compact in v1.

Metadata badges are out of v1:

- `read-only`
- `primary`
- `supporting`
- `auto`
- `xelyon`
- `shadowed`

## 11. Implementation Priority

Phase 0: Source map and package boundary check.

- Default package boundary: split Router pure logic into `internal/skills/router`.
- Default package boundary: split Doctor 2.0 diagnostics into `internal/skills/doctor`.
- Default package boundary: keep usage ledger persistence in a dedicated owner, not inside scorer logic.
- Confirm import direction and adjust package names only if current source ownership or import cycles require it.
- Confirm command rendering remains in `internal/agent`.

Phase 1: `agents/xelyon.yaml` schema and parser.

- Add optional XELYON routing metadata to parsed skill domain.
- Read `agents/xelyon.yaml` as a sidecar when present.
- Preserve existing `name` / `description` only skills.
- Add tests for valid sidecar, missing sidecar, invalid sidecar, unsupported version, unknown fields policy.
- Use the small v1 schema: `version`, `intents`, `role`, `read_only`, `modes`, `triggers`, `conflicts`, `activation`.
- Do not implement `paths`, `languages`, or `priority` in v1.
- Accept only `version: 1`; missing/wrong/unsupported `version` is invalid sidecar warning + description-only fallback.
- Validate metadata values against the initial routing vocabulary from section 6.1.
- Unknown `intent`, `mode`, `role`, or `conflict` values warn and do not invalidate the skill.
- Include `agents/xelyon.yaml` existence/content in skill catalog cache invalidation. Sidecar-only edits must refresh routing metadata.

Phase 2: Doctor 2.0 static diagnostics.

- Keep existing parse / duplicate diagnostics.
- Add sidecar metadata and description-quality diagnostics.
- Add unknown `intent`, `mode`, `role`, and `conflict` diagnostics.
- Add source shadowing and prompt budget diagnostics.
- Format output without making it noisy.

Phase 3: Router pure logic.

- Define router input, candidate, recommendation, conflict structures.
- Implement deterministic scorer.
- Score all catalog skills and keep the full ranked list.
- Use the default score bands: high `80..100`, medium `50..79`, low `25..49`, none `<25`.
- Add focused tests for primary / supporting / maybe / conflict.
- Add tests for read-only vs implementation conflict.
- Add tests for explicit skill mention boosting and conflict/safety constraints.
- Add signal collection caps: no full diff/file content, target total budget `750ms`, git/touched-file budget `500ms`, touched path cap `200`.

Phase 4: Debug command surface.

- Add `/skills suggest <text>` for inspection.
- Do not require normal users to use it.
- Use same pure router logic as runtime.
- Keep `/skills suggest` v1 output human-readable only; do not add `--json`.

Phase 5: Runtime hint injection.

- Add bounded recommendation hint block to prompt.
- Apply category limits for runtime hint rendering.
- Keep `/skills suggest` full-list rendering separate from runtime hint rendering.
- Ensure prompt refresh replaces hint block instead of duplicating it.
- Ensure project rules / mandatory system prompt remain stronger than skill hints.

Phase 6: Auto activation policy structure.

- Add config or internal policy.
- Start with hint-only product default.
- Keep auto activation disabled until dogfood and explicit product decision.
- Ensure reserved auto-load can be limited to high confidence, no conflict, budget-safe skills.

Phase 7: Local usage ledger.

- Add local-only skill routing usage ledger.
- Store no raw prompts, no raw responses, no diffs, and no file contents.
- Record recommended vs activated skill summaries.
- Store repo-scoped JSONL under `~/.xelyon/skills/router/usage/<repo-key>.jsonl`.
- Use a hash of the cleaned project root as `<repo-key>`. Agent-level current-repo diagnostics do not use a `no-repo.jsonl` fallback; rootless sessions skip per-repo usage writes/reads so unrelated directories are not mixed.
- Add retention / pruning on write and before `/skills usage` / `/skills doctor --routing` reads.
- Add `/skills usage`, `/skills usage clear`, and `/skills usage clear --all`.
- Add Doctor 2.0 aggregate diagnostics over the local ledger.
- Ensure ledger writes are failure-safe and never block the turn.
- Use `go-state-lifecycle-change` for ledger state/persistence owner and cleanup.
- Use `security-boundary-change` for ledger path, content, retention, and raw-data exclusion checks.

Phase Final-A: Impact audit.

- Verify no CODEX_HOME / external skill reading path returned.
- Verify XELYON reads only `agents/xelyon.yaml` for routing metadata and ignores other agent sidecars.
- Verify usage ledger stores no raw prompt, raw response, diff, file content, or secrets.
- Verify usage ledger failure does not break chat/runtime execution.
- Verify usage ledger retention and clear behavior.
- Verify prompt blocks do not duplicate after refresh.
- Verify read-only conflict prevents implementation guidance in review mode.
- Verify existing `/skills` behavior remains compatible.
- Verify Doctor 2.0 does not turn existing valid skills into errors by default.

Phase Final-B: Mandatory post-implementation refactor.

- Inspect production diff and test diff for wrong owner, duplicate source of truth, generic helpers, and bloated test fixtures.
- If router and doctor logic accumulate in one file, split before review.
- If scorer logic, doctor diagnostics, prompt rendering, and JSONL persistence accumulate in one package/file, split by owner before review.
- If test helpers duplicate scorer policy, extract test fixture builders.
- Run focused tests and `make ci-check`.

## 12. Responsibility Boundaries

Suggested ownership.

- `internal/skills`: skill domain model, parse, catalog, XELYON routing metadata sidecar, activate payload.
- `internal/skills/router`: pure router scoring and recommendation domain.
- `internal/skills/doctor`: deterministic doctor diagnostics over catalog + router metadata.
- `internal/agent`: slash command orchestration and prompt hint injection.
- `internal/agent`: usage ledger write orchestration for routing recommendations and successful skill activations; activation writes require a resolved project root and must be best-effort.
- `internal/tools/skills`: `activate_skill` tool remains read-only full body loader and must not write usage ledger state.
- usage ledger owner should be a dedicated package or clearly bounded component; do not mix persistence with scorer logic.
- docs: `docs/commands.md` for user-visible commands, this file for internal plan.

Do not place domain routing policy in generic helpers.

## 13. Tests

Required focused tests.

- existing `SKILL.md` without `agents/xelyon.yaml` remains valid.
- `agents/xelyon.yaml` parser accepts the v1 schema.
- invalid `agents/xelyon.yaml` yields warning and description-only fallback.
- missing / wrong / unsupported `version` yields warning and description-only fallback.
- unknown `agents/xelyon.yaml` fields yield warning, are ignored, and keep the skill valid.
- sidecar-only content changes invalidate cached catalog routing metadata.
- adding or removing `agents/xelyon.yaml` invalidates cached catalog routing metadata.
- `agents/openai.yaml` or other agent sidecars do not affect XELYON routing.
- invalid core `name` / `description` frontmatter remains parse error.
- unknown `intent`, `mode`, `role`, and `conflict` sidecar values warn and keep the skill valid.
- built-in `skill-creator` appears in router catalog.
- project skill can still override built-in `skill-creator`.
- skill-creator guidance instructs XELYON-authored skills to create/update `agents/xelyon.yaml`.
- skill-creator guidance forbids XELYON-specific `SKILL.md` frontmatter fields.
- Doctor reports parse failures and duplicate names as before.
- Doctor reports sidecar metadata warnings without failing valid legacy skills.
- Doctor reports overlapping primary candidates.
- Doctor reports read-only skill missing conflict metadata.
- Router selects primary for review request.
- Router selects supporting skill for provider/runtime touched files.
- Router marks implementation skill as conflict when read-only review is primary.
- Router stores a full ranked list for all catalog skills.
- Router applies high / medium / low / none score bands.
- Router honors runtime hint limits: `primary 2`, `supporting 5`, `conflict 5`, `maybe 0`.
- Router boosts explicit skill-name mention while respecting conflict and safety constraints.
- Router handles no git repository / no diff / timeout signals.
- Router signal collection does not read full diff or file contents in v1.
- Router signal collection caps touched paths and returns partial signals on timeout.
- `/skills suggest` and `/skills usage` are human-readable only in v1 and expose no `--json` public schema.
- usage ledger records recommended vs activated summaries without raw prompt or raw content.
- usage ledger does not count conflict-only candidates as activation recommendations.
- usage ledger prunes by retention and can be cleared.
- usage ledger write failure does not fail runtime execution.
- Doctor aggregates ledger diagnostics for unused recommended and manually activated skills.
- Prompt hint injection replaces existing hint block.
- Prompt hint injection is bounded by candidate cap.
- Prompt hint injection uses category limits and omits weak maybe candidates by default.
- `/skills suggest` can render full ranked results, not only runtime-injected candidates.
- `/skills suggest` uses the same router logic as runtime.
- `activate_skill` remains read-only and does not execute scripts.

Comprehensive verification.

```text
make ci-check
```

## 14. Config / Policy Surface

The runtime design must support a richer internal policy shape from v1, but public config exposes only the stable minimum surface.

V1 public config keys:

```yaml
skills:
  router:
    enabled: true
    activation: hint
    usage_ledger: true
    usage_retention_days: 30
```

Because v1 exposes public config, use `xelyon-config-contract-change`, update generated config docs, examples, registry, `/config` UI, and migration/default behavior.

Chosen v1 public config:

- `skills.router.enabled`
  - default: `true`
  - when `false`, runtime skill routing hint injection is disabled; explicit `/skills` and `activate_skill` remain available.
- `skills.router.activation`
  - accepted v1 values: `off`, `hint`
  - default: `hint`
  - `off`: runtime hint injection is disabled while `/skills suggest` can still use Router logic.
  - `hint`: Router injects bounded skill recommendations; full `SKILL.md` bodies are not auto-loaded.
- `auto` is intentionally not accepted in v1. It is reserved for dogfood-backed product decisions outside v1.
- `skills.router.usage_ledger`
  - default: `true`
  - records local-only routing outcome summaries for Doctor diagnostics.
  - must not store raw prompts, raw responses, diffs, file contents, or secrets.
- `skills.router.usage_retention_days`
  - default: `30`
  - bounds local ledger retention.
  - valid range: `1..365`.
  - `0` is not a disable switch; use `skills.router.usage_ledger: false` to disable persistence.

Internal policy still carries the broader shape:

- architecture: supports `manual` / `hint` / reserved `auto` / `never`
- default behavior: hint-only
- auto activation: disabled until dogfood and explicit product decision
- debug surface: `/skills suggest <text>`
- `/skills suggest`: full ranked list by default; its limit remains internal in v1
- runtime hint: category-limited, not full list
- usage ledger: local-only, enabled by default, bounded retention, clearable

Do not expose these in v1 public config:

- runtime primary/supporting/conflict limits
- `suggest_limit`
- visibility / show hints
- auto activation knobs
- doctor routing diagnostics toggles

Keep those internal until dogfood proves the names and semantics.

## 15. Docs Surface

User-facing docs must be updated when implementation lands.

Required docs updates:

- `docs/commands.md`
  - `/skills doctor` now checks routing / sidecar metadata health, not only parse/duplicate diagnostics.
  - Plain `/skills doctor` remains quiet for legacy valid skills without `agents/xelyon.yaml`.
  - Routing-specific diagnostics such as missing `agents/xelyon.yaml` are shown through `/skills doctor --routing` or the chosen routing diagnostics mode.
  - `/skills suggest <text>` is a debug/authoring command for inspecting Skill Router recommendations.
  - `/skills suggest` v1 output is human-readable text only; no `--json`.
  - `/skills usage` v1 output is human-readable text only; no `--json`.
  - `/skills usage clear` clears current repo usage ledger.
  - `/skills usage clear --all` clears all skill router usage ledgers.
  - Normal users do not need to call `/skills suggest`; runtime routing happens behind the scenes.
  - v1 default is hint-only: XELYON recommends skills to the model, but does not auto-load full skill bodies unless a non-v1 explicit setting enables it.
  - usage ledger is local-only, raw prompt/content-free, bounded, and clearable.
- `docs/config.md` / generated config docs for `skills.router.enabled`, `skills.router.activation`, `skills.router.usage_ledger`, and `skills.router.usage_retention_days`.
- `config.yaml.example` and `/config` registry entries for the public config keys.
- README only if `/skills` is already user-visible there or the release note needs a short feature mention.

Docs must avoid implying that XELYON reads Codex / Claude system skills.

## 16. Resolved Decisions And Implementation-Time Checks

No product-level open decisions remain for v1. The implementation should follow the resolved defaults below and only adjust package names if current source ownership or import cycles require it.

Resolved defaults:

- `agents/xelyon.yaml` remains optional and separate from `SKILL.md` frontmatter.
- v1 sidecar schema includes `version`, `intents`, `role`, `read_only`, `modes`, `triggers`, `conflicts`, `activation`.
- v1 sidecar schema excludes `paths`, `languages`, and `priority`.
- `version` must be `1`; missing, wrong type, or unsupported versions are invalid sidecar warnings with description-only fallback.
- Invalid sidecar warns while the skill remains usable; core `name` / `description` invalid still makes the skill invalid.
- Unknown sidecar fields warn, are ignored, and do not invalidate the skill.
- Unknown `intent`, `mode`, `role`, or `conflict` values warn and do not invalidate the skill.
- Skill catalog cache / fingerprint includes `agents/xelyon.yaml` existence and content.
- Initial known vocabulary is defined in section 6.1.
- Router pure logic defaults to `internal/skills/router`.
- Doctor diagnostics default to `internal/skills/doctor`.
- Usage ledger persistence stays out of scorer logic.
- Runtime candidates are hidden in normal UI by default.
- Activated skills may be shown in normal UI.
- `/skills suggest` shows the full ranked candidate list for debug/authoring.
- `/skills suggest` and `/skills usage` are human-readable only in v1; no `--json`.
- Score bands are high `80..100`, medium `50..79`, low `25..49`, none `<25`.
- Runtime hint limits are `primary: 2`, `supporting: 5`, `conflict: 5`, `maybe: 0`.
- Existing prompt catalog cap stays unchanged in v1. Router hint augments it instead of replacing it.
- Signal collection reads path/status/extension signals only, not full diff or file contents.
- Signal collection targets `750ms` total budget, `500ms` git/touched-file budget, and `200` touched paths max.
- Plain `/skills doctor` stays quiet for valid legacy skills without `agents/xelyon.yaml`.
- `/skills doctor --routing` shows missing sidecar and routing metadata quality diagnostics.
- Doctor severity follows the Error / Warning / Info policy in section 9.2.
- Do not update current `.agents/skills` / `~/.agents/skills` as part of XELYON repo implementation.
- After Router / Doctor lands, use `/skills doctor --routing` to identify local skills that would benefit from `agents/xelyon.yaml`.

Implementation-time checks:

- Confirm import direction between `internal/skills`, `internal/skills/router`, `internal/skills/doctor`, `internal/agent`, and `internal/tools/skills`.
- If the exact package names cause import cycles, adjust package names while preserving the owner split.
- Keep `/status` skill routing summary out of v1 unless it falls out naturally without expanding scope.

## 17. Consultation Checklist Before Goal

Recommended default for first implementation:

- `agents/xelyon.yaml` optional
- XELYON-created skills should include `agents/xelyon.yaml` when authoring context is clear enough
- v1 sidecar schema is small and stable: `version`, `intents`, `role`, `read_only`, `modes`, `triggers`, `conflicts`, `activation`
- sidecar `version` must be `1`; missing/wrong/unsupported version warns and falls back to description-only routing
- invalid `agents/xelyon.yaml` warning, not full skill failure, unless core `name` / `description` is invalid
- unknown `agents/xelyon.yaml` fields warn, are ignored, and keep the skill valid
- skill catalog cache invalidates on `agents/xelyon.yaml` add/remove/content changes
- initial intent / mode / role / conflict vocabulary is defined in section 6.1; unknown values warn but keep the skill valid
- `/skills suggest <text>` as debug command
- `/skills suggest` renders full ranked list by default
- `/skills suggest` and `/skills usage` are human-readable only in v1; no `--json`
- score bands are high `80..100`, medium `50..79`, low `25..49`, none `<25`
- runtime hint uses category limits rather than one global max candidate count
- runtime hint limits are `primary: 2`, `supporting: 5`, `conflict: 5`, `maybe: 0`
- signal collection uses path/status/extension only, with no full diff/file content reads
- signal collection targets `750ms` total budget, `500ms` git/touched-file budget, and `200` touched paths max
- existing prompt catalog cap stays unchanged in v1
- runtime hint-only default
- architecture supports auto activation policy from v1
- Router / Doctor / usage ledger persistence are separate owners by default
- Doctor severity follows Error / Warning / Info policy from section 9.2
- public config includes only stable router controls: `skills.router.enabled`, `skills.router.activation`, `skills.router.usage_ledger`, `skills.router.usage_retention_days`
- `skills.router.activation` accepts `off` and `hint` in v1; `auto` is reserved and rejected or warned as unsupported
- usage ledger is v1 scope, local-only, default enabled, raw prompt/content-free, bounded retention, clearable
- usage ledger path is `~/.xelyon/skills/router/usage/<repo-key>.jsonl`, where `<repo-key>` is a cleaned project root hash
- direct `activate_skill` tool calls do not write usage ledger state; Agent runtime activation recording writes only when project root resolution succeeds
- `/skills usage` shows usage summaries; `/skills usage clear` clears the current repo ledger; `/skills usage clear --all` clears all router usage ledgers
- retention pruning runs on ledger write and before `/skills usage` / `/skills doctor --routing` reads, not during startup
- plain `/skills doctor` stays quiet for legacy valid skills without `agents/xelyon.yaml`
- routing sidecar warnings are shown in `/skills doctor --routing` or equivalent routing diagnostics mode
- normal UI hides router candidates by default
- normal UI can show activated skills

## 18. Goal Handoff Prompt

Use this when implementation is ready:

```text
/goal Implement docs/dev/skill-router-doctor-plan.md end to end.

Use docs/dev/skill-router-doctor-plan.md as the source of truth. Re-read it after resume or context compaction. Preserve all global contracts, especially no CODEX_HOME/external system skill discovery, legacy SKILL.md compatibility, read-only conflict handling, bounded prompt/runtime hints, and deterministic Doctor diagnostics.

Start with package/source owner confirmation, then implement `agents/xelyon.yaml` sidecar parsing, Doctor 2.0 static diagnostics, Router pure logic, debug command surface, runtime hint injection, local-only usage ledger, docs/config updates, and the required focused tests. Architecture must support auto activation policy from v1, but product default must remain hint-only until dogfood and explicit product decision. Keep Router, Doctor, and usage ledger persistence as separate owners; keep the existing prompt catalog cap unchanged; accept only sidecar `version: 1` and fallback to description-only routing for missing/wrong/unsupported version; make skill catalog cache invalidate on `agents/xelyon.yaml` add/remove/content changes; use the initial routing vocabulary from section 6.1; use score bands high `80..100`, medium `50..79`, low `25..49`, none `<25`; use runtime hint limits `primary: 2`, `supporting: 5`, `conflict: 5`, `maybe: 0`; keep `/skills suggest` and `/skills usage` human-readable only in v1; collect routing signals from path/status/extension only with no full diff/file content reads and with the documented timeout/cap limits; apply the Doctor Error / Warning / Info severity policy. The usage ledger must store no raw prompts, raw responses, diffs, file contents, or secrets; store repo-scoped JSONL under `~/.xelyon/skills/router/usage/<repo-key>.jsonl`; prune on write and before usage/doctor reads; expose `/skills usage`, `/skills usage clear`, and `/skills usage clear --all`. Final-A impact audit and Final-B post-implementation refactor are mandatory. Do not commit or push unless explicitly requested.
```
