package common

import (
	"errors"
	"testing"
)

func TestIsRipgrepAvailable_Cached(t *testing.T) {
	ResetRipgrepAvailabilityForTest()
	t.Cleanup(ResetRipgrepAvailabilityForTest)

	calls := 0
	rgLookPath = func(file string) (string, error) {
		calls++
		if file != "rg" {
			t.Fatalf("unexpected lookup target: %s", file)
		}
		return "/usr/bin/rg", nil
	}

	if !IsRipgrepAvailable() {
		t.Fatal("IsRipgrepAvailable() = false, want true")
	}
	if !IsRipgrepAvailable() {
		t.Fatal("second IsRipgrepAvailable() = false, want true")
	}
	if calls != 1 {
		t.Fatalf("ripgrep lookup called %d times, want 1", calls)
	}
}

func TestIsRipgrepAvailable_ReturnsConsistentResult(t *testing.T) {
	ResetRipgrepAvailabilityForTest()
	t.Cleanup(ResetRipgrepAvailabilityForTest)

	calls := 0
	rgLookPath = func(file string) (string, error) {
		calls++
		return "", errors.New("not found")
	}

	first := IsRipgrepAvailable()
	second := IsRipgrepAvailable()
	if first {
		t.Fatal("first IsRipgrepAvailable() = true, want false")
	}
	if second {
		t.Fatal("second IsRipgrepAvailable() = true, want false")
	}
	if calls != 1 {
		t.Fatalf("ripgrep lookup called %d times, want 1", calls)
	}
}

func TestRipgrepPath_ReturnsFullPath(t *testing.T) {
	ResetRipgrepAvailabilityForTest()
	t.Cleanup(ResetRipgrepAvailabilityForTest)

	rgLookPath = func(file string) (string, error) {
		return "/usr/bin/rg", nil
	}

	if !IsRipgrepAvailable() {
		t.Fatal("IsRipgrepAvailable() = false, want true")
	}
	if got := RipgrepPath(); got != "/usr/bin/rg" {
		t.Fatalf("RipgrepPath() = %q, want /usr/bin/rg", got)
	}
}

func TestRipgrepPath_FallbackWhenNotFound(t *testing.T) {
	ResetRipgrepAvailabilityForTest()
	t.Cleanup(ResetRipgrepAvailabilityForTest)

	rgLookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	if IsRipgrepAvailable() {
		t.Fatal("IsRipgrepAvailable() = true, want false")
	}
	if got := RipgrepPath(); got != "rg" {
		t.Fatalf("RipgrepPath() = %q, want rg", got)
	}
}
