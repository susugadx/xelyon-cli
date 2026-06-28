# Headless CI v1 Implementation Master Plan

この文書は Codex Goal でまとめて実装するための内部実装仕様書である。
公開 docs ではなく、Headless CI 強化の設計・実装・handoff の source of truth として使う。

## 0. Purpose

XELYON の headless mode を、単なる JSON 出力オプションではなく、CI / script / GitHub Actions から安定利用できる公開 automation surface として固める。

この master plan で扱う主な対象は以下である。

- headless JSON schema の安定化
- CI で扱いやすい prompt 入力
- tool failure / final check failure / read-only violation を表現できる failure contract
- `changed_files` / `commands` / `final_checks` の machine-readable summary
- review-only / dry-run の安全実行
- headless image 対応に向けた段階設計
- 実装中に必要な owner 分離、test foundation、post-implementation refactor

Goal の完了条件は、v1 の stable contract が実装・テスト・docs 更新まで閉じ、後続の Eval Runner / Batch backend がこの headless contract の上に乗れる状態になっていることとする。

## 0.1 Implementation Progress Checklist

この checklist は Headless CI v1 の進捗 source of truth として更新する。

- [x] Phase 0/1: contract foundation、`schema_version`、`input` metadata、`--prompt-file`、stdin 入力、Headless docs 最小更新を実装した。
  - Commit: `42265919 headless CI 入力契約を追加`
  - Verification: focused headless tests、`git diff --check`、`make verify-fast`、`make ci-check`
- [x] Phase 2-A: `failure_reason`、`exit_policy`、`recommended_exit_code` と `--exit-code-policy legacy|ci` を追加した。
  - Commit: `f89d7cde headless CI の終了コードポリシーを追加`
  - Scope: 既存 error path の分類と exit-code mapping の土台まで。`--fail-on-tool-error` は Phase 2-B に分ける。
  - Verification: focused headless / exit policy tests、`git diff --check`、`make ci-check`
- [x] Phase 2-B: `--fail-on-tool-error` を追加し、strict mode だけ tool failure を headless failure に昇格する。
  - Commit: `47127857 headless CI の tool error strict mode を追加`
  - Scope: explicit headless option、tool error promotion、CLI flag、CI exit code 4、docs update まで。
  - Verification: focused headless / cmd exit policy tests、`go test ./cmd ./internal/agent ./internal/climode -count=1`、`git diff --check`
- [x] Phase 3: `summary.changed_files`、`summary.commands`、`summary.final_checks` を source-of-truth 経由で追加する。
  - Scope: headless summary DTO / builder、bash command summary、headless final check execution and failure classification、docs update まで。
  - Review follow-up: headless final-check cancellation、normal-mode final-check API deadline 分離、`summary.commands` cancellation classification まで修正した。
  - Verification: focused headless summary / final check / cmd exit policy tests、affected package tests、`git diff --check`、local review
- [x] Phase 4: `--read-only` / `--dry-run` no-mutation safety mode を追加する。
  - Scope: headless/JSON 専用 flags、`--dry-run` strict alias、config bootstrap read-only loader、provider tool definition からの write tool / MCP / sub-agent exclusion、read-only 時の session history / change history / audit log storage / MCP bootstrap / startup ProjectMap cache / skill-router git status signal / skill-router usage ledger 抑止、実行直前 deny、strict 時の `read_only_violation` 昇格、docs update まで。
  - Verification: focused cmd / internal/agent / internal/tools tests、affected package tests、`git diff --check`
- [x] Phase 5: public docs と GitHub Actions examples を現行 schema / flags に合わせる。
  - Scope: `docs/commands.md` の Headless reference 整理、`docs/ci.md` の GitHub Actions PR smoke 例追加、README からの入口追加まで。CLI flags、JSON schema、exit code、runtime behavior は変更しない。
  - Verification: docs flag drift scan、`git diff --check`
- [ ] Phase 6: headless image support を、scope が制御できる場合だけ実装する。
- [x] Phase Final-A: impact audit / review-hole sweep を実施する。
  - Scope: read-only startup persistence、provider-history raw output artifact materialization、startup/warmup no-write surfaces、affected caller paths の review-hole sweep まで。
  - Verification: focused cmd / internal/agent / providerhistory tests、affected package tests、`git diff --check`、local review
- [x] Phase Final-B: mandatory comprehensive refactor including tests を実施する。

## 1. Current State / Implemented Preconditions

現状の headless は「まったく使えない」状態ではない。

