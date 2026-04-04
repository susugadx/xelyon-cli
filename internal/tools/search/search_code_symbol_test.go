package search

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type mockGoSymbolLSPClient struct {
	refs  []navigation.LSPLocation
	impls []navigation.LSPLocation
}

func (m *mockGoSymbolLSPClient) FindReferences(context.Context, string, int, int, bool) ([]navigation.LSPLocation, error) {
	return m.refs, nil
}

func (m *mockGoSymbolLSPClient) GotoDefinition(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return nil, nil
}

func (m *mockGoSymbolLSPClient) GotoImplementation(context.Context, string, int, int) ([]navigation.LSPLocation, error) {
	return m.impls, nil
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

	// explicit mode=regex → symbol rescue/resolve をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", Mode: string(SearchModeRegex)})
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

const tsArrowTestSource = `const buildMap = async (): Promise<Map<string, string>> => {
  return new Map<string, string>()
}
`

const tsArrowUsageSource = `const mapPromise = buildMap()
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

func TestSearchCode_TypeScriptArrowFunctionSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"service.ts": tsArrowTestSource,
		"app.ts":     tsArrowUsageSource,
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "buildMap", Path: dir, FileType: "ts"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit for TypeScript arrow function")
	}
	if !strings.Contains(result, "buildMap") {
		t.Error("expected symbol name in result")
	}
	if !strings.Contains(result, "function") {
		t.Error("expected kind 'function' in result")
	}
	if strings.Contains(result, lineRangeHint) {
		t.Error("symbol result should not contain lineRangeHint")
	}
}

// ── TS/JS enhanced symbol path テスト ──

func TestSearchCode_TypeScriptImportCallerClassification(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"service.ts": "export class UserService {\n  async getUser(id: string) { return id }\n}\n",
		"app.ts":     "import { UserService } from './service'\nconst svc = new UserService(db)\n",
		"handler.ts": "const x: UserService = getSvc()\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "UserService", Path: dir, FileType: "ts"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Imports") {
		t.Errorf("expected Imports section, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
	// class kind preserved
	if !strings.Contains(result, "class") {
		t.Error("expected kind 'class' in result")
	}
}

func TestSearchCode_TypeScriptFunctionCallerClassification(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"api.ts": "export async function fetchUser(id: string) { return id }\n",
		"app.ts": "import { fetchUser } from './api'\nconst user = fetchUser('123')\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "fetchUser", Path: dir, FileType: "ts"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Imports") {
		t.Errorf("expected Imports section, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
}

func TestSearchCode_TypeScriptTestSeparation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"service.ts":      "export function doWork() { return 1 }\n",
		"app.ts":          "import { doWork } from './service'\ndoWork()\n",
		"service.test.ts": "import { doWork } from './service'\ndoWork()\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "doWork", Path: dir, FileType: "ts"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests section, got:\n%s", result)
	}
}

func TestSearchCode_TypeScriptNoRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"unused.ts": "export function unusedHelper() { return 1 }\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "unusedHelper", Path: dir, FileType: "ts"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit even with no refs")
	}
	if !strings.Contains(result, "No references found") {
		t.Errorf("expected 'No references found', got:\n%s", result)
	}
}

func TestSearchCode_TypeScriptFallbackToText(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"app.ts": "console.log('hello')\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "NonExistent12345", Path: dir, FileType: "ts"})
	if !strings.Contains(result, "No matches found") {
		t.Errorf("expected fallback to text search with no matches, got: %s", result)
	}
}

func TestClassifyJSRefs(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "app.ts", Line: 1, Snippet: "import { UserService } from './service'"},
		{File: "app.ts", Line: 3, Snippet: "const svc = new UserService(db)"},
		{File: "app.ts", Line: 5, Snippet: "const x: UserService = getSvc()"},
		{File: "app.ts", Line: 7, Snippet: "// UserService is great"},
		{File: "handler.ts", Line: 1, Snippet: "export { UserService } from './service'"},
		{File: "handler.ts", Line: 3, Snippet: "UserService()"},
		{File: "types.ts", Line: 1, Snippet: "class Admin extends UserService {"},
		{File: "types.ts", Line: 2, Snippet: "interface Foo<UserService> {"},
	}

	imports, callers, typeRefs, others := classifyJSRefs(refs, "UserService")

	if len(imports) != 2 {
		t.Errorf("expected 2 imports, got %d: %+v", len(imports), imports)
	}
	if len(callers) != 2 {
		t.Errorf("expected 2 callers (new + direct call), got %d: %+v", len(callers), callers)
	}
	if len(typeRefs) != 3 {
		t.Errorf("expected 3 type refs (: annotation + extends + generic), got %d: %+v", len(typeRefs), typeRefs)
	}
	if len(others) != 1 {
		t.Errorf("expected 1 other (comment), got %d: %+v", len(others), others)
	}
}

// ── Python enhanced symbol path テスト ──

func TestClassifyPythonRefs(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "views.py", Line: 1, Snippet: "from models import User"},
		{File: "views.py", Line: 3, Snippet: "user = User(request.name)"},
		{File: "views.py", Line: 5, Snippet: "@User"},
		{File: "views.py", Line: 7, Snippet: "# User is a model"},
		{File: "admin.py", Line: 1, Snippet: "import User"},
		{File: "admin.py", Line: 3, Snippet: "from User.models import Foo"},
	}

	imports, callers, decorators, others := classifyPythonRefs(refs, "User")

	if len(imports) != 3 {
		t.Errorf("expected 3 imports, got %d: %+v", len(imports), imports)
	}
	if len(callers) != 1 {
		t.Errorf("expected 1 caller, got %d: %+v", len(callers), callers)
	}
	if len(decorators) != 1 {
		t.Errorf("expected 1 decorator, got %d: %+v", len(decorators), decorators)
	}
	if len(others) != 1 {
		t.Errorf("expected 1 other, got %d: %+v", len(others), others)
	}
}

func TestSearchCode_PythonImportCallerClassification(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"models.py": "class User:\n    pass\n",
		"views.py":  "from models import User\nuser = User()\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "User", Path: dir, FileType: "py"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Imports") {
		t.Errorf("expected Imports section, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
}

func TestSearchCode_PythonDecoratorClassification(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"deco.py":  "def login_required(f):\n    return f\n",
		"views.py": "from deco import login_required\n@login_required\ndef secret():\n    pass\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "login_required", Path: dir, FileType: "py"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Decorators") {
		t.Errorf("expected Decorators section, got:\n%s", result)
	}
}

func TestSearchCode_PythonTestSeparation(t *testing.T) {
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
		t.Errorf("expected Related Tests section, got:\n%s", result)
	}
}

func TestSearchCode_PythonTestSeparation_SuffixTest(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"models.py":      "class User:\n    pass\n",
		"views.py":       "user = User()\n",
		"models_test.py": "def test_user():\n    u = User()\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "User", Path: dir, FileType: "py"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests for *_test.py, got:\n%s", result)
	}
}

func TestSearchCode_PythonTestSeparation_TestsDir(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"models.py":            "class User:\n    pass\n",
		"views.py":             "user = User()\n",
		"tests/test_models.py": "def test_user():\n    u = User()\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "User", Path: dir, FileType: "py"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests for tests/ dir, got:\n%s", result)
	}
}

func TestSearchCode_PythonNoRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"util.py": "def unused_helper():\n    pass\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "unused_helper", Path: dir, FileType: "py"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "No references found") {
		t.Errorf("expected 'No references found', got:\n%s", result)
	}
}

// ── Rust enhanced symbol path テスト ──

func TestClassifyRustRefs(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "main.rs", Line: 1, Snippet: "use crate::config::Config;"},
		{File: "main.rs", Line: 5, Snippet: "let cfg = Config::new()"},
		{File: "main.rs", Line: 7, Snippet: "impl Config {"},
		{File: "main.rs", Line: 9, Snippet: "impl Display for Config {"},
		{File: "main.rs", Line: 11, Snippet: "// Config is important"},
		{File: "main.rs", Line: 13, Snippet: "Config(value)"},
		{File: "main.rs", Line: 15, Snippet: "dyn Config"},
		{File: "main.rs", Line: 17, Snippet: "config_log!(Config)"},
	}

	uses, callers, implRefs, others := classifyRustRefs(refs, "Config")

	if len(uses) != 1 {
		t.Errorf("expected 1 use, got %d: %+v", len(uses), uses)
	}
	if len(callers) != 2 {
		t.Errorf("expected 2 callers (::new + direct call), got %d: %+v", len(callers), callers)
	}
	if len(implRefs) != 3 {
		t.Errorf("expected 3 impl refs (impl + impl for + dyn), got %d: %+v", len(implRefs), implRefs)
	}
	if len(others) != 2 {
		t.Errorf("expected 2 others (comment + macro arg), got %d: %+v", len(others), others)
	}
}

func TestSearchCode_RustSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs":  "pub struct Config {\n    pub name: String,\n}\n",
		"main.rs": "use crate::Config;\nlet c = Config { name: \"x\".into() };\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "Config", Path: dir, FileType: "rs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit for Rust struct")
	}
	if !strings.Contains(result, "Config") {
		t.Error("expected symbol name in result")
	}
	if !strings.Contains(result, "struct") {
		t.Errorf("expected kind 'struct', got:\n%s", result)
	}
}

func TestSearchCode_RustUsesAndCallers(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs":  "pub fn process(data: &str) -> String {\n    data.to_string()\n}\n",
		"main.rs": "use crate::process;\nlet result = process(\"hello\");\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "process", Path: dir, FileType: "rs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Uses") {
		t.Errorf("expected Uses section, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
}

func TestSearchCode_RustFallbackToText(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs": "fn main() {}\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "NonExistent12345", Path: dir, FileType: "rs"})
	if !strings.Contains(result, "No matches found") {
		t.Errorf("expected fallback with no matches, got: %s", result)
	}
}

func TestSearchCode_RustNoRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs": "pub fn unused_fn() {}\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "unused_fn", Path: dir, FileType: "rs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "No references found") {
		t.Errorf("expected 'No references found', got:\n%s", result)
	}
}

func TestSearchCode_RustTestSeparation_TestsDir(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs":                    "pub fn process(data: &str) -> String {\n    data.to_string()\n}\n",
		"main.rs":                   "let result = process(\"hello\");\n",
		"tests/integration_test.rs": "let result = process(\"test\");\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "process", Path: dir, FileType: "rs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests for tests/ dir, got:\n%s", result)
	}
}

// ── Java/Kotlin enhanced symbol path テスト ──

func TestClassifyJavaRefs(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "App.java", Line: 1, Snippet: "import com.example.UserService;"},
		{File: "App.java", Line: 3, Snippet: "UserService svc = new UserService();"},
		{File: "App.java", Line: 5, Snippet: "@UserService"},
		{File: "App.java", Line: 7, Snippet: "class Admin extends UserService {"},
		{File: "App.java", Line: 9, Snippet: "// UserService comment"},
	}

	imports, callers, annotations, inheritance, others := classifyJavaRefs(refs, "UserService")

	if len(imports) != 1 {
		t.Errorf("expected 1 import, got %d: %+v", len(imports), imports)
	}
	if len(callers) != 1 {
		t.Errorf("expected 1 caller, got %d: %+v", len(callers), callers)
	}
	if len(annotations) != 1 {
		t.Errorf("expected 1 annotation, got %d: %+v", len(annotations), annotations)
	}
	if len(inheritance) != 1 {
		t.Errorf("expected 1 inheritance, got %d: %+v", len(inheritance), inheritance)
	}
	if len(others) != 1 {
		t.Errorf("expected 1 other, got %d: %+v", len(others), others)
	}
}

func TestSearchCode_JavaSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"UserService.java": "public class UserService {\n    public void run() {}\n}\n",
		"App.java":         "import com.UserService;\nUserService svc = new UserService();\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "UserService", Path: dir, FileType: "java"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "class") {
		t.Errorf("expected kind 'class', got:\n%s", result)
	}
	if !strings.Contains(result, "Imports") {
		t.Errorf("expected Imports section, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
}

func TestSearchCode_JavaTestSeparation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"UserService.java":     "public class UserService {\n}\n",
		"App.java":             "UserService svc = new UserService();\n",
		"UserServiceTest.java": "UserService svc = new UserService();\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "UserService", Path: dir, FileType: "java"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests section, got:\n%s", result)
	}
}

func TestSearchCode_KotlinSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"Config.kt": "data class Config(\n    val name: String\n)\n",
		"App.kt":    "val cfg = Config(name = \"x\")\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "Config", Path: dir, FileType: "kt"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "class") {
		t.Errorf("expected kind 'class', got:\n%s", result)
	}
}

// ── C# enhanced symbol path テスト ──

func TestClassifyCSharpRefs(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "App.cs", Line: 1, Snippet: "using MyApp.Services;"},
		{File: "App.cs", Line: 3, Snippet: "var svc = new OrderService();"},
		{File: "App.cs", Line: 5, Snippet: "[OrderService]"},
		{File: "App.cs", Line: 7, Snippet: "class Admin : OrderService {"},
		{File: "App.cs", Line: 9, Snippet: "// OrderService comment"},
	}

	usings, callers, attributes, inheritance, others := classifyCSharpRefs(refs, "OrderService")

	if len(usings) != 1 {
		t.Errorf("expected 1 using, got %d: %+v", len(usings), usings)
	}
	if len(callers) != 1 {
		t.Errorf("expected 1 caller, got %d: %+v", len(callers), callers)
	}
	if len(attributes) != 1 {
		t.Errorf("expected 1 attribute, got %d: %+v", len(attributes), attributes)
	}
	if len(inheritance) != 1 {
		t.Errorf("expected 1 inheritance, got %d: %+v", len(inheritance), inheritance)
	}
	if len(others) != 1 {
		t.Errorf("expected 1 other, got %d: %+v", len(others), others)
	}
}

func TestSearchCode_CSharpSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"OrderService.cs": "public class OrderService {\n    public void Process() {}\n}\n",
		"App.cs":          "var svc = new OrderService();\nsvc.Process();\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "OrderService", Path: dir, FileType: "cs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "class") {
		t.Errorf("expected kind 'class', got:\n%s", result)
	}
	if !strings.Contains(result, "Callers") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
}

func TestSearchCode_CSharpTestSeparation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"OrderService.cs":      "public class OrderService {}\n",
		"App.cs":               "var svc = new OrderService();\n",
		"OrderServiceTests.cs": "var svc = new OrderService();\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "OrderService", Path: dir, FileType: "cs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests section, got:\n%s", result)
	}
}

// ── PHP enhanced symbol path テスト ──

func TestClassifyPHPRefs(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "app.php", Line: 1, Snippet: "use App\\UserRepository;"},
		{File: "app.php", Line: 3, Snippet: "new UserRepository()"},
		{File: "app.php", Line: 5, Snippet: "class AdminRepo extends UserRepository {"},
		{File: "app.php", Line: 7, Snippet: "// UserRepository comment"},
	}

	uses, callers, inheritance, others := classifyPHPRefs(refs, "UserRepository")

	if len(uses) != 1 {
		t.Errorf("expected 1 use, got %d: %+v", len(uses), uses)
	}
	if len(callers) != 1 {
		t.Errorf("expected 1 caller, got %d: %+v", len(callers), callers)
	}
	if len(inheritance) != 1 {
		t.Errorf("expected 1 inheritance, got %d: %+v", len(inheritance), inheritance)
	}
	if len(others) != 1 {
		t.Errorf("expected 1 other, got %d: %+v", len(others), others)
	}
}

func TestSearchCode_PHPSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"UserRepository.php": "<?php\nclass UserRepository {\n    public function find() {}\n}\n",
		"app.php":            "<?php\nuse App\\UserRepository;\n$repo = new UserRepository();\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "UserRepository", Path: dir, FileType: "php"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "class") {
		t.Errorf("expected kind 'class', got:\n%s", result)
	}
	if !strings.Contains(result, "Uses") {
		t.Errorf("expected Uses section, got:\n%s", result)
	}
}

func TestSearchCode_PHPTestSeparation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"UserRepository.php":     "<?php\nclass UserRepository {}\n",
		"app.php":                "<?php\nnew UserRepository();\n",
		"UserRepositoryTest.php": "<?php\nnew UserRepository();\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "UserRepository", Path: dir, FileType: "php"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests section, got:\n%s", result)
	}
}

// ── Ruby enhanced symbol path テスト ──

func TestClassifyRubyRefs(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "app.rb", Line: 1, Snippet: "require 'user_service'"},
		{File: "app.rb", Line: 3, Snippet: "UserService.new"},
		{File: "app.rb", Line: 5, Snippet: "include UserService"},
		{File: "app.rb", Line: 7, Snippet: "class Admin < UserService"},
		{File: "app.rb", Line: 9, Snippet: "# UserService comment"},
	}
	requires, callers, mixins, others := classifyRubyRefs(refs, "UserService")
	if len(requires) != 1 {
		t.Errorf("expected 1 require, got %d", len(requires))
	}
	if len(callers) != 1 {
		t.Errorf("expected 1 caller, got %d", len(callers))
	}
	if len(mixins) != 2 {
		t.Errorf("expected 2 mixins (include + <), got %d", len(mixins))
	}
	if len(others) != 1 {
		t.Errorf("expected 1 other, got %d", len(others))
	}
}

func TestSearchCode_RubySymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"user.rb": "class UserService\n  def run; end\nend\n",
		"app.rb":  "require_relative 'user'\nsvc = UserService.new\n",
	})
	result := ExecuteSearchCode(SearchOptions{Pattern: "UserService", Path: dir, FileType: "rb"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "class") {
		t.Errorf("expected kind 'class', got:\n%s", result)
	}
}

func TestSearchCode_RubyTestSeparation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"user.rb":      "class UserService; end\n",
		"app.rb":       "UserService.new\n",
		"user_spec.rb": "UserService.new\n",
	})
	result := ExecuteSearchCode(SearchOptions{Pattern: "UserService", Path: dir, FileType: "rb"})
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests, got:\n%s", result)
	}
}

// ── Swift enhanced symbol path テスト ──

func TestSearchCode_SwiftSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"Config.swift": "public struct Config {\n    var name: String\n}\n",
		"App.swift":    "let cfg = Config(name: \"x\")\n",
	})
	result := ExecuteSearchCode(SearchOptions{Pattern: "Config", Path: dir, FileType: "swift"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "struct") {
		t.Errorf("expected kind 'struct', got:\n%s", result)
	}
	if !strings.Contains(result, "Callers") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
}

// ── Scala enhanced symbol path テスト ──

func TestSearchCode_ScalaSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"Config.scala": "case class Config(name: String)\n",
		"App.scala":    "import com.Config\nval cfg = Config(\"x\")\n",
	})
	result := ExecuteSearchCode(SearchOptions{Pattern: "Config", Path: dir, FileType: "scala"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "class") {
		t.Errorf("expected kind 'class', got:\n%s", result)
	}
	if !strings.Contains(result, "Imports") {
		t.Errorf("expected Imports section, got:\n%s", result)
	}
}

// ── Elixir enhanced symbol path テスト ──

func TestSearchCode_ElixirSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"user.ex": "defmodule UserService do\n  def run, do: :ok\nend\n",
		"app.ex":  "alias MyApp.UserService\nUserService.run()\n",
	})
	result := ExecuteSearchCode(SearchOptions{Pattern: "UserService", Path: dir, FileType: "ex"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "module") {
		t.Errorf("expected kind 'module', got:\n%s", result)
	}
}

func TestSearchCode_ElixirTestSeparation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"user.ex":       "defmodule UserService do\nend\n",
		"app.ex":        "UserService.run()\n",
		"user_test.exs": "UserService.run()\n",
	})
	result := ExecuteSearchCode(SearchOptions{Pattern: "UserService", Path: dir, FileType: "ex"})
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests, got:\n%s", result)
	}
}

// ── Lua enhanced symbol path テスト ──

func TestSearchCode_LuaSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"utils.lua": "function process(data)\n  return data\nend\n",
		"app.lua":   "local u = require('utils')\nprocess('hello')\n",
	})
	result := ExecuteSearchCode(SearchOptions{Pattern: "process", Path: dir, FileType: "lua"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "function") {
		t.Errorf("expected kind 'function', got:\n%s", result)
	}
}

// ── C/C++ enhanced symbol path テスト ──

func TestSearchCode_CppSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.h": "struct Config {\n    int value;\n};\n",
		"main.cpp": "#include \"config.h\"\nConfig cfg;\ncfg.value = 1;\n",
	})
	result := ExecuteSearchCode(SearchOptions{Pattern: "Config", Path: dir, FileType: "cpp"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "struct") {
		t.Errorf("expected kind 'struct', got:\n%s", result)
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

func TestBuildGoSymbolBundleIncludesEditSurface(t *testing.T) {
	bundle := buildGoSymbolBundle("Close", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:     "Close",
			Kind:     "method",
			File:     "agent.go",
			Line:     10,
			EndLine:  14,
			Receiver: "*Agent",
		},
		Body: []string{
			"10: func (a *Agent) Close() error {",
			"11: \treturn nil",
			"12: }",
		},
		Callers: []navigation.Reference{
			{File: "runner.go", Line: 22, Scope: "shutdown", Snippet: "return agent.Close()"},
		},
		TotalCallers: 1,
		Tests: []navigation.TestRef{
			{File: "agent_test.go", Line: 8, Name: "TestClose"},
		},
		TotalTests: 1,
	})
	result := formatSymbolBundle(bundle, nil, nil)

	if !strings.Contains(result, "Definition:") {
		t.Errorf("expected Definition section, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers (1):") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
	if !strings.Contains(result, "Related Tests (1):") {
		t.Errorf("expected Related Tests section, got:\n%s", result)
	}
}

func TestBuildGoSymbolBundleCarriesDiagnostics(t *testing.T) {
	bundle := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Run",
			Kind:    "function",
			File:    "run.go",
			Line:    10,
			EndLine: 12,
		},
		Body: []string{
			"10: func Run() {",
			"11: }",
		},
		ResolvedViaLSP:     true,
		UpstreamIncomplete: true,
	})
	result := formatSymbolBundle(bundle, nil, nil)
	if !strings.Contains(result, "Warning: upstream search may be incomplete.") {
		t.Fatalf("expected incomplete warning in bundle output, got:\n%s", result)
	}
	if !strings.Contains(result, "Note: resolved via gopls.") {
		t.Fatalf("expected LSP note in bundle output, got:\n%s", result)
	}
}

func TestBuildGoSymbolBundleCarriesTruncatedDiagnostic(t *testing.T) {
	bundle := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Run",
			Kind:    "function",
			File:    "run.go",
			Line:    10,
			EndLine: 12,
		},
		Body: []string{
			"10: func Run() {",
			"11: }",
		},
		UpstreamTruncated: true,
	})
	result := formatSymbolBundle(bundle, nil, nil)
	if !strings.Contains(result, "Note: upstream results were truncated.") {
		t.Fatalf("expected truncation note in bundle output, got:\n%s", result)
	}
}

func TestBuildGoSymbolBundleCanonicalIsStableAcrossLineMoves(t *testing.T) {
	first := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:               "Run",
			Kind:               "method",
			File:               "pkg/run.go",
			Line:               10,
			EndLine:            12,
			Receiver:           "*Agent",
			ReceiverNorm:       "Agent",
			Signature:          "func (a *Agent) Run() error",
			PackageDir:         "pkg",
			StableKey:          stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error"),
			StableKeyCollision: false,
		},
	})
	second := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:               "Run",
			Kind:               "method",
			File:               "pkg/run.go",
			Line:               40,
			EndLine:            42,
			Receiver:           "*Agent",
			ReceiverNorm:       "Agent",
			Signature:          "func (a *Agent) Run() error",
			PackageDir:         "pkg",
			StableKey:          stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error"),
			StableKeyCollision: false,
		},
	})

	if first.Identity.Canonical == "" || second.Identity.Canonical == "" {
		t.Fatal("expected canonical identity to be populated")
	}
	if first.Identity.Canonical != second.Identity.Canonical {
		t.Fatalf("expected stable canonical identity across line moves, got %q vs %q", first.Identity.Canonical, second.Identity.Canonical)
	}
}

func TestBuildGoSymbolBundleCanonicalAddsFileDisambiguatorOnCollision(t *testing.T) {
	stableKey := stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error")
	first := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:               "Run",
			Kind:               "method",
			File:               "pkg/run_linux.go",
			Line:               10,
			EndLine:            12,
			Receiver:           "*Agent",
			ReceiverNorm:       "Agent",
			Signature:          "func (a *Agent) Run() error",
			PackageDir:         "pkg",
			StableKey:          stableKey,
			StableKeyCollision: true,
		},
	})
	second := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:               "Run",
			Kind:               "method",
			File:               "pkg/run_darwin.go",
			Line:               10,
			EndLine:            12,
			Receiver:           "*Agent",
			ReceiverNorm:       "Agent",
			Signature:          "func (a *Agent) Run() error",
			PackageDir:         "pkg",
			StableKey:          stableKey,
			StableKeyCollision: true,
		},
	})

	if first.Identity.Canonical == second.Identity.Canonical {
		t.Fatalf("expected file disambiguator for colliding stable keys, got %q", first.Identity.Canonical)
	}
	if !strings.Contains(first.Identity.Canonical, "file=pkg/run_linux.go") {
		t.Fatalf("expected linux file disambiguator, got %q", first.Identity.Canonical)
	}
	if !strings.Contains(second.Identity.Canonical, "file=pkg/run_darwin.go") {
		t.Fatalf("expected darwin file disambiguator, got %q", second.Identity.Canonical)
	}
}

func TestGoSymbolResolver_UsesBundleDiagnostics(t *testing.T) {
	setupSymbolTestDir(t, "example.go", `package example

func Run() {}
`)

	resolved := goSymbolResolver{}.Resolve("Run", SearchOptions{
		Path:            ".",
		LSPClient:       &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "example.go", Line: 3, Character: 1, EndLine: 3, EndChar: 5}}},
		LocatorRegistry: nil,
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}
	if resolved.Bundle == nil {
		t.Fatal("expected bundle in go symbol resolution")
	}
	if !strings.Contains(resolved.Output, "Note: resolved via gopls.") {
		t.Fatalf("expected LSP note in go symbol output, got:\n%s", resolved.Output)
	}
}

func TestGoSymbolResolver_UsesProjectMapSnapshot(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"broken.go": "package example\n\nfunc (\n",
	})

	resolved := goSymbolResolver{}.Resolve("Run", SearchOptions{
		Path: dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "broken.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "go-snapshot-fast-path",
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected snapshot-backed single resolution, got %s", resolved.Status)
	}
	if resolved.Bundle == nil || resolved.Bundle.Definition.File != "broken.go" {
		t.Fatalf("expected snapshot bundle for broken.go, got %+v", resolved.Bundle)
	}
}

func TestGoSymbolResolver_LocatorRegistryDoesNotRegisterHiddenIDs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"builder.go": `package example

type Builder interface {
	Build() string
}
`,
		"builder_impl.go": `package example

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }
`,
		"builder_test.go": `package example

