package applypatch

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func confirmApplyPatch(execCtx tools.ExecutionContext, parsed *ParsedPatch, autoApproved bool) common.ConfirmDecision {
	const message = "Apply this patch? / このパッチを適用しますか？"

	if autoApproved {
		safety := getApplyPatchSafety(parsed)
		execCtx.Output().Green.Printf("Auto-approved (%s): %s\n", common.GetSafetyDescription(safety), "apply_patch")
		return common.ConfirmDecision{Action: common.ConfirmYes}
	}

	tc := buildApplyPatchConfirmContext(parsed)
	return common.ConfirmToolAction(execCtx.PromptIO(), execCtx.ConfirmOptions(), "apply_patch", message, tc)
}

// buildApplyPatchConfirmContext は patch の全 hunk から ToolConfirmContext を構築する。
func buildApplyPatchConfirmContext(parsed *ParsedPatch) common.ToolConfirmContext {
	targetPaths, hasMove := collectApplyPatchTargetPaths(parsed)
	return common.ToolConfirmContext{
		TargetPaths:   targetPaths,
		HasMove:       hasMove,
		AffectedFiles: len(targetPaths),
	}
}

func collectApplyPatchTargetPaths(parsed *ParsedPatch) ([]string, bool) {
	if parsed == nil {
		return nil, false
	}

	targetPaths := make([]string, 0, len(parsed.Hunks)*2)
	seen := make(map[string]bool, len(parsed.Hunks)*2)
	hasMove := false

	addPath := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		targetPaths = append(targetPaths, path)
	}

	for _, hunk := range parsed.Hunks {
		addPath(hunk.Path)
		if hunk.MovePath != "" {
			hasMove = true
			addPath(hunk.MovePath)
		}
	}

	return targetPaths, hasMove
}

// shouldAutoApproveApplyPatch は apply_patch のプレビュー表示をスキップするかを判定する。
// execution policy → 旧 ToolConfirm の順で判定し、auto-approve される場合はプレビュー不要。
// always_confirm / workspace_outside / mass_change / move に該当する場合は false を返す。
func shouldAutoApproveApplyPatch(execCtx tools.ExecutionContext, parsed *ParsedPatch) bool {
	options := execCtx.ConfirmOptions()
	policy := options.EffectivePolicy()
	tc := buildApplyPatchConfirmContext(parsed)

	if requiresApplyPatchConfirmation(policy, tc) {
		return false
	}
	if common.IsAutoApprovable("apply_patch", options.AutoApprove) {
		return true
	}
	if isAutoApprovedByExecutionPolicy(policy, tc) {
		return true
	}
	return isAutoApprovedByLegacyConfig(options.Config, getApplyPatchSafety(parsed))
}

func requiresApplyPatchConfirmation(policy config.ExecutionPolicy, tc common.ToolConfirmContext) bool {
	if cat := common.ToolNameToConfirmCategory("apply_patch"); cat != "" && policy.ShouldConfirm(cat) {
		return true
	}
	for _, cat := range common.ResolveContextCategories("apply_patch", tc) {
		if policy.ShouldConfirm(cat) {
			return true
		}
	}
	return false
}

func isAutoApprovedByExecutionPolicy(policy config.ExecutionPolicy, tc common.ToolConfirmContext) bool {
	if policy.AutoApproveFullAuto {
		return true
	}
	if policy.AutoApproveTrustedWrite && config.TrustedWriteTools["apply_patch"] && common.AllPathsInsideWorkspace(tc) {
		return true
	}
	return false
}

func isAutoApprovedByLegacyConfig(cfg *config.Config, safety common.ToolSafety) bool {
	return cfg != nil && cfg.ToolConfirm.AutoApproveMedium && safety == common.SafetyMedium
}

func getApplyPatchSafety(parsed *ParsedPatch) common.ToolSafety {
	if parsed == nil {
		return common.SafetyLow
	}

	for _, hunk := range parsed.Hunks {
		if hunk.Type == "delete" {
			return common.SafetyLow
		}
		if hunk.Type == "update" && hunk.MovePath != "" {
			srcPath, srcErr := common.ValidatePath(hunk.Path)
			dstPath, dstErr := common.ValidatePath(hunk.MovePath)
			if srcErr != nil || dstErr != nil || srcPath != dstPath {
				return common.SafetyLow
			}
		}
	}

	return common.SafetyMedium
}