- `--headless` または `--output-format json` は headless mode に解決される。
- headless 実行は stdout に JSON を出力する。
- `status: "error"` の headless result は CLI error として返る。
- provider setup 不足は `provider_setup_required` の headless JSON として stdout に出せる。
- error JSON 後に Cobra usage を stderr に混ぜない regression test がある。
- JSON には `schema_version`, `status`, `provider`, `model`, `response`, `input`, `summary`, `tool_calls`, `tokens`, `web_search`, `duration_ms`, `timestamp`, `error`, `cost`, `pricing_unavailable` がある。
- `--prompt-file <path>`、`--prompt-file -`、headless / JSON mode での bare `-` stdin 入力に対応している。
- tool loop limit、API error、token/cost、web search usage、stdout pure JSON などの focused tests がある。

一方で、CI 用の公開 contract としては以下が弱い。

- review-only / dry-run で workspace mutation を禁止できない。
- headless image は現在 `--headless` / `--output-format json` と併用禁止。

## 2. Shared Contract Preflight

### Shared change decision

YES. この計画は shared contract change である。

理由:

- CLI flags / args / stdin / file input の user-visible behavior を変える。
- stdout JSON schema という serialized public contract を拡張する。
- process exit code と failure policy を CI 向け contract として定義する。
- runtime task ledger、tool execution result、final checks、docs/tests に波及する。
- `--read-only` / `--dry-run` は tool visibility / execution policy / mutation tracking に影響する。

### Contracts to change

- Headless JSON に `schema_version` を追加する。
- Headless JSON に `input`, `summary`, `failure_reason`, `exit_policy`, `recommended_exit_code` を追加する。
- `--prompt-file` と stdin 入力を headless / JSON mode の入力 source として追加する。
- `--fail-on-tool-error` により tool failure を headless status / exit result に反映できるようにする。
- `--exit-code-policy` により legacy と CI 向け detailed exit code を選べるようにする。
- `--read-only` / `--dry-run` により write-capable tool execution を禁止または apply しない mode を提供する。
- final checks の結果を JSON summary に含める。
- headless image を将来段階で JSON contract に載せる。

### Contracts not to change

- 既存の top-level fields は削除しない。
- `status: "success"` / `status: "error"` の文字列は維持する。
- provider setup 不足と unknown provider は混同しない。
- stdout は headless JSON だけを出す。
- stderr に Cobra usage や human-readable progress を混ぜない path を維持する。
- API provider request / provider-facing history / raw session persistence の既存 contract は、この計画の v1 では変更しない。
- Batch API / Eval Runner はこの計画では実装しない。

### Backward compatibility policy

- v1 は additive JSON schema とする。
- 既存 fields の意味を変えない。
- default process exit behavior は `legacy` policy として維持し、詳細 exit code は opt-in にする。
- CI 利用者向け docs では `--exit-code-policy ci` を推奨する。
- unknown new fields を ignore できる consumer を前提に、schema version は breaking change の判断材料として使う。

### Refactor decision

SHOULD.

理由:

- `cmd/root.go` に CLI mode resolution、input validation、provider setup JSON、headless execution、exit error mapping が同居している。
- `internal/agent/headless.go` は result struct owner だが、今後 `schema_version`, summary, failure policy, image metadata まで入ると肥大化しやすい。
- `internal/agent/headless_runner.go` は runner orchestration owner だが、CI summary / policy formatting まで持つと責務が混ざる。

ただし、現時点で owner は説明できるため、実装前 blocker の MUST refactor ではない。
Phase 0 で小さく owner を分け、実装後 Final-B で構造整理を必須にする。

## 3. Global Contracts

- Headless stdout は常に machine-readable JSON だけにする。
- Human-readable warning / diagnostics / progress は stderr へ出すか、headless では suppress する。
- 既存 JSON fields は削除・rename しない。
- New fields are additive.
- `schema_version` は top-level に置く。
- JSON 内の failure classification は process exit code よりも詳細な source of truth とする。
- Process exit code は `--exit-code-policy` の結果であり、JSON 内にも `recommended_exit_code` として出す。
- `--read-only` は workspace mutation を実行しない hard safety mode として扱う。
- `--dry-run` は user-visible intent と JSON contract では no-write mode とする。内部実装で read-only と同じ policy に寄せるか、将来 apply preview を分けるかは Open decisions で扱う。
- Tool result raw output や provider-facing history の保存形式は、この plan では変更しない。
- `changed_files` の source of truth は実行済み tool の `tools.FileChange` / runtime task ledger とし、AI の自然文要約から推測しない。
- `commands` の source of truth は bash tool execution observation と final checks observation とし、AI の自然文から推測しない。
- `final_checks` の source of truth は `runFinalCheckCommands` / `taskstate.TestObservation` とする。
- Failure reason は enum-like string とし、consumer が分岐できる粒度で固定する。
- Unknown / unsupported / unsafe mode は成功に丸めない。

