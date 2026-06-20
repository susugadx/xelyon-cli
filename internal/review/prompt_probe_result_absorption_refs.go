package review

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

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

func reviewProbeResultAbsorptionRefs(report reviewreport.ReviewReport) reviewProbeResultAbsorptionRefSet {
	refs := reviewProbeResultAbsorptionRefSet{
		safeProbes:            make(map[string][]string),
		safeCommands:          make(map[reviewmodelinput.ProbeCommandResultKey][]string),
		unsafeProbes:          make(map[string]struct{}),
		unsafeCommands:        make(map[reviewmodelinput.ProbeCommandResultKey]struct{}),
		unsafeCommandProbeIDs: make(map[string]struct{}),
	}
	addRefs := func(evidenceRefs []reviewreport.ReviewEvidenceRef, owner string, safe bool) {
		for _, ref := range evidenceRefs {
			probeID := strings.TrimSpace(ref.ProbeID)
			if probeID == "" {
				continue
			}
			switch ref.Kind {
			case reviewreport.ReviewEvidenceKindProbe:
				if safe {
					refs.safeProbes[probeID] = append(refs.safeProbes[probeID], owner)
				} else {
					refs.unsafeProbes[probeID] = struct{}{}
				}
			case reviewreport.ReviewEvidenceKindProbeCommand:
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
			addRefs(surface.EvidenceRefs, owner, surface.Status == reviewreport.ReviewReportImpactSurfaceChecked)
		}
		for _, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
			owner := "scope_coverage.risk." + strings.TrimSpace(risk.RiskID)
			addRefs(risk.EvidenceRefs, owner, risk.Status == reviewreport.ReviewReportCandidateRiskDismissed)
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
		refs.safeProbes[probeID] = reviewpromptreduction.DedupeSortedReviewPromptAbsorptionRefs(values)
	}
	for key, values := range refs.safeCommands {
		refs.safeCommands[key] = reviewpromptreduction.DedupeSortedReviewPromptAbsorptionRefs(values)
	}
	return refs
}

func reviewProbeCommandResultKeyFromEvidenceRef(ref reviewreport.ReviewEvidenceRef) (reviewmodelinput.ProbeCommandResultKey, bool) {
	probeID := strings.TrimSpace(ref.ProbeID)
	if probeID == "" || ref.CommandIndex == nil || *ref.CommandIndex < 0 {
		return reviewmodelinput.ProbeCommandResultKey{}, false
	}
	return reviewmodelinput.ProbeCommandResultKey{ProbeID: probeID, CommandIndex: *ref.CommandIndex}, true
}

func reviewProbeResultSafeForAbsorbedPrompt(result reviewprobe.ReviewProbeResult) bool {
	return result.Status == domain.ReviewProbePassed &&
		!result.MutatedWorktree &&
		!result.OutputTruncated &&
		strings.TrimSpace(result.Error) == ""
}

func reviewProbeCommandResultSafeForAbsorbedPrompt(result reviewprobe.ReviewProbeCommandResult) bool {
	return result.Status == domain.ReviewProbePassed &&
		!result.OutputTruncated &&
		strings.TrimSpace(result.Error) == ""
}