func TestBuild(t *testing.T) {
	var b FileBuilder
	_ = b.Build()
}
`,
	})

	reg := locator.NewRegistry()
	resolved := goSymbolResolver{}.Resolve("Builder", SearchOptions{
		Path:            dir,
		LocatorRegistry: reg,
		LSPClient: &mockGoSymbolLSPClient{
			refs:  []navigation.LSPLocation{{File: "builder_test.go", Line: 5, Character: 1, EndLine: 5, EndChar: 6}},
			impls: []navigation.LSPLocation{{File: "builder_impl.go", Line: 3, Character: 1, EndLine: 3, EndChar: 11}},
		},
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}

	ids := visibleLocatorIDs(resolved.Output)
	if len(ids) == 0 {
		t.Fatalf("expected visible locator IDs in output, got:\n%s", resolved.Output)
	}
	for i, id := range ids {
		want := "[L" + strconv.Itoa(i+1) + "]"
		if id != want {
			t.Fatalf("expected sequential locator %s, got %s in output:\n%s", want, id, resolved.Output)
		}
	}
	if _, ok := reg.Resolve("[L" + strconv.Itoa(len(ids)+1) + "]"); ok {
		t.Fatalf("expected no hidden locator beyond visible IDs, got extra registry entry after %d visible IDs", len(ids))
	}
}

func TestGoSymbolResolver_LocatorRegistryMatchesImplementation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"builder.go": `package example

