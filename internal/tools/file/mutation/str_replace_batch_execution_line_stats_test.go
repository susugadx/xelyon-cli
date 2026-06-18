package mutation

import "testing"

func TestResolveBatchExecutionLineStats_UsesExactWhenStdoutVisible(t *testing.T) {
	edits := []EditEntry{
		{
			OldStr: "a\nb\nc",
			NewStr: "a\nx\nc",
		},
	}

	added, removed := resolveBatchExecutionLineStats("a\nb\nc", "a\nx\nc", edits, false)
	if added != 1 || removed != 1 {
		t.Fatalf("expected exact +1/-1 when stdout is visible, got +%d/-%d", added, removed)
	}
}

func TestResolveBatchExecutionLineStats_UsesFallbackWhenStdoutSuppressed(t *testing.T) {
	edits := []EditEntry{
		{
			OldStr: "a\nb\nc",
			NewStr: "a\nx\nc",
		},
	}

	added, removed := resolveBatchExecutionLineStats("a\nb\nc", "a\nx\nc", edits, true)
	if added != 3 || removed != 3 {
		t.Fatalf("expected fallback +3/-3 when stdout is suppressed, got +%d/-%d", added, removed)
	}
}

func TestResolveBatchExecutionLineStats_ForceExactWhenSuppressedByEnv(t *testing.T) {
	t.Setenv(envBatchExactLineStats, "1")
	edits := []EditEntry{
		{
			OldStr: "a\nb\nc",
			NewStr: "a\nx\nc",
		},
	}

	added, removed := resolveBatchExecutionLineStats("a\nb\nc", "a\nx\nc", edits, true)
	if added != 1 || removed != 1 {
		t.Fatalf("expected forced exact +1/-1 when env is enabled, got +%d/-%d", added, removed)
	}
}
