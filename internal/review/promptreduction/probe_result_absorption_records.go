package promptreduction

import (
	"fmt"

	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// ProbeResultAbsorptionReductionRecord は runner が stats/state へ記録する候補行を表す。
type ProbeResultAbsorptionReductionRecord struct {
	Classifier  string
	SavedBytes  int
	SavedTokens int
	Item        ReviewPromptReductionItem
}

// ReductionRecords は absorption 候補を stats/state 記録用の reduction record に変換する。
func (p ProbeResultAbsorptionPlan) ReductionRecords(phase ReviewModelPhase, applied bool, refs ProbeResultAbsorptionArtifactRefs) []ProbeResultAbsorptionReductionRecord {
	records := make([]ProbeResultAbsorptionReductionRecord, 0, len(p.probes)+len(p.commands))
	for _, probeID := range p.ProbeIDs() {
		candidate := p.probes[probeID].withRawArtifactRef(refs.probeResultRef(probeID))
		records = append(records, ProbeResultAbsorptionReductionRecord{
			Classifier:  "probe_result_absorption_candidate",
			SavedBytes:  candidate.savedBytes,
			SavedTokens: candidate.savedTokens,
			Item: ReviewPromptReductionItem{
				ID:               "probe_result:" + probeID,
				Family:           ReviewPromptReductionFamilyProbeResult,
				Phase:            phase,
				Status:           probeResultAbsorptionStatus(applied),
				AbsorbedBy:       ReviewPromptAbsorptionRefsFromOwners(candidate.summary.AbsorbedBy),
				EvidenceRefs:     []reviewreport.ReviewEvidenceRef{{Kind: reviewreport.ReviewEvidenceKindProbe, ProbeID: probeID}},
				RawArtifactRef:   candidate.summary.RawArtifactRef,
				Summary:          candidate.summary.Summary,
				OriginalBytes:    candidate.originalBytes,
				ReplacementBytes: candidate.replacementBytes,
			},
		})
	}
	for _, key := range p.CommandKeys() {
		candidate := p.commands[key].withRawArtifactRef(refs.probeCommandRef(key))
		records = append(records, ProbeResultAbsorptionReductionRecord{
			Classifier:  "probe_command_result_absorption_candidate",
			SavedBytes:  candidate.savedBytes,
			SavedTokens: candidate.savedTokens,
			Item: ReviewPromptReductionItem{
				ID:               fmt.Sprintf("probe_result:%s:command:%d", key.ProbeID, key.CommandIndex),
				Family:           ReviewPromptReductionFamilyProbeResult,
				Phase:            phase,
				Status:           probeResultAbsorptionStatus(applied),
				AbsorbedBy:       ReviewPromptAbsorptionRefsFromOwners(candidate.summary.AbsorbedBy),
				EvidenceRefs:     []reviewreport.ReviewEvidenceRef{{Kind: reviewreport.ReviewEvidenceKindProbeCommand, ProbeID: key.ProbeID, CommandIndex: reviewreport.ReviewCommandIndex(key.CommandIndex)}},
				RawArtifactRef:   candidate.summary.RawArtifactRef,
				Summary:          candidate.summary.Summary,
				OriginalBytes:    candidate.originalBytes,
				ReplacementBytes: candidate.replacementBytes,
			},
		})
	}
	return records
}

func probeResultAbsorptionAppliedSummary(probeID string, commandLevel bool, commandIndex int) string {
	if commandLevel {
		return fmt.Sprintf("passed probe result %q command[%d] was reflected by report evidence; raw output is available in Review Probe Raw Output Context", probeID, commandIndex)
	}
	return fmt.Sprintf("passed probe result %q was reflected by report evidence; raw output is available in Review Probe Raw Output Context", probeID)
}

func probeResultAbsorptionStatus(applied bool) ReviewPromptReductionItemStatus {
	if applied {
		return ReviewPromptReductionItemAbsorbed
	}
	return ReviewPromptReductionItemCandidate
}