type Builder interface {
	Build() string
}
`,
		"builder_impl.go": `package example

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }
`,
	})

	reg := locator.NewRegistry()
	resolved := goSymbolResolver{}.Resolve("Builder", SearchOptions{
		Path:            dir,
		LocatorRegistry: reg,
		LSPClient: &mockGoSymbolLSPClient{
			refs:  []navigation.LSPLocation{{File: "builder_test.go", Line: 5, Character: 1, EndLine: 5, EndChar: 6}},
			impls: []navigation.LSPLocation{{File: "builder_impl.go", Line: 3, Character: 1, EndLine: 3, EndChar: 11}},
		},
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}

	implID := locatorIDForLine(t, resolved.Output, "builder_impl.go:3")
	implLoc, ok := reg.Resolve(implID)
	if !ok {
		t.Fatalf("expected implementation locator %s to resolve", implID)
	}
	if implLoc.FilePath != "builder_impl.go" || implLoc.Line != 3 {
		t.Fatalf("unexpected implementation locator target: %+v", implLoc)
	}
}

func TestFormatSymbolBundle_LocatorRegistryMatchesRelatedTest(t *testing.T) {
	reg := locator.NewRegistry()
	bundle := buildGoSymbolBundle("Close", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:     "Close",
			Kind:     "method",
			File:     "agent.go",
			Line:     5,
			EndLine:  7,
			Receiver: "*Agent",
		},
		Body: []string{
			"5: func (a *Agent) Close() error {",
			"6: \treturn nil",
			"7: }",
		},
		Tests: []navigation.TestRef{
			{File: "agent_test.go", Line: 4, Name: "TestClose"},
		},
		TotalTests: 1,
	})
	output := formatSymbolBundle(bundle, reg, nil)

	testID := locatorIDForLine(t, output, "agent_test.go:4")
	testLoc, ok := reg.Resolve(testID)
	if !ok {
		t.Fatalf("expected test locator %s to resolve", testID)
	}
	if testLoc.FilePath != "agent_test.go" || testLoc.Line != 4 {
		t.Fatalf("unexpected test locator target: %+v", testLoc)
	}
}

func TestSearchCode_MultiPatternGoSymbolPreservesDiagnostics(t *testing.T) {
	setupSymbolTestDir(t, "example.go", `package example

