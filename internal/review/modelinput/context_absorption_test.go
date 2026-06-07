package modelinput

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func TestBuildReviewProbeResultPromptContextsAbsorbsSingleCommand(t *testing.T) {
	absorbedOutput := "absorbed command output should be omitted"
	keptOutput := "neighbor command output must remain visible"

	contexts := BuildProbeResultPromptContextsWithOptions([]reviewprobe.ReviewProbeResult{
		{
			ID:     "probe-1",
			Mode:   domain.ReviewProbeHostReadOnly,
			Status: domain.ReviewProbePassed,
			CommandResults: []reviewprobe.ReviewProbeCommandResult{
				{
					Command: "go",
					Args:    []string{"test", "./internal/review"},
					Status:  domain.ReviewProbePassed,
					Output:  absorbedOutput,
				},
				{
					Command: "git",
					Args:    []string{"diff"},
					Status:  domain.ReviewProbePassed,
					Output:  keptOutput,
				},
			},
		},
	}, nil, ProbeResultPromptContextOptions{
		AbsorbedProbeCommands: map[ProbeCommandResultKey]ProbeResultAbsorptionSummary{
			{ProbeID: "probe-1", CommandIndex: 0}: {
				Summary:        "command[0] reflected by latest report",
				AbsorbedBy:     []string{"scope_coverage.surface.surface-1"},
				RawArtifactRef: "probe_results.json",
			},
		},
	})

	if got, want := len(contexts[0].Commands), 2; got != want {
		t.Fatalf("commands = %d, want %d", got, want)
	}
	absorbedCommand := contexts[0].Commands[0]
	if !absorbedCommand.Absorbed {
		t.Fatal("command[0].Absorbed = false, want true")
	}
	if absorbedCommand.Output != "" || strings.Contains(absorbedCommand.Output, absorbedOutput) {
		t.Fatalf("command[0] output = %q, want omitted absorbed output", absorbedCommand.Output)
	}
	if absorbedCommand.Command != "go" || strings.Join(absorbedCommand.Args, " ") != "test ./internal/review" {
		t.Fatalf("command[0] metadata = (%q, %#v), want original command metadata", absorbedCommand.Command, absorbedCommand.Args)
	}
	if absorbedCommand.AbsorptionSummary == "" || absorbedCommand.RawArtifactRef != "probe_results.json" {
		t.Fatalf("command[0] absorption metadata = %#v, want summary and raw artifact ref", absorbedCommand)
	}

	keptCommand := contexts[0].Commands[1]
	if keptCommand.Absorbed {
		t.Fatal("command[1].Absorbed = true, want false")
	}
	if keptCommand.Output != keptOutput {
		t.Fatalf("command[1].Output = %q, want neighbor output kept", keptCommand.Output)
	}
}

func TestBuildReviewProbeResultPromptContextsSkipsCommandCompactorOnlyForAbsorbedCommandCandidate(t *testing.T) {
	candidateOutput := "raw candidate command output kept while only reported"
	neighborOutput := "neighbor command output may be compacted"
	compactor := &recordingCommandOutputCompactorForTest{
		result: CommandOutputCompactResult{
			Output:     "[compacted neighbor output]",
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
					Command: "rg",
					Args:    []string{"TODO"},
					Status:  domain.ReviewProbePassed,
					Output:  candidateOutput,
				},
				{
					Command: "go",
					Args:    []string{"test", "./..."},
					Status:  domain.ReviewProbePassed,
					Output:  neighborOutput,
				},
			},
		},
	}, nil, ProbeResultPromptContextOptions{
		CommandOutputCompactor: compactor,
		AbsorptionCandidateCommands: map[ProbeCommandResultKey]struct{}{
			{ProbeID: "probe-1", CommandIndex: 0}: {},
		},
	})

	candidateCommand := contexts[0].Commands[0]
	if candidateCommand.OutputCompacted {
		t.Fatal("command[0].OutputCompacted = true, want false for command-level absorption candidate")
	}
	if candidateCommand.Output != candidateOutput {
		t.Fatalf("command[0].Output = %q, want raw candidate output", candidateCommand.Output)
	}

	neighborCommand := contexts[0].Commands[1]
	if !neighborCommand.OutputCompacted {
		t.Fatal("command[1].OutputCompacted = false, want neighbor compaction still enabled")
	}
	if neighborCommand.Output != "[compacted neighbor output]" {
		t.Fatalf("command[1].Output = %q, want compacted neighbor output", neighborCommand.Output)
	}
	if compactor.command != "go test ./..." {
		t.Fatalf("compactor command = %q, want only neighbor command compacted", compactor.command)
	}
}
