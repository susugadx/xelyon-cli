package analysis

import (
	"fmt"
	"sort"
	"strings"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

// ValidateProbePlanAgainstEvidence は Pass1 probe plan が evidence input の
// material path と evidence pressure を扱っていることを検証する。
func ValidateProbePlanAgainstEvidence(plan reviewprobe.ReviewProbePlan, input EvidenceInput) error {
	if err := reviewprobe.ValidateReviewProbePlan(plan); err != nil {
		return err
	}
	if err := ValidateProbePlanExternalDocRefs(plan, input.WebSearchEvidence.ExternalDocs); err != nil {
		return err
	}

	index := newReviewProbePlanImpactSurfaceEvidenceIndex(plan.ImpactSurfaces)

	if err := validateReviewProbePlanMaterialPathCoverage(input, index); err != nil {
		return err
	}
	if err := validateReviewProbePlanInventoryCategoryCoverage(input, index); err != nil {
		return err
	}
	if err := validateReviewProbePlanUntrackedCoverage(input, plan, index); err != nil {
		return err
	}
	if err := validateReviewProbePlanGenericImpactCoverage(input, index); err != nil {
		return err
	}
	if err := validateReviewProbePlanTruncationPressure(input, plan); err != nil {
		return err
	}
	if err := validateReviewProbePlanGenericImpactTruncationPressure(input, plan); err != nil {
		return err
	}
	if err := validateReviewProbePlanNoProbeRequiresRelatedEvidence(input, plan); err != nil {
		return err
	}
	return nil
}

type reviewProbePlanImpactSurfaceEvidenceIndex struct {
	evidenceSummaries []string
	surfaceTexts      []string
	refPaths          map[string]struct{}
}

func newReviewProbePlanImpactSurfaceEvidenceIndex(surfaces []reviewprobe.ReviewProbeImpactSurface) reviewProbePlanImpactSurfaceEvidenceIndex {
	index := reviewProbePlanImpactSurfaceEvidenceIndex{
		evidenceSummaries: make([]string, 0, len(surfaces)),
		surfaceTexts:      make([]string, 0, len(surfaces)*3),
		refPaths:          make(map[string]struct{}),
	}
	for _, surface := range surfaces {
		index.evidenceSummaries = append(index.evidenceSummaries, surface.EvidenceSummary)
		index.surfaceTexts = append(index.surfaceTexts, surface.Summary, surface.EvidenceSummary, surface.Reason)
		for _, ref := range surface.EvidenceRefs {
			if ref.Path == "" {
				continue
			}
			index.refPaths[normalizeReviewProbePlanEvidencePath(ref.Path)] = struct{}{}
		}
	}
	return index
}

func (i reviewProbePlanImpactSurfaceEvidenceIndex) coversMaterialEvidencePath(path string) bool {
	path = normalizeReviewProbePlanEvidencePath(path)
	if path == "" {
		return true
	}
	if _, exists := i.refPaths[path]; exists {
		return true
	}
	for _, summary := range i.evidenceSummaries {
		if strings.Contains(summary, path) {
			return true
		}
	}
	return false
}

func (i reviewProbePlanImpactSurfaceEvidenceIndex) mentionsSurfaceTextToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return true
	}
	for _, text := range i.surfaceTexts {
		if reviewProbePlanSurfaceTextContainsToken(text, token) {
			return true
		}
	}
	return false
}

func (i reviewProbePlanImpactSurfaceEvidenceIndex) coversGenericImpactCandidatePath(path string) bool {
	path = normalizeReviewProbePlanEvidencePath(path)
	if path == "" {
		return false
	}
	if i.coversMaterialEvidencePath(path) {
		return true
	}
	for _, text := range i.surfaceTexts {
		if reviewProbePlanSurfaceTextContainsPath(text, path) {
			return true
		}
	}
	return false
}

func (i reviewProbePlanImpactSurfaceEvidenceIndex) mentionsSurfaceTextRole(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return true
	}
	roleWords := strings.ReplaceAll(strings.ReplaceAll(role, "_", " "), "-", " ")
	for _, summary := range i.surfaceTexts {
		normalized := strings.ToLower(summary)
		if strings.Contains(normalized, role) || strings.Contains(strings.ReplaceAll(normalized, "-", " "), roleWords) {
			return true
		}
	}
	return false
}

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