type Agent struct{}

func (a *Agent) Close() error { return nil }

func run(a *Agent) error {
	return a.Close()
}
`)

	result := ExecuteSearchCode(SearchOptions{
		Pattern: "Close,(*Agent).Close,\\.Close\\(\\)",
		Path:    ".",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{{File: "example.go", Line: 5, Character: 1, EndLine: 5, EndChar: 10}},
		},
	})

	if !strings.Contains(result, "Matched patterns:") {
		t.Fatalf("expected multi-pattern bundle output, got:\n%s", result)
	}
	if !strings.Contains(result, "Note: resolved via gopls.") {
		t.Fatalf("expected LSP note in multi-pattern output, got:\n%s", result)
	}
}

func TestSearchCode_SymbolFastPathCachesAffectedFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"run.go": "package example\n\nfunc Run() {\n\thelper()\n}\n\nfunc helper() {}\n",
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern: "Run",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 5, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "affected-single",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Run") {
		t.Fatalf("expected symbol result, got:\n%s", result)
	}

	want := filepath.Join(dir, "run.go")
	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	if !containsAffectedFile(affected, want) {
		t.Fatalf("expected exact cache key %q to track %s, got %v", searchKey, want, affected)
	}
}

func TestSearchCode_MultiPatternCacheTracksBundleAndTextAffectedFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"run.go": "package example\n\nfunc Run() {\n\thelper()\n}\n\nfunc helper() {}\n",
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern: "Run,helper()",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 5, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "affected-multi",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Run") || !strings.Contains(result, "helper()") {
		t.Fatalf("expected mixed multi-pattern result, got:\n%s", result)
	}

	want := filepath.Join(dir, "run.go")
	searchKey := singlePatternBundleCacheKey(buildMultiCacheKey(splitPatterns(opts.Pattern)), cache.lastSetPath)
	affected := cache.affected[searchKey]
	if !containsAffectedFile(affected, want) {
		t.Fatalf("expected exact multi cache key %q to track %s, got %v", searchKey, want, affected)
	}
}

func TestSearchCode_MultiPatternCacheSupplementsSymbolMultipleAffectedFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/helper.go":     "package pkg\n\nfunc helper() {}\n",
		"pkg/run_linux.go":  "package pkg\n\nfunc Run() {}\n",
		"pkg/run_darwin.go": "package pkg\n\nfunc Run() {}\n",
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern: "helper(,Run",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/run_linux.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
				{
					Path: "pkg/run_darwin.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "affected-multi-symbol-multiple",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Multiple symbols matched") || !strings.Contains(result, "helper") {
		t.Fatalf("expected mixed text/symbol-multiple result, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey(buildMultiCacheKey(splitPatterns(opts.Pattern)), cache.lastSetPath)
	affected := cache.affected[searchKey]
	wantHelper := filepath.Join(dir, "pkg", "helper.go")
	wantLinux := filepath.Join(dir, "pkg", "run_linux.go")
	wantDarwin := filepath.Join(dir, "pkg", "run_darwin.go")
	for _, want := range []string{wantHelper, wantLinux, wantDarwin} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected exact multi cache key %q to track %s, got %v", searchKey, want, affected)
		}
	}

	cache.InvalidateSearchCacheForFile(wantDarwin)
	if _, ok := cache.GetSearch(buildMultiCacheKey(splitPatterns(opts.Pattern)), cache.lastSetPath); ok {
		t.Fatalf("expected multi-pattern cache entry to be invalidated after editing %s", wantDarwin)
	}
}

func TestSearchCode_SymbolBundleAffectedFilesStayRepoRelativeFromSubdir(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": "package pkg\n\nfunc Run() {}\n",
	})
	subdir := filepath.Join(dir, "pkg")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:            "Run",
		Path:               ".",
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapStateKey: "symbol-subdir-root",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "in pkg/run.go") {
		t.Fatalf("expected repo-relative symbol bundle path, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	want := filepath.Join(dir, "pkg", "run.go")
	if !containsAffectedFile(affected, want) {
		t.Fatalf("expected symbol bundle affected files to include %s, got %v", want, affected)
	}
	if containsAffectedFile(affected, filepath.Join(dir, "run.go")) {
		t.Fatalf("did not expect wrongly rebased root path in affected files: %v", affected)
	}
}

func TestSearchCode_SnapshotBackedSectionItemAffectedFilesUseProjectRoot(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": `package pkg

