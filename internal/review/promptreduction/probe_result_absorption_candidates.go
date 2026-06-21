package promptreduction

import (
	"fmt"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const reviewPromptProbeResultAbsorptionMinSavedTokens = 128

// AbsorbedBy は候補が吸収済みと判断された report owner refs を返す。
func (c ProbeResultAbsorptionCandidate) AbsorbedBy() []string {
	return append([]string(nil), c.summary.AbsorbedBy...)
}

// OriginalBytes は削減前の概算 byte 数を返す。
func (c ProbeResultAbsorptionCandidate) OriginalBytes() int {
	return c.originalBytes
}

// ReplacementBytes は削減後の概算 byte 数を返す。
func (c ProbeResultAbsorptionCandidate) ReplacementBytes() int {
	return c.replacementBytes
}

func buildProbeResultAbsorptionCandidate(probeID string, absorbedBy []string, originalBytes int, commandLevel bool, commandIndex int) (ProbeResultAbsorptionCandidate, bool) {
	summaryText := fmt.Sprintf("passed probe result %q is reflected by review evidence and can be reduced when review raw output rehydrate is available", probeID)
	if commandLevel {
		summaryText = fmt.Sprintf("passed probe result %q command[%d] is reflected by review evidence and can be reduced when review raw output rehydrate is available", probeID, commandIndex)
	}
	summary := reviewmodelinput.ProbeResultAbsorptionSummary{
		Summary:    summaryText,
		AbsorbedBy: append([]string(nil), absorbedBy...),
	}
	replacementBytes := len(summary.Summary) + len(strings.Join(summary.AbsorbedBy, "\n")) + len(summary.RawArtifactRef)
	savedBytes, savedTokens, ok := promptAbsorptionSavings(originalBytes, replacementBytes)
	if !ok {
		return ProbeResultAbsorptionCandidate{}, false
	}
	return ProbeResultAbsorptionCandidate{
		summary:          summary,
		originalBytes:    originalBytes,
		replacementBytes: replacementBytes,
		savedBytes:       savedBytes,
		savedTokens:      savedTokens,
	}, true
}

func (c ProbeResultAbsorptionCandidate) summaryCopy() reviewmodelinput.ProbeResultAbsorptionSummary {
	summary := c.summary
	summary.AbsorbedBy = append([]string(nil), summary.AbsorbedBy...)
	return summary
}

func (c ProbeResultAbsorptionCandidate) withRawArtifactRef(refID string) ProbeResultAbsorptionCandidate {
	refID = strings.TrimSpace(refID)
	if refID == "" {
		return c
	}
	c.summary.RawArtifactRef = refID
	c.replacementBytes = len(c.summary.Summary) + len(strings.Join(c.summary.AbsorbedBy, "\n")) + len(c.summary.RawArtifactRef)
	return c
}

func probeResultPromptOriginalBytes(result reviewprobe.ReviewProbeResult) int {
	total := len(result.ID) + len(string(result.Mode)) + len(string(result.Status)) + len(result.Error)
	for _, path := range result.MutatedFiles {
		total += len(path)
	}
	for _, command := range result.CommandResults {
		total += len(command.Command)
		for _, arg := range command.Args {
			total += len(arg)
		}
		total += len(command.WorkDir)
		total += len(string(command.Status))
		total += len(command.Output)
		total += len(command.Error)
	}
	return total
}

func probeCommandResultPromptOriginalBytes(command reviewprobe.ReviewProbeCommandResult) int {
	return len(command.Output) + len(command.Error)
}

func promptAbsorptionSavings(originalBytes, replacementBytes int) (int, int, bool) {
	if originalBytes <= replacementBytes {
		return 0, 0, false
	}
	savedBytes := originalBytes - replacementBytes
	savedTokens := token.EstimateTokenCount(strings.Repeat("x", originalBytes)) - token.EstimateTokenCount(strings.Repeat("x", replacementBytes))
	if savedTokens < reviewPromptProbeResultAbsorptionMinSavedTokens {
		return 0, 0, false
	}
	return savedBytes, savedTokens, true
}

func sortedProbeAbsorptionProbeIDs(candidates map[string]ProbeResultAbsorptionCandidate) []string {
	ids := make([]string, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedProbeCommandAbsorptionKeys(candidates map[reviewmodelinput.ProbeCommandResultKey]ProbeResultAbsorptionCandidate) []reviewmodelinput.ProbeCommandResultKey {
	keys := make([]reviewmodelinput.ProbeCommandResultKey, 0, len(candidates))
	for key := range candidates {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].ProbeID != keys[j].ProbeID {
			return keys[i].ProbeID < keys[j].ProbeID
		}
		return keys[i].CommandIndex < keys[j].CommandIndex
	})
	return keys
}

func probeResultSafeForAbsorbedPrompt(result reviewprobe.ReviewProbeResult) bool {
	return result.Status == domain.ReviewProbePassed &&
		!result.MutatedWorktree &&
		!result.OutputTruncated &&
		strings.TrimSpace(result.Error) == ""
}

func probeCommandResultSafeForAbsorbedPrompt(result reviewprobe.ReviewProbeCommandResult) bool {
	return result.Status == domain.ReviewProbePassed &&
		!result.OutputTruncated &&
		strings.TrimSpace(result.Error) == ""
}
