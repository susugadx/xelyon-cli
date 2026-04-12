package investigation

import "testing"

func TestResolveSurface(t *testing.T) {
	tests := []struct {
		name                      string
		allowLowLevelOverrides    bool
		allowReadFileExactControl bool
		want                      Surface
	}{
		{
			name: "default",
			want: SurfaceDefault,
		},
		{
			name:                      "edit exact control",
			allowReadFileExactControl: true,
			want:                      SurfaceEditExactControl,
		},
		{
			name:                   "legacy overrides win",
			allowLowLevelOverrides: true,
			want:                   SurfaceLegacyOverrides,
		},
		{
			name:                      "legacy overrides win over exact control",
			allowLowLevelOverrides:    true,
			allowReadFileExactControl: true,
			want:                      SurfaceLegacyOverrides,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveSurface(tt.allowLowLevelOverrides, tt.allowReadFileExactControl); got != tt.want {
				t.Fatalf("ResolveSurface(%v, %v) = %q, want %q", tt.allowLowLevelOverrides, tt.allowReadFileExactControl, got, tt.want)
			}
		})
	}
}

func TestSurfaceToolRoles_CurrentContract(t *testing.T) {
	tests := []struct {
		surface  Surface
		toolName string
		want     ToolRole
	}{
		{surface: SurfaceDefault, toolName: ToolGatherContext, want: ToolRoleDefault},
		{surface: SurfaceDefault, toolName: ToolSearchCode, want: ToolRoleHidden},
		{surface: SurfaceDefault, toolName: ToolReadFile, want: ToolRoleHidden},
		{surface: SurfaceDefault, toolName: ToolListDir, want: ToolRoleHidden},
		{surface: SurfaceEditExactControl, toolName: ToolGatherContext, want: ToolRoleDefault},
		{surface: SurfaceEditExactControl, toolName: ToolSearchCode, want: ToolRoleHidden},
		{surface: SurfaceEditExactControl, toolName: ToolReadFile, want: ToolRoleEditExactControl},
		{surface: SurfaceEditExactControl, toolName: ToolListDir, want: ToolRoleHidden},
		{surface: SurfaceLegacyOverrides, toolName: ToolGatherContext, want: ToolRoleDefault},
		{surface: SurfaceLegacyOverrides, toolName: ToolSearchCode, want: ToolRoleLowLevelOverride},
		{surface: SurfaceLegacyOverrides, toolName: ToolReadFile, want: ToolRoleLowLevelOverride},
		{surface: SurfaceLegacyOverrides, toolName: ToolListDir, want: ToolRoleHidden},
	}

	for _, tt := range tests {
		if got := tt.surface.ToolRole(tt.toolName); got != tt.want {
			t.Fatalf("%s/%s role = %q, want %q", tt.surface, tt.toolName, got, tt.want)
		}
	}
}

func TestSurfaceSummary_CurrentContract(t *testing.T) {
	tests := []struct {
		surface Surface
		want    string
	}{
		{surface: SurfaceDefault, want: "gather_context default"},
		{surface: SurfaceEditExactControl, want: "gather_context default + read_file exact-control override"},
		{surface: SurfaceLegacyOverrides, want: "gather_context default + search_code/read_file low-level overrides"},
	}

	for _, tt := range tests {
		if got := tt.surface.Summary(); got != tt.want {
			t.Fatalf("%s summary = %q, want %q", tt.surface, got, tt.want)
		}
	}
}

func TestSurfaceHelpSummary_CoreInvestigationTools(t *testing.T) {
	tests := []struct {
		surface  Surface
		toolName string
		want     string
	}{
		{surface: SurfaceDefault, toolName: ToolGatherContext, want: "Primary/default investigation tool"},
		{surface: SurfaceLegacyOverrides, toolName: ToolSearchCode, want: "Low-level code search override on legacy surfaces when exposed"},
		{surface: SurfaceEditExactControl, toolName: ToolReadFile, want: "Exact file reader for edit/apply_patch exact-control override"},
		{surface: SurfaceLegacyOverrides, toolName: ToolReadFile, want: "Low-level exact file reader override when exposed"},
		{surface: SurfaceDefault, toolName: ToolListDir, want: "Low-level directory listing; current gather_context-first surfaces keep it hidden"},
	}

	for _, tt := range tests {
		got, ok := tt.surface.HelpSummary(tt.toolName)
		if !ok {
			t.Fatalf("%s/%s should have help summary", tt.surface, tt.toolName)
		}
		if got != tt.want {
			t.Fatalf("%s/%s summary = %q, want %q", tt.surface, tt.toolName, got, tt.want)
		}
	}

	if _, ok := SurfaceDefault.HelpSummary("bash"); ok {
		t.Fatal("non-investigation tools should not be owned by investigation.HelpSummary")
	}
}