func Run() {}
`,
		"pkg/run_test.go": `package pkg

func TestRun() {
	Run()
}
`,
	})
	subdir := filepath.Join(dir, "pkg")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:            "Run",
		Path:               ".",
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapStateKey: "snapshot-section-item-subdir",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Related Tests") || !strings.Contains(result, "pkg/run_test.go") {
		t.Fatalf("expected normalized related test path in symbol bundle, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	testFile := filepath.Join(dir, "pkg", "run_test.go")
	if !containsAffectedFile(affected, testFile) {
		t.Fatalf("expected section item affected files to include %s, got %v", testFile, affected)
	}
	if containsAffectedFile(affected, filepath.Join(dir, "run_test.go")) {
		t.Fatalf("did not expect wrongly rebased section item path in affected files: %v", affected)
	}

	cache.InvalidateSearchCacheForFile(testFile)
	if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
		t.Fatalf("expected symbol cache entry to be invalidated after editing %s", testFile)
	}
}

func TestSearchCode_SnapshotBackedLSPSectionItemKeepsRepoRelativePath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": `package pkg

func Run() {}
`,
		"pkg/run_test.go": `package pkg

func TestRun() {
	Run()
}
`,
	})
	subdir := filepath.Join(dir, "pkg")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:            "Run",
		Path:               ".",
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapStateKey: "snapshot-lsp-section-item-subdir",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "pkg/run_test.go", Line: 4, Character: 1, EndLine: 4, EndChar: 5},
			},
		},
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Related Tests") || !strings.Contains(result, "pkg/run_test.go:3") {
		t.Fatalf("expected repo-relative LSP section item path, got:\n%s", result)
	}
	if strings.Contains(result, "pkg/pkg/run_test.go") {
		t.Fatalf("did not expect doubly rebased LSP path, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	testFile := filepath.Join(dir, "pkg", "run_test.go")
	if !containsAffectedFile(affected, testFile) {
		t.Fatalf("expected LSP section item affected files to include %s, got %v", testFile, affected)
	}

	cache.InvalidateSearchCacheForFile(testFile)
	if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
		t.Fatalf("expected symbol cache entry to be invalidated after editing %s", testFile)
	}
}

func TestSearchCode_SnapshotBackedLSPSectionItemPreservesInvocationRelativePath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": `package pkg

