package analysis

import (
	"strings"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

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