func validateReviewProbePlanTruncationPressure(input EvidenceInput, plan reviewprobe.ReviewProbePlan) error {
	if !reviewEvidenceInputHasDiffContextOrSearchTruncation(input) || !reviewProbePlanAllImpactSurfacesChecked(plan) {
		return nil
	}
	return fmt.Errorf("impact_surfaces cannot all be checked when diff, context, or search evidence was truncated")
}

func validateReviewProbePlanGenericImpactTruncationPressure(input EvidenceInput, plan reviewprobe.ReviewProbePlan) error {
	if !input.GenericImpact.Truncated || len(plan.Probes) > 0 || !reviewProbePlanAllImpactSurfacesChecked(plan) {
		return nil
	}
	return fmt.Errorf("impact_surfaces cannot all be checked without probes when generic impact candidates were truncated")
}

func reviewEvidenceInputHasDiffContextOrSearchTruncation(input EvidenceInput) bool {
	for _, diff := range input.TruncationFlags.Diffs {
		if diff.Stat || diff.NameStatus || diff.Diff {
			return true
		}
	}
	for _, file := range input.TruncationFlags.ChangedFileContext {
		if file.Truncated {
			return true
		}
	}
	for _, file := range input.TruncationFlags.RelatedContextFiles {
		if file.Truncated {
			return true
		}
	}
	return input.TruncationFlags.RelatedCandidates || input.TruncationFlags.RelatedSearch
}

func validateReviewProbePlanNoProbeRequiresRelatedEvidence(input EvidenceInput, plan reviewprobe.ReviewProbePlan) error {
	if len(plan.Probes) > 0 || !reviewProbePlanAllImpactSurfacesChecked(plan) {
		return nil
	}
	if len(input.RelatedContextFiles) > 0 || len(input.RelatedSearchHits) > 0 {
		return nil
	}
	return fmt.Errorf("no-probe all-checked plan requires related context files or related search hits; absence of related evidence is not proof of no impact")
}

func reviewProbePlanAllImpactSurfacesChecked(plan reviewprobe.ReviewProbePlan) bool {
	if len(plan.ImpactSurfaces) == 0 {
		return false
	}
	for _, surface := range plan.ImpactSurfaces {
		if surface.Status != reviewprobe.ReviewProbeImpactSurfaceChecked {
			return false
		}
	}
	return true
}

func uniqueReviewProbePlanEvidencePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = normalizeReviewProbePlanEvidencePath(path)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}

func normalizeReviewProbePlanEvidencePath(path string) string {
	return strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
}

func reviewProbePlanSurfaceTextContainsToken(text, token string) bool {
	text = strings.ToLower(text)
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return true
	}
	return reviewProbePlanSurfaceTextContainsDelimitedLiteral(text, token, isReviewProbePlanTokenContinuationByte)
}

func reviewProbePlanSurfaceTextContainsPath(text, path string) bool {
	text = strings.ReplaceAll(text, "\\", "/")
	path = normalizeReviewProbePlanEvidencePath(path)
	if path == "" {
		return false
	}
	return reviewProbePlanSurfaceTextContainsDelimitedLiteral(text, path, isReviewProbePlanPathContinuationByte)
}

func reviewProbePlanSurfaceTextContainsDelimitedLiteral(text, literal string, isContinuation func(byte) bool) bool {
	start := 0
	for {
		index := strings.Index(text[start:], literal)
		if index < 0 {
			return false
		}
		index += start
		beforeOK := index == 0 || !isContinuation(text[index-1])
		afterIndex := index + len(literal)
		afterOK := afterIndex >= len(text) || !isContinuation(text[afterIndex])
		if beforeOK && afterOK {
			return true
		}
		start = afterIndex
	}
}

func isReviewProbePlanTokenContinuationByte(ch byte) bool {
	return ch == '_' || ch == '-' || ch == '/' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

func isReviewProbePlanPathContinuationByte(ch byte) bool {
	return ch == '_' || ch == '-' || ch == '/' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}
