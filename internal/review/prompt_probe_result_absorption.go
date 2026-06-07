package review

import (
	"context"
	"fmt"
	"sort"
	"strings"

	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	reviewPromptProbeResultAbsorptionMinSavedTokens = 128
)

type reviewProbeResultAbsorptionCandidate struct {
	summary          reviewmodelinput.ProbeResultAbsorptionSummary
	originalBytes    int
	replacementBytes int
	savedBytes       int
	savedTokens      int
}

type reviewProbeResultAbsorptionCandidates struct {
	probes   map[string]reviewProbeResultAbsorptionCandidate
	commands map[reviewmodelinput.ProbeCommandResultKey]reviewProbeResultAbsorptionCandidate
}

func (c reviewProbeResultAbsorptionCandidates) empty() bool {
	return len(c.probes) == 0 && len(c.commands) == 0
}

type reviewProbeResultPromptContextBuild struct {
	options          reviewmodelinput.ProbeResultPromptContextOptions
	rawOutputContext string
	rawOutputLedger  *ReviewProbeRawOutputLedger
}

func (r *ReviewRunner) probeResultPromptContextBuildForAbsorbedReport(ctx context.Context, phase ReviewModelPhase, promptKind string, report ReviewReport, probeResults []ReviewProbeResult, redactor reviewmodelinput.Redactor) reviewProbeResultPromptContextBuild {
	opts := r.probeResultPromptContextOptions()
	if r == nil {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	mode := normalizeReviewPromptReductionMode(r.promptReductionMode)
	if mode == ReviewPromptReductionModeOff {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	candidates := buildReviewProbeResultAbsorptionCandidates(report, probeResults)
	if candidates.empty() {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	if len(candidates.probes) > 0 {
		opts.AbsorptionCandidateProbeIDs = make(map[string]struct{}, len(candidates.probes))
		for probeID := range candidates.probes {
			opts.AbsorptionCandidateProbeIDs[probeID] = struct{}{}
		}
	}
	if len(candidates.commands) > 0 {
		opts.AbsorptionCandidateCommands = make(map[reviewmodelinput.ProbeCommandResultKey]struct{}, len(candidates.commands))
		for key := range candidates.commands {
			opts.AbsorptionCandidateCommands[key] = struct{}{}
		}
	}
	if r.promptReductionStats == nil {
		r.promptReductionStats = newReviewPromptReductionStats(mode)
	}
	rawOutput := r.buildReviewProbeRawOutputForCandidates(ctx, phase, promptKind, candidates, probeResults, redactor)
	applied := rawOutput.applyAllowed
	for _, probeID := range sortedReviewProbeAbsorptionProbeIDs(candidates.probes) {
		candidate := candidates.probes[probeID]
		if ref, ok := rawOutput.probeRefs[probeID]; ok {
			candidate.summary.RawArtifactRef = ref.RefID
			candidate.replacementBytes = len(candidate.summary.Summary) + len(strings.Join(candidate.summary.AbsorbedBy, "\n")) + len(candidate.summary.RawArtifactRef)
		}
		r.promptReductionStats.record("probe_result_absorption_candidate", candidate.savedBytes, candidate.savedTokens, applied)
		if !applied {
			r.promptReductionStats.recordKeepReason(reviewProbeResultAbsorptionKeepReason(rawOutput))
		}
		r.recordPromptReductionItem(ReviewPromptReductionItem{
			ID:               "probe_result:" + probeID,
			Family:           ReviewPromptReductionFamilyProbeResult,
			Phase:            phase,
			Status:           reviewPromptProbeResultAbsorptionStatus(applied),
			AbsorbedBy:       reviewPromptAbsorptionRefsFromOwners(candidate.summary.AbsorbedBy),
			EvidenceRefs:     []ReviewEvidenceRef{{Kind: ReviewEvidenceKindProbe, ProbeID: probeID}},
			RawArtifactRef:   candidate.summary.RawArtifactRef,
			Summary:          candidate.summary.Summary,
			OriginalBytes:    candidate.originalBytes,
			ReplacementBytes: candidate.replacementBytes,
		})
	}
	for _, key := range sortedReviewProbeCommandAbsorptionKeys(candidates.commands) {
		candidate := candidates.commands[key]
		if ref, ok := rawOutput.commandRefs[key]; ok {
			candidate.summary.RawArtifactRef = ref.RefID
			candidate.replacementBytes = len(candidate.summary.Summary) + len(strings.Join(candidate.summary.AbsorbedBy, "\n")) + len(candidate.summary.RawArtifactRef)
		}
		r.promptReductionStats.record("probe_command_result_absorption_candidate", candidate.savedBytes, candidate.savedTokens, applied)
		if !applied {
			r.promptReductionStats.recordKeepReason(reviewProbeResultAbsorptionKeepReason(rawOutput))
		}
		r.recordPromptReductionItem(ReviewPromptReductionItem{
			ID:               fmt.Sprintf("probe_result:%s:command:%d", key.ProbeID, key.CommandIndex),
			Family:           ReviewPromptReductionFamilyProbeResult,
			Phase:            phase,
			Status:           reviewPromptProbeResultAbsorptionStatus(applied),
			AbsorbedBy:       reviewPromptAbsorptionRefsFromOwners(candidate.summary.AbsorbedBy),
			EvidenceRefs:     []ReviewEvidenceRef{{Kind: ReviewEvidenceKindProbeCommand, ProbeID: key.ProbeID, CommandIndex: ReviewCommandIndex(key.CommandIndex)}},
			RawArtifactRef:   candidate.summary.RawArtifactRef,
			Summary:          candidate.summary.Summary,
			OriginalBytes:    candidate.originalBytes,
			ReplacementBytes: candidate.replacementBytes,
		})
	}
	rawOutputLedger := reviewProbeRawOutputLedgerPtr(rawOutput.ledger)
	if rawOutputLedger != nil {
		r.promptReductionStats.recordRawOutputLedger(*rawOutputLedger)
		if r.promptReductionState != nil {
			r.promptReductionState.Report = r.promptReductionStats.reportValue()
		}
	}
	if !applied {
		return reviewProbeResultPromptContextBuild{options: opts}
	}
	if len(candidates.probes) > 0 {
		opts.AbsorbedProbeResults = make(map[string]reviewmodelinput.ProbeResultAbsorptionSummary, len(candidates.probes))
		for probeID, candidate := range candidates.probes {
			if ref, ok := rawOutput.probeRefs[probeID]; ok {
				candidate.summary.RawArtifactRef = ref.RefID
				candidate.summary.Summary = reviewProbeResultAbsorptionAppliedSummary(probeID, false, 0)
			}
			opts.AbsorbedProbeResults[probeID] = candidate.summary
		}
	}
	if len(candidates.commands) > 0 {
		opts.AbsorbedProbeCommands = make(map[reviewmodelinput.ProbeCommandResultKey]reviewmodelinput.ProbeResultAbsorptionSummary, len(candidates.commands))
		for key, candidate := range candidates.commands {
			if ref, ok := rawOutput.commandRefs[key]; ok {
				candidate.summary.RawArtifactRef = ref.RefID
				candidate.summary.Summary = reviewProbeResultAbsorptionAppliedSummary(key.ProbeID, true, key.CommandIndex)
			}
			opts.AbsorbedProbeCommands[key] = candidate.summary
		}
	}
	return reviewProbeResultPromptContextBuild{
		options:          opts,
		rawOutputContext: rawOutput.context,
		rawOutputLedger:  rawOutputLedger,
	}
}

func reviewPromptProbeResultAbsorptionStatus(applied bool) ReviewPromptReductionItemStatus {
	if applied {
		return ReviewPromptReductionItemAbsorbed
	}
	return ReviewPromptReductionItemCandidate
}

func buildReviewProbeResultAbsorptionCandidates(report ReviewReport, probeResults []ReviewProbeResult) reviewProbeResultAbsorptionCandidates {
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

type reviewProbeResultAbsorptionRefSet struct {
	safeProbes            map[string][]string
	safeCommands          map[reviewmodelinput.ProbeCommandResultKey][]string
	unsafeProbes          map[string]struct{}
	unsafeCommands        map[reviewmodelinput.ProbeCommandResultKey]struct{}
	unsafeCommandProbeIDs map[string]struct{}
}

func (refs reviewProbeResultAbsorptionRefSet) probeUnsafeForFullAbsorption(probeID string) bool {
	if _, unsafe := refs.unsafeProbes[probeID]; unsafe {
		return true
	}
	_, unsafe := refs.unsafeCommandProbeIDs[probeID]
	return unsafe
}

func (refs reviewProbeResultAbsorptionRefSet) commandUnsafeForAbsorption(key reviewmodelinput.ProbeCommandResultKey) bool {
	if _, unsafe := refs.unsafeProbes[key.ProbeID]; unsafe {
		return true
	}
	_, unsafe := refs.unsafeCommands[key]
	return unsafe
}

func reviewProbeResultAbsorptionRefs(report ReviewReport) reviewProbeResultAbsorptionRefSet {
	refs := reviewProbeResultAbsorptionRefSet{
		safeProbes:            make(map[string][]string),
		safeCommands:          make(map[reviewmodelinput.ProbeCommandResultKey][]string),
		unsafeProbes:          make(map[string]struct{}),
		unsafeCommands:        make(map[reviewmodelinput.ProbeCommandResultKey]struct{}),
		unsafeCommandProbeIDs: make(map[string]struct{}),
	}
	addRefs := func(evidenceRefs []ReviewEvidenceRef, owner string, safe bool) {
		for _, ref := range evidenceRefs {
			probeID := strings.TrimSpace(ref.ProbeID)
			if probeID == "" {
				continue
			}
			switch ref.Kind {
			case ReviewEvidenceKindProbe:
				if safe {
					refs.safeProbes[probeID] = append(refs.safeProbes[probeID], owner)
				} else {
					refs.unsafeProbes[probeID] = struct{}{}
				}
			case ReviewEvidenceKindProbeCommand:
				key, ok := reviewProbeCommandResultKeyFromEvidenceRef(ref)
				if !ok {
					refs.unsafeProbes[probeID] = struct{}{}
					continue
				}
				if safe {
					refs.safeCommands[key] = append(refs.safeCommands[key], owner)
				} else {
					refs.unsafeCommands[key] = struct{}{}
					refs.unsafeCommandProbeIDs[probeID] = struct{}{}
				}
			default:
				continue
			}
		}
	}

	if report.ScopeCoverage != nil {
		for _, surface := range report.ScopeCoverage.ReviewedImpactSurfaces {
			owner := "scope_coverage.surface." + strings.TrimSpace(surface.SurfaceID)
			addRefs(surface.EvidenceRefs, owner, surface.Status == ReviewReportImpactSurfaceChecked)
		}
		for _, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
			owner := "scope_coverage.risk." + strings.TrimSpace(risk.RiskID)
			addRefs(risk.EvidenceRefs, owner, risk.Status == ReviewReportCandidateRiskDismissed)
		}
		for _, finding := range report.ScopeCoverage.NewFindingsFromReportPass {
			addRefs(finding.EvidenceRefs, "scope_coverage.new_finding", false)
		}
	}

	for _, surface := range report.CheckedSurfaces {
		addRefs(surface.EvidenceRefs, "checked_surfaces", false)
	}
	for _, surface := range report.UnverifiedSurfaces {
		addRefs(surface.EvidenceRefs, "unverified_surfaces", false)
	}
	for _, risk := range report.ResidualRisks {
		addRefs(risk.EvidenceRefs, "residual_risks", false)
	}
	for _, group := range report.RootCauseGroups {
		for _, finding := range group.Findings {
			addRefs(finding.EvidenceRefs, "root_cause.finding", false)
			for _, surface := range finding.CheckedSurfaces {
				addRefs(surface.EvidenceRefs, "root_cause.finding.checked_surface", false)
			}
			for _, surface := range finding.UnverifiedSurfaces {
				addRefs(surface.EvidenceRefs, "root_cause.finding.unverified_surface", false)
			}
			for _, risk := range finding.ResidualRisks {
				addRefs(risk.EvidenceRefs, "root_cause.finding.residual_risk", false)
			}
		}
		for _, surface := range group.CheckedSurfaces {
			addRefs(surface.EvidenceRefs, "root_cause.checked_surface", false)
		}
		for _, surface := range group.UnverifiedSurfaces {
			addRefs(surface.EvidenceRefs, "root_cause.unverified_surface", false)
		}
		for _, risk := range group.ResidualRisks {
			addRefs(risk.EvidenceRefs, "root_cause.residual_risk", false)
		}
	}

	for probeID, values := range refs.safeProbes {
		refs.safeProbes[probeID] = dedupeSortedReviewPromptAbsorptionRefs(values)
	}
	for key, values := range refs.safeCommands {
		refs.safeCommands[key] = dedupeSortedReviewPromptAbsorptionRefs(values)
	}
	return refs
}

func reviewProbeCommandResultKeyFromEvidenceRef(ref ReviewEvidenceRef) (reviewmodelinput.ProbeCommandResultKey, bool) {
	probeID := strings.TrimSpace(ref.ProbeID)
	if probeID == "" || ref.CommandIndex == nil || *ref.CommandIndex < 0 {
		return reviewmodelinput.ProbeCommandResultKey{}, false
	}
	return reviewmodelinput.ProbeCommandResultKey{ProbeID: probeID, CommandIndex: *ref.CommandIndex}, true
}

func reviewProbeResultSafeForAbsorbedPrompt(result ReviewProbeResult) bool {
	return result.Status == ReviewProbePassed &&
		!result.MutatedWorktree &&
		!result.OutputTruncated &&
		strings.TrimSpace(result.Error) == ""
}

func reviewProbeCommandResultSafeForAbsorbedPrompt(result ReviewProbeCommandResult) bool {
	return result.Status == ReviewProbePassed &&
		!result.OutputTruncated &&
		strings.TrimSpace(result.Error) == ""
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
