package modelinput

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func TestBuildReviewProbeResultPromptContextsCapsCommandOutput(t *testing.T) {
	output := strings.Repeat("z", maxReviewProbeResultPromptCommandOutputBytes+1)
	contexts := BuildProbeResultPromptContexts([]reviewprobe.ReviewProbeResult{
		{
			ID:     "probe-1",
			Mode:   domain.ReviewProbeHostReadOnly,
			Status: domain.ReviewProbeFailed,
			CommandResults: []reviewprobe.ReviewProbeCommandResult{
				{
					Command: "go",
					Status:  domain.ReviewProbeFailed,
					Output:  output,
				},
			},
		},
	}, nil)

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
	commandResults := make([]reviewprobe.ReviewProbeCommandResult, 0, commandCount)
	for i := 0; i < commandCount; i++ {
		commandResults = append(commandResults, reviewprobe.ReviewProbeCommandResult{
			Command: "printf",
			Status:  domain.ReviewProbePassed,
			Output:  strings.Repeat("z", maxReviewProbeResultPromptCommandOutputBytes),
		})
	}

	contexts := BuildProbeResultPromptContexts([]reviewprobe.ReviewProbeResult{
		{
			ID:             "probe-1",
			Mode:           domain.ReviewProbeHostReadOnly,
			Status:         domain.ReviewProbePassed,
			CommandResults: commandResults,
		},
	}, nil)

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

func TestBuildReviewProbeResultPromptContextsCompactsValidationOutputWhenEnabled(t *testing.T) {
	compactor := &recordingCommandOutputCompactorForTest{
		result: CommandOutputCompactResult{
			Output:     `[omitted old successful validation command output; command="go test ./..."; exit=0; classifier=validation]`,
			Classifier: "validation",
		},
		ok: true,
	}
	contexts := BuildProbeResultPromptContextsWithOptions([]reviewprobe.ReviewProbeResult{
		{
			ID:     "probe-1",
			Mode:   domain.ReviewProbeHostReadOnly,
			Status: domain.ReviewProbePassed,
			CommandResults: []reviewprobe.ReviewProbeCommandResult{
				{
					Command: "go",
					Args:    []string{"test", "./..."},
					Status:  domain.ReviewProbePassed,
					Output: strings.Join([]string{
						"ok   github.com/susugadx/xelyon-cli/internal/review 0.123s",
						"verbose detail that should not be sent after compaction",
					}, "\n"),
				},
			},
		},
	}, nil, ProbeResultPromptContextOptions{CommandOutputCompactor: compactor})

	if got, want := len(contexts), 1; got != want {
		t.Fatalf("contexts = %d, want %d", got, want)
	}
	gotCommand := contexts[0].Commands[0]
	if !gotCommand.OutputCompacted {
		t.Fatal("OutputCompacted = false, want true")
	}
	if gotCommand.OutputCompactClassifier != "validation" {
		t.Fatalf("OutputCompactClassifier = %q, want validation", gotCommand.OutputCompactClassifier)
	}
	if !gotCommand.OutputTruncated {
		t.Fatal("OutputTruncated = false, want compacted output marked")
	}
	if !strings.Contains(gotCommand.Output, `[omitted old successful validation command output; command="go test ./..."; exit=0; classifier=validation`) {
		t.Fatalf("compacted output missing validation placeholder:\n%s", gotCommand.Output)
	}
	if strings.Contains(gotCommand.Output, "verbose detail that should not be sent") {
		t.Fatalf("compacted output leaked raw verbose detail:\n%s", gotCommand.Output)
	}
	if gotCommand.Command != "go" || strings.Join(gotCommand.Args, " ") != "test ./..." {
		t.Fatalf("command metadata = (%q, %#v), want original command and args", gotCommand.Command, gotCommand.Args)
	}
	if compactor.command != "go test ./..." {
		t.Fatalf("compactor command = %q, want quoted command display", compactor.command)
	}
}

func TestBuildReviewProbeResultPromptContextsDoesNotCompactByDefault(t *testing.T) {
	output := strings.Join([]string{
		"ok   github.com/susugadx/xelyon-cli/internal/review 0.123s",
		"verbose detail kept while prompt reduction is dry-run or off",
	}, "\n")

	contexts := BuildProbeResultPromptContexts([]reviewprobe.ReviewProbeResult{
		{
			ID:     "probe-1",
			Mode:   domain.ReviewProbeHostReadOnly,
			Status: domain.ReviewProbePassed,
			CommandResults: []reviewprobe.ReviewProbeCommandResult{
				{
					Command: "go",
					Args:    []string{"test", "./..."},
					Status:  domain.ReviewProbePassed,
					Output:  output,
				},
			},
		},
	}, nil)

	gotCommand := contexts[0].Commands[0]
	if gotCommand.OutputCompacted {
		t.Fatal("OutputCompacted = true, want false")
	}
	if gotCommand.Output != output {
		t.Fatalf("Output = %q, want raw output", gotCommand.Output)
	}
}

func TestBuildReviewProbeResultPromptContextsSkipsCommandCompactorForAbsorptionCandidate(t *testing.T) {
	output := "raw output kept while absorbed probe candidate is only reported"
	compactor := &recordingCommandOutputCompactorForTest{
		result: CommandOutputCompactResult{
			Output:     "[compacted output]",
			Classifier: "validation",
		},
		ok: true,
	}

	contexts := BuildProbeResultPromptContextsWithOptions([]reviewprobe.ReviewProbeResult{
		{
			ID:     "probe-1",
			Mode:   domain.ReviewProbeHostReadOnly,
			Status: domain.ReviewProbePassed,
			CommandResults: []reviewprobe.ReviewProbeCommandResult{
				{
					Command: "go",
					Args:    []string{"test", "./..."},
					Status:  domain.ReviewProbePassed,
					Output:  output,
				},
			},
		},
	}, nil, ProbeResultPromptContextOptions{
		CommandOutputCompactor:      compactor,
		AbsorptionCandidateProbeIDs: map[string]struct{}{"probe-1": {}},
	})

	gotCommand := contexts[0].Commands[0]
	if gotCommand.OutputCompacted {
		t.Fatal("OutputCompacted = true, want false for absorbed candidate dry-run context")
	}
	if gotCommand.Output != output {
		t.Fatalf("Output = %q, want raw output", gotCommand.Output)
	}
	if compactor.command != "" {
		t.Fatalf("compactor command = %q, want not called", compactor.command)
	}
}

func TestBuildReviewProbeResultPromptContextsCompactsFailureOutputAndRedactsSecrets(t *testing.T) {
	compactor := &recordingCommandOutputCompactorForTest{
		result: CommandOutputCompactResult{
			Output:     "[compacted failed output with api_key=[redacted]]",
			Classifier: "validation_failure",
		},
		ok: true,
	}

	contexts := BuildProbeResultPromptContextsWithOptions([]reviewprobe.ReviewProbeResult{
		{
			ID:     "probe-1",
			Mode:   domain.ReviewProbeHostReadOnly,
			Status: domain.ReviewProbeFailed,
			CommandResults: []reviewprobe.ReviewProbeCommandResult{
				{
					Command:  "go",
					Args:     []string{"test", "./..."},
					Status:   domain.ReviewProbeFailed,
					ExitCode: 1,
					Output:   "api_key=super-secret-token",
				},
			},
		},
	}, secretRedactorForModelInputTest{}, ProbeResultPromptContextOptions{CommandOutputCompactor: compactor})

	gotCommand := contexts[0].Commands[0]
	if !gotCommand.OutputCompacted {
		t.Fatal("OutputCompacted = false, want true")
	}
	if gotCommand.OutputCompactClassifier != "validation_failure" {
		t.Fatalf("OutputCompactClassifier = %q, want validation_failure", gotCommand.OutputCompactClassifier)
	}
	if !strings.Contains(gotCommand.Output, "[compacted failed output") {
		t.Fatalf("compacted output missing failure header:\n%s", gotCommand.Output)
	}
	if strings.Contains(compactor.output, "super-secret-token") {
		t.Fatalf("compactor received unredacted output: %q", compactor.output)
	}
}

type recordingCommandOutputCompactorForTest struct {
	result  CommandOutputCompactResult
	ok      bool
	command string
	output  string
}

func (c *recordingCommandOutputCompactorForTest) CompactCommandOutput(command, output string) (CommandOutputCompactResult, bool) {
	c.command = command
	c.output = output
	return c.result, c.ok
}

type secretRedactorForModelInputTest struct{}

func (secretRedactorForModelInputTest) RedactText(text string) string {
	return strings.ReplaceAll(text, "super-secret-token", "[redacted]")
}

func (secretRedactorForModelInputTest) RedactTexts(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, secretRedactorForModelInputTest{}.RedactText(value))
	}
	return out
}

func (secretRedactorForModelInputTest) RedactPath(path string) string {
	return secretRedactorForModelInputTest{}.RedactText(path)
}

func (secretRedactorForModelInputTest) RedactPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, secretRedactorForModelInputTest{}.RedactPath(path))
	}
	return out
}
