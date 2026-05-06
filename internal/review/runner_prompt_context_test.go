package review

import (
	"strings"
	"testing"
)

func TestBuildReviewProbeResultPromptContextsCapsCommandOutput(t *testing.T) {
	output := strings.Repeat("z", maxReviewProbeResultPromptCommandOutputBytes+1)
	contexts := buildReviewProbeResultPromptContexts([]ReviewProbeResult{
		{
			ID:     "probe-1",
			Mode:   ReviewProbeHostReadOnly,
			Status: ReviewProbeFailed,
			CommandResults: []ReviewProbeCommandResult{
				{
					Command: "go",
					Status:  ReviewProbeFailed,
					Output:  output,
				},
			},
		},
	}, reviewRunnerPromptRedactor{})

	if got, want := len(contexts), 1; got != want {
		t.Fatalf("contexts = %d, want %d", got, want)
	}
	gotCommand := contexts[0].Commands[0]
	if !gotCommand.OutputTruncated {
		t.Fatal("OutputTruncated = false, want true")
	}
	if strings.Count(gotCommand.Output, "z") != maxReviewProbeResultPromptCommandOutputBytes {
		t.Fatalf("prompt output contains %d raw output bytes, want %d", strings.Count(gotCommand.Output, "z"), maxReviewProbeResultPromptCommandOutputBytes)
	}
	if !strings.Contains(gotCommand.Output, reviewProbeResultPromptOutputTruncatedMarker) {
		t.Fatalf("prompt output missing truncation marker:\n%s", gotCommand.Output)
	}
}

func TestBuildReviewProbeResultPromptContextsCapsAggregateCommandOutput(t *testing.T) {
	commandCount := maxReviewProbeResultPromptTotalOutputBytes/maxReviewProbeResultPromptCommandOutputBytes + 2
	commandResults := make([]ReviewProbeCommandResult, 0, commandCount)
	for i := 0; i < commandCount; i++ {
		commandResults = append(commandResults, ReviewProbeCommandResult{
			Command: "printf",
			Status:  ReviewProbePassed,
			Output:  strings.Repeat("z", maxReviewProbeResultPromptCommandOutputBytes),
		})
	}

	contexts := buildReviewProbeResultPromptContexts([]ReviewProbeResult{
		{
			ID:             "probe-1",
			Mode:           ReviewProbeHostReadOnly,
			Status:         ReviewProbePassed,
			CommandResults: commandResults,
		},
	}, reviewRunnerPromptRedactor{})

	rawOutputBytes := 0
	truncatedCommands := 0
	for _, command := range contexts[0].Commands {
		rawOutputBytes += strings.Count(command.Output, "z")
		if command.OutputTruncated {
			truncatedCommands++
		}
	}
	if rawOutputBytes != maxReviewProbeResultPromptTotalOutputBytes {
		t.Fatalf("prompt raw output bytes = %d, want %d", rawOutputBytes, maxReviewProbeResultPromptTotalOutputBytes)
	}
	if truncatedCommands == 0 {
		t.Fatal("truncated commands = 0, want at least one aggregate-truncated command")
	}
	if !strings.Contains(contexts[0].Commands[len(contexts[0].Commands)-1].Output, reviewProbeResultPromptOutputOmittedMarker) {
		t.Fatalf("last command output missing omitted marker:\n%s", contexts[0].Commands[len(contexts[0].Commands)-1].Output)
	}
}
