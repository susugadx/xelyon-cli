package common

import (
	"sync/atomic"
	"testing"
)

func TestEnterExitQuietMode_ReferenceCount(t *testing.T) {
	saved := atomic.LoadInt32(&quietModeCount)
	defer atomic.StoreInt32(&quietModeCount, saved)

	atomic.StoreInt32(&quietModeCount, 0)

	EnterQuietMode()
	EnterQuietMode()
	if !IsQuietMode() {
		t.Fatal("expected quiet mode enabled after EnterQuietMode")
	}

	ExitQuietMode()
	if !IsQuietMode() {
		t.Fatal("expected quiet mode to remain enabled until all references exit")
	}

	ExitQuietMode()
	if IsQuietMode() {
		t.Fatal("expected quiet mode disabled after final ExitQuietMode")
	}
}

func TestExitQuietMode_AtZero_NoOp(t *testing.T) {
	saved := atomic.LoadInt32(&quietModeCount)
	defer atomic.StoreInt32(&quietModeCount, saved)

	atomic.StoreInt32(&quietModeCount, 0)

	ExitQuietMode()
	if IsQuietMode() {
		t.Fatal("expected quiet mode to stay disabled when exiting at zero")
	}
	if got := atomic.LoadInt32(&quietModeCount); got != 0 {
		t.Fatalf("expected quietModeCount to remain 0, got %d", got)
	}
}
