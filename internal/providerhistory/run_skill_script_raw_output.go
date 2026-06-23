package providerhistory

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

const (
	runSkillScriptRawOutputReason                     = "artifact_backed_run_skill_script_result"
	runSkillScriptRawOutputReplacementKind            = "compact_old_run_skill_script_result"
	runSkillScriptRawOutputArtifactsDisabledReason    = "run_skill_script_raw_output_artifacts_disabled"
	runSkillScriptRawOutputArtifactMissingReason      = "run_skill_script_raw_output_artifact_missing"
	runSkillScriptRawOutputRehydrateUnavailableReason = "run_skill_script_raw_output_rehydrate_not_available"
	runSkillScriptRawOutputMissingIdentityReason      = "run_skill_script_missing_skill_or_script"
)

type runSkillScriptProjectionArgs struct {
	skill    string
	script   string
	args     string
	argsJSON string
}

func recordProviderHistoryRunSkillScriptArtifactCandidate(report *ProjectionReport, policy Policy, entry ReductionCandidate, arguments, content string, messages []api.Message) bool {
	if report == nil {
		return false
	}
	entry.FutureApplyCandidate = true
	entry.Reason = runSkillScriptRawOutputReason
	entry.SuggestedReplacementKind = runSkillScriptRawOutputReplacementKind

	if !providerHistoryHasLaterAssistant(messages, entry.HistoryIndex) {
		entry.KeepReason = "no_later_assistant_message"
		report.Kept = append(report.Kept, entry)
		return true
	}
	if strings.TrimSpace(content) == "" {
		entry.KeepReason = "empty_run_skill_script_output"
		entry.FailClosedReason = entry.KeepReason
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return true
	}

	parsed, ok := providerHistoryRunSkillScriptProjectionArgs(arguments)
	if !ok {
		entry.KeepReason = runSkillScriptRawOutputMissingIdentityReason
		entry.FailClosedReason = entry.KeepReason
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return true
	}
	if providerHistoryRunSkillScriptArgsLookSensitive(arguments, parsed) {
		entry.SafetyStatus = "sensitive"
		entry.KeepReason = string(rawoutputs.ReasonSensitiveArtifactForbidden)
		entry.FailClosedReason = entry.KeepReason
		entry.ArtifactGateStatus = "sensitive_args"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return true
	}
	if rawoutputs.LooksSensitiveContent(content) || providerHistoryLooksBareSecret(content) {
		entry.SafetyStatus = "sensitive"
		entry.KeepReason = string(rawoutputs.ReasonSensitiveArtifactForbidden)
		entry.FailClosedReason = entry.KeepReason
		entry.ArtifactGateStatus = "sensitive_body"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return true
	}

	spec := providerHistoryDataBearingToolArtifactCandidateSpec{
		Surface: rawoutputs.SurfaceCommandOutput,
		Source: rawoutputs.SourceMetadata{
			CommandHash:    commandHash("run_skill_script\x00" + parsed.skill + "\x00" + parsed.script),
			CommandPreview: providerHistoryRunSkillScriptCommandPreview(parsed),
			ToolName:       entry.ToolName,
			ToolCallID:     entry.ToolCallID,
			EventID:        fmt.Sprintf("history:%d", entry.HistoryIndex),
			HistoryIndex:   entry.HistoryIndex,
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "run_skill_script",
			Classifier:   "skill_script_output",
		},
		Reason:                     runSkillScriptRawOutputReason,
		ReplacementKind:            runSkillScriptRawOutputReplacementKind,
		ArtifactsDisabledReason:    runSkillScriptRawOutputArtifactsDisabledReason,
		MissingArtifactReason:      runSkillScriptRawOutputArtifactMissingReason,
		RehydrateUnavailableReason: runSkillScriptRawOutputRehydrateUnavailableReason,
		BuildPlaceholder: func(ref rawoutputs.RawOutputRef) string {
			return buildProviderHistoryRawOutputPlaceholder("run_skill_script result", ref, content)
		},
	}
	recordProviderHistoryDataBearingToolArtifactCandidate(report, policy, entry, content, spec)
	return true
}

func providerHistoryRunSkillScriptProjectionArgs(arguments string) (runSkillScriptProjectionArgs, bool) {
	fields, err := providerHistoryCommandArgumentFields(arguments)
	if err != nil {
		return runSkillScriptProjectionArgs{}, false
	}
	skill, ok := providerHistoryCommandJSONStringArgument(fields, "skill")
	if !ok || strings.TrimSpace(skill) == "" {
		return runSkillScriptProjectionArgs{}, false
	}
	script, ok := providerHistoryCommandJSONStringArgument(fields, "script")
	if !ok || strings.TrimSpace(script) == "" {
		return runSkillScriptProjectionArgs{}, false
	}
	args, _ := providerHistoryCommandJSONStringArgument(fields, "args")
	argsJSON, _ := providerHistoryCommandJSONStringArgument(fields, "args_json")
	return runSkillScriptProjectionArgs{
		skill:    strings.TrimSpace(skill),
		script:   strings.TrimSpace(script),
		args:     strings.TrimSpace(args),
		argsJSON: strings.TrimSpace(argsJSON),
	}, true
}

func providerHistoryRunSkillScriptCommandPreview(parsed runSkillScriptProjectionArgs) string {
	return rawoutputs.SanitizeDisplayPreview(
		fmt.Sprintf("run_skill_script skill=%s script=%s", parsed.skill, parsed.script),
		160,
	)
}

func providerHistoryRunSkillScriptArgsLookSensitive(arguments string, parsed runSkillScriptProjectionArgs) bool {
	if rawoutputs.LooksSensitiveContent(arguments) ||
		rawoutputs.LooksSensitiveContent(parsed.args) ||
		rawoutputs.LooksSensitiveContent(parsed.argsJSON) {
		return true
	}
	lower := strings.ToLower(parsed.args + "\n" + parsed.argsJSON)
	for _, marker := range []string{
		"authorization",
		"api-key",
		"apikey",
		"access-token",
		"refresh-token",
		"id-token",
		"session-token",
		"auth-token",
		"client-secret",
		"client_secret",
		"password",
		"passwd",
		"secret",
		"token",
		"jwt",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
