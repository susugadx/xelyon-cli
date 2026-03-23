package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

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

func TestSearchCode_SymbolSingleHit(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: "."})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit, got no matches")
	}
	// symbol auto hit は inspect 形式の結果を返す（lineRangeHint を含まない）
	if strings.Contains(result, lineRangeHint) {
		t.Error("symbol auto result should not contain lineRangeHint")
	}
	if !strings.Contains(result, "NewAgent") {
		t.Error("expected symbol name in result")
	}
}

func TestSearchCode_SymbolMultipleHit(t *testing.T) {
	setupSymbolTestDir(t, "multi.go", symbolTestMultiSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "Build", Path: "."})
	if !strings.Contains(result, "Multiple symbols matched") {
		t.Errorf("expected multiple symbols result, got: %s", result)
	}
}

func TestSearchCode_SymbolFallbackToText(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "NonExistentXYZ12345", Path: "."})
	if !strings.Contains(result, "No matches found") {
		t.Errorf("expected 'No matches found', got: %s", result)
	}
}

func TestSearchCode_RegexSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// is_regex=true → symbol auto をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", IsRegex: true})
	// text search 結果は lineRangeHint を含む（symbol auto は IsRegex=true で完全スキップ）
	if !strings.Contains(result, lineRangeHint) {
		t.Error("regex search should fall back to text search with lineRangeHint")
	}
}

func TestSearchCode_UnsupportedLangSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// 非対応言語 → symbol auto をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", FileType: "haskell"})
	if !strings.Contains(result, "No matches found") && !strings.Contains(result, lineRangeHint) {
		t.Error("unsupported language should fall back to text search")
	}
}

func TestSearchCode_FilePatternSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// glob パターン指定 → symbol auto をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", FilePattern: "*.go"})
	if !strings.Contains(result, lineRangeHint) {
		t.Error("file pattern should fall back to text search with lineRangeHint")
	}
}

// ── 多言語シンボル解決テスト ──

const pythonTestSource = `class User:
    def __init__(self, name):
        self.name = name

    def authenticate(self, password):
        return check_password(self.name, password)
`

const pythonTestUsageSource = `from models import User

def login_view(request):
    user = User(request.name)
    if user.authenticate(request.password):
        return "OK"
`

func TestSearchCode_PythonSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"models.py": pythonTestSource,
		"views.py":  pythonTestUsageSource,
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "authenticate", Path: dir, FileType: "py"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit for Python function")
	}
	if !strings.Contains(result, "authenticate") {
		t.Error("expected symbol name in result")
	}
	if !strings.Contains(result, "function") {
		t.Error("expected kind 'function' in result")
	}
	// symbol 解決結果は lineRangeHint を含まない
	if strings.Contains(result, lineRangeHint) {
		t.Error("symbol result should not contain lineRangeHint")
	}
}

func TestSearchCode_PythonSymbolMultipleDefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"a.py": "def process(data):\n    return data\n",
		"b.py": "def process(items):\n    return items\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "process", Path: dir, FileType: "py"})
	if !strings.Contains(result, "Multiple definitions found") {
		t.Errorf("expected multiple definitions, got: %s", result)
	}
}

func TestSearchCode_PythonSymbolFallbackToText(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"app.py": "print('hello')\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "NonExistentXYZ12345", Path: dir, FileType: "py"})
	if !strings.Contains(result, "No matches found") {
		t.Errorf("expected no matches, got: %s", result)
	}
}

const tsTestSource = `export class UserService {
  constructor(private db: Database) {}

  async getUser(id: string): Promise<User> {
    return this.db.find(id)
  }
}
`

const tsTestUsageSource = `import { UserService } from './service'

const svc = new UserService(db)
const user = svc.getUser("123")
`

func TestSearchCode_TypeScriptSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"service.ts": tsTestSource,
		"app.ts":     tsTestUsageSource,
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "UserService", Path: dir, FileType: "ts"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit for TypeScript class")
	}
	if !strings.Contains(result, "UserService") {
		t.Error("expected symbol name in result")
	}
	if !strings.Contains(result, "class") {
		t.Error("expected kind 'class' in result")
	}
}

func TestSearchCode_GenericSymbolTestSeparation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"models.py":      "class User:\n    pass\n",
		"views.py":       "user = User()\n",
		"test_models.py": "def test_user():\n    u = User()\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "User", Path: dir, FileType: "py"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Error("expected test references to be separated into Related Tests section")
	}
}

func TestSearchCode_GenericSymbolNoRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"util.py": "def unused_helper():\n    pass\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "unused_helper", Path: dir, FileType: "py"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit even with no refs")
	}
	if !strings.Contains(result, "No references found") {
		t.Error("expected 'No references found' message")
	}
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
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
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

// ── multi-pattern + symbol fast path テスト ──

func TestSearchCode_MultiPatternGoSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// カンマ区切りで Go シンボル2つ → 両方 symbol 解決される
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent,Run", Path: "."})
	if !strings.Contains(result, "Pattern 1/2") {
		t.Error("expected Pattern 1/2 header")
	}
	if !strings.Contains(result, "Pattern 2/2") {
		t.Error("expected Pattern 2/2 header")
	}
	if !strings.Contains(result, "NewAgent") {
		t.Error("expected NewAgent in result")
	}
	if !strings.Contains(result, "Run") {
		t.Error("expected Run in result")
	}
}

func TestSearchCode_MultiPatternPythonSymbol(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"models.py": pythonTestSource,
		"views.py":  pythonTestUsageSource,
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "authenticate,login_view", Path: dir, FileType: "py"})
	if !strings.Contains(result, "Pattern 1/2") {
		t.Error("expected Pattern 1/2 header")
	}
	if !strings.Contains(result, "authenticate") {
		t.Error("expected authenticate in result")
	}
	if !strings.Contains(result, "login_view") {
		t.Error("expected login_view in result")
	}
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