func Run() {}
`,
		"shared/run_test.go": `package shared

func TestRunFromShared(t *testing.T) {
	pkg.Run()
}
`,
	})
	subdir := filepath.Join(dir, "pkg")

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:            "Run",
		Path:               ".",
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapStateKey: "snapshot-lsp-parent-relative",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "../shared/run_test.go", Line: 4, Character: 2, EndLine: 4, EndChar: 6},
			},
		},
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Related Tests") || !strings.Contains(result, "shared/run_test.go:3") {
		t.Fatalf("expected parent-relative LSP path to normalize to repo-relative, got:\n%s", result)
	}
	if strings.Contains(result, "../shared/run_test.go") {
		t.Fatalf("did not expect invocation-relative path to leak into output, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	testFile := filepath.Join(dir, "shared", "run_test.go")
	if !containsAffectedFile(affected, testFile) {
		t.Fatalf("expected invocation-relative LSP affected files to include %s, got %v", testFile, affected)
	}

	cache.InvalidateSearchCacheForFile(testFile)
	if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
		t.Fatalf("expected symbol cache entry to be invalidated after editing %s", testFile)
	}
}

func TestSearchCode_SymbolBundleAffectedFilesUseInvocationCWDOnASTFallback(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": "package pkg\n\nfunc Run() {}\n",
	})
	subdir := filepath.Join(dir, "pkg")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:            "Run",
		Path:               ".",
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path:    "pkg/other.go",
					Symbols: []repomap.Symbol{{Name: "Other", Kind: "function", Line: 3, EndLine: 3, Signature: "func Other()", Exported: true}},
				},
			},
		},
		ProjectMapStateKey: "symbol-ast-fallback-subdir",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "func Run()") {
		t.Fatalf("expected AST fallback symbol result, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	want := filepath.Join(dir, "pkg", "run.go")
	if !containsAffectedFile(affected, want) {
		t.Fatalf("expected AST fallback affected files to include %s, got %v", want, affected)
	}
	if containsAffectedFile(affected, filepath.Join(dir, "run.go")) {
		t.Fatalf("did not expect wrongly rebased repo-root path in affected files: %v", affected)
	}

	cache.InvalidateSearchCacheForFile(want)
	if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
		t.Fatalf("expected symbol cache entry to be invalidated after editing %s", want)
	}
}

func TestBuildGoSymbolBundleLimitsImplementations(t *testing.T) {
	bundle := buildGoSymbolBundle("Closer", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Closer",
			Kind:    "interface",
			File:    "closer.go",
			Line:    5,
			EndLine: 7,
		},
		Body: []string{
			"5: type Closer interface {",
			"6: \tClose() error",
			"7: }",
		},
		Implementations: []navigation.ImplementationRef{
			{File: "agent.go", Line: 10, Name: "Agent"},
			{File: "service.go", Line: 20, Name: "Service"},
			{File: "worker.go", Line: 30, Name: "Worker"},
			{File: "job.go", Line: 40, Name: "Job"},
			{File: "task.go", Line: 50, Name: "Task"},
		},
	})

	var implSection *SymbolBundleSection
	for i := range bundle.Sections {
		if bundle.Sections[i].Kind == "implementations" {
			implSection = &bundle.Sections[i]
			break
		}
	}
	if implSection == nil {
		t.Fatal("expected implementations section")
	}
	if len(implSection.Items) != goImplementationLimit {
		t.Fatalf("expected %d implementation items, got %d", goImplementationLimit, len(implSection.Items))
	}
	if implSection.Total != 5 {
		t.Fatalf("expected Total=5, got %d", implSection.Total)
	}
	if !implSection.More {
		t.Fatal("expected More=true when implementations are truncated")
	}
}

func TestBuildGoSymbolBundleKeepsAllImplementationsWhenUnderLimit(t *testing.T) {
	bundle := buildGoSymbolBundle("Closer", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Closer",
			Kind:    "interface",
			File:    "closer.go",
			Line:    5,
			EndLine: 7,
		},
		Body: []string{
			"5: type Closer interface {",
			"6: \tClose() error",
			"7: }",
		},
		Implementations: []navigation.ImplementationRef{
			{File: "agent.go", Line: 10, Name: "Agent"},
			{File: "service.go", Line: 20, Name: "Service"},
		},
	})

	var implSection *SymbolBundleSection
	for i := range bundle.Sections {
		if bundle.Sections[i].Kind == "implementations" {
			implSection = &bundle.Sections[i]
			break
		}
	}
	if implSection == nil {
		t.Fatal("expected implementations section")
	}
	if len(implSection.Items) != 2 {
		t.Fatalf("expected 2 implementation items, got %d", len(implSection.Items))
	}
	if implSection.Total != 2 {
		t.Fatalf("expected Total=2, got %d", implSection.Total)
	}
	if implSection.More {
		t.Fatal("expected More=false when implementations are not truncated")
	}
}

func TestResolvePythonSymbol_UsesBundleAsOutputSource(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"models.py": pythonTestSource,
		"views.py":  pythonTestUsageSource,
	})

	result := resolvePythonSymbol("authenticate", SearchOptions{Path: dir, FileType: "py"})
	if result.Status != genericSymbolSingle {
		t.Fatalf("expected genericSymbolSingle, got %s", result.Status)
	}
	if result.Bundle == nil {
		t.Fatal("expected bundle in resolve result")
	}
	want := formatSymbolBundle(result.Bundle, nil, nil)
	if result.Output != want {
		t.Fatalf("expected output to be formatted from the returned bundle\nwant:\n%s\n\ngot:\n%s", want, result.Output)
	}
}

func TestPrioritizeGenericRefs_PrefersRepresentativeFiles(t *testing.T) {
	def := genericSymbolDef{Name: "Close", File: "pkg/agent.go", Line: 10}
	refs := []genericSymbolRef{
		{File: "pkg/agent.go", Line: 20, Snippet: "a.Close()"},
		{File: "pkg/agent.go", Line: 30, Snippet: "a.Close()"},
		{File: "pkg/handler.go", Line: 40, Snippet: "svc.Close()"},
		{File: "pkg/controller.go", Line: 50, Snippet: "svc.Close()"},
		{File: "pkg/ui.go", Line: 60, Snippet: "svc.Close()"},
	}

	selected := prioritizeGenericRefs(def, refs, 3, false)
	if len(selected) != 3 {
		t.Fatalf("expected 3 representative refs, got %d", len(selected))
	}

	seenFiles := make(map[string]bool)
	for _, ref := range selected {
		if seenFiles[ref.File] {
			t.Fatalf("expected file diversity in prioritized refs, got duplicate file %q in %+v", ref.File, selected)
		}
		seenFiles[ref.File] = true
	}
}

func TestSearchCode_MultiPatternGoSymbolBundleDedupe(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}

func run(a *Agent) error {
	return a.Close()
}
`,
		"agent_test.go": `package example

func TestClose() {
	var a Agent
	_ = a.Close()
}
`,
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: `Close,(*Agent).Close,\.Close\(\)`, Path: dir})
	if count := strings.Count(result, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected a single deduped symbol bundle header, got %d:\n%s", count, result)
	}
	for _, want := range []string{"Matched patterns:", "Close", "(*Agent).Close", `\.Close\(\)`} {
		if !strings.Contains(result, want) {
			t.Errorf("expected %q in deduped bundle output, got:\n%s", want, result)
		}
	}
}