## 4. Non-goals

- Batch API は実装しない。
- Eval / Regression Runner は実装しない。
- Provider capability matrix 全体は実装しない。ただし docs/doctor と矛盾しない hooks は残す。
- provider-native request schema は変更しない。
- session persistence format は変更しない。
- existing interactive TUI / legacy `--no-tui` の user experience は変更しない。
- setup / auth flow 全体の再設計はしない。
- headless image は v1 stable contract の後続 phase として扱い、最初の CI v1 gate を塞がない。

## 5. Source Findings

### CLI mode / command owner

- `cmd/root.go` が root flags、mode dispatch、headless execution、provider setup JSON を持つ。
- `internal/climode/mode.go` が `--headless` / `--output-format json` / `--image` の mode validation を持つ。
- 現在 `--image` は JSON output と併用禁止である。
- `cmd/root_test.go` に JSON mode path、provider setup JSON、unknown provider distinction、stdout pure JSON、headless error JSON の regression tests がある。

### Headless result / runner owner

- `internal/agent/headless.go` が `HeadlessResult`, `ToolCallResult`, `TokenUsage`, `WebSearchUsage`, `ErrorInfo` を持つ。
- `internal/agent/headless_runner.go` が `RunHeadlessWithConfig`、tool loop、tool call execution、stats attachment を持つ。
- `internal/app/entrypoints.go` は `agent` の headless types / runner を app layer へ re-export している。
- `internal/agent/headless_integration_test.go` と `internal/agent/headless_test.go` に headless behavior tests がある。

### Tool execution / failure owner

- `internal/tools/execute_core.go` の `ExecutionResult` が `Result`, `Change`, `Observation`, `StartedAt`, `Duration`, `Error` を持つ。
- `tools.IsErrorResult` は trimmed output prefix `Error:` を failure と判定する。
- `internal/agent/headless_runner.go` の `isHeadlessToolCallSuccess` は `ExecutionResult.Error` と `tools.IsErrorResult` を source of truth としている。
- 現在 tool failure があっても、最終 assistant response が tool call なしなら headless result は `success` になり得る。

### Changed files / task ledger owner

- `internal/agent/recorded_task_changes.go` は `changeStack` から current task の changed files を snapshot できる。
- `internal/taskstate` は runtime task ledger を持ち、`ChangedFiles`, `TouchedFiles`, `LastFailedTests`, `LastPassedTests` などを snapshot できる。
- `internal/agent/mutation_tracker.go` は tool result から task ledger と change stack を更新する owner である。
- headless runner は tool execution result を `MutationTracker` へ渡し、changed files summary は task ledger snapshot を source of truth にする。

### Final checks owner

- `internal/agent/final_check_commands.go` が `final_checks.commands` を実行し、`taskstate.TestObservation` を記録する。
- headless は変更ファイルがある最終 no-tool 応答時に `runFinalCheckCommands` を再利用し、結果を `summary.final_checks` に出す。
- `final_checks.commands` は `XELYON_CHANGED_FILES` を env として渡す。

### Docs owner

- `docs/commands.md` に headless の使用例と JSON output 例がある。
- `docs/providers.md` に headless JSON の cost / web_search の説明がある。
- `README.md` は final checks と MCP headless config を説明している。
- Headless CI v1 の公開 docs は `docs/commands.md` と README を最小更新対象にする。

## 6. Responsibility Boundaries

### CLI input / mode owner

- `cmd/root.go`: flags registration, root command dispatch, stdout/stderr separation, process exit mapping.
- `internal/climode`: mode validation and conflict errors.
- New focused owner candidate: `cmd/headless_input.go` or `internal/climode/headless_input.go` for prompt source resolution.

### Headless contract owner

- `internal/agent/headless.go`: public JSON shape and constructor helpers.
- New focused owner candidate: `internal/agent/headless_summary.go` for summary construction.
- New focused owner candidate: `internal/agent/headless_policy.go` for failure policy result classification.

### Runtime execution owner

- `internal/agent/headless_runner.go`: one headless task execution, tool loop, provider requests.
- It should not become the owner of CLI flag parsing or process exit code policy.

### Tool observation / mutation owner

- `internal/tools`: tool execution result and file change source of truth.
- `internal/agent/mutation_tracker.go`: tool observation, project map invalidation, change stack, task ledger update.
- `internal/taskstate`: runtime summary data model for changed files / command-like observations / final check observations.

### Final checks owner

- `internal/agent/final_check_commands.go`: final check command execution and observation.
- If headless triggers final checks, reuse this owner rather than duplicating shell execution in headless runner.

