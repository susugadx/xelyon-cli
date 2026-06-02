package evidence

import (
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
)

type reviewInventorySurface int

const (
	reviewInventorySurfaceGenerated reviewInventorySurface = iota
	reviewInventorySurfaceTests
	reviewInventorySurfaceDocs
	reviewInventorySurfaceConfig
	reviewInventorySurfaceProduction
)

type reviewInventoryPath struct {
	path string
	base string
	ext  string
	dirs map[string]struct{}
}

type reviewInventorySurfaceClassifier struct {
	surface reviewInventorySurface
	match   func(reviewInventoryPath) bool
}

var reviewInventorySurfaceClassifiers = []reviewInventorySurfaceClassifier{
	{surface: reviewInventorySurfaceGenerated, match: matchGeneratedReviewInventoryPath},
	{surface: reviewInventorySurfaceTests, match: matchTestReviewInventoryPath},
	{surface: reviewInventorySurfaceDocs, match: matchDocsReviewInventoryPath},
	{surface: reviewInventorySurfaceConfig, match: matchConfigReviewInventoryPath},
}

type reviewChangeInventoryBuilder struct {
	generated    map[string]struct{}
	tests        map[string]struct{}
	docs         map[string]struct{}
	config       map[string]struct{}
	production   map[string]struct{}
	newFiles     map[string]struct{}
	deletedFiles map[string]struct{}
	renamedFiles map[string]struct{}
	untracked    map[string]struct{}
}

func buildReviewChangeInventory(changedFiles []ReviewChangedFile, untrackedPaths []string) ReviewChangeInventory {
	builder := newReviewChangeInventoryBuilder()

	for _, file := range changedFiles {
		builder.addChangedFile(file)
	}
	for _, path := range untrackedPaths {
		builder.addUntrackedPath(path)
	}

	return builder.build()
}

func newReviewChangeInventoryBuilder() *reviewChangeInventoryBuilder {
	return &reviewChangeInventoryBuilder{
		generated:    make(map[string]struct{}),
		tests:        make(map[string]struct{}),
		docs:         make(map[string]struct{}),
		config:       make(map[string]struct{}),
		production:   make(map[string]struct{}),
		newFiles:     make(map[string]struct{}),
		deletedFiles: make(map[string]struct{}),
		renamedFiles: make(map[string]struct{}),
		untracked:    make(map[string]struct{}),
	}
}

func (b *reviewChangeInventoryBuilder) addChangedFile(file ReviewChangedFile) {
	b.addSurfacePath(file.Path)
	if file.OldPath != "" {
		b.addSurfacePath(file.OldPath)
	}
	if reviewStatusHasPrefix(file.Status, "A") {
		addReviewInventoryPath(b.newFiles, file.Path)
	}
	if reviewStatusHasPrefix(file.Status, "D") {
		addReviewInventoryPath(b.deletedFiles, file.Path)
	}
	if reviewStatusHasPrefix(file.Status, "R") {
		addReviewInventoryPath(b.renamedFiles, file.Path)
	}
}

func (b *reviewChangeInventoryBuilder) addUntrackedPath(path string) {
	normalized := normalizeReviewEvidenceDisplayPath(path)
	addReviewInventoryPath(b.untracked, normalized)
	b.addSurfacePath(normalized)
}

func (b *reviewChangeInventoryBuilder) addSurfacePath(path string) {
	switch classifyReviewInventorySurface(path) {
	case reviewInventorySurfaceGenerated:
		addReviewInventoryPath(b.generated, path)
	case reviewInventorySurfaceTests:
		addReviewInventoryPath(b.tests, path)
	case reviewInventorySurfaceDocs:
		addReviewInventoryPath(b.docs, path)
	case reviewInventorySurfaceConfig:
		addReviewInventoryPath(b.config, path)
	default:
		addReviewInventoryPath(b.production, path)
	}
}

func (b *reviewChangeInventoryBuilder) build() ReviewChangeInventory {
	return ReviewChangeInventory{
		Generated:    sortedReviewInventorySet(b.generated),
		Tests:        sortedReviewInventorySet(b.tests),
		Docs:         sortedReviewInventorySet(b.docs),
		Config:       sortedReviewInventorySet(b.config),
		Production:   sortedReviewInventorySet(b.production),
		NewFiles:     sortedReviewInventorySet(b.newFiles),
		DeletedFiles: sortedReviewInventorySet(b.deletedFiles),
		RenamedFiles: sortedReviewInventorySet(b.renamedFiles),
		Untracked:    sortedReviewInventorySet(b.untracked),
	}
}

func addReviewInventoryPath(set map[string]struct{}, path string) {
	if path == "" || path == "." {
		return
	}
	set[path] = struct{}{}
}

func sortedReviewInventorySet(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func reviewStatusHasPrefix(status, prefix string) bool {
	for _, part := range strings.Split(status, "/") {
		if strings.HasPrefix(part, prefix) {
			return true
		}
	}
	return false
}

func classifyReviewInventorySurface(candidate string) reviewInventorySurface {
	path := newReviewInventoryPath(candidate)
	for _, classifier := range reviewInventorySurfaceClassifiers {
		if classifier.match(path) {
			return classifier.surface
		}
	}
	return reviewInventorySurfaceProduction
}

func newReviewInventoryPath(candidate string) reviewInventoryPath {
	normalized := strings.ToLower(filepath.ToSlash(candidate))
	base := pathpkg.Base(normalized)
	return reviewInventoryPath{
		path: normalized,
		base: base,
		ext:  pathpkg.Ext(base),
		dirs: reviewInventoryPathDirs(normalized),
	}
}

func reviewInventoryPathDirs(path string) map[string]struct{} {
	parts := strings.Split(path, "/")
	dirs := make(map[string]struct{}, len(parts))
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "" || parts[i] == "." {
			continue
		}
		dirs[parts[i]] = struct{}{}
	}
	return dirs
}

func (p reviewInventoryPath) hasDir(dir string) bool {
	_, ok := p.dirs[dir]
	return ok
}

func matchGeneratedReviewInventoryPath(candidate reviewInventoryPath) bool {
	return candidate.hasDir("generated") ||
		strings.Contains(candidate.base, "generated") ||
		strings.HasSuffix(candidate.base, ".pb.go") ||
		strings.HasSuffix(candidate.base, ".gen.go")
}

func matchTestReviewInventoryPath(candidate reviewInventoryPath) bool {
	return strings.HasSuffix(candidate.base, "_test.go") ||
		candidate.hasDir("test") ||
		candidate.hasDir("tests") ||
		candidate.hasDir("__tests__") ||
		candidate.hasDir("testdata")
}

func matchDocsReviewInventoryPath(candidate reviewInventoryPath) bool {
	switch candidate.ext {
	case ".md", ".mdx", ".rst", ".adoc":
		return true
	}
	return strings.HasPrefix(candidate.path, "docs/") ||
		candidate.base == "readme" ||
		candidate.base == "license" ||
		candidate.base == "changelog"
}

func matchConfigReviewInventoryPath(candidate reviewInventoryPath) bool {
	switch candidate.ext {
	case ".yaml", ".yml", ".toml", ".json", ".ini", ".env":
		return true
	}
	return candidate.base == "makefile" ||
		candidate.base == "go.mod" ||
		candidate.base == "go.sum" ||
		strings.HasPrefix(candidate.path, ".github/") ||
		strings.HasPrefix(candidate.path, ".codex/")
}
