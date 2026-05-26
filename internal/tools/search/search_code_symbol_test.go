package search

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type mockGoSymbolLSPClient struct {
	mu                      sync.Mutex
	refs                    []navigation.LSPLocation
	impls                   []navigation.LSPLocation
	findReferencesCalls     int
	gotoImplementationCalls int
}

func (m *mockGoSymbolLSPClient) FindReferences(context.Context, string, int, int, bool) ([]navigation.LSPLocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.findReferencesCalls++
	return append([]navigation.LSPLocation(nil), m.refs...), nil
}

func (m *mockGoSymbolLSPClient) GotoDefinition(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func (m *mockGoSymbolLSPClient) GotoImplementation(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gotoImplementationCalls++
	return append([]navigation.LSPLocation(nil), m.impls...), nil
}

const symbolTestSource = `package example

// NewAgent creates a new Agent.
func NewAgent(name string) *Agent {
	return &Agent{Name: name}
}

// Agent is an agent.
type Agent struct {
	Name string
}

// Run executes the agent.
func (a *Agent) Run() {
	// do something
}

// Setup calls NewAgent.
func Setup() {
	NewAgent("test")
}
`

const symbolTestMultiSource = `package example

func Build() {}

type Config struct{}

func (c *Config) Build() string { return "" }
`

func setupSymbolTestDir(t *testing.T, filename, source string) string {
	t.Helper()
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore directory: %v", err)
		}
	})
	return dir
}

func setupMultiLangDir(t *testing.T, files map[string]string) string {
	t.Helper()
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	dir := t.TempDir()
	for name, content := range files {
		fullPath := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSearchCode_MetaCharSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// regex メタ文字を含むパターン → looksLikeIdentifier で拒否 → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "Set[A-Z]Tools", Path: "."})
	if !strings.Contains(result, "No matches found") && !strings.Contains(result, lineRangeHint) {
		t.Error("pattern with regex meta chars should fall back to text search")
	}
}

func TestLooksLikeGoIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"NewAgent", true},
		{"Config.Build", true},
		{"(*Config).Build", true},
		{"SetMCPTools", true},
		{"myFunc_v2", true},
		{"Set.*Tools", false},    // regex .* pattern
		{"Build.+Config", false}, // regex .+ pattern
		{"hello world", false},   // whitespace
		{"", false},              // empty
		{"foo[0]", false},        // regex meta []
		{"a|b", false},           // regex meta |
		{"foo\\bar", false},      // regex meta backslash
		{"name^2", false},        // regex meta ^
		{"end$", false},          // regex meta $
		{"foo+bar", false},       // regex meta +
		{"foo?bar", false},       // regex meta ?
		{"foo(bar)", false},      // ( without leading (*
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestContainsRegexMeta(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"NewAgent", false},
		{"Config.Build", false},    // . is not in meta set
		{"(*Config).Build", false}, // * ( ) are not in meta set
		{"foo+bar", true},
		{"foo?bar", true},
		{"foo[0]", true},
		{"a|b", true},
		{"foo\\bar", true},
		{"^start", true},
		{"end$", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := containsRegexMeta(tt.input)
			if got != tt.want {
				t.Errorf("containsRegexMeta(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func countSinglePatternBundleCacheEntries() int {
	count := 0
	singlePatternBundleCache.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func countSearchAffectedFilesCacheEntries() int {
	count := 0
	searchAffectedFilesCache.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func visibleLocatorIDs(output string) []string {
	re := regexp.MustCompile(`\[L\d+\]`)
	matches := re.FindAllString(output, -1)
	seen := make(map[string]bool)
	ids := make([]string, 0, len(matches))
	for _, id := range matches {
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func locatorIDForLine(t *testing.T, output, needle string) string {
	t.Helper()
	re := regexp.MustCompile(`\[L\d+\]`)
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, needle) {
			continue
		}
		id := re.FindString(line)
		if id == "" {
			t.Fatalf("expected locator ID on line containing %q in output:\n%s", needle, output)
		}
		return id
	}
	t.Fatalf("expected line containing %q in output:\n%s", needle, output)
	return ""
}

func containsAffectedFile(affected []string, want string) bool {
	for _, file := range affected {
		if file == want {
			return true
		}
	}
	return false
}

func TestIsSymbolResolvableLanguage(t *testing.T) {
	supported := []string{"py", "python", "ts", "tsx", "js", "jsx", "mjs", "rs", "rust", "java", "kt", "rb", "ruby", "php", "c", "cpp", "swift", "scala", "sh", "bash"}
	for _, lang := range supported {
		if !isSymbolResolvableLanguage(lang) {
			t.Errorf("expected %q to be resolvable", lang)
		}
	}
	unsupported := []string{"haskell", "erlang", "prolog", ""}
	for _, lang := range unsupported {
		if isSymbolResolvableLanguage(lang) {
			t.Errorf("expected %q to NOT be resolvable", lang)
		}
	}
}