### Docs / generated owner

- `docs/commands.md`: command behavior and headless JSON schema examples.
- `README.md`: high-level CLI automation usage.
- `docs/providers.md`: provider-specific cost / web_search caveats.

## 7. Implementation Priority

1. Phase 0: contract/test foundation and owner cleanup.
2. Phase 1: schema version and prompt input sources.
3. Phase 2: failure policy and exit-code policy.
4. Phase 3: summary JSON for changed files, commands, final checks.
5. Phase 4: read-only / dry-run safety mode.
6. Phase 5: docs and GitHub Actions examples.
7. Phase 6: headless image compatibility design and implementation if scope remains controlled.
8. Phase Final-A: impact audit / review-hole sweep.
9. Phase Final-B: mandatory comprehensive refactor including tests.

## 8. Phase 0: Contract Foundation / Owner Cleanup

### Purpose

Headless CI featuresを `cmd/root.go` と `internal/agent/headless_runner.go` へ直接積み増さないため、最小限の土台を作る。

### Non-goals

- JSON schema の user-visible field をこの phase だけで増やさない。
- exit code を変更しない。
- final checks の実行 path を変更しない。

### Design contract

- CLI input parsing / exit mapping / JSON shape / runtime summary construction を分ける。
- `cmd/root.go` は orchestration に留める。
- headless runner は runtime execution owner に留める。
- result struct と summary builder は test しやすい pure-ish owner に寄せる。

### Implementation owner candidates

- `cmd/root.go`
- `cmd/execution_mode.go`
- new `cmd/headless_input.go`
- `internal/agent/headless.go`
- new `internal/agent/headless_summary.go`
- new `internal/agent/headless_policy.go`
- `internal/agent/headless_runner.go`
- `cmd/root_test.go`
- `internal/agent/headless_test.go`

### Tests

- Existing headless JSON tests stay green.
- New owner helpers have focused unit tests.
- `go test ./cmd ./internal/agent ./internal/climode -run 'Headless|OutputFormat|Mode' -count=1`

## 9. Phase 1: Schema Version and Prompt Input Sources

### Purpose

CI から長い prompt を扱えるようにし、JSON schema の互換性判断を可能にする。

### Design contract

- Add top-level `schema_version`.
- Initial value: `"xelyon.headless.v1"`.
- Add top-level `input` object.
- `input.source` enum candidates: `args`, `prompt_file`, `stdin`.
- `input.prompt_file` is present only for `prompt_file`.
- `input.bytes` may be included as prompt byte length, not prompt body.
- Never echo full prompt body into JSON by default.

### CLI behavior

- Add `--prompt-file <path>` for headless / JSON mode.
- Support stdin with `-` or explicit `--prompt-file -`.
- Keep positional query support.
- Reject ambiguous multiple prompt sources.
- Empty input returns usage/config style error before provider request.
- File read errors produce machine-readable headless JSON when possible.

### Safety gates

- Prompt file path is external input. Resolve and read it through a single owner.
- Do not expand shell syntax in prompt file paths.
- Do not read directories.
- Do not silently read stdin in interactive mode.
- Do not block forever on stdin unless stdin was explicitly selected.

### Tests

- Positional query still works.
- `--prompt-file` reads prompt content.
- `--prompt-file -` reads stdin.
- Positional query plus `--prompt-file` errors.
- Empty stdin errors.
- Missing prompt file returns JSON error in headless mode.
- JSON includes `schema_version` and `input.source`.
- JSON does not include full prompt body.

## 10. Phase 2: Failure Policy and Exit Code Policy

### Purpose

CI が `status` だけでなく failure reason と process exit behavior を安定的に扱えるようにする。

### Design contract

Add JSON fields:

```json
{
  "failure_reason": "tool_error",
  "exit_policy": "ci",
  "recommended_exit_code": 4
}
```

`failure_reason` is omitted or empty on success.

Failure reason enum candidates:

- `usage_error`
- `config_error`
- `provider_setup_required`
- `api_error`
- `cancelled`
- `tool_loop_limit`
- `tool_error`
- `final_check_failed`
- `read_only_violation`
- `unsupported_capability`
- `unknown_error`

### Exit code policy

Add `--exit-code-policy`.

Policy values:

- `legacy`: preserve current behavior as much as possible. Success = 0, any command error = 1.
- `ci`: detailed process exit code for CI.

Recommended CI mapping:

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | unknown/general failure |
| 2 | usage error / invalid flags / invalid prompt source |
| 3 | config or provider setup required |
| 4 | tool error when `--fail-on-tool-error` is enabled |
| 5 | final check failed |
| 6 | API/provider runtime error |
| 7 | cancelled / timeout |
| 8 | read-only violation |
| 9 | unsupported capability |

