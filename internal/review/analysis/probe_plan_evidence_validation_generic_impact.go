package analysis

import (
	"fmt"
	"sort"
	"strings"
)

func validateReviewProbePlanGenericImpactCoverage(input EvidenceInput, index reviewProbePlanImpactSurfaceEvidenceIndex) error {
	if len(input.GenericImpact.Candidates) == 0 {
		return nil
	}
	candidatesByRole := make(map[string][]GenericImpactCandidate)
	roles := make([]string, 0)
	for _, candidate := range input.GenericImpact.Candidates {
		role := strings.TrimSpace(candidate.Role)
		if role == "" {
			continue
		}
		if _, ok := candidatesByRole[role]; !ok {
			roles = append(roles, role)
		}
		candidatesByRole[role] = append(candidatesByRole[role], candidate)
	}
	sort.Strings(roles)
	for _, role := range roles {
		if index.mentionsSurfaceTextRole(role) {
			continue
		}
		covered := false
		for _, candidate := range candidatesByRole[role] {
			token := strings.TrimSpace(candidate.Token)
			if index.coversGenericImpactCandidatePath(candidate.Path) || (token != "" && index.mentionsSurfaceTextToken(token)) {
				covered = true
				break
			}
		}
		if !covered {
			return fmt.Errorf("impact_surfaces must cover generic impact candidates role %q by role, candidate path, or token", role)
		}
	}
	return nil
}
