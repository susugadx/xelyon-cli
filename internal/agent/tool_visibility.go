package agent

import "github.com/susugadx/xelyon-cli/internal/prompt"

type toolSurfacePhase string

const (
	toolSurfacePhaseNormal toolSurfacePhase = "normal"
	toolSurfacePhasePlan   toolSurfacePhase = "plan"
)

type toolVisibilityOptions struct {
	allowSubAgents bool
}

type toolVisibilityPolicy struct {
	editToolMode               string
	excludedTools              []string
	allowLowLevelInvestigation bool
	allowReadFileOverride      bool
}

type normalModeRecoveryPromptKind string

const (
	normalModeRecoveryPromptDirectExecution normalModeRecoveryPromptKind = "direct_execution"
	normalModeRecoveryPromptStopPlanning    normalModeRecoveryPromptKind = "stop_planning"
	normalModeRecoveryPromptNoTextPlan      normalModeRecoveryPromptKind = "no_text_plan"
)

func resolveToolVisibilityPolicy(providerName string, modelName string, phase toolSurfacePhase, opts toolVisibilityOptions) toolVisibilityPolicy {
	return newToolVisibilityPolicy(ResolveEditToolMode(providerName, modelName), phase, opts)
}

func newToolVisibilityPolicy(editToolMode string, phase toolSurfacePhase, opts toolVisibilityOptions) toolVisibilityPolicy {
	policy := toolVisibilityPolicy{
		editToolMode:               editToolMode,
		allowLowLevelInvestigation: editToolMode == EditToolModeLegacy,
		allowReadFileOverride:      true,
	}

	excluded := baseToolSurfaceExclusions(editToolMode, phase)
	excluded = appendUniqueStrings(excluded, investigationToolSurfaceExclusions(policy.allowLowLevelInvestigation, policy.allowReadFileOverride)...)
	if !opts.allowSubAgents {
		excluded = appendUniqueStrings(excluded, "spawn_agent", "wait_agent")
	}
	policy.excludedTools = excluded
	return policy
}

func (p toolVisibilityPolicy) excluded() []string {
	return append([]string(nil), p.excludedTools...)
}

func (p toolVisibilityPolicy) normalModeRecoveryPrompt(kind normalModeRecoveryPromptKind) string {
	toolExamples := p.normalModeRecoveryToolExamples()
	switch kind {
	case normalModeRecoveryPromptStopPlanning:
		return "[SYSTEM] STOP planning. Pick the FIRST change and execute it NOW using the appropriate visible tools (" + toolExamples + ", etc). One tool call, no explanation."
	case normalModeRecoveryPromptNoTextPlan:
		return "[SYSTEM] Do NOT output plans as numbered text. Execute the required changes directly using visible tools (" + toolExamples + ", etc)."
	default:
		return "[SYSTEM] You are in NORMAL MODE. Do NOT output JSON directly. Execute the required changes directly using visible tools (" + toolExamples + ", etc)."
	}
}

func (p toolVisibilityPolicy) normalModeRecoveryToolExamples() string {
	if p.allowLowLevelInvestigation {
		return "gather_context, read_file, str_replace"
	}
	if p.allowReadFileOverride {
		return "gather_context, read_file, apply_patch"
	}
	return "gather_context, apply_patch"
}

func baseToolSurfaceExclusions(editToolMode string, phase toolSurfacePhase) []string {
	excluded := make([]string, 0, 6)
	if phase == toolSurfacePhaseNormal {
		excluded = appendUniqueStrings(excluded, prompt.PlanningToolNames...)
	}

	excluded = filterStrings(excluded, "apply_patch", "str_replace", "write_file", "delete_file")
	if editToolMode == EditToolModeLegacy {
		return appendUniqueStrings(excluded, "apply_patch")
	}
	return appendUniqueStrings(excluded, "str_replace", "write_file", "delete_file")
}

func investigationToolSurfaceExclusions(allowLowLevelInvestigation bool, allowReadFileOverride bool) []string {
	if allowLowLevelInvestigation {
		return []string{"list_dir"}
	}
	if allowReadFileOverride {
		return []string{"search_code", "list_dir"}
	}
	return []string{"search_code", "read_file", "list_dir"}
}

func (a *Agent) toolVisibilityPolicy(phase toolSurfacePhase, opts toolVisibilityOptions) toolVisibilityPolicy {
	if a == nil {
		return newToolVisibilityPolicy(EditToolModeApplyPatch, phase, opts)
	}
	return resolveToolVisibilityPolicy(a.ProviderName, a.CurrentModel, phase, opts)
}
