package mutation

import "testing"

func TestResolveBatchDiffLineStatsPolicy_DisablesExactWhenSuppressedByDefault(t *testing.T) {
	policy := resolveBatchDiffLineStatsPolicy(true)
	if policy.resolveExact {
		t.Fatal("expected exact stats to be disabled when stdout is suppressed by default")
	}
}

func TestResolveBatchDiffLineStatsPolicy_ForceExactWhenEnvironmentEnabled(t *testing.T) {
	t.Setenv(envBatchExactLineStats, "1")
	policy := resolveBatchDiffLineStatsPolicy(true)
	if !policy.resolveExact {
		t.Fatal("expected exact stats to be enabled when force env is set")
	}
}

func TestResolveBatchDiffLineStatsWithPolicy_SkipsExactWhenDisabled(t *testing.T) {
	policy := batchDiffLineStatsPolicy{
		resolveExact: false,
		tuning:       resolveMyersDiffTuning(),
	}

	_, _, exact := resolveBatchDiffLineStatsWithPolicy("a\nx", "a\ny", policy)
	if exact {
		t.Fatal("expected exact=false when policy disables exact stats")
	}
}
