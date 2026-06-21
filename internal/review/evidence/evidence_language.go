package evidence

import (
	pathpkg "path"
	"strings"
)

type reviewRelatedSearchTermAdder func(term, reason string, priority int) bool

type reviewEvidenceLanguageSpec struct {
	fileExtension             string
	testFileSuffix            string
	relatedCandidatePathspec  string
	relatedSourceContextRole  string
	relatedTestContextRole    string
	extractRelatedSearchTerms func(reviewEvidenceLanguageSpec, ReviewContextFileEvidence, reviewRelatedSearchTermAdder) bool
}

var reviewEvidenceLanguageCatalog = []reviewEvidenceLanguageSpec{
	reviewEvidenceGoLanguage,
}

func reviewEvidenceLanguageSpecForRelatedPath(relPath string) (reviewEvidenceLanguageSpec, bool) {
	for _, language := range reviewEvidenceLanguageCatalog {
		if language.isRelatedPath(relPath) {
			return language, true
		}
	}
	return reviewEvidenceLanguageSpec{}, false
}

func reviewEvidenceLanguageSpecForPath(relPath string) (reviewEvidenceLanguageSpec, bool) {
	for _, language := range reviewEvidenceLanguageCatalog {
		if language.matchesFileExtension(relPath) {
			return language, true
		}
	}
	return reviewEvidenceLanguageSpec{}, false
}

func reviewEvidenceRelatedCandidatePathspecs() []string {
	pathspecs := make([]string, 0, len(reviewEvidenceLanguageCatalog))
	for _, language := range reviewEvidenceLanguageCatalog {
		if language.relatedCandidatePathspec == "" {
			continue
		}
		pathspecs = append(pathspecs, language.relatedCandidatePathspec)
	}
	return pathspecs
}

func (s reviewEvidenceLanguageSpec) isRelatedPath(relPath string) bool {
	return s.matchesFileExtension(relPath) &&
		!isReviewContextGeneratedPath(relPath) &&
		!isReviewContextVendorPath(relPath)
}

func (s reviewEvidenceLanguageSpec) matchesFileExtension(relPath string) bool {
	return s.fileExtension != "" && pathpkg.Ext(relPath) == s.fileExtension
}

func (s reviewEvidenceLanguageSpec) isTestPath(relPath string) bool {
	return s.testFileSuffix != "" &&
		strings.HasSuffix(strings.ToLower(pathpkg.Base(relPath)), s.testFileSuffix)
}

func (s reviewEvidenceLanguageSpec) stem(relPath string) string {
	base := pathpkg.Base(relPath)
	if s.isTestPath(relPath) {
		return base[:len(base)-len(s.testFileSuffix)]
	}
	return strings.TrimSuffix(base, pathpkg.Ext(base))
}

func (s reviewEvidenceLanguageSpec) relatedContextRole(relPath string) string {
	if s.isTestPath(relPath) {
		return s.relatedTestContextRole
	}
	return s.relatedSourceContextRole
}

func isReviewContextGeneratedPath(path string) bool {
	return matchGeneratedReviewInventoryPath(newReviewInventoryPath(path))
}

func isReviewContextVendorPath(path string) bool {
	return newReviewInventoryPath(path).hasDir("vendor")
}