### Tool error policy

Add `--fail-on-tool-error`.

Default:

- Existing behavior remains: a failed tool call is represented in `tool_calls[].success=false`, but final assistant response can still make overall status success.

With `--fail-on-tool-error`:

- If any tool call has `success=false`, final result becomes `status: "error"`.
- `error.type` and `failure_reason` become `tool_error`.
- Response can still be included if available, but CI must not treat it as success.

Phase 2-B implementation brief:

- Shared change: YES. This adds a user-visible CLI flag and changes headless JSON / process exit behavior only when strict tool failure policy is explicitly enabled.
- Contract owner: `internal/agent` remains the headless JSON and failure policy source of truth. `cmd` only parses `--fail-on-tool-error` and passes the selected headless run option.
- Runtime owner: `internal/agent/headless_runner.go` owns tool execution observations and should be the place that detects failed `ToolCallResult` entries for strict-mode promotion.
- Recommended shape: add an additive headless run options type, keep `RunHeadlessWithConfig` as the default-compatible wrapper, and use an options-aware entrypoint for CLI headless mode.
- Contract additions: add typed `HeadlessErrorTypeToolError` and `HeadlessRunOptions{FailOnToolError bool}` in `internal/agent`, with `internal/app` aliases only where `cmd` needs them.
- Default compatibility: without `--fail-on-tool-error`, failed tool calls remain in `tool_calls[].success=false` and the overall result can still be success.
- Strict compatibility: with `--fail-on-tool-error`, an otherwise-successful headless result containing at least one failed tool call is promoted to `status: "error"`, `error.type: "tool_error"`, `failure_reason: "tool_error"`, and `recommended_exit_code` follows the selected `exit_policy`.
- Error precedence: Phase 2-B should not reclassify pre-existing API, cancelled, config, provider setup, or tool loop limit errors just because a failed tool call was observed earlier.
- Serialization: no schema version bump; this is additive and uses existing Phase 2-A fields.
- Out of scope: final checks, read-only / dry-run, summary extraction, and changing tool success detection rules.

### Tests

- Default tool error behavior remains backward compatible.
- `--fail-on-tool-error` converts a failed tool call into headless error.
- `legacy` policy returns current non-zero style.
- `ci` policy returns mapped code.
- JSON includes `failure_reason`, `exit_policy`, `recommended_exit_code`.
- Provider setup required maps to setup/config failure without becoming unknown provider.

## 11. Phase 3: Summary JSON for changed_files / commands / final_checks

### Purpose

CI report や markdown summary を作れるだけの structured result を headless JSON に出す。

### Design contract

Add top-level `summary` object:

```json
{
  "summary": {
    "changed_files": ["internal/example.go"],
    "commands": [
      {
        "command": "go test ./internal/agent -run TestHeadless -count=1",
        "exit_code": 0,
        "status": "passed",
        "source": "tool"
      }
    ],
    "final_checks": [
      {
        "command": "make verify-fast",
        "exit_code": 0,
        "status": "passed"
      }
    ]
  }
}
```

### Source of truth

- `changed_files`: runtime task ledger / `tools.FileChange`.
- `commands`: tool observation for bash-like commands and test-like commands.
- `final_checks`: `runFinalCheckCommands` observations.
- Do not parse assistant final text for these fields.

### Safety gates

- Large command output is not included by default.
- If output excerpts are added, they must be bounded and safe.
- Do not include secrets/env values from command output.
- Paths should be normalized relative to repo root when possible.
- Summary ordering should be deterministic and first-observed where practical.

### Tests

- Editing tool populates `summary.changed_files`.
- Bash test command populates `summary.commands`.
- Non-test shell command can be summarized without pretending it is a final check.
- final checks populate `summary.final_checks`.
- Large output is omitted or bounded.
- No assistant text parsing is required for summary.

Phase 3 implementation notes:

- `internal/agent/headless_summary.go` owns summary construction and command/final-check DTO conversion.
- Headless tool execution records through `MutationTracker`, so `summary.changed_files` comes from the task ledger rather than direct runner state.
- Headless runs configured `final_checks.commands` once when a final no-tool response arrives after observed file changes.
- Final check failures are promoted to `status:"error"` with `error.type` / `failure_reason` = `final_check_failed`; output text is not included in summary.
- Headless final-check parent cancellation is promoted to `cancelled`; per-command timeout remains `final_check_failed`.
- Normal-mode final checks use request explicit-cancel context without inheriting the API request deadline.
- `summary.commands` classifies bash cancellation markers and tool errors as `failed` / `-1` rather than `passed` / `0`.

