package review

import (
	"sort"
	"strings"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func reviewProbeResultAbsorptionKeepReason(build reviewProbeRawOutputBuild) string {
	if strings.TrimSpace(build.disabledReason) != "" {
		return build.disabledReason
	}
	if strings.TrimSpace(build.ledger.FailClosedReason) != "" {
		return build.ledger.FailClosedReason
	}
	return reviewProbeRawOutputReasonRehydrateUnavailable
}

func reviewProbeRawOutputLedgerPtr(ledger ReviewProbeRawOutputLedger) *ReviewProbeRawOutputLedger {
	if ledger.empty() {
		return nil
	}
	return &ledger
}

func reviewProbeResultPromptOriginalBytes(result ReviewProbeResult) int {
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

func reviewProbeCommandResultPromptOriginalBytes(command ReviewProbeCommandResult) int {
	return len(command.Output) + len(command.Error)
}

func reviewPromptAbsorptionSavings(originalBytes, replacementBytes int) (int, int, bool) {
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

func dedupeSortedReviewPromptAbsorptionRefs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedReviewProbeAbsorptionProbeIDs(candidates map[string]reviewProbeResultAbsorptionCandidate) []string {
	ids := make([]string, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedReviewProbeCommandAbsorptionKeys(candidates map[reviewmodelinput.ProbeCommandResultKey]reviewProbeResultAbsorptionCandidate) []reviewmodelinput.ProbeCommandResultKey {
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
