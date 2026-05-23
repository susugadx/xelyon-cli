package search

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type goMethodCrossPackageTestCollector struct {
	ctx     goMethodTestProbeContext
	matcher *goMethodCrossPackageTestMatcher
}

type goMethodCrossPackageTestMatcher struct {
	ctx        goMethodTestProbeContext
	testFiles  map[string]crossPackageMethodTestFile
	graphCache *crossPackageHelperGraphCache
}

type crossPackageMethodTestFile struct {
	src    []byte
	parsed *ast.ParsedFile
	valid  bool
}

func (ctx goMethodTestProbeContext) collectCrossPackageTests(appendTest func(navigation.TestRef)) {
	newGoMethodCrossPackageTestCollector(ctx).collect(appendTest)
}

func newGoMethodCrossPackageTestCollector(ctx goMethodTestProbeContext) *goMethodCrossPackageTestCollector {
	return &goMethodCrossPackageTestCollector{
		ctx:     ctx,
		matcher: newGoMethodCrossPackageTestMatcher(ctx),
	}
}

func (c *goMethodCrossPackageTestCollector) collect(appendTest func(navigation.TestRef)) {
	if !c.ctx.symbol.Exported {
		return
	}

	for _, candidate := range c.broadNameProbeTests() {
		absPath, ok := c.crossPackageTestPath(candidate)
		if !ok {
			continue
		}
		if !c.matcher.matches(absPath, candidate) {
			continue
		}
		appendTest(candidate)
	}
}

// broadNameProbeTests は現行 contract として候補数をここで打ち切らない。
// semantic helper 判定後の TotalTests/MoreTests を保つため、budget は appendTest 側で適用する。
func (c *goMethodCrossPackageTestCollector) broadNameProbeTests() []navigation.TestRef {
	candidates, _ := findGoImpactTestsByNameProbe(c.ctx.probe, c.ctx.symbol.RootPath, c.ctx.opts, 0)
	return candidates
}

func (c *goMethodCrossPackageTestCollector) crossPackageTestPath(test navigation.TestRef) (string, bool) {
	absPath := absoluteAffectedFilePathWithPreferredBases(
		test.File,
		c.ctx.symbol.RootPath,
		affectedFileBasePath(c.ctx.opts, affectedFileSourceText),
		structuredGoImpactProbeRootPath(c.ctx.opts, c.ctx.packageDir),
	)
	c.ctx.dependencies.add(absPath)
	if absPath == "" || filepath.Clean(filepath.Dir(absPath)) == filepath.Clean(c.ctx.packageDir) {
		return "", false
	}
	return absPath, true
}

func newGoMethodCrossPackageTestMatcher(ctx goMethodTestProbeContext) *goMethodCrossPackageTestMatcher {
	return &goMethodCrossPackageTestMatcher{
		ctx:        ctx,
		testFiles:  make(map[string]crossPackageMethodTestFile),
		graphCache: newCrossPackageHelperGraphCache(),
	}
}

func (m *goMethodCrossPackageTestMatcher) matches(absPath string, test navigation.TestRef) bool {
	src, parsed, ok := m.testFile(absPath)
	if !ok {
		return false
	}
	testSymbol, ok := findCrossPackageMethodProbeTestSymbol(absPath, src, test)
	if !ok {
		return false
	}
	if methodTestBodyMatchesSymbol(m.ctx.matchContext(absPath, src), parsed, testSymbol, false) {
		return true
	}
	graph := newCrossPackageHelperGraphWithCache(m.ctx, filepath.Dir(absPath), true, m.graphCache)
	helper := packageHelper{
		key:    helperCacheKeyFromFields(absPath, testSymbol.Name, testSymbol.Line, testSymbol.EndLine),
		name:   testSymbol.Name,
		abs:    absPath,
		src:    src,
		parsed: parsed,
		sym:    testSymbol,
	}
	return graph.matchesSymbol(helper, make(map[string]struct{}))
}

func (m *goMethodCrossPackageTestMatcher) testFile(absPath string) ([]byte, *ast.ParsedFile, bool) {
	absPath = filepath.Clean(strings.TrimSpace(absPath))
	if absPath == "" {
		return nil, nil, false
	}
	if cached, ok := m.testFiles[absPath]; ok {
		return cached.src, cached.parsed, cached.valid
	}

	m.ctx.dependencies.add(absPath)
	src, err := os.ReadFile(absPath)
	if err != nil {
		m.testFiles[absPath] = crossPackageMethodTestFile{}
		return nil, nil, false
	}
	parsed, err := ast.ParseBytesForReuse(absPath, src)
	if err != nil {
		m.testFiles[absPath] = crossPackageMethodTestFile{}
		return nil, nil, false
	}

	cached := crossPackageMethodTestFile{src: src, parsed: parsed, valid: true}
	m.testFiles[absPath] = cached
	return cached.src, cached.parsed, cached.valid
}
