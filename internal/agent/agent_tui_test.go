package agent

import (
	"strings"
	"testing"
)

func TestBuildTUIHeader_ContainsGradientLogoAndSubtext(t *testing.T) {
	header := buildTUIHeader()

	checks := []string{
		"██╗",                     // ロゴのブロック文字
		"AI-powered coding agent", // サブテキスト
		"Type /help for commands", // コマンド発見用の案内
	}
	// ANSI コードを除去してテキスト検証
	stripped := stripANSI(header)
	for _, want := range checks {
		if !strings.Contains(stripped, want) {
			t.Fatalf("header should contain %q, got:\n%s", want, stripped)
		}
	}
}

func TestDefaultToolCollapsed_KeepsApplyPatchExpanded(t *testing.T) {
	if collapsed := defaultToolCollapsed("apply_patch", "*** Begin Patch", false); collapsed {
		t.Fatal("apply_patch output should stay expanded")
	}
}

func TestTUIAutoApproveReader_ReadReturnsConfirmation(t *testing.T) {
	var buf [8]byte
	n, err := (tuiAutoApproveReader{}).Read(buf[:])
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got := string(buf[:n]); got != "y\n" {
		t.Fatalf("Read() = %q, want %q", got, "y\n")
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

// stripANSI は ANSI エスケープシーケンスを除去する。
func stripANSI(s string) string {
	var sb strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
