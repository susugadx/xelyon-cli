# MCP Approval Policy Implementation Plan

この文書は Codex Goal でまとめて実装するための内部実装仕様書である。
公開 docs ではなく、MCP approval policy 実装前の設計・handoff の source of truth として使う。

## 0. Purpose

MCP tool は外部 MCP server が公開する動的 tool であり、XELYON built-in tool と同じ安全分類だけでは信頼境界を表せない。
この計画では、`~/.xelyon/mcp.json` に MCP 専用の approval policy を追加し、server/tool 単位で `confirm | auto | deny` を明示できるようにする。

Goal で完了とみなす条件は次の通り。

- MCP approval mode `confirm | auto | deny` が `mcp.json` で設定できる。
- 未設定の MCP tool は必ず `confirm` として扱われる。
- `execution.mode: full_auto` や `--auto-approve` でも、MCP は明示 `auto` なしに自動実行されない。
- `deny` の MCP tool は provider/prompt/registry surface から除外され、stale call でも実行されない。
- headless では `confirm` tool を実行せず、approval required として扱う。
- docs/tests/generated/config surface が runtime contract と一致する。

## 1. Current State / Implemented Preconditions

現在の MCP 実装は以下を前提にしている。

- `internal/mcp.ServerConfig` は `command`, `args`, `env`, `disabled`, `tools`, `startupTimeoutSeconds`, `toolTimeoutSeconds` を持つ。
- `tools.include` / `tools.exclude` は MCP server が公開する raw tool name を対象にする。
- `internal/mcp.Manager` が接続、`tools/list`、filter、name collision skip、tool metadata commit を管理する。
- `internal/agent` が `mcp.MCPTool` を `mcptool.Definition` と provider tool definition に変換する。
- `internal/mcptool.Wrapper.Run` は args validation 後、`ConfirmWithAutoApproveDecisionAndOptions` で実行確認を行う。
- dynamic MCP tool は `common.GetToolSafety` の fallback により `SafetyLow` として扱われる。
- `SafetyLow` は balanced/trusted では基本 confirm、`full_auto` や `--auto-approve` では auto approve され得る。
- headless runner は runtime auto approve を有効化するため、現状のままだと MCP も global auto approve に巻き込まれ得る。

今回の前提として、既に以下は実装済みである。

- MCP timeout は startup 120 秒、tool call 600 秒の長め default を持つ。
- MCP tool surface は tool count / token / schema bytes で bounded。
- MCP large result は履歴/provider/UI に入る前に compact される。
- empty/null/invalid MCP schema は empty object schema に fallback する。
- `properties.<name>.required=true` の旧互換は削除済みで、required source of truth は top-level `required` だけである。

## 2. Global Contracts

- MCP approval policy の値は `confirm`, `auto`, `deny` の 3 つだけにする。
- `prompt` という値は使わない。承認確認を意味する値は `confirm` に統一する。
- MCP approval 未設定時の default は `confirm`。
- global `execution.mode`, `--auto-approve`, legacy `tool_confirm` は、MCP を未設定のまま `auto` に引き上げてはならない。
- MCP を自動実行できるのは、server-level または tool-level で明示 `auto` された場合だけ。
- `deny` は `auto` / `confirm` より強い。provider/prompt/registry に出さず、stale call でも実行しない。
- `disabled: true` は approval より強く、server 全体を無効にする。
- `tools.include` / `tools.exclude` は公開候補の filter、`approval` / `toolApprovals` は実行許可 policy として責務を分ける。
- `toolApprovals` の key は MCP server が公開する raw tool name を使う。`mcp_<server>_<tool>` の exported name ではない。
- server-level `approval: "deny"` は server 全体の broad deny とし、`toolApprovals` では解除しない。
- invalid approval value は warning を出して `confirm` に fallback する。
- 既存 `mcp.json` の migration は不要。未設定は `confirm` として読む。
- 既存 config が `full_auto` に依存して MCP を自動実行していた場合は挙動が変わる。これは意図的な security tightening として扱う。
- secret/env/output redaction、large result guard、timeout/cancel contract はこの変更で弱めない。

## 3. Non-goals

