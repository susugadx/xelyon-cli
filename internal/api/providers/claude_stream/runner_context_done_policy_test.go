package claudestream

import (
	"context"
	"errors"
	"testing"
)

func TestRunnerContextDonePolicy_ResolveWithPartialAsSuccess(t *testing.T) {
	policy := newRunnerContextDonePolicy(RunnerOptions{
		CancelMode:    CancelModePartialAsSuccess,
		WarnOnPartial: true,
	})

	resolution := policy.resolve("partial", nil, context.Canceled)
	if resolution.err != nil {
		t.Fatalf("resolve() err = %v, want nil", resolution.err)
	}
	if !resolution.printTrailingNewline {
		t.Fatal("resolve() printTrailingNewline = false, want true")
	}
	if !resolution.warnPartial {
		t.Fatal("resolve() warnPartial = false, want true")
	}
}

func TestRunnerContextDonePolicy_ResolveWithPartialAsError(t *testing.T) {
	policy := newRunnerContextDonePolicy(RunnerOptions{
		CancelMode:    CancelModePartialAsError,
		WarnOnPartial: false,
	})

	resolution := policy.resolve("partial", nil, context.Canceled)
	if !errors.Is(resolution.err, context.Canceled) {
		t.Fatalf("resolve() err = %v, want %v", resolution.err, context.Canceled)
	}
	if !resolution.printTrailingNewline {
		t.Fatal("resolve() printTrailingNewline = false, want true")
	}
	if resolution.warnPartial {
		t.Fatal("resolve() warnPartial = true, want false")
	}
}

func TestRunnerContextDonePolicy_ResolveWithoutPartialPrefersEventErr(t *testing.T) {
	policy := newRunnerContextDonePolicy(RunnerOptions{})
	eventErr := errors.New("event error")

	resolution := policy.resolve("", eventErr, context.Canceled)
	if !errors.Is(resolution.err, eventErr) {
		t.Fatalf("resolve() err = %v, want %v", resolution.err, eventErr)
	}
	if resolution.printTrailingNewline {
		t.Fatal("resolve() printTrailingNewline = true, want false")
	}
	if resolution.warnPartial {
		t.Fatal("resolve() warnPartial = true, want false")
	}
}

func TestRunnerContextDonePolicy_ResolveWithoutPartialFallsBackToContextErr(t *testing.T) {
	policy := newRunnerContextDonePolicy(RunnerOptions{})

	resolution := policy.resolve("", nil, context.Canceled)
	if !errors.Is(resolution.err, context.Canceled) {
		t.Fatalf("resolve() err = %v, want %v", resolution.err, context.Canceled)
	}
	if resolution.printTrailingNewline {
		t.Fatal("resolve() printTrailingNewline = true, want false")
	}
	if resolution.warnPartial {
		t.Fatal("resolve() warnPartial = true, want false")
	}
}
