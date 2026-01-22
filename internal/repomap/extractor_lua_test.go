//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExtractLuaLocalFunction(t *testing.T) {
	content := `local function greet(name)
    print("Hello, " .. name)
end

local function add(a, b)
    return a + b
end
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "utils.lua", content)
	testFile := filepath.Join(tmpDir, "utils.lua")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをマップに集約
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// local function greet
	if sym, ok := symbolMap["greet"]; !ok {
		t.Error("Expected to find 'greet' function")
	} else {
		if sym.Kind != "function" {
			t.Errorf("Expected kind 'function', got '%s'", sym.Kind)
		}
		if !strings.Contains(sym.Signature, "local") {
			t.Errorf("Expected signature to contain 'local', got '%s'", sym.Signature)
		}
		if !strings.Contains(sym.Signature, "(name)") {
			t.Errorf("Expected signature to contain '(name)', got '%s'", sym.Signature)
		}
	}

	// local function add
	if sym, ok := symbolMap["add"]; !ok {
		t.Error("Expected to find 'add' function")
	} else {
		if !strings.Contains(sym.Signature, "(a, b)") {
			t.Errorf("Expected signature to contain '(a, b)', got '%s'", sym.Signature)
		}
	}
}

func TestExtractLuaGlobalFunction(t *testing.T) {
	content := `function globalFunc()
    return 42
end

function calculate(x, y, z)
    return x + y + z
end
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "global.lua", content)
	testFile := filepath.Join(tmpDir, "global.lua")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルをマップに集約
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// function globalFunc（localなし）
	if sym, ok := symbolMap["globalFunc"]; !ok {
		t.Error("Expected to find 'globalFunc' function")
	} else {
		// globalなのでlocalが含まれない
		if strings.Contains(sym.Signature, "local") {
			t.Errorf("Expected signature NOT to contain 'local', got '%s'", sym.Signature)
		}
	}

	// function calculate
	if _, ok := symbolMap["calculate"]; !ok {
		t.Error("Expected to find 'calculate' function")
	}
}

func TestExtractLuaModuleFunction(t *testing.T) {
	content := `local M = {}

function M.init()
    M.data = {}
end

function M.add(item)
    table.insert(M.data, item)
end

function M.get(index)
    return M.data[index]
end

return M
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "module.lua", content)
	testFile := filepath.Join(tmpDir, "module.lua")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// モジュール関数を探す
	moduleFuncs := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if strings.HasPrefix(sym.Name, "M.") {
			moduleFuncs[sym.Name] = sym
		}
	}

	// M.init, M.add, M.get が抽出されるか
	expectedFuncs := []string{"M.init", "M.add", "M.get"}
	for _, name := range expectedFuncs {
		if _, ok := moduleFuncs[name]; !ok {
			t.Errorf("Expected to find '%s' module function", name)
		}
	}
}

func TestExtractLuaNeovimConfig(t *testing.T) {
	// Neovim設定ファイルのパターン
	content := `local opts = { noremap = true, silent = true }

local function map(mode, lhs, rhs)
    vim.keymap.set(mode, lhs, rhs, opts)
end

function Setup()
    map('n', '<leader>ff', '<cmd>Telescope find_files<cr>')
    map('n', '<leader>fg', '<cmd>Telescope live_grep<cr>')
end

return {
    setup = Setup,
    map = map,
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "keymaps.lua", content)
	testFile := filepath.Join(tmpDir, "keymaps.lua")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数を探す
	functions := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "function" {
			functions[sym.Name] = sym
		}
	}

	// map（local）とSetup（global）が抽出されるか
	if _, ok := functions["map"]; !ok {
		t.Error("Expected to find 'map' function")
	}
	if _, ok := functions["Setup"]; !ok {
		t.Error("Expected to find 'Setup' function")
	}
}

func TestLuaIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"init.lua", true},
		{"config/keymaps.lua", true},
		{"plugin.lua", true},
		{"main.js", true},   // JSはサポート
		{"test.txt", false}, // txtはサポート外
	}

	for _, tt := range tests {
		result := IsSupportedFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsSupportedFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}
