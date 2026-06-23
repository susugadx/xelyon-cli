package finalcheck

import "testing"

func TestRetryState_StallsOnRepeatedFailureWithoutChanges(t *testing.T) {
	state := &RetryState{}
	result := RunResult{
		NeedsContinue:      true,
		Feedback:           "Final check failed",
		FailureFingerprint: "same failure",
	}
	progressFingerprint := "change-a"

	if stalled := state.RecordFailure(result, progressFingerprint); stalled {
		t.Fatal("first repeated failure should not stall")
	}
	if stalled := state.RecordFailure(result, progressFingerprint); !stalled {
		t.Fatal("second identical failure without changes should stall")
	}
}

func TestRetryState_ResetsWhenChangedFilesAdvance(t *testing.T) {
	state := &RetryState{}
	result := RunResult{
		NeedsContinue:      true,
		Feedback:           "Final check failed",
		FailureFingerprint: "same failure",
	}

	if stalled := state.RecordFailure(result, "change-a"); stalled {
		t.Fatal("first failure should not stall")
	}
	if stalled := state.RecordFailure(result, "change-b"); stalled {
		t.Fatal("changed fingerprint advance should reset no-progress detection")
	}
}

func TestRetryState_DoesNotStallWithoutProgressFingerprint(t *testing.T) {
	state := &RetryState{}
	result := RunResult{
		NeedsContinue:      true,
		Feedback:           "Final check failed",
		FailureFingerprint: "same failure",
	}

	if stalled := state.RecordFailure(result, ""); stalled {
		t.Fatal("first failure without progress fingerprint should not stall")
	}
	if stalled := state.RecordFailure(result, ""); stalled {
		t.Fatal("unknown progress should not be treated as no progress")
	}
}

func TestRetryState_BlankFailureFingerprintResets(t *testing.T) {
	state := &RetryState{}
	result := RunResult{FailureFingerprint: "same failure"}
	if state.RecordFailure(result, "change-a") {
		t.Fatal("first failure should not stall")
	}
	if !state.RecordFailure(result, "change-a") {
		t.Fatal("second identical failure should stall")
	}
	if state.RecordFailure(RunResult{}, "change-a") {
		t.Fatal("blank failure fingerprint should reset instead of stall")
	}
	if state.RecordFailure(result, "change-a") {
		t.Fatal("first failure after reset should not stall")
	}
}
