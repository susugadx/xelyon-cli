package replaceengine

import "testing"

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
