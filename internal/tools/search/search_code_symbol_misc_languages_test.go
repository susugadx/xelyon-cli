package search

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

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

func TestSearchCode_JavaScriptAliasSymbolContract(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}

	dir := setupMultiLangDir(t, map[string]string{
		"service.js":    "class UserService {}\n",
		"consumer.jsx":  "const svc = new UserService()\n",
		"component.vue": "<script>\nclass UserService {}\n</script>\n",
	})

	result := ExecuteSearchCode(SearchOptions{
		Pattern:  "UserService",
		Mode:     string(SearchModeSymbol),
		Path:     dir,
		FileType: "javascript",
	})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected javascript alias symbol hit")
	}
	if !strings.Contains(result, "service.js") {
		t.Fatalf("expected javascript alias symbol result to include service.js, got:\n%s", result)
	}
	if strings.Contains(result, ".vue") {
		t.Fatalf("expected javascript alias symbol result to exclude .vue files, got:\n%s", result)
	}
	if strings.Contains(result, "Multiple definitions found") {
		t.Fatalf("expected javascript alias symbol result to avoid widened ambiguous defs, got:\n%s", result)
	}
}

func TestSearchCode_TypeScriptAliasSymbolContract(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}

	dir := setupMultiLangDir(t, map[string]string{
		"App.tsx":    "export function RenderCard() { return <div /> }\n",
		"use.ts":     "RenderCard()\n",
		"legacy.cts": "export function RenderCard() { return null }\n",
		"legacy.mts": "export function RenderCard() { return null }\n",
	})

	result := ExecuteSearchCode(SearchOptions{
		Pattern:  "RenderCard",
		Mode:     string(SearchModeSymbol),
		Path:     dir,
		FileType: "typescript",
	})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected typescript alias symbol hit")
	}
	if !strings.Contains(result, "App.tsx") {
		t.Fatalf("expected typescript alias symbol result to include App.tsx, got:\n%s", result)
	}
	if strings.Contains(result, "legacy.cts") || strings.Contains(result, "legacy.mts") {
		t.Fatalf("expected typescript alias symbol result to exclude .cts/.mts files, got:\n%s", result)
	}
	if strings.Contains(result, "Multiple definitions found") {
		t.Fatalf("expected typescript alias symbol result to avoid widened ambiguous defs, got:\n%s", result)
	}
}

func TestSearchCode_GenericSymbolHonorsGlobFilePatternWithAbsolutePath(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}

	dir := setupMultiLangDir(t, map[string]string{
		"pkg/service.js": "class UserService {}\n",
		"pkg/use.js":     "const svc = new UserService()\n",
		"pkg/other.jsx":  "const UserService = () => null\n",
	})

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "UserService",
		Mode:        string(SearchModeSymbol),
		Path:        dir,
		FilePattern: "pkg/*.js",
	})
	if strings.Contains(result, "No matches found") {
		t.Fatalf("expected absolute-path glob symbol search hit, got:\n%s", result)
	}
	if !strings.Contains(result, "pkg/service.js") {
		t.Fatalf("expected absolute-path glob symbol search to include pkg/service.js, got:\n%s", result)
	}
	if strings.Contains(result, "pkg/other.jsx") {
		t.Fatalf("expected absolute-path glob symbol search to exclude pkg/other.jsx, got:\n%s", result)
	}
}

func TestSearchCode_GenericSymbolHonorsWorkspaceRelativeGlobWithAbsoluteScopedPath(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}

	dir := setupMultiLangDir(t, map[string]string{
		"pkg/service.js": "class UserService {}\n",
		"pkg/use.js":     "const svc = new UserService()\n",
		"pkg/other.jsx":  "const UserService = () => null\n",
	})

	result := ExecuteSearchCode(SearchOptions{
		Pattern:       "UserService",
		Mode:          string(SearchModeSymbol),
		Path:          filepath.Join(dir, "pkg"),
		FilePattern:   "pkg/*.js",
		InvocationCWD: dir,
	})
	if strings.Contains(result, "No matches found") {
		t.Fatalf("expected absolute-scoped glob symbol search hit, got:\n%s", result)
	}
	if !strings.Contains(result, "pkg/service.js") {
		t.Fatalf("expected absolute-scoped glob symbol search to include pkg/service.js, got:\n%s", result)
	}
	if strings.Contains(result, "pkg/other.jsx") {
		t.Fatalf("expected absolute-scoped glob symbol search to exclude pkg/other.jsx, got:\n%s", result)
	}
}
