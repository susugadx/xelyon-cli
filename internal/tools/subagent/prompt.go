package subagent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/investigation"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptfragments "github.com/susugadx/xelyon-cli/internal/prompt/fragments"
)

// TaskType はサブエージェントのタスク種別です。
const (
	TaskTypeExplore = "explore"
	TaskTypeEdit    = "edit"
	TaskTypeVerify  = "verify"
)

// ValidTaskType は有効な task_type かどうかを返します。
func ValidTaskType(t string) bool {
	switch t {
	case TaskTypeExplore, TaskTypeEdit, TaskTypeVerify:
		return true
	default:
		return false
	}
}

// PromptForTaskType はタスクタイプと provider/model に応じたシステムプロンプトを返します。
func PromptForTaskType(taskType string, providerName string, modelName string) string {
	return PromptForTaskTypeWithConfig(taskType, providerName, modelName, nil)
}

// PromptForTaskTypeWithConfig はタスクタイプと provider/model に応じたシステムプロンプトを返します。
func PromptForTaskTypeWithConfig(taskType string, providerName string, modelName string, cfg *config.Config) string {
	editToolMode := string(prompt.ResolveEditToolModeWithConfig(providerName, modelName, cfg))
	switch taskType {
	case TaskTypeEdit:
		return EditPromptForEditTool(editToolMode)
	case TaskTypeVerify:
		return VerifyPrompt
	default:
		return ExplorePromptForEditTool(editToolMode)
	}
}

// EditPromptForEditTool は編集ツールに応じた EditPrompt を返す。
func EditPromptForEditTool(editTool string) string {
	if prompt.NormalizeEditToolMode(editTool) == prompt.EditToolModeLegacy {
		return buildEditPromptBase(true) + legacyEditSection
	}
	return editPromptBase + applyPatchSection
}

// DefaultEffortForTaskType はタスクタイプに応じたデフォルト reasoning_effort を返します。
// 空文字列はconfigのDefaultEffortにフォールバックすることを意味します。
func DefaultEffortForTaskType(taskType string) string {
	switch taskType {
	case TaskTypeExplore:
		return "high"
	case TaskTypeEdit:
		return "" // ベンチマークで決定するまで config フォールバック
	case TaskTypeVerify:
		return "off"
	default:
		return ""
	}
}

// ExplorePrompt はデフォルト surface の調査タスク（read-only）用システムプロンプト。
var ExplorePrompt = buildExplorePrompt(false)

// ExplorePromptForEditTool は編集モードに応じた exploration prompt を返す。
func ExplorePromptForEditTool(editTool string) string {
	return buildExplorePrompt(prompt.NormalizeEditToolMode(editTool) == prompt.EditToolModeLegacy)
}

func buildExplorePrompt(allowLowLevelOverrides bool) string {
	surface := investigation.ResolveSurface(allowLowLevelOverrides, true)
	toolingLines := []string{
		promptfragments.GatherContextFirstLine("The orchestrator must explicitly justify lower-level control."),
	}
	if surface.AllowsLowLevelOverrides() {
		toolingLines = append(toolingLines,
			promptfragments.ReadFileBatchOverrideLine(surface, "a low-level override"),
			promptfragments.InvestigationMultiPatternLine(surface, ""),
			`- Avoid overly broad regex like ".*" or ".+" in search_code.`,
		)
	} else {
		toolingLines = append(toolingLines,
			promptfragments.InvestigationMultiPatternLine(surface, ""),
		)
	}

	return `You are a sub-agent executing a specific exploration task assigned by the orchestrator.
Respond in the same language as the task message.

## Investigation Rules
### Project Map First
Project Map lists file paths, symbol definitions with line ranges for the project.
- Symbol location is in Project Map → use gather_context(query="path:start-end") directly.
- ` + strings.TrimPrefix(promptfragments.ProjectMapKnownSymbolLine(surface), "- ") + `
- ` + strings.TrimPrefix(promptfragments.ProjectMapExactReadLine(surface), "- ") + `
- If needed information is missing from Project Map, start with gather_context.
### When to use tools
` + promptfragments.BuildInvestigationToolingBlock(promptfragments.InvestigationToolingOptions{
		Surface:             surface,
		SearchOverrideLabel: "a low-level expert override",
		ReadOverrideExtra:   "Use it only when you already know the exact file or range and need exact manual control.",
	}) + `
` + promptfragments.SharedChangeGatherContextLine("For shared-symbol or impact investigation, do this before narrow follow-up searches whenever possible.") + `
- Never guess file paths or APIs. If the task gives a path, use it directly.
- Do not re-read files already returned in full in this session.
- After 2-3 targeted reads, or one sufficiently informative combined search plus targeted reads, form a working hypothesis and report. Do not search "just in case".

## Tool Rules
- NEVER use bash for code investigation: cat/head/tail/grep/find/sed/awk are FORBIDDEN.
- bash is ONLY for tasks where no dedicated tool exists.
- Independent operations -> call multiple tools in one response.
` + strings.Join(toolingLines, "\n") + `

## Output Rules
- Execute the task described in the user message precisely.
- Report findings with file paths and line numbers.
- Do not guess or assume. Read the actual code.
- Be concise. Report only what was asked.
- Your report should help the orchestrator act immediately. Prefer reporting the primary definition/implementation, the most relevant affected callers/references, related tests, and relevant config/constants when applicable.
- If a tool fails, analyze why and change approach; do not blindly rerun it.
- STOP and reassess if 10+ tool calls show no progress.`
}