func TestSearchCode_MultiPatternGoSymbolBundleDedupeOnWarmSinglePatternCache(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}

func run(a *Agent) error {
	return a.Close()
}
`,
		"agent_test.go": `package example

func TestClose() {
	var a Agent
	_ = a.Close()
}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: `Close,(*Agent).Close,\.Close\(\)`, Path: dir}

	coldResult := ExecuteSearchCodeWithCache(cache, opts)
	if count := strings.Count(coldResult, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected a single deduped symbol bundle header on cold cache, got %d:\n%s", count, coldResult)
	}

	patterns := splitPatterns(opts.Pattern)
	delete(cache.data, buildMultiCacheKey(patterns)+"|"+buildMultiSearchCacheKey(opts, patterns))

	warmResult := ExecuteSearchCodeWithCache(cache, opts)
	if count := strings.Count(warmResult, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected a single deduped symbol bundle header on warm single-pattern cache, got %d:\n%s", count, warmResult)
	}
	for _, want := range []string{"Matched patterns:", "Close", "(*Agent).Close", `\.Close\(\)`} {
		if !strings.Contains(warmResult, want) {
			t.Errorf("expected %q in warm-cache deduped bundle output, got:\n%s", want, warmResult)
		}
	}
}

func TestSearchCode_MultiPatternDedupeUnaffectedByUnrelatedInvalidation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}

