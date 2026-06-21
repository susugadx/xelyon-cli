package mutation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteFileMutationWorkflow_PreviewTerminalSkipsConfirmAndApply(t *testing.T) {
	setupTestEnvironment(t)

	var steps []string
	result, err := executeFileMutationWorkflow(fileMutationContext{}, common.ConfirmOptions{}, fileMutationWorkflow{
		preview: func() fileMutationResult {
			steps = append(steps, "preview")
			return newErrorMutationResult("stop")
		},
		apply: func() (fileMutationResult, error) {
			steps = append(steps, "apply")
			return newAppliedMutationResult("applied"), nil
		},
	})
	if err != nil {
		t.Fatalf("executeFileMutationWorkflow returned error: %v", err)
	}
	if result.status != fileMutationStatusError || result.message != "stop" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !reflect.DeepEqual(steps, []string{"preview"}) {
		t.Fatalf("unexpected execution order: %#v", steps)
	}
}

func TestExecuteFileMutationWorkflow_ConfirmedRunsApply(t *testing.T) {
	setupTestEnvironment(t)

	var steps []string
	result, err := executeFileMutationWorkflow(fileMutationContext{
		promptIO: testPromptIO(nil, nil),
		absPath:  "target.txt",
	}, common.ConfirmOptions{AutoApprove: true}, fileMutationWorkflow{
		toolName:       "write_file",
		confirmMessage: "apply change?",
		preview: func() fileMutationResult {
			steps = append(steps, "preview")
			return fileMutationResult{}
		},
		apply: func() (fileMutationResult, error) {
			steps = append(steps, "apply")
			return newAppliedMutationResult("applied"), nil
		},
	})
	if err != nil {
		t.Fatalf("executeFileMutationWorkflow returned error: %v", err)
	}
	if result.status != fileMutationStatusApplied || result.message != "applied" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !reflect.DeepEqual(steps, []string{"preview", "apply"}) {
		t.Fatalf("unexpected execution order: %#v", steps)
	}
}

func TestExecuteFileMutationWorkflow_CancelledStopsBeforeApply(t *testing.T) {
	setupTestEnvironment(t)
	setupTestConfirm(t, false)

	var applyCalled bool
	result, err := executeFileMutationWorkflow(fileMutationContext{
		promptIO: testPromptIO(nil, nil),
		absPath:  "target.txt",
	}, common.ConfirmOptions{}, fileMutationWorkflow{
		toolName:       "write_file",
		confirmMessage: "apply change?",
		preview: func() fileMutationResult {
			return fileMutationResult{}
		},
		confirm: mutationConfirmHandlers{
			onCancel: func() fileMutationResult {
				return newCancelledMutationResult("cancelled")
			},
		},
		apply: func() (fileMutationResult, error) {
			applyCalled = true
			return newAppliedMutationResult("applied"), nil
		},
	})
	if err != nil {
		t.Fatalf("executeFileMutationWorkflow returned error: %v", err)
	}
	if applyCalled {
		t.Fatal("apply should not be called when confirmation is rejected")
	}
	if result.status != fileMutationStatusCancelled || result.message != "cancelled" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestExecuteFileMutationWorkflow_ApplyErrorIsReturned(t *testing.T) {
	setupTestEnvironment(t)

	wantErr := errors.New("apply failed")
	result, err := executeFileMutationWorkflow(fileMutationContext{
		promptIO: testPromptIO(nil, nil),
		absPath:  "target.txt",
	}, common.ConfirmOptions{AutoApprove: true}, fileMutationWorkflow{
		toolName:       "write_file",
		confirmMessage: "apply change?",
		apply: func() (fileMutationResult, error) {
			return fileMutationResult{}, wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if result.IsTerminal() {
		t.Fatalf("expected zero result on apply error, got %+v", result)
	}
}
