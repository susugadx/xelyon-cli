package search

import (
	"strings"
	"testing"
)

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
