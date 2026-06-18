package bundlekeys

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestStableGoSymbolKeyNormalizesInputsAndHashesSignature(t *testing.T) {
	got := StableGoSymbolKey("./pkg/../pkg", " Agent ", " Run ", " method ", " func (a *Agent) Run() error ")

	for _, want := range []string{"go|pkg|Agent|Run|method|", "41ebaeebbf96e573"} {
		if !strings.Contains(got, want) {
			t.Fatalf("StableGoSymbolKey = %q, want to contain %q", got, want)
		}
	}
}

func TestCanonicalGoSymbolKeyUsesStableKeyAndCollisionFile(t *testing.T) {
	got := CanonicalGoSymbolKey(navigation.SymbolCandidate{
		StableKey:          "stable",
		StableKeyCollision: true,
		File:               "pkg/run.go",
	})
	if got != "stable|file=pkg/run.go" {
		t.Fatalf("CanonicalGoSymbolKey = %q, want collision file suffix", got)
	}
}

func TestCanonicalSymbolKey(t *testing.T) {
	got := CanonicalSymbolKey("typescript", "src/build.ts", 12, "buildUser")
	if got != "typescript|src/build.ts|12|buildUser" {
		t.Fatalf("CanonicalSymbolKey = %q", got)
	}
}
