package search

import (
	"strings"
	"testing"
)

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
