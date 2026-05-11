package review

import (
	"context"
	pathpkg "path"
	"sort"
	"strings"
)

const (
	reviewContextFileRoleChanged     = "changed_file"
	reviewContextFileRoleRelatedGo   = "related_go"
	reviewContextFileRoleRelatedTest = "related_test"

	reviewContextSkipGenerated           = "generated"
	reviewContextSkipVendor              = "vendor"
	reviewContextSkipSymlink             = "symlink"
	reviewContextSkipNonRegular          = "non_regular"
	reviewContextSkipBinary              = "binary"
	reviewContextSkipTooLarge            = "too_large"
	reviewContextSkipTotalBudgetExceeded = "total_budget_exceeded"
	reviewContextSkipMaxFilesExceeded    = "max_files_exceeded"
	reviewContextSkipInvalidPath         = "invalid_path"
	reviewContextSkipStatFailed          = "stat_failed"
	reviewContextSkipReadFailed          = "read_failed"
)

type reviewContextEvidence struct {
	changedFileContext     []ReviewContextFileEvidence
	relatedContextFiles    []ReviewContextFileEvidence
	relatedSearchHits      []ReviewRelatedSearchHit
	relatedSearchTruncated bool
}

type reviewContextEvidenceCollector struct {
	repoRoot string
	ctx      context.Context
	limits   ReviewEvidenceLimits

	contextFileCount              int
	totalContextRead              int64
	maxContextFilesExceededLogged bool

	changedPaths          map[string]struct{}
	relatedCandidatePaths []string
}

type reviewContextRelatedCandidate struct {
	path     string
	role     string
	priority int
}

type reviewContextChangedGoStem struct {
	source bool
	test   bool
}

type reviewContextChangedGoScope struct {
	stemsByDir map[string]map[string]reviewContextChangedGoStem
}

func buildReviewContextEvidence(ctx context.Context, repoRoot string, changedFiles []ReviewChangedFile, relatedCandidatePaths []string, limits ReviewEvidenceLimits) (reviewContextEvidence, error) {
	collector := newReviewContextEvidenceCollector(ctx, repoRoot, limits, changedFiles, relatedCandidatePaths)
	changedFileContext := collector.collectChangedFileContext(changedFiles)
	if err := collector.contextErr(); err != nil {
		return reviewContextEvidence{}, err
	}
	relatedContextFiles := collector.collectRelatedContextFiles(changedFiles)
	if err := collector.contextErr(); err != nil {
		return reviewContextEvidence{}, err
	}
	relatedSearchCollector := newReviewRelatedSearchCollector(collector.ctx, collector.repoRoot, collector.limits, collector.changedPaths, collector.relatedCandidatePaths)
	relatedSearch, err := relatedSearchCollector.collect(changedFileContext)
	if err != nil {
		return reviewContextEvidence{}, err
	}

	return reviewContextEvidence{
		changedFileContext:     changedFileContext,
		relatedContextFiles:    relatedContextFiles,
		relatedSearchHits:      relatedSearch.hits,
		relatedSearchTruncated: relatedSearch.truncated,
	}, nil
}

