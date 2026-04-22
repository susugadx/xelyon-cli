package file

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestResolveBatchDiffLineStats(t *testing.T) {
	cases := []struct {
		name        string
		oldContent  string
		newContent  string
		wantAdded   int
		wantRemoved int
	}{
		{
			name:        "no changes",
			oldContent:  "alpha\nbeta",
			newContent:  "alpha\nbeta",
			wantAdded:   0,
			wantRemoved: 0,
		},
		{
			name:        "single line replacement",
			oldContent:  "x",
			newContent:  "z",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "line insertion",
			oldContent:  "x",
			newContent:  "x\ny",
			wantAdded:   1,
			wantRemoved: 0,
		},
		{
			name:        "line deletion",
			oldContent:  "x\ny",
			newContent:  "x",
			wantAdded:   0,
			wantRemoved: 1,
		},
		{
			name:        "reorder within replaced block",
			oldContent:  "A\nB",
			newContent:  "B\nA",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "batch intermediate change cancels out",
			oldContent:  "x",
			newContent:  "z",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "duplicate lines keep maximal common subsequence",
			oldContent:  "A\nB\nA",
			newContent:  "A\nA",
			wantAdded:   0,
			wantRemoved: 1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotAdded, gotRemoved, exact := resolveBatchDiffLineStats(tc.oldContent, tc.newContent)
			if !exact {
				t.Fatal("expected exact stats, but fallback was used")
			}
			if gotAdded != tc.wantAdded || gotRemoved != tc.wantRemoved {
				t.Fatalf("expected +%d/-%d, got +%d/-%d", tc.wantAdded, tc.wantRemoved, gotAdded, gotRemoved)
			}

			uiAdded, uiRemoved := ui.CountDiffLines(strings.Split(tc.oldContent, "\n"), strings.Split(tc.newContent, "\n"))
			if gotAdded != uiAdded || gotRemoved != uiRemoved {
				t.Fatalf("expected parity with ui.CountDiffLines (+%d/-%d), got +%d/-%d", uiAdded, uiRemoved, gotAdded, gotRemoved)
			}
		})
	}
}

func TestResolveDynamicMyersStepLimit_GrowsWithSpan(t *testing.T) {
	originalMin := myersMinDiagonalStepLimit
	originalMax := myersDiagonalStepLimit
	originalDivisor := myersRewriteBudgetDivisor
	originalFloor := myersRewriteBudgetFloor
	myersMinDiagonalStepLimit = 100
	myersDiagonalStepLimit = 1_000_000
	myersRewriteBudgetDivisor = 6
	myersRewriteBudgetFloor = 32
	t.Cleanup(func() {
		myersMinDiagonalStepLimit = originalMin
		myersDiagonalStepLimit = originalMax
		myersRewriteBudgetDivisor = originalDivisor
		myersRewriteBudgetFloor = originalFloor
	})
	tuning := resolveMyersDiffTuning()

	small := resolveDynamicMyersStepLimit(40, 40, tuning)
	medium := resolveDynamicMyersStepLimit(2000, 2000, tuning)
	if medium <= small {
		t.Fatalf("expected dynamic step limit to grow with span: small=%d medium=%d", small, medium)
	}
}

func TestResolveDynamicMyersStepLimit_GrowsWithImbalance(t *testing.T) {
	originalMin := myersMinDiagonalStepLimit
	originalMax := myersDiagonalStepLimit
	originalDivisor := myersRewriteBudgetDivisor
	originalFloor := myersRewriteBudgetFloor
	myersMinDiagonalStepLimit = 100
	myersDiagonalStepLimit = 1_000_000
	myersRewriteBudgetDivisor = 6
	myersRewriteBudgetFloor = 32
	t.Cleanup(func() {
		myersMinDiagonalStepLimit = originalMin
		myersDiagonalStepLimit = originalMax
		myersRewriteBudgetDivisor = originalDivisor
		myersRewriteBudgetFloor = originalFloor
	})
	tuning := resolveMyersDiffTuning()

	balanced := resolveDynamicMyersStepLimit(1000, 1000, tuning)
	imbalanced := resolveDynamicMyersStepLimit(1000, 1800, tuning)
	if imbalanced <= balanced {
		t.Fatalf("expected dynamic step limit to grow with imbalance: balanced=%d imbalanced=%d", balanced, imbalanced)
	}
}

