package analysis

import (
	"fmt"
	"strings"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func validateReviewProbePlanMaterialPathCoverage(input EvidenceInput, index reviewProbePlanImpactSurfaceEvidenceIndex) error {
	untrackedPaths := reviewProbePlanEvidenceUntrackedPathSet(input)
	for _, path := range reviewProbePlanEvidenceMaterialPaths(input) {
		if reviewProbePlanEvidencePathSetContains(untrackedPaths, path) {
			continue
		}
		if !index.coversMaterialEvidencePath(path) {
			return fmt.Errorf("impact_surfaces must cover material changed path %q in evidence_summary or evidence_refs", path)
		}
	}
	return nil
}

func reviewProbePlanEvidenceMaterialPaths(input EvidenceInput) []string {
	var paths []string
	for _, file := range input.ChangedFiles {
		paths = append(paths, file.Path, file.OldPath)
	}
	paths = append(paths, input.ChangeInventory.Generated...)
	paths = append(paths, input.ChangeInventory.Tests...)
	paths = append(paths, input.ChangeInventory.Docs...)
	paths = append(paths, input.ChangeInventory.Config...)
	paths = append(paths, input.ChangeInventory.Production...)
	paths = append(paths, input.ChangeInventory.NewFiles...)
	paths = append(paths, input.ChangeInventory.DeletedFiles...)
	paths = append(paths, input.ChangeInventory.RenamedFiles...)
	return uniqueReviewProbePlanEvidencePaths(paths)
}

func validateReviewProbePlanInventoryCategoryCoverage(input EvidenceInput, index reviewProbePlanImpactSurfaceEvidenceIndex) error {
	untrackedPaths := reviewProbePlanEvidenceUntrackedPathSet(input)
	categories := []struct {
		name  string
		paths []string
	}{
		{name: "production", paths: input.ChangeInventory.Production},
		{name: "config", paths: input.ChangeInventory.Config},
		{name: "tests", paths: input.ChangeInventory.Tests},
		{name: "docs", paths: input.ChangeInventory.Docs},
		{name: "generated", paths: input.ChangeInventory.Generated},
	}
	for _, category := range categories {
		paths := reviewProbePlanEvidencePathsExcluding(category.paths, untrackedPaths)
		if len(paths) == 0 {
			continue
		}
		if index.mentionsSurfaceTextToken(category.name) {
			continue
		}
		coveredByPath := false
		for _, path := range paths {
			if index.coversMaterialEvidencePath(path) {
				coveredByPath = true
				break
			}
		}
		if !coveredByPath {
			return fmt.Errorf("impact_surfaces must cover %s inventory category by category name or category path", category.name)
		}
	}
	return nil
}

func validateReviewProbePlanUntrackedCoverage(input EvidenceInput, plan reviewprobe.ReviewProbePlan, index reviewProbePlanImpactSurfaceEvidenceIndex) error {
	untrackedPaths := reviewProbePlanEvidenceUntrackedPaths(input)
	if len(untrackedPaths) == 0 {
		return nil
	}

	reasons := []string{plan.NoCandidateRiskReason, plan.NoProbeReason}
	for _, path := range untrackedPaths {
		if index.coversMaterialEvidencePath(path) || index.mentionsSurfaceTextToken("untracked") || reviewProbePlanReasonsCoverUntracked(path, reasons) {
			continue
		}
		return fmt.Errorf("untracked path %q must be covered by impact_surfaces or no_candidate_risk_reason/no_probe_reason", path)
	}
	return nil
}

func reviewProbePlanEvidenceUntrackedPaths(input EvidenceInput) []string {
	paths := append([]string{}, input.ChangeInventory.Untracked...)
	for _, file := range input.UntrackedFiles {
		paths = append(paths, file.Path)
	}
	return uniqueReviewProbePlanEvidencePaths(paths)
}

func reviewProbePlanEvidenceUntrackedPathSet(input EvidenceInput) map[string]struct{} {
	return newReviewProbePlanEvidencePathSet(reviewProbePlanEvidenceUntrackedPaths(input))
}

func newReviewProbePlanEvidencePathSet(paths []string) map[string]struct{} {
	set := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = normalizeReviewProbePlanEvidencePath(path)
		if path == "" {
			continue
		}
		set[path] = struct{}{}
	}
	return set
}

func reviewProbePlanEvidencePathSetContains(set map[string]struct{}, path string) bool {
	path = normalizeReviewProbePlanEvidencePath(path)
	if path == "" {
		return false
	}
	_, exists := set[path]
	return exists
}

func reviewProbePlanEvidencePathsExcluding(paths []string, excluded map[string]struct{}) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if reviewProbePlanEvidencePathSetContains(excluded, path) {
			continue
		}
		result = append(result, path)
	}
	return result
}

func reviewProbePlanReasonsCoverUntracked(path string, reasons []string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, path) || strings.Contains(strings.ToLower(reason), "untracked") {
			return true
		}
	}
	return false
}
