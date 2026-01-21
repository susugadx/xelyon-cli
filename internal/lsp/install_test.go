package lsp

import (
	"testing"
)

func TestGetInstallInfo(t *testing.T) {
	tests := []struct {
		serverKey   string
		expectOk    bool
		packageName string
	}{
		// Original 4 languages
		{"go", true, "gopls"},
		{"typescript", true, "vtsls"},
		{"python", true, "pyright"},
		{"rust", true, "rust-analyzer"},
		// Tier 1: Backend languages
		{"java", true, "jdtls"},
		{"c", true, "clangd"},
		{"cpp", true, "clangd"},
		{"ruby", true, "solargraph"},
		{"kotlin", true, "kotlin-language-server"},
		{"csharp", true, "csharp-ls"},
		{"php", true, "intelephense"},
		{"lua", true, "lua-language-server"},
		// Tier 2: Frontend languages
		{"css", true, "vscode-css-language-server"},
		{"html", true, "vscode-html-language-server"},
		{"vue", true, "@vue/language-server"},
		{"svelte", true, "svelte-language-server"},
		// Tier 3: Config/Script languages
		{"yaml", true, "yaml-language-server"},
		{"toml", true, "taplo"},
		{"sql", true, "sqls"},
		{"bash", true, "bash-language-server"},
		{"markdown", true, "marksman"},
		// Unknown
		{"unknown", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.serverKey, func(t *testing.T) {
			info, ok := GetInstallInfo(tt.serverKey)
			if ok != tt.expectOk {
				t.Errorf("Expected ok=%v, got %v", tt.expectOk, ok)
			}
			if ok && info.PackageName != tt.packageName {
				t.Errorf("Expected package %s, got %s", tt.packageName, info.PackageName)
			}
		})
	}
}

func TestGetAllInstallInfos(t *testing.T) {
	infos := GetAllInstallInfos()

	// 23言語分のサーバーが定義されていることを確認
	// 4 (existing) + 11 (Tier 1) + 4 (Tier 2) + 5 (Tier 3) = 24
	// Note: c and cpp share clangd but are separate entries
	if len(infos) < 23 {
		t.Errorf("Expected at least 23 install infos, got %d", len(infos))
	}

	// 各サーバーにコマンドが設定されていることを確認
	// 例外: swift は Xcode/Swift toolchain に含まれるためコマンドなし
	for key, info := range infos {
		if info.ServerKey != key {
			t.Errorf("ServerKey mismatch: expected %s, got %s", key, info.ServerKey)
		}
		if info.PackageName == "" {
			t.Errorf("PackageName is empty for %s", key)
		}
		// Swift は Xcode/Swift toolchain に含まれるためコマンドなしを許可
		if len(info.Commands) == 0 && key != "swift" {
			t.Errorf("No install commands for %s", key)
		}
	}
}

func TestInstallCommands_GoServer(t *testing.T) {
	info, ok := GetInstallInfo("go")
	if !ok {
		t.Fatal("go server info not found")
	}

	if info.ServerKey != "go" {
		t.Errorf("Expected ServerKey 'go', got %s", info.ServerKey)
	}

	if info.PackageName != "gopls" {
		t.Errorf("Expected PackageName 'gopls', got %s", info.PackageName)
	}

	if len(info.Commands) == 0 {
		t.Error("Expected at least one command")
	}

	// コマンドに "gopls" が含まれていることを確認
	found := false
	for _, cmd := range info.Commands {
		if containsString(cmd, "gopls") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected install command to contain 'gopls'")
	}
}

func TestInstallCommands_TypeScriptServer(t *testing.T) {
	info, ok := GetInstallInfo("typescript")
	if !ok {
		t.Fatal("typescript server info not found")
	}

	if info.PackageName != "vtsls" {
		t.Errorf("Expected PackageName 'vtsls', got %s", info.PackageName)
	}

	// コマンドに "npm" が含まれていることを確認
	found := false
	for _, cmd := range info.Commands {
		if containsString(cmd, "npm") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected install command to contain 'npm'")
	}
}

func TestInstallCommands_PythonServer(t *testing.T) {
	info, ok := GetInstallInfo("python")
	if !ok {
		t.Fatal("python server info not found")
	}

	if info.PackageName != "pyright" {
		t.Errorf("Expected PackageName 'pyright', got %s", info.PackageName)
	}

	// 複数のインストールオプションがあることを確認
	if len(info.Commands) < 2 {
		t.Error("Expected multiple install commands for python (pip and npm)")
	}
}

func TestInstallCommands_RustServer(t *testing.T) {
	info, ok := GetInstallInfo("rust")
	if !ok {
		t.Fatal("rust server info not found")
	}

	if info.PackageName != "rust-analyzer" {
		t.Errorf("Expected PackageName 'rust-analyzer', got %s", info.PackageName)
	}

	// コマンドに "rustup" が含まれていることを確認
	found := false
	for _, cmd := range info.Commands {
		if containsString(cmd, "rustup") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected install command to contain 'rustup'")
	}
}

func TestIsServerInstalled(t *testing.T) {
	// 存在しないサーバーキー
	configs := map[string]ServerConfig{}
	if IsServerInstalled("unknown", configs) {
		t.Error("Expected false for unknown server")
	}

	// disabled なサーバー
	configs = map[string]ServerConfig{
		"go": {Command: "gopls", Disabled: true},
	}
	if !IsServerInstalled("go", configs) {
		t.Error("Expected true for disabled server (treated as installed)")
	}

	// 存在しないコマンド
	configs = map[string]ServerConfig{
		"test": {Command: "nonexistent-command-12345"},
	}
	if IsServerInstalled("test", configs) {
		t.Error("Expected false for non-existent command")
	}
}

// containsString は文字列に部分文字列が含まれているかチェック
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
