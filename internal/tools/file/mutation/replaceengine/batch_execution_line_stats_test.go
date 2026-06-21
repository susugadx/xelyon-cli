package replaceengine

import "testing"

func TestResolveBatchExecutionLineStats_UsesExactWhenStdoutVisible(t *testing.T) {
	edits := []Edit{
		{
			OldStr: "a\nb\nc",
			NewStr: "a\nx\nc",
		},
	}

	stats := ResolveBatchExecutionLineStats("a\nb\nc", "a\nx\nc", edits, false)
	if stats.LinesAdded != 1 || stats.LinesRemoved != 1 {
		t.Fatalf("expected exact +1/-1 when stdout is visible, got +%d/-%d", stats.LinesAdded, stats.LinesRemoved)
	}
}

func TestResolveBatchExecutionLineStats_UsesFallbackWhenStdoutSuppressed(t *testing.T) {
	edits := []Edit{
		{
			OldStr: "a\nb\nc",
			NewStr: "a\nx\nc",
		},
	}

	stats := ResolveBatchExecutionLineStats("a\nb\nc", "a\nx\nc", edits, true)
	if stats.LinesAdded != 3 || stats.LinesRemoved != 3 {
		t.Fatalf("expected fallback +3/-3 when stdout is suppressed, got +%d/-%d", stats.LinesAdded, stats.LinesRemoved)
	}
}

func TestResolveBatchExecutionLineStats_ForceExactWhenSuppressedByEnv(t *testing.T) {
	t.Setenv(envBatchExactLineStats, "1")
	edits := []Edit{
		{
			OldStr: "a\nb\nc",
			NewStr: "a\nx\nc",
		},
	}

	stats := ResolveBatchExecutionLineStats("a\nb\nc", "a\nx\nc", edits, true)
	if stats.LinesAdded != 1 || stats.LinesRemoved != 1 {
		t.Fatalf("expected forced exact +1/-1 when env is enabled, got +%d/-%d", stats.LinesAdded, stats.LinesRemoved)
	}
}
