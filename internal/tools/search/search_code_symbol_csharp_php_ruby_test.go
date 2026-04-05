package search

import (
	"strings"
	"testing"
)

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