- MCP server の tool description から read/write/network/browser/secret 権限を自動推定しない。
- MCP protocol 自体に権限スコープ拡張を実装しない。
- OAuth/token/sandbox/container isolation は扱わない。
- `tools.include` / `tools.exclude` の意味を変えない。
- built-in tool の approval model 全体を再設計しない。
- provider schema に approval metadata を載せない。
- `/mcp status` はこの plan の完了条件に含めない。ただし後続で表示しやすい data owner にする。
- PR/push/merge は Goal 完了条件に含めない。commit もユーザー明示指示がある場合だけ行う。

## 4. Source Findings

### MCP config owner

`internal/mcp/client.go` の `ServerConfig` が `mcp.json` の server-level JSON schema owner である。
ここに `approval` と `toolApprovals` を追加する。

`internal/mcp/config_loader.go` は default `mcp.json` 作成と read を担う。
新 default sample には `approval: "confirm"` を明示してよいが、未設定 default の source of truth は runtime fallback である必要がある。

### MCP tool metadata owner

`internal/mcp/manager_tools.go` が `tools/list` result から `MCPTool` を作る。
approval resolution はここで行うのが自然である。

`MCPTool` には effective approval mode を持たせ、`internal/agent` 経由で `mcptool.Definition` と provider surface selection に渡す。

### MCP execution owner

`internal/mcptool/wrapper.go` が MCP tool execution の owner である。
ここで `auto`, `confirm`, `deny`, headless approval required を最終 enforcement する。

### Global confirmation owner

`internal/tools/common/confirm.go` は built-in tool と generic safety policy の owner である。
MCP の default `confirm` を保つため、MCP wrapper が無条件に `ConfirmWithAutoApproveDecisionAndOptions` を呼ぶだけの構造は避ける。

MCP 専用 policy は `common.GetToolSafety` の fallback に依存させない。

### Config/generated/docs owner

MCP server config は `~/.xelyon/mcp.json` であり、main `config.yaml` ではない。
この plan では、MCP 専用 config は `mcp.json` に置き、`execution.always_confirm` に `mcp` category は追加しない。
main config generated surface は変更しない想定である。

## 5. Responsibility Boundaries

```text
internal/mcp:
  mcp.json parsing, default sample, approval value normalization, tool-level effective approval resolution, denied tool filtering before manager tools commit.

internal/mcptool:
  wrapper execution enforcement, confirm/auto/deny behavior, headless approval required result, stale denied call defense.

internal/agent:
  MCP current surface propagation to prompt/provider/request context. denied tools must not be exposed.

internal/tools/common:
  built-in tool confirmation policy remains owner for built-in tools. It should not become the MCP approval source of truth.

internal/config:
  not an MCP approval source of truth in this plan. Do not add execution.always_confirm category for MCP in the first implementation.

docs/mcp.md:
  user-facing MCP approval config docs.

docs/dev/mcp-approval-policy-plan.md:
  implementation source of truth for this change.
```

Avoid putting MCP approval policy in a generic `utils` package.
If shared constants are needed to avoid import cycles, use a small MCP-specific package such as `internal/mcpapproval`, not a broad common helper.

## 6. Approval Config Contract

### Server-level config

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-github"],
      "approval": "confirm"
    }
  }
}
```

`approval` is optional.
Allowed values:

- `confirm`: ask before running each tool call.
- `auto`: run without confirmation, only for explicitly trusted MCP server/tool.
- `deny`: do not expose or run tools.

Missing value means `confirm`.

### Tool-level override

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-github"],
      "approval": "confirm",
      "toolApprovals": {
        "get_issue": "auto",
        "list_issues": "auto",
        "create_issue": "confirm",
        "delete_repository": "deny"
      }
    }
  }
}
```

`toolApprovals` key uses the raw MCP tool name from `tools/list`.
It does not use the exported XELYON name `mcp_<server>_<tool>`.

Unknown tool names in `toolApprovals` should not fail startup.
They may produce a warning or be reported in future status output, but the initial implementation may leave them as inert config.

### Precedence

1. `disabled: true` disables the server.
2. `tools.include` / `tools.exclude` filter visible candidate tools.
3. server-level `approval: "deny"` removes all remaining tools from that server.
4. `toolApprovals.<rawToolName>` overrides approval for non-denied servers.
5. server-level `approval` applies to remaining tools.
6. missing approval defaults to `confirm`.

`deny` after filtering removes the tool from provider/prompt/registry surface.
If a stale call still reaches the wrapper, it must not call the MCP server.

