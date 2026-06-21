package analysis

import (
	"path/filepath"
	"strconv"
	"strings"
)

func reviewPressureSignalGenericImpactCandidatesPresentEvidence(input EvidenceInput) []string {
	if len(input.GenericImpact.Candidates) == 0 {
		return nil
	}
	return reviewPressureSignalGenericImpactCandidateEvidence(input.GenericImpact.Candidates)
}

func reviewPressureSignalGenericImpactCandidatesTruncatedEvidence(input EvidenceInput) []string {
	if !input.GenericImpact.Truncated {
		return nil
	}
	evidence := []string{"generic_impact_candidates: truncated"}
	evidence = append(evidence, reviewPressureSignalGenericImpactCandidateEvidence(input.GenericImpact.Candidates)...)
	return evidence
}

func reviewPressureSignalGenericImpactCandidatesTestsOrDocsEvidence(input EvidenceInput) []string {
	evidence := make([]string, 0)
	for _, candidate := range input.GenericImpact.Candidates {
		switch candidate.Role {
		case "same_stem_test_or_spec", "nearby_test_or_tests_dir", "docs_reference":
			evidence = append(evidence, "generic_impact_candidate: "+candidate.Role+" "+candidate.Path)
		}
		if len(evidence) == reviewPressureSignalMaxPathEvidence {
			break
		}
	}
	return evidence
}

func reviewPressureSignalGenericImpactCandidatesEmptyForNonGoEvidence(input EvidenceInput) []string {
	if len(input.GenericImpact.Candidates) > 0 || !reviewPressureSignalHasNonGoChangedPath(input) {
		return nil
	}
	return []string{"generic_impact_candidates: []", "non_go_changed_paths: present"}
}

func reviewPressureSignalGenericImpactCandidateEvidence(candidates []GenericImpactCandidate) []string {
	evidence := make([]string, 0, minReviewAnalysisInt(len(candidates), reviewPressureSignalMaxPathEvidence)+1)
	for i, candidate := range candidates {
		if i >= reviewPressureSignalMaxPathEvidence {
			evidence = append(evidence, "generic_impact_candidates: ... ("+strconv.Itoa(len(candidates)-i)+" more)")
			break
		}
		item := "generic_impact_candidate: " + candidate.Role + " " + candidate.Path
		if candidate.Token != "" {
			item += " token=" + candidate.Token
		}
		evidence = append(evidence, item)
	}
	return evidence
}

func reviewPressureSignalHasNonGoChangedPath(input EvidenceInput) bool {
	for _, path := range reviewPressureSignalAllInventoryPaths(input.ChangeInventory) {
		normalized := strings.ToLower(filepath.ToSlash(path))
		if strings.TrimSpace(normalized) == "" || normalized == outsideRepoPathDisplay {
			continue
		}
		if filepath.Ext(normalized) != ".go" {
			return true
		}
	}
	return false
}