## 12. Phase 4: Read-only / Dry-run Safety Mode

### Purpose

OSS 利用者が PR smoke / review-only CI で workspace mutation を禁止できるようにする。

### Design contract

Add flags:

- `--read-only`
- `--dry-run`

Initial v1 behavior:

- `--read-only` means write-capable tools and unclassified execution surfaces are unavailable or denied.
- `--dry-run` is an alias or near-alias for no-write execution in headless CI v1.
- If the model attempts a write-capable tool, result should be a structured denied tool result.
- With `--fail-on-tool-error`, denied write attempt becomes `read_only_violation`.
- Without `--fail-on-tool-error`, denied write is still visible in `tool_calls[]` and `summary`, and should not mutate files.

### Owner requirements

- Tool visibility / execution policy should be owned by runtime/tool visibility policy, not ad hoc checks in each write tool.
- Write tool classification should use existing `tools.IsWriteTool` or a single source of truth.
- Bash is fail-closed in read-only headless v1; shell command read-only classification is not used for this mode.

### Safety gates

- No production hook that pretends to write.
- No sleep/retry workaround.
- No global test-only bypass.
- No path mutation in dry-run.
- File system before/after checks should be used in tests where practical.

### Tests

- `--read-only` allows read_file / search_code / gather_context.
- `--read-only` denies apply_patch / write_file / delete_file / str_replace.
- Bash commands are denied, including commands that look read-only under the normal shell policy.
- Heuristic-bypass bash examples such as command substitution and `find . -delete` must not execute.
- Denied mutation does not change files.
- JSON has `failure_reason: "read_only_violation"` under strict policy.
- First-run HOME does not get `~/.xelyon/config.yaml` / `~/.xelyon/AGENTS.md` under `--headless --read-only` or `--headless --dry-run`; normal headless keeps existing bootstrap behavior.
- First-run HOME does not get `~/.xelyon/history`, `~/.xelyon/changes`, or `~/.xelyon/audit` from read-only startup, including when `XELYON_AUDIT_LOG=1`.
- XML-form `<mcp_...>` examples inside Markdown code blocks and unmatched open tags remain final text, not denied attempts.
- Strict read-only violation remains `read_only_violation` even if a denied call is followed by tool loop limit.

Phase 4 implementation notes:

- `cmd/root.go` parses `--read-only` / `--dry-run` and normalizes both to `HeadlessRunOptions.ReadOnly`.
- `--read-only` / `--dry-run` are usage errors outside `--headless` / `--output-format json`.
- `cmd` selects `cliruntime.LoadConfigSelectionReadOnly` before runner startup when headless read-only / dry-run is active, so missing config bootstrap does not create `~/.xelyon/config.yaml` or the default global `AGENTS.md`.
- `internal/agent/headless_runner.go` excludes provider-visible write tools using `tools.IsWriteTool`, plus bash, `run_skill_script`, MCP exported tools, and sub-agent tools, without changing normal headless edit surface behavior.
- Read-only headless does not initialize session history storage, change storage, or file audit logging, so first-run HOME does not get `~/.xelyon/history`, `~/.xelyon/changes`, or `~/.xelyon/audit` from startup.
- Read-only headless uses a runtime config copy with `mcp.headless=false`, so MCP server processes are not started and `~/.xelyon/mcp.json` is not created by MCP bootstrap during read-only runs.
- Read-only headless uses the same runtime config copy with `project_map.enabled=false`, so startup project-map build, prompt injection, and `~/.xelyon/cache/projectmap` persistence do not run during read-only runs.
- Read-only headless uses the same runtime config copy with `lsp.enabled=false`, so LSP client startup and warmup processes do not run during read-only runs.
- Read-only headless sets the internal runtime read-only policy, so skill-router runtime hints do not run `git status --porcelain` and skill-router recommendation / activation usage ledger writes are skipped.
- Read-only headless propagates the same internal runtime read-only policy to provider-history projection, so raw output artifact candidates can be reported but artifact materialization and artifact-backed provider-facing replacements are not applied.
- MCP exported tools are fail-closed in read-only mode until a later per-server/tool read-only capability contract exists.
- `spawn_agent` / `wait_agent` are fail-closed in read-only mode until sub-agent read-only inheritance is designed.
- Read-only headless uses a non-persistent ToolCache so cache load/save cannot create, overwrite, or remove `.xelyon/cache/tool_cache.json`.
- Read-only headless skips startup dev artifact cleanup so old `.xelyon/artifacts/*` files are not removed during no-write runs.
- Execution-time guard rejects `tools.IsWriteTool(tc.Tool)`, all `bash` tool calls, and `run_skill_script` before invoking the real tool.
- Execution-time guard also rejects `mcp_*` tool calls before registry execution, including manually emitted tool JSON.
- Read-only headless also rescues XML-form `<mcp_...>` attempts as synthetic denied tool calls before an unknown XML tag can become a benign final response; candidate detection lives in `internal/tools` and only returns matching-close-tag XML outside Markdown code blocks.
- Execution-time guard rejects `spawn_agent` / `wait_agent` before registry execution, including manually emitted tool JSON.
- Denied calls are recorded as failed `tool_calls[]`; denied bash also appears in `summary.commands` as `failed` / `-1`.
- Denied calls do not call `MutationTracker` and therefore do not record file mutations or invalidate project map as if a write occurred.
- Strict promotion precedence for otherwise-successful results is `cancelled` -> `final_check_failed` -> `read_only_violation` -> `tool_error`; loop-limit results also preserve `read_only_violation` when the loop was caused after a denied read-only attempt.