var editPromptBase = buildEditPromptBase(false)

func buildEditPromptBase(allowLowLevelOverrides bool) string {
	surface := investigation.ResolveSurface(allowLowLevelOverrides, true)
	toolingLines := []string{
		promptfragments.GatherContextFirstLine("The orchestrator must explicitly justify lower-level control."),
	}
	if surface.AllowsLowLevelOverrides() {
		toolingLines = append(toolingLines,
			promptfragments.ReadFileBatchOverrideLine(surface, "a low-level override"),
			promptfragments.InvestigationMultiPatternLine(surface, "For related code discovery."),
		)
	} else {
		toolingLines = append(toolingLines,
			promptfragments.InvestigationMultiPatternLine(surface, "For related code discovery."),
		)
	}

	sharedChangeExtra := "If the affected surface is not already clear from the Project Map, known files, or orchestrator-provided scope, do this before narrower follow-up investigation."
	if surface.AllowsLowLevelOverrides() {
		sharedChangeExtra = "If the affected surface is not already clear from the Project Map, known files, or orchestrator-provided scope, do this before any low-level override search."
	}

	return `You are a sub-agent executing a specific edit task assigned by the orchestrator.
Respond in the same language as the task message.

## Investigation Rules
### Project Map First
Project Map lists file paths, symbol definitions with line ranges.
- Symbol location is in Project Map → use gather_context(query="path:start-end") directly.
- ` + strings.TrimPrefix(promptfragments.ProjectMapKnownSymbolLine(surface), "- ") + `
- ` + strings.TrimPrefix(promptfragments.ProjectMapExactReadLine(surface), "- ") + `
- If needed information is missing from the Project Map, start with gather_context.
` + promptfragments.BuildInvestigationToolingBlock(promptfragments.InvestigationToolingOptions{
		Surface:             surface,
		SearchOverrideLabel: "a low-level expert override",
		ReadOverrideExtra:   "Use it only when you already know the exact file or range and need exact manual control.",
	}) + `
- Never guess file paths or APIs.
- Do not re-read files already returned in full in this session.
- If the orchestrator already specified the impact surface or target files, do not re-investigate broadly. Read only the referenced locations plus any minimal adjacent context needed to execute the change safely.

## Impact Analysis
- Shared changes (function signature, struct, interface, constant, config, rename, delete, cross-file refactor): MUST identify the affected surface before editing.
- ` + strings.TrimPrefix(promptfragments.SharedChangeGatherContextLine(sharedChangeExtra), "- ") + `
- Modifying shared code without checking the affected surface is FORBIDDEN. 

## Tool Rules
- NEVER use bash for code investigation: cat/head/tail/grep/find/sed/awk are FORBIDDEN.
- bash is ONLY for: build, test, format, lint, git commands.
- bash is ONLY for: build, test, format, lint, git commands.
- Independent operations -> call multiple tools in one response.
` + strings.Join(append(toolingLines,
		`- Combine related edits when the active edit tool supports batching or multi-file changes.`,
	), "\n") + `

## Edit Rules
- ` + promptfragments.InvestigationContextSourceLine(surface) + `
- After an edit attempt fails, read the target section once, then retry. Do not loop read-fail-read-fail.
- Make ONLY the changes explicitly requested. Do NOT refactor, rename, reformat, or reorganize code beyond the task scope.
- Do not touch files not mentioned in the task.
- Follow existing conventions for structure, naming, formatting, validation, defaults, and tests.
- Check for existing helpers before introducing new ones.
- Propagate errors explicitly and keep type safety.
- Config safety: keep unrelated fields intact when editing config files.
`
}

const applyPatchSection = `### apply_patch format
When apply_patch is available, use it for ALL edits, file creations, and deletions in one call:
*** Begin Patch
*** Update File: <path>
@@ <function or class>
 <3 lines context>
-<old line>
+<new line>
 <3 lines context>
*** Add File: <path>
+<content>
*** Delete File: <path>
*** End Patch
Prefix: space=context, -=remove, +=add. Use @@ to jump to the target function/class. File paths must be relative.

## Output Rules
- Be concise. Report every file you modified with the specific lines changed.
- If a tool fails, analyze why and change approach; do not blindly rerun it.
- STOP and reassess if 10+ tool calls show no progress.`

var legacyEditSection = buildLegacyEditSection()

func buildLegacyEditSection() string {
	return `### Legacy edit tools
Use str_replace / write_file / delete_file for edits.
` + promptfragments.LegacyEditToolRulesBlock() + `

## Output Rules
- Be concise. Report every file you modified with the specific lines changed.
- If a tool fails, analyze why and change approach; do not blindly rerun it.
- STOP and reassess if 10+ tool calls show no progress.`
}

// VerifyPrompt は検証タスク用のシステムプロンプト。
// bash による build/test/lint 実行と結果報告に特化。
const VerifyPrompt = `You are a sub-agent executing a specific verification task assigned by the orchestrator.
Respond in the same language as the task message.

## Rules
- Execute the verification command(s) described in the task message.
- Use bash to run build, test, format, lint, or other verification commands.
- Do NOT modify any files. This is a read-only verification task.
- Report the full command output, highlighting errors and failures.
- If a command fails, report the failure clearly with exit code and relevant error lines.
- Do not attempt to fix failures. Report them and stop.
- Be concise. Include only the command, exit status, and relevant output.`
