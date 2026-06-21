package replaceengine

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/uifileview"
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

			uiAdded, uiRemoved := uifileview.CountDiffLines(strings.Split(tc.oldContent, "\n"), strings.Split(tc.newContent, "\n"))
			if gotAdded != uiAdded || gotRemoved != uiRemoved {
				t.Fatalf("expected parity with uifileview.CountDiffLines (+%d/-%d), got +%d/-%d", uiAdded, uiRemoved, gotAdded, gotRemoved)
			}
		})
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
