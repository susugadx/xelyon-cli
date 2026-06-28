package mcpnames

import "testing"

func TestMCPSanitizePart(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "ascii", in: "server_name_123", want: "server_name_123"},
		{name: "symbols", in: "server-name.tool name!", want: "server_name_tool_name_"},
		{name: "unicode", in: "ツール名", want: "____"},
		{name: "empty", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizePart(tt.in); got != tt.want {
				t.Fatalf("SanitizePart(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMCPExportedToolName(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		toolName   string
		want       string
	}{
		{name: "server tool join", serverName: "github.server", toolName: "create-issue", want: "mcp_github_server_create_issue"},
		{name: "unicode parts", serverName: "外部", toolName: "検索", want: "mcp______"},
		{name: "empty parts", serverName: "", toolName: "", want: "mcp__"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExportedToolName(tt.serverName, tt.toolName); got != tt.want {
				t.Fatalf("ExportedToolName(%q, %q) = %q, want %q", tt.serverName, tt.toolName, got, tt.want)
			}
		})
	}
}

func TestMCPIsExportedToolName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "exported", in: "mcp_github_get_issue", want: true},
		{name: "minimal prefix", in: "mcp_", want: true},
		{name: "builtin", in: "read_file", want: false},
		{name: "similar", in: "xmcp_github_get_issue", want: false},
		{name: "empty", in: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExportedToolName(tt.in); got != tt.want {
				t.Fatalf("IsExportedToolName(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