## 7. Runtime Behavior Contract

### `confirm`

- In interactive mode, show the existing confirm UI with server, tool, and bounded args preview.
- Args validation may still run before confirm for non-denied tools, preserving current validation-before-prompt behavior.
- User `No` returns the existing rejection result.
- User `Comment` returns feedback to the model without calling the MCP server.

### `auto`

- Does not call the global `ConfirmWithAutoApproveDecisionAndOptions` path for approval.
- Should print/log a clear auto approval reason such as `Auto-approved (MCP config): mcp_github_get_issue`.
- Still validates args before call.
- Still uses timeout/cancel from `tools.ExecutionContext.EffectiveContext()`.
- Still passes through large result guard after execution.

### `deny`

- Denied tools are filtered out before provider/prompt/registry surface.
- Defense-in-depth: if a denied tool wrapper is somehow invoked, return a denied result and error without calling the MCP server.
- Denied result should include server/tool identity and a stable reason phrase such as `MCP tool execution denied by approval policy`.

### Headless

- `mcp.headless: true` only allows MCP availability in headless mode. It does not imply approval.
- `confirm` in headless must not call the MCP server.
- Headless confirm should return an approval-required error/result with a stable marker such as `approval_required`.
- Headless MCP automation requires explicit `approval: "auto"` at server or tool level.

## 8. Provider / Prompt / Registry Surface

Denied tools must not be shown to the model.

Apply denial before these surfaces are built:

- `tools.Registry`
- MCP system prompt / request context surface
- provider-specific tool definitions
- current MCP surface budget

If a whole server is `approval: "deny"`, it should connect only if needed to read/filter tools.
The implementation may skip registering all tools from that server after list.
Do not call any denied tool.

`confirm` and `auto` tools remain visible to the model.
The provider schema does not need to expose approval metadata.

## 9. Config / Docs / Generated Metadata Surface

### `mcp.json`

Add to `internal/mcp.ServerConfig`:

```go
Approval      string            `json:"approval,omitempty"`
ToolApprovals map[string]string `json:"toolApprovals,omitempty"`
```

Implementation may replace `string` with an MCP-specific mode type if it does not create import cycles.

### Default sample

The default disabled sample may include:

```json
"approval": "confirm"
```

This is documentation-by-example only.
Runtime default must still be `confirm` when the field is absent.

### `config.yaml`

Do not add MCP approval to main `config.yaml`.
The owner is `~/.xelyon/mcp.json`.
Do not add `execution.always_confirm: ["mcp"]` in this implementation.
No generated config metadata change is expected.

### `docs/mcp.md`

Update public MCP docs with:

- `approval` server-level examples.
- `toolApprovals` examples.
- default `confirm`.
- `confirm | auto | deny` table.
- headless behavior.
- `--auto-approve` / `execution.mode: full_auto` do not auto-run MCP unless MCP approval is explicit `auto`.

## 10. Implementation Priority

1. Add approval mode parsing/normalization and config tests.
2. Resolve effective server/tool approval during MCP tool listing.
3. Filter `deny` tools before provider/prompt/registry surface.
4. Enforce `confirm | auto | deny` in `internal/mcptool.Wrapper`.
5. Add headless approval-required behavior.
6. Update docs and default sample.
7. Run focused tests and `make ci-check`.

## 11. Tests

### `internal/mcp`

- load config with server-level `approval`.
- load config with `toolApprovals`.
- missing approval defaults to `confirm`.
- invalid approval warns/falls back to `confirm`.
- `deny` tools are excluded after include/exclude.
- include/exclude still uses raw tool names and keeps its existing precedence.
- default disabled sample stays disabled and does not connect.

### `internal/mcptool`

- `confirm` prompts in interactive mode.
- `auto` does not prompt and calls the MCP server.
- `deny` does not prompt and does not call the MCP server.
- `confirm` in headless returns approval required and does not call the MCP server.
- validation still runs for `confirm` and `auto`.
- timeout/cancel still uses caller context.

### `internal/agent`

- denied MCP tools do not appear in provider tool definitions.
- denied MCP tools do not appear in MCP prompt/current surface.
- full_auto / auto-approve does not auto-run MCP with default `confirm`.
- explicit MCP `auto` allows execution in headless.

### config/docs/generated

