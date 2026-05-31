package evidence

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

const (
	ReviewGenericImpactRoleSameStemTestOrSpec       = "same_stem_test_or_spec"
	ReviewGenericImpactRoleNearbyTestOrTestsDir     = "nearby_test_or_tests_dir"
	ReviewGenericImpactRoleNearbyProjectConfig      = "nearby_project_config"
	ReviewGenericImpactRoleDocsReference            = "docs_reference"
	ReviewGenericImpactRoleTextualReference         = "textual_reference"
	ReviewGenericImpactRoleChangedPathStemReference = "changed_path_stem_reference"

	reviewGenericImpactMaxTokens            = 12
	reviewGenericImpactMaxCandidatesTotal   = 40
	reviewGenericImpactMaxCandidatesPerRole = 8
	reviewGenericImpactMaxHitsPerToken      = 5
)

var (
	reviewGenericImpactCommandTokenRE    = regexp.MustCompile(`/[A-Za-z][A-Za-z0-9_-]*`)
	reviewGenericImpactFlagTokenRE       = regexp.MustCompile(`--[A-Za-z][A-Za-z0-9-]*`)
	reviewGenericImpactKeyTokenRE        = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_-]*)\s*[:=]`)
	reviewGenericImpactIdentifierTokenRE = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

type reviewGenericImpactToken struct {
	value    string
	fromDiff bool
	fromStem bool
}

type reviewGenericImpactTokenExtraction struct {
	commands    []string
	flags       []string
	keys        []string
	identifiers []string
}

type reviewGenericImpactTokenExtractor struct {
	extraction reviewGenericImpactTokenExtraction
	seen       map[string]struct{}
}

type reviewGenericImpactReferenceSearch struct {
	role   string
	reason string
	filter func(string) bool
}

var (
	reviewGenericImpactDiffTokenReferenceSearches = []reviewGenericImpactReferenceSearch{
		{
			role:   ReviewGenericImpactRoleDocsReference,
			reason: "docs/readme reference to changed token",
			filter: reviewGenericImpactDocsSearchFilter,
		},
		{
			role:   ReviewGenericImpactRoleTextualReference,
			reason: "bounded token reference",
			filter: reviewGenericImpactTextualSearchFilter,
		},
	}
	reviewGenericImpactStemTokenReferenceSearch = reviewGenericImpactReferenceSearch{
		role:   ReviewGenericImpactRoleChangedPathStemReference,
		reason: "changed path stem reference",
		filter: reviewGenericImpactAllSearchFilter,
	}
	reviewGenericImpactExcludedPathParts = map[string]struct{}{
		".git":         {},
		"node_modules": {},
		"vendor":       {},
		"dist":         {},
		"build":        {},
		"coverage":     {},
	}
	reviewGenericImpactNearbyTestDirNames = []string{"test", "tests", "__tests__"}
	reviewGenericImpactDefaultIgnore      = pathmatch.NewMatcher(pathmatch.DefaultIgnorePatterns())
)

type reviewGenericImpactCandidateBuilder struct {
	bundle ReviewEvidenceBundle
	limits ReviewEvidenceLimits

	repoRoot string

	repoPaths    []string
	changedPaths map[string]struct{}
	changedDirs  []string
	changedStems []string

	tokens []reviewGenericImpactToken

	candidates []ReviewGenericImpactCandidate
	seen       map[string]struct{}
	roleCounts map[string]int
	tokenHits  map[string]int
	truncated  bool

	searchFileCount int
	totalSearchRead int64
}

// BuildReviewGenericImpactCandidates は言語非依存 heuristic で review 用 impact 候補を作る。
// これは import/caller graph ではなく、Pass1 の impact surface 検討に使う bounded lead である。
func BuildReviewGenericImpactCandidates(bundle ReviewEvidenceBundle) ReviewGenericImpactCandidates {
	builder := newReviewGenericImpactCandidateBuilder(bundle)
	builder.collectRepoPaths()
	builder.collectPathCandidates()
	builder.collectSearchCandidates()
	return builder.build()
}

func newReviewGenericImpactCandidateBuilder(bundle ReviewEvidenceBundle) *reviewGenericImpactCandidateBuilder {
	limits := normalizeReviewEvidenceLimits(bundle.Limits)
	repoRoot := strings.TrimSpace(bundle.RepoRoot)
	if repoRoot != "" {
		repoRoot = filepath.Clean(repoRoot)
	}
	changedPaths := reviewGenericImpactChangedPathSet(bundle)
	changedDirs := reviewGenericImpactChangedDirs(changedPaths)
	changedStems := reviewGenericImpactChangedStems(changedPaths)
	builder := &reviewGenericImpactCandidateBuilder{
		bundle:       bundle,
		limits:       limits,
		repoRoot:     repoRoot,
		changedPaths: changedPaths,
		changedDirs:  changedDirs,
		changedStems: changedStems,
		seen:         make(map[string]struct{}),
		roleCounts:   make(map[string]int),
		tokenHits:    make(map[string]int),
		truncated:    bundle.GenericImpactCandidateListTruncated,
	}
	builder.collectTokens()
	return builder
}

func (b *reviewGenericImpactCandidateBuilder) collectTokens() {
	diffTokens := extractReviewGenericImpactDiffTokens(b.bundle.Diffs)
	for _, token := range diffTokens.commands {
		b.addToken(token, true, false)
	}
	for _, token := range diffTokens.flags {
		b.addToken(token, true, false)
	}
	for _, token := range diffTokens.keys {
		b.addToken(token, true, false)
	}
	untrackedTokens := extractReviewGenericImpactUntrackedTokens(b.bundle.UntrackedFiles)
	if b.bundle.UntrackedSnapshotsTruncated || reviewGenericImpactUntrackedSnapshotsHaveTruncation(b.bundle.UntrackedFiles) {
		b.truncated = true
	}
	for _, token := range untrackedTokens.commands {
		b.addToken(token, true, false)
	}
	for _, token := range untrackedTokens.flags {
		b.addToken(token, true, false)
	}
	for _, token := range untrackedTokens.keys {
		b.addToken(token, true, false)
	}
	for _, stem := range b.changedStems {
		b.addToken(stem, false, true)
	}
	for _, token := range diffTokens.identifiers {
		b.addToken(token, true, false)
	}
	for _, token := range untrackedTokens.identifiers {
		b.addToken(token, true, false)
	}
}

func (b *reviewGenericImpactCandidateBuilder) addToken(token string, fromDiff, fromStem bool) {
	token = strings.TrimSpace(token)
	if !isReviewGenericImpactUsefulToken(token) {
		return
	}
	for i := range b.tokens {
		if b.tokens[i].value == token {
			b.tokens[i].fromDiff = b.tokens[i].fromDiff || fromDiff
			b.tokens[i].fromStem = b.tokens[i].fromStem || fromStem
			return
		}
	}
	if len(b.tokens) >= reviewGenericImpactMaxTokens {
		b.truncated = true
		return
	}
	b.tokens = append(b.tokens, reviewGenericImpactToken{
		value:    token,
		fromDiff: fromDiff,
		fromStem: fromStem,
	})
}

func (b *reviewGenericImpactCandidateBuilder) collectRepoPaths() {
	if strings.TrimSpace(b.repoRoot) == "" {
		return
	}
	if b.bundle.GenericImpactCandidatePathsCollected {
		b.collectRepoPathsFromCandidateList(b.bundle.GenericImpactCandidatePaths)
		return
	}
	err := filepath.WalkDir(b.repoRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			b.truncated = true
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, err := filepath.Rel(b.repoRoot, path)
		if err != nil {
			b.truncated = true
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}
		if entry.IsDir() {
			if isReviewGenericImpactExcludedPath(relPath) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeType != 0 {
			return nil
		}
		if isReviewGenericImpactExcludedPath(relPath) {
			return nil
		}
		b.repoPaths = append(b.repoPaths, relPath)
		return nil
	})
	if err != nil {
		b.truncated = true
	}
	sort.Strings(b.repoPaths)
}

func (b *reviewGenericImpactCandidateBuilder) collectRepoPathsFromCandidateList(paths []string) {
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(b.repoRoot, path)
		if err != nil || isReviewGenericImpactExcludedPath(relPath) {
			continue
		}
		if _, ok := seen[relPath]; ok {
			continue
		}
		info, err := os.Lstat(absPath)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		seen[relPath] = struct{}{}
		b.repoPaths = append(b.repoPaths, relPath)
	}
	sort.Strings(b.repoPaths)
}

func (b *reviewGenericImpactCandidateBuilder) collectPathCandidates() {
	b.collectSameStemTestOrSpecCandidates()
	b.collectNearbyTestCandidates()
	b.collectNearbyProjectConfigCandidates()
}

func (b *reviewGenericImpactCandidateBuilder) collectSameStemTestOrSpecCandidates() {
	stems := reviewGenericImpactStringSet(b.changedStems)
	for _, path := range b.repoPaths {
		if b.isChangedPath(path) || !isReviewGenericImpactTestOrSpecPath(path) {
			continue
		}
		stem := reviewGenericImpactPathStem(path)
		if _, ok := stems[stem]; !ok {
			continue
		}
		b.addCandidate(ReviewGenericImpactCandidate{
			Path:   path,
			Role:   ReviewGenericImpactRoleSameStemTestOrSpec,
			Reason: "test/spec file shares changed path stem",
			Token:  stem,
		})
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectNearbyTestCandidates() {
	for _, path := range b.repoPaths {
		if b.isChangedPath(path) || !isReviewGenericImpactNearbyTestPath(path) {
			continue
		}
		if !b.isNearChangedDir(path) {
			continue
		}
		b.addCandidate(ReviewGenericImpactCandidate{
			Path:   path,
			Role:   ReviewGenericImpactRoleNearbyTestOrTestsDir,
			Reason: "test/spec file is in the same directory or a nearby test directory",
			Token:  reviewGenericImpactPathStem(path),
		})
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectNearbyProjectConfigCandidates() {
	ancestorDirs := reviewGenericImpactAncestorDirSet(b.changedDirs)
	for _, path := range b.repoPaths {
		if b.isChangedPath(path) || !isReviewGenericImpactProjectConfigPath(path) {
			continue
		}
		if _, ok := ancestorDirs[reviewGenericImpactPathDir(path)]; !ok {
			continue
		}
		b.addCandidate(ReviewGenericImpactCandidate{
			Path:   path,
			Role:   ReviewGenericImpactRoleNearbyProjectConfig,
			Reason: "project config is in the changed path directory or one of its ancestors",
			Token:  reviewGenericImpactConfigToken(path),
		})
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectSearchCandidates() {
	for _, token := range b.tokens {
		if token.fromDiff {
			for _, search := range reviewGenericImpactDiffTokenReferenceSearches {
				b.collectTokenReferences(search, token.value)
			}
		}
		if token.fromStem {
			b.collectTokenReferences(reviewGenericImpactStemTokenReferenceSearch, token.value)
		}
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectTokenReferences(search reviewGenericImpactReferenceSearch, token string) {
	if !isReviewGenericImpactUsefulToken(token) {
		return
	}
	for _, path := range b.repoPaths {
		if b.searchHitLimitReached(search.role, token) {
			b.truncated = true
			return
		}
		if b.isChangedPath(path) || !search.filter(path) {
			continue
		}
		content, ok := b.readSearchFile(path)
		if !ok {
			continue
		}
		b.collectTokenReferencesInContent(search, path, token, content)
	}
}

func (b *reviewGenericImpactCandidateBuilder) collectTokenReferencesInContent(search reviewGenericImpactReferenceSearch, path, token, content string) {
	lines := strings.SplitAfter(content, "\n")
	for i, line := range lines {
		if b.searchHitLimitReached(search.role, token) {
			b.truncated = true
			return
		}
		lineText := strings.TrimRight(line, "\r\n")
		if lineText == "" || !reviewGenericImpactLineContainsToken(lineText, token) {
			continue
		}
		snippet := strings.TrimSpace(lineText)
		if snippet == "" {
			snippet = lineText
		}
		var snippetTruncated bool
		snippet, snippetTruncated = truncateReviewEvidenceStringPrefix(snippet, b.limits.MaxSearchSnippetBytes)
		if snippetTruncated {
			b.truncated = true
		}
		b.addCandidate(ReviewGenericImpactCandidate{
			Path:    path,
			Role:    search.role,
			Reason:  search.reason,
			Token:   token,
			Line:    i + 1,
			Snippet: snippet,
		})
	}
}

func (b *reviewGenericImpactCandidateBuilder) readSearchFile(path string) (string, bool) {
	if b.searchFileCount >= b.limits.MaxRelatedSearchFiles || b.totalSearchRead >= b.limits.MaxTotalRelatedSearchBytes {
		b.truncated = true
		return "", false
	}
	absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(b.repoRoot, path)
	if err != nil {
		return "", false
	}
	remainingTotal := b.limits.MaxTotalRelatedSearchBytes - b.totalSearchRead
	if remainingTotal <= 0 {
		b.truncated = true
		return "", false
	}
	maxBytes := minReviewEvidenceInt64(b.limits.MaxRelatedSearchFileBytes, remainingTotal)
	file := readReviewEvidenceRegularFile(reviewEvidenceRegularFileReadInput{
		repoRoot: b.repoRoot,
		absPath:  absPath,
		relPath:  relPath,
		maxBytes: maxBytes,
	})
	if file.regular {
		b.searchFileCount++
	}
	if file.status != reviewEvidenceRegularFileReadOK {
		return "", false
	}
	b.totalSearchRead += file.readBytes
	if file.truncated {
		b.truncated = true
	}
	if file.binary {
		return "", false
	}
	return string(file.data), true
}

func (b *reviewGenericImpactCandidateBuilder) addCandidate(candidate ReviewGenericImpactCandidate) bool {
	if strings.TrimSpace(candidate.Path) == "" || strings.TrimSpace(candidate.Role) == "" {
		return false
	}
	key := candidate.Role + "\x00" + candidate.Path + "\x00" + candidate.Token + "\x00" + strconvReviewGenericImpactLine(candidate.Line)
	if _, ok := b.seen[key]; ok {
		return true
	}
	if len(b.candidates) >= reviewGenericImpactMaxCandidatesTotal {
		b.truncated = true
		return false
	}
	if b.roleCounts[candidate.Role] >= reviewGenericImpactMaxCandidatesPerRole {
		b.truncated = true
		return false
	}
	if candidate.Line > 0 && candidate.Token != "" {
		hitKey := candidate.Role + "\x00" + candidate.Token
		if b.tokenHits[hitKey] >= reviewGenericImpactMaxHitsPerToken {
			b.truncated = true
			return false
		}
		b.tokenHits[hitKey]++
	}
	b.seen[key] = struct{}{}
	b.roleCounts[candidate.Role]++
	b.candidates = append(b.candidates, candidate)
	return true
}

func (b *reviewGenericImpactCandidateBuilder) searchHitLimitReached(role, token string) bool {
	return b.tokenHits[role+"\x00"+token] >= reviewGenericImpactMaxHitsPerToken
}

func (b *reviewGenericImpactCandidateBuilder) isChangedPath(path string) bool {
	_, ok := b.changedPaths[path]
	return ok
}

func (b *reviewGenericImpactCandidateBuilder) isNearChangedDir(path string) bool {
	dir := reviewGenericImpactPathDir(path)
	for _, changedDir := range b.changedDirs {
		if dir == changedDir {
			return true
		}
		for _, testDir := range reviewGenericImpactNearbyTestDirNames {
			if dir == reviewGenericImpactJoinPath(changedDir, testDir) ||
				dir == reviewGenericImpactJoinPath(reviewGenericImpactPathDir(changedDir), testDir) {
				return true
			}
		}
	}
	return false
}

func (b *reviewGenericImpactCandidateBuilder) build() ReviewGenericImpactCandidates {
	tokens := make([]string, 0, len(b.tokens))
	for _, token := range b.tokens {
		tokens = append(tokens, token.value)
	}
	if tokens == nil {
		tokens = []string{}
	}
	if b.candidates == nil {
		b.candidates = []ReviewGenericImpactCandidate{}
	}
	return ReviewGenericImpactCandidates{
		Tokens:     tokens,
		Candidates: b.candidates,
		Truncated:  b.truncated,
	}
}

func extractReviewGenericImpactDiffTokens(diffs []ReviewDiffEvidence) reviewGenericImpactTokenExtraction {
	extractor := newReviewGenericImpactTokenExtractor()
	for _, diff := range diffs {
		for _, line := range strings.Split(diff.Diff, "\n") {
			line, ok := reviewGenericImpactChangedDiffContentLine(line)
			if !ok {
				continue
			}
			extractor.addLine(line)
		}
	}
	return extractor.extraction
}

func extractReviewGenericImpactUntrackedTokens(files []ReviewUntrackedFile) reviewGenericImpactTokenExtraction {
	extractor := newReviewGenericImpactTokenExtractor()
	for _, file := range files {
		if file.Symlink || file.Binary || strings.TrimSpace(file.Snapshot) == "" ||
			isReviewGenericImpactExcludedPath(file.Path) || !isReviewGenericImpactSearchableTextPath(file.Path) {
			continue
		}
		for _, line := range strings.Split(file.Snapshot, "\n") {
			extractor.addLine(line)
		}
	}
	return extractor.extraction
}

func newReviewGenericImpactTokenExtractor() reviewGenericImpactTokenExtractor {
	return reviewGenericImpactTokenExtractor{
		seen: make(map[string]struct{}),
	}
}

func (e *reviewGenericImpactTokenExtractor) addToken(values *[]string, token string) {
	token = strings.TrimSpace(token)
	if !isReviewGenericImpactUsefulToken(token) {
		return
	}
	if _, ok := e.seen[token]; ok {
		return
	}
	e.seen[token] = struct{}{}
	*values = append(*values, token)
}

func (e *reviewGenericImpactTokenExtractor) addLine(line string) {
	for _, token := range reviewGenericImpactCommandTokenRE.FindAllString(line, -1) {
		e.addToken(&e.extraction.commands, token)
	}
	for _, token := range reviewGenericImpactFlagTokenRE.FindAllString(line, -1) {
		e.addToken(&e.extraction.flags, token)
	}
	for _, match := range reviewGenericImpactKeyTokenRE.FindAllStringSubmatch(line, -1) {
		if len(match) > 1 {
			e.addToken(&e.extraction.keys, match[1])
		}
	}
	for _, token := range reviewGenericImpactIdentifierTokenRE.FindAllString(line, -1) {
		if isReviewGenericImpactIdentifierLikeToken(token) {
			e.addToken(&e.extraction.identifiers, token)
		}
	}
}

func reviewGenericImpactUntrackedSnapshotsHaveTruncation(files []ReviewUntrackedFile) bool {
	for _, file := range files {
		if file.Truncated {
			return true
		}
	}
	return false
}

func reviewGenericImpactChangedDiffContentLine(line string) (string, bool) {
	if len(line) == 0 {
		return "", false
	}
	switch line[0] {
	case '+':
		if isReviewGenericImpactDiffFileHeader(line, '+') {
			return "", false
		}
	case '-':
		if isReviewGenericImpactDiffFileHeader(line, '-') {
			return "", false
		}
	default:
		return "", false
	}
	return line[1:], true
}

func isReviewGenericImpactDiffFileHeader(line string, marker byte) bool {
	if len(line) < 4 || line[0] != marker || line[1] != marker || line[2] != marker {
		return false
	}
	switch line[3] {
	case ' ', '\t':
		return true
	default:
		return false
	}
}

func reviewGenericImpactChangedPathSet(bundle ReviewEvidenceBundle) map[string]struct{} {
	paths := make(map[string]struct{})
	add := func(path string) {
		relPath, ok := reviewGenericImpactBundleRelativePath(bundle.RepoRoot, path)
		if ok {
			paths[relPath] = struct{}{}
		}
	}
	for _, file := range bundle.ChangedFiles {
		add(file.Path)
		add(file.OldPath)
	}
	for _, file := range bundle.UntrackedFiles {
		add(file.Path)
	}
	for _, path := range bundle.Inventory.Untracked {
		add(path)
	}
	return paths
}

func reviewGenericImpactChangedDirs(paths map[string]struct{}) []string {
	dirs := make(map[string]struct{})
	for path := range paths {
		dirs[reviewGenericImpactPathDir(path)] = struct{}{}
	}
	return reviewGenericImpactSortedSet(dirs)
}

func reviewGenericImpactChangedStems(paths map[string]struct{}) []string {
	stems := make(map[string]struct{})
	for path := range paths {
		stem := reviewGenericImpactPathStem(path)
		if isReviewGenericImpactUsefulToken(stem) {
			stems[stem] = struct{}{}
		}
	}
	return reviewGenericImpactSortedSet(stems)
}

func reviewGenericImpactBundleRelativePath(repoRoot, path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	if _, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, path); err == nil {
		return relPath, true
	}
	display := formatReviewEvidencePathDisplay(repoRoot, path)
	if display == "" || display == reviewEvidenceOutsideRepoPathDisplay || display == "." {
		return "", false
	}
	return display, true
}

func reviewGenericImpactAncestorDirSet(dirs []string) map[string]struct{} {
	ancestors := make(map[string]struct{})
	for _, dir := range dirs {
		for {
			if dir == "" {
				dir = "."
			}
			ancestors[dir] = struct{}{}
			if dir == "." {
				break
			}
			dir = reviewGenericImpactPathDir(dir)
		}
	}
	return ancestors
}

func isReviewGenericImpactExcludedPath(path string) bool {
	normalized := filepath.ToSlash(path)
	if normalized == ".xelyon/review-runs" || strings.HasPrefix(normalized, ".xelyon/review-runs/") {
		return true
	}
	if reviewGenericImpactDefaultIgnore.Match(normalized, false) {
		return true
	}
	if isReviewGenericImpactSensitivePath(normalized) {
		return true
	}
	for _, part := range strings.Split(normalized, "/") {
		if _, ok := reviewGenericImpactExcludedPathParts[part]; ok {
			return true
		}
	}
	return false
}

func isReviewGenericImpactTestOrSpecPath(path string) bool {
	base := strings.ToLower(reviewGenericImpactPathBase(path))
	return strings.HasSuffix(base, "_test.go") ||
		strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.")
}

func isReviewGenericImpactNearbyTestPath(path string) bool {
	if isReviewGenericImpactTestOrSpecPath(path) {
		return true
	}
	for _, part := range strings.Split(strings.ToLower(filepath.ToSlash(path)), "/") {
		switch part {
		case "test", "tests", "__tests__":
			return true
		}
	}
	return false
}

func isReviewGenericImpactProjectConfigPath(path string) bool {
	base := strings.ToLower(reviewGenericImpactPathBase(path))
	switch base {
	case "package.json", "pyproject.toml", "cargo.toml", "makefile", "go.mod", "readme.md":
		return true
	}
	return strings.HasPrefix(base, "tsconfig") && strings.HasSuffix(base, ".json") ||
		strings.HasPrefix(base, "vite.config.") ||
		strings.HasPrefix(base, "next.config.")
}

func reviewGenericImpactDocsSearchFilter(path string) bool {
	return isReviewGenericImpactSearchableTextPath(path) && matchDocsReviewInventoryPath(newReviewInventoryPath(path))
}

func reviewGenericImpactTextualSearchFilter(path string) bool {
	return isReviewGenericImpactSearchableTextPath(path) && !reviewGenericImpactDocsSearchFilter(path)
}

func reviewGenericImpactAllSearchFilter(path string) bool {
	return isReviewGenericImpactSearchableTextPath(path)
}

func isReviewGenericImpactSearchableTextPath(path string) bool {
	switch strings.ToLower(reviewGenericImpactPathBase(path)) {
	case ".gitignore", ".ignore":
		return false
	default:
		return !isReviewGenericImpactSensitivePath(path)
	}
}

func isReviewGenericImpactSensitivePath(path string) bool {
	normalized := strings.ToLower(filepath.ToSlash(path))
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		switch part {
		case ".aws", ".azure", ".gnupg", ".kube", ".ssh", "credential", "credentials", "secret", "secrets":
			return true
		}
	}
	base := reviewGenericImpactPathBase(normalized)
	switch base {
	case ".env", ".envrc", ".netrc", ".npmrc", ".pypirc", "credentials", "credential", "secret", "secrets",
		"id_dsa", "id_ecdsa", "id_ed25519", "id_rsa":
		return true
	}
	if strings.HasPrefix(base, ".env.") ||
		strings.HasSuffix(base, ".env") ||
		strings.HasPrefix(base, "credential.") ||
		strings.HasPrefix(base, "credentials.") ||
		strings.HasPrefix(base, "secret.") ||
		strings.HasPrefix(base, "secrets.") ||
		strings.HasSuffix(base, ".key") ||
		strings.HasSuffix(base, ".pem") ||
		strings.HasSuffix(base, ".p12") ||
		strings.HasSuffix(base, ".pfx") {
		return true
	}
	return false
}

func reviewGenericImpactLineContainsToken(line, token string) bool {
	if strings.HasPrefix(token, "--") || strings.HasPrefix(token, "/") || strings.Contains(token, "-") {
		return strings.Contains(line, token)
	}
	start := 0
	for {
		index := strings.Index(line[start:], token)
		if index < 0 {
			return false
		}
		index += start
		beforeOK := index == 0 || !isReviewGenericImpactIdentifierByte(line[index-1])
		afterIndex := index + len(token)
		afterOK := afterIndex >= len(line) || !isReviewGenericImpactIdentifierByte(line[afterIndex])
		if beforeOK && afterOK {
			return true
		}
		start = afterIndex
	}
}

func isReviewGenericImpactUsefulToken(token string) bool {
	token = strings.TrimSpace(token)
	if len(token) < 3 || len(token) > 80 {
		return false
	}
	if _, ok := reviewGenericImpactStopWords[strings.ToLower(token)]; ok {
		return false
	}
	return true
}

func isReviewGenericImpactIdentifierLikeToken(token string) bool {
	if !isReviewGenericImpactUsefulToken(token) {
		return false
	}
	if strings.Contains(token, "_") {
		return true
	}
	hasLower := false
	hasUpper := false
	for i := 0; i < len(token); i++ {
		if token[i] >= 'a' && token[i] <= 'z' {
			hasLower = true
		}
		if token[i] >= 'A' && token[i] <= 'Z' {
			hasUpper = true
		}
	}
	return hasLower && hasUpper
}

func isReviewGenericImpactIdentifierByte(ch byte) bool {
	return ch == '_' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

func reviewGenericImpactPathStem(path string) string {
	base := reviewGenericImpactPathBase(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	lower := strings.ToLower(stem)
	for _, suffix := range []string{".test", ".spec", "_test"} {
		if strings.HasSuffix(lower, suffix) {
			return stem[:len(stem)-len(suffix)]
		}
	}
	return stem
}

func reviewGenericImpactConfigToken(path string) string {
	base := reviewGenericImpactPathBase(path)
	if stem := reviewGenericImpactPathStem(path); stem != "" {
		return stem
	}
	return base
}

func reviewGenericImpactPathBase(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	return parts[len(parts)-1]
}

func reviewGenericImpactPathDir(path string) string {
	normalized := filepath.ToSlash(path)
	index := strings.LastIndex(normalized, "/")
	if index < 0 {
		return "."
	}
	dir := normalized[:index]
	if dir == "" {
		return "."
	}
	return dir
}

func reviewGenericImpactJoinPath(dir, name string) string {
	if dir == "" || dir == "." {
		return name
	}
	return dir + "/" + name
}

func reviewGenericImpactStringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func reviewGenericImpactSortedSet(set map[string]struct{}) []string {
	if len(set) == 0 {
		return []string{}
	}
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func strconvReviewGenericImpactLine(line int) string {
	if line == 0 {
		return "0"
	}
	return strconv.Itoa(line)
}

var reviewGenericImpactStopWords = map[string]struct{}{
	"and": {}, "are": {}, "case": {}, "const": {}, "else": {}, "false": {}, "for": {}, "func": {},
	"function": {}, "import": {}, "let": {}, "nil": {}, "package": {}, "return": {}, "struct": {},
	"the": {}, "true": {}, "type": {}, "var": {}, "with": {},
}