func run(a *Agent) error {
	return a.Close()
}
`,
		"agent_test.go": `package example

func TestClose() {
	var a Agent
	_ = a.Close()
}
`,
		"unrelated.go": `package example

func noop() {}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: `Close,(*Agent).Close,\.Close\(\)`, Path: dir}

	coldResult := ExecuteSearchCodeWithCache(cache, opts)
	if count := strings.Count(coldResult, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected a single deduped symbol bundle header on cold cache, got %d:\n%s", count, coldResult)
	}

	patterns := splitPatterns(opts.Pattern)
	delete(cache.data, buildMultiCacheKey(patterns)+"|"+buildMultiSearchCacheKey(opts, patterns))

	cache.InvalidateSearchCacheForFile(filepath.Join(dir, "unrelated.go"))

	warmResult := ExecuteSearchCodeWithCache(cache, opts)
	if count := strings.Count(warmResult, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected deduped symbol bundle after unrelated invalidation, got %d:\n%s", count, warmResult)
	}
}

func TestSinglePatternBundleCacheClearedWithSearchCache(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: "Close", Path: dir}
	ExecuteSearchCodeWithCache(cache, opts)

	if got := countSinglePatternBundleCacheEntries(); got == 0 {
		t.Fatal("expected bundle cache entry before clear")
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got == 0 {
		t.Fatal("expected affected-files cache entry before clear")
	}

	cache.ClearSearchCache()

	if got := countSinglePatternBundleCacheEntries(); got != 0 {
		t.Fatalf("expected bundle cache to be cleared, got %d entries", got)
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got != 0 {
		t.Fatalf("expected affected-files cache to be cleared, got %d entries", got)
	}
}

func TestSinglePatternBundleCacheInvalidatedWithFileInvalidation(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: "Close", Path: dir}
	normOpts, ok := normalizeSearchOptions(opts)
	if !ok {
		t.Fatal("expected normalized options")
	}
	normOpts.CtxLines = 3
	normOpts.TokenBudget = 15000
	cacheKey := buildSearchCacheKeyWithRoute(normOpts, planSearchRoute("Close", normOpts).cacheSignature())

	storeSinglePatternBundle("Close", cacheKey, &SymbolBundle{
		Identity: SymbolBundleIdentity{Language: "go", Query: "Close", Canonical: "go|agent.go|5|Close", DisplayName: "Close", Kind: "function", File: "agent.go", Line: 5, EndLine: 7},
	})
	storeSinglePatternAffectedFiles("Close", cacheKey, []string{filepath.Join(dir, "agent.go")})
	otherKey := buildSearchCacheKeyWithRoute(normOpts, planSearchRoute("OtherClose", normOpts).cacheSignature())
	storeSinglePatternBundle("OtherClose", otherKey, &SymbolBundle{
		Identity: SymbolBundleIdentity{Language: "go", Query: "OtherClose", Canonical: "go|other.go|5|OtherClose", DisplayName: "OtherClose", Kind: "function", File: "other.go", Line: 5, EndLine: 7},
	})
	storeSinglePatternAffectedFiles("OtherClose", otherKey, []string{filepath.Join(dir, "other.go")})
	cache.SetSearch("Close", cacheKey, "cached", []string{filepath.Join(dir, "agent.go")})
	cache.SetSearch("OtherClose", otherKey, "cached", []string{filepath.Join(dir, "other.go")})

	if got := countSinglePatternBundleCacheEntries(); got != 2 {
		t.Fatalf("expected 2 bundle cache entries before invalidate, got %d", got)
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got != 2 {
		t.Fatalf("expected 2 affected-files cache entries before invalidate, got %d", got)
	}

	cache.InvalidateSearchCacheForFile(filepath.Join(dir, "agent.go"))

	if got := countSinglePatternBundleCacheEntries(); got != 1 {
		t.Fatalf("expected targeted bundle cache invalidation, got %d entries", got)
	}
	if loadSinglePatternBundle("OtherClose", otherKey) == nil {
		t.Fatal("expected unrelated bundle cache entry to remain")
	}
	if loadSinglePatternAffectedFiles("Close", cacheKey) != nil {
		t.Fatal("expected targeted affected-files cache entry to be removed")
	}
	if loadSinglePatternAffectedFiles("OtherClose", otherKey) == nil {
		t.Fatal("expected unrelated affected-files cache entry to remain")
	}
}

func TestSinglePatternBundleCacheClearedOnSearchCacheEviction(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	storeSinglePatternBundle("keep", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "keep"}})
	storeSinglePatternBundle("drop", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "drop"}})

	if got := countSinglePatternBundleCacheEntries(); got != 2 {
		t.Fatalf("expected 2 bundle cache entries before eviction, got %d", got)
	}

	tools.NotifySearchCacheEvicted([]string{singlePatternBundleCacheKey("drop", "key")})

	if got := countSinglePatternBundleCacheEntries(); got != 1 {
		t.Fatalf("expected targeted bundle cache eviction, got %d entries", got)
	}
	if loadSinglePatternBundle("keep", "key") == nil {
		t.Fatal("expected unrelated bundle cache entry to remain after eviction")
	}
}

func TestSinglePatternBundleCachePreservesUnrelatedKeysOnTargetedInvalidation(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	storeSinglePatternBundle("keep", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "keep"}})
	storeSinglePatternBundle("drop", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "drop"}})

	tools.NotifySearchCacheInvalidatedKeys([]string{singlePatternBundleCacheKey("drop", "key")})

	if loadSinglePatternBundle("keep", "key") == nil {
		t.Fatal("expected unrelated bundle cache entry to remain after targeted invalidation")
	}
	if loadSinglePatternBundle("drop", "key") != nil {
		t.Fatal("expected targeted bundle cache entry to be removed")
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

func countSinglePatternAffectedFilesCacheEntries() int {
	count := 0
	singlePatternAffectedFilesCache.Range(func(key, value any) bool {
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