- no generated config change is expected.
- if implementation discovers an unavoidable main config change, stop and reclassify the plan before editing generated config metadata.
- docs examples match runtime values exactly: `confirm`, `auto`, `deny`.

## 12. Verification Commands

Focused:

```sh
go test ./internal/mcp ./internal/mcptool ./internal/agent -run '([Mm][Cc][Pp]|Approval|Confirm|Headless|ToolDefinitions|RuntimeSurface)' -count=1
```

Broader MCP/provider:

```sh
go test ./internal/mcp ./internal/mcptool ./internal/mcpnames ./internal/prompt ./internal/agent ./internal/api ./internal/api/providers/openai ./internal/api/providers/openai_responses ./internal/api/providers/openai_compat ./internal/api/providers/claude ./internal/api/providers/gemini ./internal/api/providers/bedrock -run '([Mm][Cc][Pp]|ToolDefinitions|SetMCPTools|Prompt|schema|Config|Headless|RuntimeSurface|Approval)' -count=1
```

Final:

```sh
make ci-check
```

## 13. Phase Plan

### Phase 0: pre-implementation contract map

- Re-read `internal/mcp/client.go`, `config_loader.go`, `manager_tools.go`, `manager_session.go`.
- Re-read `internal/agent/agent_mcp.go`, `mcp_tool_surface.go`, `runtime_surface_sync.go`.
- Re-read `internal/mcptool/wrapper.go`.
- Confirm whether adding a small `internal/mcpapproval` package reduces duplicated strings without causing package boundary issues.
- Confirm headless error/result owner before implementation.

### Phase 1: config and normalization

- Add approval fields to `ServerConfig`.
- Add normalization helper and tests.
- Keep absent value as `confirm`.
- Warn and fallback to `confirm` for invalid values.

### Phase 2: effective approval and surface filtering

- Resolve per-tool approval after include/exclude.
- Attach effective approval to `MCPTool`.
- Exclude `deny` from committed tools and skipped/summary accounting in a way visible enough for diagnostics.

### Phase 3: execution enforcement

- Pass approval through `mcptool.Definition` / `WrapperOptions`.
- Implement `auto`, `confirm`, `deny` in wrapper.
- Ensure `--auto-approve` and `full_auto` cannot auto-run default-confirm MCP tools.
- Implement headless approval-required path.

### Phase 4: docs/tests

- Update `docs/mcp.md`.
- Add focused tests for config, wrapper, agent surface, and headless path.
- Run verification commands.

### Phase Final-A: impact audit

Before final report, check:

- denied tool cannot appear in provider surface.
- stale denied call cannot execute.
- default confirm remains confirm under `full_auto` and `--auto-approve`.
- explicit auto still works in headless.
- include/exclude semantics did not drift.
- timeout/cancel and large output guard still apply.
- docs match runtime values.

### Phase Final-B: mandatory refactor

After tests pass, review production and test diff for:

- duplicated `confirm | auto | deny` string parsing.
- mixed config parsing and runtime enforcement owner.
- generic helper names hiding MCP approval policy.
- oversized wrapper or manager functions.
- brittle test fixtures.

Run `post-implementation-refactor` if new helper/fallback/policy code accumulates in the wrong owner.
Run `test-boundary-refactor` if MCP approval tests make existing wrapper/agent test files materially harder to read.

## 14. Implementer Freedom

The implementer may choose exact helper/type names and package placement, as long as these contracts do not change:

- public config values are exactly `confirm`, `auto`, `deny`.
- default is `confirm`.
- global auto approve does not auto-run default MCP.
- `deny` is not visible to the model and cannot execute through stale calls.
- `toolApprovals` keys are raw MCP tool names.
- `mcp.json` remains the owner of MCP approval policy.

## 15. Open Decisions

No product-level open decisions remain.

## 16. Goal Handoff Prompt

```text
/goal Implement docs/dev/mcp-approval-policy-plan.md end to end.

Use docs/dev/mcp-approval-policy-plan.md as the source of truth. Start with Phase 0 contract map, then implement the planned sections, then run the impact audit and mandatory post-implementation refactor described in the plan.

Final-B is mandatory, not optional. If the plan and latest source structure conflict, preserve the MCP safety contracts and adapt to the existing owner boundaries. Re-read the plan after resume or context compaction. Do not commit or push unless explicitly requested.
```
