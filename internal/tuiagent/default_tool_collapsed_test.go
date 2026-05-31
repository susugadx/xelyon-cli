package tuiagent

import "testing"

func TestDefaultToolCollapsed_KeepsApplyPatchExpanded(t *testing.T) {
	if collapsed := defaultToolCollapsed("apply_patch", "*** Begin Patch", false); collapsed {
		t.Fatal("apply_patch output should stay expanded")
	}
}

func TestDefaultToolCollapsed_OtherBranches(t *testing.T) {
	cases := []struct {
		name     string
		tool     string
		isError  bool
		wantOpen bool
	}{
		{name: "bash success collapses", tool: "bash", wantOpen: false},
		{name: "gather_context success collapses", tool: "gather_context", wantOpen: false},
		{name: "search success collapses", tool: "search_code", wantOpen: false},
		{name: "web_search success collapses", tool: "web_search", wantOpen: false},
		{name: "errors stay open", tool: "bash", isError: true, wantOpen: true},
		{name: "unknown defaults collapsed", tool: "custom_tool", wantOpen: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			collapsed := defaultToolCollapsed(tc.tool, "result", tc.isError)
			if gotOpen := !collapsed; gotOpen != tc.wantOpen {
				t.Fatalf("defaultToolCollapsed(%q, error=%v) open = %v, want %v", tc.tool, tc.isError, gotOpen, tc.wantOpen)
			}
		})
	}
}