func TestResolveMyersDiffTuning_UsesEnvironmentOverrides(t *testing.T) {
	t.Setenv(envMyersDiagonalStepLimit, "1234")
	t.Setenv(envMyersLineSpanLimit, "5678")
	t.Setenv(envMyersMinStepLimit, "90")
	t.Setenv(envMyersRewriteDivisor, "4")
	t.Setenv(envMyersRewriteFloor, "12")

	tuning := resolveMyersDiffTuning()
	if tuning.diagonalStepLimit != 1234 {
		t.Fatalf("expected diagonal step limit 1234, got %d", tuning.diagonalStepLimit)
	}
	if tuning.lineSpanLimit != 5678 {
		t.Fatalf("expected line span limit 5678, got %d", tuning.lineSpanLimit)
	}
	if tuning.minDiagonalStep != 90 {
		t.Fatalf("expected min step limit 90, got %d", tuning.minDiagonalStep)
	}
	if tuning.rewriteBudgetDivisor != 4 {
		t.Fatalf("expected rewrite divisor 4, got %d", tuning.rewriteBudgetDivisor)
	}
	if tuning.rewriteBudgetFloor != 12 {
		t.Fatalf("expected rewrite floor 12, got %d", tuning.rewriteBudgetFloor)
	}
}

func TestResolveMyersDiffTuning_InvalidEnvironmentFallsBackToDefaults(t *testing.T) {
	t.Setenv(envMyersDiagonalStepLimit, "x")
	t.Setenv(envMyersLineSpanLimit, "y")
	t.Setenv(envMyersMinStepLimit, "z")
	t.Setenv(envMyersRewriteDivisor, "w")
	t.Setenv(envMyersRewriteFloor, "v")

	tuning := resolveMyersDiffTuning()
	if tuning.diagonalStepLimit != myersDiagonalStepLimit {
		t.Fatalf("expected diagonal step default %d, got %d", myersDiagonalStepLimit, tuning.diagonalStepLimit)
	}
	if tuning.lineSpanLimit != myersLineSpanLimit {
		t.Fatalf("expected line span default %d, got %d", myersLineSpanLimit, tuning.lineSpanLimit)
	}
	if tuning.minDiagonalStep != myersMinDiagonalStepLimit {
		t.Fatalf("expected min step default %d, got %d", myersMinDiagonalStepLimit, tuning.minDiagonalStep)
	}
	if tuning.rewriteBudgetDivisor != myersRewriteBudgetDivisor {
		t.Fatalf("expected rewrite divisor default %d, got %d", myersRewriteBudgetDivisor, tuning.rewriteBudgetDivisor)
	}
	if tuning.rewriteBudgetFloor != myersRewriteBudgetFloor {
		t.Fatalf("expected rewrite floor default %d, got %d", myersRewriteBudgetFloor, tuning.rewriteBudgetFloor)
	}
}

