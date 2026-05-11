package search

import (
	"strings"
	"testing"
)

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
		{File: "types.ts", Line: 3, Snippet: "const maybe = UserService?.(db)"},
		{File: "types.ts", Line: 4, Snippet: "const typed = value as UserService"},
		{File: "types.ts", Line: 5, Snippet: "const checked = value satisfies UserService"},
		{File: "types.ts", Line: 6, Snippet: "const list: UserService[] = []"},
		{File: "types.ts", Line: 7, Snippet: "const array: Array<UserService> = []"},
		{File: "types.ts", Line: 8, Snippet: "const record: Record<string, UserService> = {}"},
		{File: "types.ts", Line: 9, Snippet: "export type { UserService } from './service'"},
		{File: "types.ts", Line: 10, Snippet: "export { UserService }"},
		{File: "types.ts", Line: 11, Snippet: "const asserted = <UserService>raw"},
		{File: "callbacks.ts", Line: 1, Snippet: `register("service", UserService)`},
	}

	imports, callers, typeRefs, others := classifyJSRefs(refs, "UserService")

	if len(imports) != 4 {
		t.Errorf("expected 4 imports, got %d: %+v", len(imports), imports)
	}
	if len(callers) != 3 {
		t.Errorf("expected 3 callers (new + direct + optional call), got %d: %+v", len(callers), callers)
	}
	if len(typeRefs) != 9 {
		t.Errorf("expected 9 type refs (: annotation + extends + generic + TS operators + angle assertion), got %d: %+v", len(typeRefs), typeRefs)
	}
	if len(others) != 2 {
		t.Errorf("expected 2 others (comment + comma-separated value ref), got %d: %+v", len(others), others)
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