## 13. Phase 5: Docs and GitHub Actions Examples

### Purpose

Headless CI v1 を OSS 利用者がすぐ使えるようにする。

### Docs targets

- `docs/commands.md`
- `README.md`
- `docs/providers.md` if cost/web_search wording changes
- optional new `docs/ci.md` if examples become too large

### Required docs

- Basic one-shot JSON usage.
- `--prompt-file` and stdin usage.
- `--exit-code-policy legacy|ci`.
- `--fail-on-tool-error`.
- `--read-only` / `--dry-run`.
- JSON schema example with `schema_version`.
- GitHub Actions PR smoke example.
- Nightly eval is not implemented yet, but docs may mention it as future-compatible only if clearly marked.

### Tests / docs sync

- Add docs contract tests only if the repo already has similar command docs tests for this surface.
- Avoid brittle snapshot of full JSON; prefer required field examples.

## 14. Phase 6: Headless Image Support

### Purpose

Headless CI でも screenshot / error image / UI artifact を扱えるようにする。

### Current constraint

`--image` is currently incompatible with `--headless` / `--output-format json`.

### Design contract

- Do not force this into initial v1 if it destabilizes CLI mode contract.
- When implemented, JSON must include bounded image metadata, not raw image bytes.
- Suggested `input.image` fields:
  - `path`
  - `mime_type`
  - `bytes`
  - `provider_supported`
- Unsupported provider should return `unsupported_capability`, not generic API error.

### Owner candidates

- `internal/climode/mode.go` for mode validation.
- `cmd/root.go` or focused input owner for CLI image path acceptance.
- `internal/agent/agent_run.go` image loading logic may need reuse or extraction.
- Provider capability check should reuse existing provider image support contract.

### Tests

- `--headless --image` with supported provider runs image path.
- Unsupported provider returns structured JSON error.
- Missing image returns structured JSON error.
- JSON does not include raw base64 image.
- Existing interactive and one-shot image behavior remains unchanged.

## 15. Mode / Policy / Defaults

### Default behavior

- Existing `--headless` and `--output-format json` remain valid.
- Existing positional query remains valid.
- Default exit policy remains `legacy`.
- Default tool error policy remains backward compatible.
- `schema_version` is always present once v1 lands.

### CI recommended behavior

Recommended command shape:

```sh
xelyon --headless \
  --prompt-file prompt.md \
  --exit-code-policy ci \
  --fail-on-tool-error \
  --read-only
```

### Compatibility behavior

- Consumers that only read `status` and `response` continue to work.
- Consumers that require stable CI failure classification should read `schema_version`, `status`, `failure_reason`, and `recommended_exit_code`.

## 16. Config / Docs / Generated Metadata Surface

No config key is required for v1.

Flags are preferred over config for CI surface because:

- CI behavior should be explicit in workflow YAML.
- Per-run policy is safer than global user config.
- It avoids silently changing local interactive behavior.

If a future config key is added, that is a separate `xelyon-config-contract-change` task.

## 17. Report / Status / Observability

Headless JSON should make these observable:

- input source and prompt length
- provider and model
- total duration
- token usage and cost
- web search usage if present
- all tool calls with success/failure
- changed files
- commands run by tools
- final checks
- failure reason
- exit policy and recommended exit code
- pricing unavailable
- read-only/dry-run policy state

Do not include:

- full prompt body
- raw command output by default
- raw image bytes
- secrets / env values
- provider raw response body

## 18. Tests

### Focused tests