func newReviewContextEvidenceCollector(ctx context.Context, repoRoot string, limits ReviewEvidenceLimits, changedFiles []ReviewChangedFile, relatedCandidatePaths []string) *reviewContextEvidenceCollector {
	if ctx == nil {
		ctx = context.Background()
	}
	collector := &reviewContextEvidenceCollector{
		repoRoot:              repoRoot,
		ctx:                   ctx,
		limits:                normalizeReviewEvidenceLimits(limits),
		changedPaths:          make(map[string]struct{}, len(changedFiles)),
		relatedCandidatePaths: normalizeReviewRelatedCandidatePaths(relatedCandidatePaths),
	}
	for _, file := range changedFiles {
		_, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, file.Path)
		if err != nil {
			continue
		}
		collector.changedPaths[relPath] = struct{}{}
	}
	return collector
}

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
		if !isReviewContextRelatedGoPath(relPath) {
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

func (c *reviewContextEvidenceCollector) contextErr() error {
	if c.ctx == nil {
		return nil
	}
	return c.ctx.Err()
}

func (c *reviewContextEvidenceCollector) collectChangedFileContext(changedFiles []ReviewChangedFile) []ReviewContextFileEvidence {
	files := make([]ReviewContextFileEvidence, 0, minReviewEvidenceInt(len(changedFiles), c.limits.MaxContextFiles))
	for _, file := range changedFiles {
		if c.contextErr() != nil {
			break
		}
		evidence, ok := c.collectChangedFile(file)
		if !ok {
			continue
		}
		files = append(files, evidence)
	}
	return files
}

func (c *reviewContextEvidenceCollector) collectChangedFile(file ReviewChangedFile) (ReviewContextFileEvidence, bool) {
	if reviewStatusHasPrefix(file.Status, "D") {
		return ReviewContextFileEvidence{}, false
	}
	return c.collectContextFile(file.Path, reviewContextFileRoleChanged)
}

func (c *reviewContextEvidenceCollector) collectRelatedContextFiles(changedFiles []ReviewChangedFile) []ReviewContextFileEvidence {
	scope := c.changedGoScope(changedFiles)

	files := make([]ReviewContextFileEvidence, 0)
	candidates := make([]reviewContextRelatedCandidate, 0)

	for _, relPath := range c.relatedCandidatePaths {
		if c.maxContextFilesExceededLogged || c.contextErr() != nil {
			break
		}
		if !isReviewContextRelatedGoPath(relPath) {
			continue
		}
		if !scope.hasDir(pathpkg.Dir(relPath)) {
			continue
		}
		if _, changed := c.changedPaths[relPath]; changed {
			continue
		}

		role := reviewContextFileRoleRelatedGo
		isTest := isReviewContextTestGoPath(relPath)
		if isTest {
			role = reviewContextFileRoleRelatedTest
		}
		candidates = append(candidates, reviewContextRelatedCandidate{
			path:     relPath,
			role:     role,
			priority: scope.relatedCandidatePriority(relPath, isTest),
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

func (s reviewContextChangedGoScope) relatedCandidatePriority(relPath string, isTest bool) int {
	dir := pathpkg.Dir(relPath)
	stem := reviewContextGoStem(relPath)
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

func (c *reviewContextEvidenceCollector) changedGoScope(changedFiles []ReviewChangedFile) reviewContextChangedGoScope {
	scope := reviewContextChangedGoScope{
		stemsByDir: make(map[string]map[string]reviewContextChangedGoStem),
	}
	for _, file := range changedFiles {
		scope.addPath(c.repoRoot, file.Path)
		scope.addPath(c.repoRoot, file.OldPath)
	}
	return scope
}

func (s reviewContextChangedGoScope) hasDir(dir string) bool {
	_, ok := s.stemsByDir[dir]
	return ok
}

func (s reviewContextChangedGoScope) hasSourceStem(dir, stem string) bool {
	changed, ok := s.changedStem(dir, stem)
	return ok && changed.source
}

func (s reviewContextChangedGoScope) hasTestStem(dir, stem string) bool {
	changed, ok := s.changedStem(dir, stem)
	return ok && changed.test
}

func (s reviewContextChangedGoScope) changedStem(dir, stem string) (reviewContextChangedGoStem, bool) {
	stems, ok := s.stemsByDir[dir]
	if !ok {
		return reviewContextChangedGoStem{}, false
	}
	changed, ok := stems[stem]
	return changed, ok
}

func (s *reviewContextChangedGoScope) addPath(repoRoot, path string) {
	dir, stem, isTest, ok := relatedContextChangedGoPath(repoRoot, path)
	if !ok {
		return
	}
	if s.stemsByDir == nil {
		s.stemsByDir = make(map[string]map[string]reviewContextChangedGoStem)
	}
	if _, ok := s.stemsByDir[dir]; !ok {
		s.stemsByDir[dir] = make(map[string]reviewContextChangedGoStem)
	}
	changed := s.stemsByDir[dir][stem]
	if isTest {
		changed.test = true
	} else {
		changed.source = true
	}
	s.stemsByDir[dir][stem] = changed
}

func relatedContextChangedGoPath(repoRoot, path string) (string, string, bool, bool) {
	_, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, path)
	if err != nil {
		return "", "", false, false
	}
	if !isReviewContextRelatedGoPath(relPath) {
		return "", "", false, false
	}
	return pathpkg.Dir(relPath), reviewContextGoStem(relPath), isReviewContextTestGoPath(relPath), true
}

func (c *reviewContextEvidenceCollector) collectContextFile(path, role string) (ReviewContextFileEvidence, bool) {
	absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(c.repoRoot, path)
	evidence := ReviewContextFileEvidence{
		Path: normalizeReviewEvidenceDisplayPath(path),
		Role: role,
	}
	if err != nil {
		return c.skipContextFile(evidence, reviewContextSkipInvalidPath)
	}
	evidence.Path = relPath

	if isReviewContextGeneratedPath(relPath) {
		return c.skipContextFile(evidence, reviewContextSkipGenerated)
	}
	if isReviewContextVendorPath(relPath) {
		return c.skipContextFile(evidence, reviewContextSkipVendor)
	}

	return c.readContextFile(absPath, evidence)
}

func (c *reviewContextEvidenceCollector) skipContextFile(evidence ReviewContextFileEvidence, reason string) (ReviewContextFileEvidence, bool) {
	reserved, ok := c.reserveContextFileSlot(evidence)
	if !ok {
		return reserved, false
	}
	return markReviewContextFileSkipped(reserved, reason), true
}

func (c *reviewContextEvidenceCollector) readContextFile(absPath string, evidence ReviewContextFileEvidence) (ReviewContextFileEvidence, bool) {
	reserved, ok := c.reserveContextFileSlot(evidence)
	if !ok {
		return reserved, false
	}
	evidence = reserved

	remainingTotal := c.limits.MaxTotalContextBytes - c.totalContextRead
	maxBytes := minReviewEvidenceInt64(c.limits.MaxContextFileBytes, remainingTotal)
	file := readReviewEvidenceRegularFile(reviewEvidenceRegularFileReadInput{
		repoRoot:       c.repoRoot,
		absPath:        absPath,
		relPath:        evidence.Path,
		maxBytes:       maxBytes,
		maxFileBytes:   c.limits.MaxContextFileBytes,
		enforceFileMax: true,
		maxBudgetBytes: remainingTotal,
		enforceBudget:  true,
	})
	evidence.SizeBytes = file.sizeBytes
	evidence.ReadBytes = file.readBytes
	evidence.Truncated = file.truncated

	switch file.status {
	case reviewEvidenceRegularFileReadOK:
		c.totalContextRead += evidence.ReadBytes
		if file.binary {
			return markReviewContextFileSkipped(evidence, reviewContextSkipBinary), true
		}
		evidence.Content = string(file.data)
		return evidence, true
	case reviewEvidenceRegularFileReadMissing, reviewEvidenceRegularFileReadStatFailed:
		return markReviewContextFileSkipped(evidence, reviewContextSkipStatFailed), true
	case reviewEvidenceRegularFileReadSymlink:
		return markReviewContextFileSkipped(evidence, reviewContextSkipSymlink), true
	case reviewEvidenceRegularFileReadDir, reviewEvidenceRegularFileReadNonRegular:
		return markReviewContextFileSkipped(evidence, reviewContextSkipNonRegular), true
	case reviewEvidenceRegularFileReadInvalidPath:
		return markReviewContextFileSkipped(evidence, reviewContextSkipInvalidPath), true
	case reviewEvidenceRegularFileReadFileTooLarge:
		return markReviewContextFileSkipped(evidence, reviewContextSkipTooLarge), true
	case reviewEvidenceRegularFileReadBudgetExceeded:
		return markReviewContextFileSkipped(evidence, reviewContextSkipTotalBudgetExceeded), true
	case reviewEvidenceRegularFileReadFailed:
		return markReviewContextFileSkipped(evidence, reviewContextSkipReadFailed), true
	}
	return markReviewContextFileSkipped(evidence, reviewContextSkipReadFailed), true
}

func (c *reviewContextEvidenceCollector) reserveContextFileSlot(evidence ReviewContextFileEvidence) (ReviewContextFileEvidence, bool) {
	if c.contextFileCount < c.limits.MaxContextFiles {
		c.contextFileCount++
		return evidence, true
	}
	if c.maxContextFilesExceededLogged {
		return evidence, false
	}
	c.maxContextFilesExceededLogged = true
	return markReviewContextFileSkipped(evidence, reviewContextSkipMaxFilesExceeded), true
}

func markReviewContextFileSkipped(evidence ReviewContextFileEvidence, reason string) ReviewContextFileEvidence {
	evidence.Content = ""
	evidence.Skipped = true
	evidence.SkipReason = reason
	return evidence
}

func isReviewContextGeneratedPath(path string) bool {
	return matchGeneratedReviewInventoryPath(newReviewInventoryPath(path))
}

func isReviewContextVendorPath(path string) bool {
	return newReviewInventoryPath(path).hasDir("vendor")
}

func isReviewContextRelatedGoPath(relPath string) bool {
	return pathpkg.Ext(relPath) == ".go" &&
		!isReviewContextGeneratedPath(relPath) &&
		!isReviewContextVendorPath(relPath)
}

func isReviewContextTestGoPath(relPath string) bool {
	return strings.HasSuffix(strings.ToLower(pathpkg.Base(relPath)), "_test.go")
}

func reviewContextGoStem(relPath string) string {
	base := pathpkg.Base(relPath)
	if isReviewContextTestGoPath(relPath) {
		return base[:len(base)-len("_test.go")]
	}
	return strings.TrimSuffix(base, pathpkg.Ext(base))
}
