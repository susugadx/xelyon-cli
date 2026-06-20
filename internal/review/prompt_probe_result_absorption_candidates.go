package review

import (
	"fmt"
	"strings"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func buildReviewProbeResultAbsorptionCandidates(report reviewreport.ReviewReport, probeResults []reviewprobe.ReviewProbeResult) reviewProbeResultAbsorptionCandidates {
	candidates := reviewProbeResultAbsorptionCandidates{}
	if strings.TrimSpace(report.SchemaVersion) == "" || report.ScopeCoverage == nil || len(probeResults) == 0 {
		return candidates
	}
	refs := reviewProbeResultAbsorptionRefs(report)
	for _, result := range probeResults {
		probeID := strings.TrimSpace(result.ID)
		if probeID == "" {
			continue
		}
		if !reviewProbeResultSafeForAbsorbedPrompt(result) {
			continue
		}

		probeAbsorbed := false
		absorbedBy := refs.safeProbes[probeID]
		if len(absorbedBy) > 0 && !refs.probeUnsafeForFullAbsorption(probeID) {
			if candidate, ok := buildReviewProbeResultAbsorptionCandidate(probeID, absorbedBy, reviewProbeResultPromptOriginalBytes(result), false, 0); ok {
				if candidates.probes == nil {
					candidates.probes = make(map[string]reviewProbeResultAbsorptionCandidate)
				}
				candidates.probes[probeID] = candidate
				probeAbsorbed = true
			}
		}

		if probeAbsorbed {
			continue
		}

		for commandIndex, command := range result.CommandResults {
			key := reviewmodelinput.ProbeCommandResultKey{ProbeID: probeID, CommandIndex: commandIndex}
			absorbedBy := refs.safeCommands[key]
			if len(absorbedBy) == 0 || refs.commandUnsafeForAbsorption(key) {
				continue
			}
			if !reviewProbeCommandResultSafeForAbsorbedPrompt(command) {
				continue
			}
			originalBytes := reviewProbeCommandResultPromptOriginalBytes(command)
			if candidate, ok := buildReviewProbeResultAbsorptionCandidate(probeID, absorbedBy, originalBytes, true, commandIndex); ok {
				if candidates.commands == nil {
					candidates.commands = make(map[reviewmodelinput.ProbeCommandResultKey]reviewProbeResultAbsorptionCandidate)
				}
				candidates.commands[key] = candidate
			}
		}
	}
	return candidates
}

func buildReviewProbeResultAbsorptionCandidate(probeID string, absorbedBy []string, originalBytes int, commandLevel bool, commandIndex int) (reviewProbeResultAbsorptionCandidate, bool) {
	summaryText := fmt.Sprintf("passed probe result %q is reflected by review evidence and can be reduced when review raw output rehydrate is available", probeID)
	if commandLevel {
		summaryText = fmt.Sprintf("passed probe result %q command[%d] is reflected by review evidence and can be reduced when review raw output rehydrate is available", probeID, commandIndex)
	}
	summary := reviewmodelinput.ProbeResultAbsorptionSummary{
		Summary:    summaryText,
		AbsorbedBy: absorbedBy,
	}
	replacementBytes := len(summary.Summary) + len(strings.Join(summary.AbsorbedBy, "\n")) + len(summary.RawArtifactRef)
	savedBytes, savedTokens, ok := reviewPromptAbsorptionSavings(originalBytes, replacementBytes)
	if !ok {
		return reviewProbeResultAbsorptionCandidate{}, false
	}
	return reviewProbeResultAbsorptionCandidate{
		summary:          summary,
		originalBytes:    originalBytes,
		replacementBytes: replacementBytes,
		savedBytes:       savedBytes,
		savedTokens:      savedTokens,
	}, true
}

func reviewProbeResultAbsorptionAppliedSummary(probeID string, commandLevel bool, commandIndex int) string {
	if commandLevel {
		return fmt.Sprintf("passed probe result %q command[%d] was reflected by report evidence; raw output is available in Review Probe Raw Output Context", probeID, commandIndex)
	}
	return fmt.Sprintf("passed probe result %q was reflected by report evidence; raw output is available in Review Probe Raw Output Context", probeID)
}
