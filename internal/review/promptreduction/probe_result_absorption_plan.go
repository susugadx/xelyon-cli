package promptreduction

import (
	"fmt"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const reviewPromptProbeResultAbsorptionMinSavedTokens = 128

// ProbeResultAbsorptionPlan は report evidence に吸収済みの probe result 削減候補を表す。
type ProbeResultAbsorptionPlan struct {
	probes   map[string]ProbeResultAbsorptionCandidate
	commands map[reviewmodelinput.ProbeCommandResultKey]ProbeResultAbsorptionCandidate
}

// ProbeResultAbsorptionCandidate は probe result prompt context の削減候補を表す。
type ProbeResultAbsorptionCandidate struct {
	summary          reviewmodelinput.ProbeResultAbsorptionSummary
	originalBytes    int
	replacementBytes int
	savedBytes       int
	savedTokens      int
}

// ProbeResultAbsorptionArtifactRefs は absorption を適用する raw output artifact ref を表す。
type ProbeResultAbsorptionArtifactRefs struct {
	ProbeResults  map[string]string
	ProbeCommands map[reviewmodelinput.ProbeCommandResultKey]string
}

// ProbeResultAbsorptionReductionRecord は runner が stats/state へ記録する候補行を表す。
type ProbeResultAbsorptionReductionRecord struct {
	Classifier  string
	SavedBytes  int
	SavedTokens int
	Item        ReviewPromptReductionItem
}

// BuildProbeResultAbsorptionPlan は report evidence から probe result absorption 候補を構築する。
func BuildProbeResultAbsorptionPlan(report reviewreport.ReviewReport, probeResults []reviewprobe.ReviewProbeResult) ProbeResultAbsorptionPlan {
	plan := ProbeResultAbsorptionPlan{}
	if strings.TrimSpace(report.SchemaVersion) == "" || report.ScopeCoverage == nil || len(probeResults) == 0 {
		return plan
	}
	refs := probeResultAbsorptionRefs(report)
	for _, result := range probeResults {
		probeID := strings.TrimSpace(result.ID)
		if probeID == "" {
			continue
		}
		if !probeResultSafeForAbsorbedPrompt(result) {
			continue
		}

		probeAbsorbed := false
		absorbedBy := refs.safeProbes[probeID]
		if len(absorbedBy) > 0 && !refs.probeUnsafeForFullAbsorption(probeID) {
			if candidate, ok := buildProbeResultAbsorptionCandidate(probeID, absorbedBy, probeResultPromptOriginalBytes(result), false, 0); ok {
				if plan.probes == nil {
					plan.probes = make(map[string]ProbeResultAbsorptionCandidate)
				}
				plan.probes[probeID] = candidate
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
			if !probeCommandResultSafeForAbsorbedPrompt(command) {
				continue
			}
			originalBytes := probeCommandResultPromptOriginalBytes(command)
			if candidate, ok := buildProbeResultAbsorptionCandidate(probeID, absorbedBy, originalBytes, true, commandIndex); ok {
				if plan.commands == nil {
					plan.commands = make(map[reviewmodelinput.ProbeCommandResultKey]ProbeResultAbsorptionCandidate)
				}
				plan.commands[key] = candidate
			}
		}
	}
	return plan
}

// Empty は absorption 候補がないかを返す。
func (p ProbeResultAbsorptionPlan) Empty() bool {
	return len(p.probes) == 0 && len(p.commands) == 0
}

// ProbeCount は full probe result absorption 候補数を返す。
func (p ProbeResultAbsorptionPlan) ProbeCount() int {
	return len(p.probes)
}

// CommandCount は command-level absorption 候補数を返す。
func (p ProbeResultAbsorptionPlan) CommandCount() int {
	return len(p.commands)
}

// ProbeIDs は full probe result absorption 候補 ID を安定順で返す。
func (p ProbeResultAbsorptionPlan) ProbeIDs() []string {
	return sortedProbeAbsorptionProbeIDs(p.probes)
}

// CommandKeys は command-level absorption 候補 key を安定順で返す。
func (p ProbeResultAbsorptionPlan) CommandKeys() []reviewmodelinput.ProbeCommandResultKey {
	return sortedProbeCommandAbsorptionKeys(p.commands)
}

// ProbeCandidate は probe ID に対応する absorption 候補を返す。
func (p ProbeResultAbsorptionPlan) ProbeCandidate(probeID string) (ProbeResultAbsorptionCandidate, bool) {
	candidate, ok := p.probes[probeID]
	return candidate, ok
}

// CommandCandidate は command key に対応する absorption 候補を返す。
func (p ProbeResultAbsorptionPlan) CommandCandidate(key reviewmodelinput.ProbeCommandResultKey) (ProbeResultAbsorptionCandidate, bool) {
	candidate, ok := p.commands[key]
	return candidate, ok
}

// CandidatePromptContextOptions は未適用の absorption 候補を prompt context options へ反映する。
func (p ProbeResultAbsorptionPlan) CandidatePromptContextOptions(opts reviewmodelinput.ProbeResultPromptContextOptions) reviewmodelinput.ProbeResultPromptContextOptions {
	if len(p.probes) > 0 {
		opts.AbsorptionCandidateProbeIDs = make(map[string]struct{}, len(p.probes))
		for _, probeID := range p.ProbeIDs() {
			opts.AbsorptionCandidateProbeIDs[probeID] = struct{}{}
		}
	}
	if len(p.commands) > 0 {
		opts.AbsorptionCandidateCommands = make(map[reviewmodelinput.ProbeCommandResultKey]struct{}, len(p.commands))
		for _, key := range p.CommandKeys() {
			opts.AbsorptionCandidateCommands[key] = struct{}{}
		}
	}
	return opts
}

// AbsorbedPromptContextOptions は適用済み absorption summary を prompt context options へ反映する。
func (p ProbeResultAbsorptionPlan) AbsorbedPromptContextOptions(opts reviewmodelinput.ProbeResultPromptContextOptions, refs ProbeResultAbsorptionArtifactRefs) reviewmodelinput.ProbeResultPromptContextOptions {
	if len(p.probes) > 0 {
		opts.AbsorbedProbeResults = make(map[string]reviewmodelinput.ProbeResultAbsorptionSummary, len(p.probes))
		for _, probeID := range p.ProbeIDs() {
			refID := refs.probeResultRef(probeID)
			candidate := p.probes[probeID].withRawArtifactRef(refID)
			if refID != "" {
				candidate.summary.Summary = probeResultAbsorptionAppliedSummary(probeID, false, 0)
			}
			opts.AbsorbedProbeResults[probeID] = candidate.summaryCopy()
		}
	}
	if len(p.commands) > 0 {
		opts.AbsorbedProbeCommands = make(map[reviewmodelinput.ProbeCommandResultKey]reviewmodelinput.ProbeResultAbsorptionSummary, len(p.commands))
		for _, key := range p.CommandKeys() {
			refID := refs.probeCommandRef(key)
			candidate := p.commands[key].withRawArtifactRef(refID)
			if refID != "" {
				candidate.summary.Summary = probeResultAbsorptionAppliedSummary(key.ProbeID, true, key.CommandIndex)
			}
			opts.AbsorbedProbeCommands[key] = candidate.summaryCopy()
		}
	}
	return opts
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

func (r ProbeResultAbsorptionArtifactRefs) probeResultRef(probeID string) string {
	if len(r.ProbeResults) == 0 {
		return ""
	}
	return strings.TrimSpace(r.ProbeResults[probeID])
}

func (r ProbeResultAbsorptionArtifactRefs) probeCommandRef(key reviewmodelinput.ProbeCommandResultKey) string {
	if len(r.ProbeCommands) == 0 {
		return ""
	}
	return strings.TrimSpace(r.ProbeCommands[key])
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