- `go test ./cmd -run 'Headless|OutputFormat|PromptFile|ExitCode|ReadOnly' -count=1`
- `go test ./internal/climode -run 'Mode|OutputFormat|Image' -count=1`
- `go test ./internal/agent -run 'Headless|FinalCheck|ReadOnly|ToolError|Summary' -count=1`
- `go test ./internal/tools -run 'XML.*Tool|ParseToolCalls|XMLRescue' -count=1`
- `go test ./internal/taskstate -run 'ChangedFiles|TestObservation|Snapshot' -count=1`

### Broader tests

- `go test ./cmd ./internal/agent ./internal/climode ./internal/taskstate -count=1`
- `make verify-fast`
- `git diff --check`

### Final gate

Before commit, run:

```sh
make ci-check
```

## 19. Final-A: Impact Audit / Review-hole Sweep

After feature implementation, run an impact audit before final reporting.

Check:

- stdout remains pure JSON.
- stderr does not contain Cobra usage after headless JSON errors.
- provider setup required remains distinct from unknown provider.
- existing JSON consumers still have old fields.
- `schema_version` is present in all headless result paths, including pre-run setup errors.
- prompt file / stdin failures are structured.
- tool failure strict policy does not break default behavior.
- read-only mode actually prevents file changes.
- final checks summary cannot leak large output or secrets by default.
- docs examples match current flags and schema.
- image compatibility, if implemented, does not break existing image one-shot / interactive paths.

If a counterexample is found, fix it in this phase rather than leaving a TODO.

## 20. Final-B: Mandatory Comprehensive Refactor Including Tests

Final-B is mandatory, not optional.

After tests pass once, review production and test diffs for owner quality.

Minimum checklist:

- Inventory production diff and test diff.
- List new/changed helper, struct, flag, enum string, fallback, summary field, fixture, fake, table test.
- Decide `MUST` / `SHOULD` / `NO` refactor for each clustered area.
- Perform behavior-preserving MUST refactors.
- If tests/fixtures/fakes/table tests became bulky or duplicated, run `test-boundary-refactor`.
- If headless result/policy/summary all accumulated in one file, split by owner.
- If `cmd/root.go` gained too much orchestration, extract focused helpers.
- If failure reason strings are duplicated, consolidate source of truth.
- If summary extraction parses assistant text, remove it and return to ledger/tool observation sources.
- Run focused tests and broader tests again.

Final-B report must include:

- owner map
- MUST/SHOULD/NO refactor judgment
- behavior-preserving refactors performed
- concrete remaining debt by file/function, if any

## 21. Implementer Freedom

The implementer may choose final names and exact file splits based on current source structure.

Allowed implementation choices:

- exact helper/type names
- whether headless input helper lives in `cmd` or `internal/climode`
- whether summary builder lives in `internal/agent/headless_summary.go` or nearby owner file
- whether exit code mapping uses a small internal package or cmd-local type
- exact JSON struct layout as long as documented fields and compatibility contracts hold

Do not change:

- top-level `schema_version` requirement
- additive JSON compatibility
- stdout pure JSON contract
- provider setup vs unknown provider distinction
- read-only no-mutation contract
- source-of-truth rules for summary
- Final-A / Final-B requirement

## 22. Open Decisions

- Should `--dry-run` be a strict alias for `--read-only` in v1, or should it eventually mean "preview planned writes"? Recommendation: v1 alias/no-write, future preview mode separate.
- Should detailed process exit codes become default in a future major version? Recommendation: keep `legacy` default for now, document `ci` policy.
- Final checks now run automatically in headless when changed files are observed and `final_checks.commands` is configured; implementation reuses `runFinalCheckCommands`.
- Should command summaries include bounded output excerpts? Recommendation: omit by default in v1; add opt-in later.
- Should headless image be part of the first implementation Goal or a follow-up Goal? Recommendation: keep it in the master plan but do not block Headless CI v1 stable contract on image support.

## 23. Goal Handoff Prompt

```text
/goal Implement docs/dev/headless-ci-v1-master-plan.md end to end.

Use docs/dev/headless-ci-v1-master-plan.md as the source of truth. Start with Phase 0 pre-implementation refactor / test foundation, then implement the planned sections, then run the impact audit and mandatory post-implementation refactor described in the plan.

Final-B is mandatory, not optional. After tests pass, run post-implementation-refactor; if tests, fixtures, fakes, table tests, or assertion helpers changed, also run test-boundary-refactor. Same-file repeated findings, large files, or generic helpers mixing semantic roles trigger a file/test split audit before strict review.

If the plan and latest source structure conflict, preserve the safety contracts and adapt to the existing owner boundaries. Re-read the plan after resume or context compaction. Do not commit or push unless explicitly requested.
```