func TestResolveMyersDiffTuning_NormalizesUnsafeEnvironmentOverrides(t *testing.T) {
	t.Setenv(envMyersDiagonalStepLimit, "0")
	t.Setenv(envMyersLineSpanLimit, "-10")
	t.Setenv(envMyersMinStepLimit, "0")
	t.Setenv(envMyersRewriteDivisor, "0")
	t.Setenv(envMyersRewriteFloor, "-5")

	tuning := resolveMyersDiffTuning()
	if tuning.diagonalStepLimit != myersDiagonalStepLimit {
		t.Fatalf("expected diagonal step fallback %d, got %d", myersDiagonalStepLimit, tuning.diagonalStepLimit)
	}
	if tuning.lineSpanLimit != myersLineSpanLimit {
		t.Fatalf("expected line span fallback %d, got %d", myersLineSpanLimit, tuning.lineSpanLimit)
	}
	if tuning.minDiagonalStep != 1 {
		t.Fatalf("expected min step clamp 1, got %d", tuning.minDiagonalStep)
	}
	if tuning.rewriteBudgetDivisor != 1 {
		t.Fatalf("expected rewrite divisor clamp 1, got %d", tuning.rewriteBudgetDivisor)
	}
	if tuning.rewriteBudgetFloor != 0 {
		t.Fatalf("expected rewrite floor clamp 0, got %d", tuning.rewriteBudgetFloor)
	}
}

func TestResolveMyersDiffTuning_ClampsDiagonalLimitToMinimum(t *testing.T) {
	t.Setenv(envMyersDiagonalStepLimit, "50")
	t.Setenv(envMyersMinStepLimit, "120")

	tuning := resolveMyersDiffTuning()
	if tuning.diagonalStepLimit != 120 {
		t.Fatalf("expected diagonal step to clamp to min 120, got %d", tuning.diagonalStepLimit)
	}
}

func TestResolveBatchDiffLineStats_FallbackWhenMyersLimitExceeded(t *testing.T) {
	originalLimit := myersDiagonalStepLimit
	originalMin := myersMinDiagonalStepLimit
	myersDiagonalStepLimit = 1
	myersMinDiagonalStepLimit = 1
	t.Cleanup(func() {
		myersDiagonalStepLimit = originalLimit
		myersMinDiagonalStepLimit = originalMin
	})

	_, _, exact := resolveBatchDiffLineStats("a\nx\nb\ny\nc", "a\np\nb\nq\nc")
	if exact {
		t.Fatal("expected fallback when myers limit is exceeded")
	}
}

func TestResolveBatchDiffLineStats_UsesTrimmedLinesForSpanLimit(t *testing.T) {
	originalLimit := myersLineSpanLimit
	myersLineSpanLimit = 20
	t.Cleanup(func() {
		myersLineSpanLimit = originalLimit
	})

	oldLines := make([]string, 0, 101)
	newLines := make([]string, 0, 101)
	for i := 0; i < 50; i++ {
		oldLines = append(oldLines, "same")
		newLines = append(newLines, "same")
	}
	oldLines = append(oldLines, "TARGET_OLD")
	newLines = append(newLines, "TARGET_NEW")
	for i := 0; i < 50; i++ {
		oldLines = append(oldLines, "same")
		newLines = append(newLines, "same")
	}

	added, removed, exact := resolveBatchDiffLineStats(strings.Join(oldLines, "\n"), strings.Join(newLines, "\n"))
	if !exact {
		t.Fatal("expected exact stats with trimmed shared context")
	}
	if added != 1 || removed != 1 {
		t.Fatalf("expected +1/-1, got +%d/-%d", added, removed)
	}
}

func TestResolveBatchDiffLineStats_FallbackWhenTrimmedSpanStillLarge(t *testing.T) {
	originalLimit := myersLineSpanLimit
	myersLineSpanLimit = 10
	t.Cleanup(func() {
		myersLineSpanLimit = originalLimit
	})

	oldLines := make([]string, 0, 30)
	newLines := make([]string, 0, 30)
	for i := 0; i < 30; i++ {
		oldLines = append(oldLines, "old_"+strings.Repeat("x", i+1))
		newLines = append(newLines, "new_"+strings.Repeat("y", i+1))
	}

	_, _, exact := resolveBatchDiffLineStats(strings.Join(oldLines, "\n"), strings.Join(newLines, "\n"))
	if exact {
		t.Fatal("expected fallback when trimmed span exceeds limit")
	}
}
