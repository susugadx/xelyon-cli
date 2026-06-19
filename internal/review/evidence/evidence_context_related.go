package evidence

import (
	pathpkg "path"
	"sort"
)

func normalizeReviewRelatedCandidatePaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		relPath, err := normalizeReviewEvidenceRelativePath(path)
		if err != nil {
			continue
		}
		if _, ok := reviewEvidenceLanguageSpecForRelatedPath(relPath); !ok {
			continue
		}
		if _, ok := seen[relPath]; ok {
			continue
		}
		seen[relPath] = struct{}{}
		result = append(result, relPath)
	}
	sort.Strings(result)
	return result
}

func (c *reviewContextEvidenceCollector) collectRelatedContextFiles(changedFiles []ReviewChangedFile) []ReviewContextFileEvidence {
	scope := c.changedRelatedLanguageScope(changedFiles)

	files := make([]ReviewContextFileEvidence, 0)
	candidates := make([]reviewContextRelatedCandidate, 0)

	for _, relPath := range c.relatedCandidatePaths {
		if c.maxContextFilesExceededLogged || c.contextErr() != nil {
			break
		}
		language, ok := reviewEvidenceLanguageSpecForRelatedPath(relPath)
		if !ok {
			continue
		}
		if !scope.hasDir(pathpkg.Dir(relPath)) {
			continue
		}
		if _, changed := c.changedPaths[relPath]; changed {
			continue
		}

		candidates = append(candidates, reviewContextRelatedCandidate{
			path:     relPath,
			role:     language.relatedContextRole(relPath),
			priority: scope.relatedCandidatePriority(relPath, language),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].path < candidates[j].path
	})

	for _, candidate := range candidates {
		if c.maxContextFilesExceededLogged || c.contextErr() != nil {
			break
		}
		evidence, ok := c.collectContextFile(candidate.path, candidate.role)
		if ok {
			files = append(files, evidence)
		}
	}

	return files
}

func (s reviewContextChangedLanguageScope) relatedCandidatePriority(relPath string, language reviewEvidenceLanguageSpec) int {
	dir := pathpkg.Dir(relPath)
	stem := language.stem(relPath)
	isTest := language.isTestPath(relPath)
	sameSourceStem := s.hasSourceStem(dir, stem)
	sameTestStem := s.hasTestStem(dir, stem)
	switch {
	case isTest && sameSourceStem:
		return 0
	case !isTest && sameTestStem:
		return 0
	case isTest:
		return 1
	case sameSourceStem || sameTestStem:
		return 2
	default:
		return 3
	}
}

func (c *reviewContextEvidenceCollector) changedRelatedLanguageScope(changedFiles []ReviewChangedFile) reviewContextChangedLanguageScope {
	scope := reviewContextChangedLanguageScope{
		stemsByDir: make(map[string]map[string]reviewContextChangedLanguageStem),
	}
	for _, file := range changedFiles {
		scope.addPath(c.repoRoot, file.Path)
		scope.addPath(c.repoRoot, file.OldPath)
	}
	return scope
}

func (s reviewContextChangedLanguageScope) hasDir(dir string) bool {
	_, ok := s.stemsByDir[dir]
	return ok
}

func (s reviewContextChangedLanguageScope) hasSourceStem(dir, stem string) bool {
	changed, ok := s.changedStem(dir, stem)
	return ok && changed.source
}

func (s reviewContextChangedLanguageScope) hasTestStem(dir, stem string) bool {
	changed, ok := s.changedStem(dir, stem)
	return ok && changed.test
}

func (s reviewContextChangedLanguageScope) changedStem(dir, stem string) (reviewContextChangedLanguageStem, bool) {
	stems, ok := s.stemsByDir[dir]
	if !ok {
		return reviewContextChangedLanguageStem{}, false
	}
	changed, ok := stems[stem]
	return changed, ok
}

func (s *reviewContextChangedLanguageScope) addPath(repoRoot, path string) {
	dir, stem, isTest, ok := relatedContextChangedLanguagePath(repoRoot, path)
	if !ok {
		return
	}
	if s.stemsByDir == nil {
		s.stemsByDir = make(map[string]map[string]reviewContextChangedLanguageStem)
	}
	if _, ok := s.stemsByDir[dir]; !ok {
		s.stemsByDir[dir] = make(map[string]reviewContextChangedLanguageStem)
	}
	changed := s.stemsByDir[dir][stem]
	if isTest {
		changed.test = true
	} else {
		changed.source = true
	}
	s.stemsByDir[dir][stem] = changed
}

func relatedContextChangedLanguagePath(repoRoot, path string) (string, string, bool, bool) {
	_, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, path)
	if err != nil {
		return "", "", false, false
	}
	language, ok := reviewEvidenceLanguageSpecForRelatedPath(relPath)
	if !ok {
		return "", "", false, false
	}
	return pathpkg.Dir(relPath), language.stem(relPath), language.isTestPath(relPath), true
}
