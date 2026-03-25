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
		"Type /help for commands", // ヘルプ案内
	}
	// ANSI コードを除去してテキスト検証
	stripped := stripANSI(header)
	for _, want := range checks {
		if !strings.Contains(stripped, want) {
			t.Fatalf("header should contain %q, got:\n%s", want, stripped)
		}
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
