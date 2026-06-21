package modelinput

import (
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// BuildProbeResultPromptContexts は probe result を redacted prompt DTO へ変換する。
func BuildProbeResultPromptContexts(results []reviewprobe.ReviewProbeResult, redactor Redactor) []ProbeResultPromptContext {
	return BuildProbeResultPromptContextsWithOptions(results, redactor, ProbeResultPromptContextOptions{})
}

// BuildProbeResultPromptContextsWithOptions は probe result を redacted prompt DTO へ変換する。
func BuildProbeResultPromptContextsWithOptions(results []reviewprobe.ReviewProbeResult, redactor Redactor, opts ProbeResultPromptContextOptions) []ProbeResultPromptContext {
	redactor = normalizeRedactor(redactor)
	outputLimiter := newReviewProbeResultPromptOutputLimiter()
	contexts := make([]ProbeResultPromptContext, 0, len(results))
	for _, result := range results {
		if absorption, ok := opts.AbsorbedProbeResults[result.ID]; ok {
			contexts = append(contexts, ProbeResultPromptContext{
				ProbeID:           result.ID,
				Mode:              result.Mode,
				Status:            result.Status,
				MutatedWorktree:   result.MutatedWorktree,
				MutatedFiles:      redactor.RedactPaths(result.MutatedFiles),
				OutputTruncated:   result.OutputTruncated,
				Error:             redactor.RedactText(result.Error),
				Absorbed:          true,
				AbsorptionSummary: redactor.RedactText(absorption.Summary),
				AbsorbedBy:        redactor.RedactTexts(absorption.AbsorbedBy),
				RawArtifactRef:    absorption.RawArtifactRef,
			})
			continue
		}
		probeOpts := opts
		if _, ok := opts.AbsorptionCandidateProbeIDs[result.ID]; ok {
			probeOpts.CommandOutputCompactor = nil
		}
		contexts = append(contexts, ProbeResultPromptContext{
			ProbeID:         result.ID,
			Mode:            result.Mode,
			Status:          result.Status,
			MutatedWorktree: result.MutatedWorktree,
			MutatedFiles:    redactor.RedactPaths(result.MutatedFiles),
			OutputTruncated: result.OutputTruncated,
			Error:           redactor.RedactText(result.Error),
			Commands:        buildReviewProbeCommandResultPromptContexts(result.ID, result.CommandResults, redactor, outputLimiter, probeOpts),
		})
	}
	return contexts
}

func buildReviewProbeCommandResultPromptContexts(probeID string, results []reviewprobe.ReviewProbeCommandResult, redactor Redactor, outputLimiter *reviewProbeResultPromptOutputLimiter, opts ProbeResultPromptContextOptions) []ProbeCommandResultPromptContext {
	contexts := make([]ProbeCommandResultPromptContext, 0, len(results))
	for commandIndex, result := range results {
		redactedCommand := redactor.RedactText(result.Command)
		redactedArgs := redactor.RedactTexts(result.Args)
		key := ProbeCommandResultKey{ProbeID: probeID, CommandIndex: commandIndex}
		if absorption, ok := opts.AbsorbedProbeCommands[key]; ok {
			contexts = append(contexts, ProbeCommandResultPromptContext{
				Command:           redactedCommand,
				Args:              redactedArgs,
				WorkDir:           redactor.RedactPath(result.WorkDir),
				Status:            result.Status,
				ExitCode:          result.ExitCode,
				OutputTruncated:   result.OutputTruncated,
				Error:             "",
				DurationMs:        result.Duration.Milliseconds(),
				Absorbed:          true,
				AbsorptionSummary: redactor.RedactText(absorption.Summary),
				AbsorbedBy:        redactor.RedactTexts(absorption.AbsorbedBy),
				RawArtifactRef:    absorption.RawArtifactRef,
			})
			continue
		}

		redactedOutput := redactor.RedactText(result.Output)
		commandOpts := opts
		if _, ok := opts.AbsorptionCandidateCommands[key]; ok {
			commandOpts.CommandOutputCompactor = nil
		}
		output, outputCompacted, compactClassifier, promptOutputTruncated := reviewProbeCommandPromptOutput(
			redactedCommand,
			redactedArgs,
			redactedOutput,
			outputLimiter,
			commandOpts,
		)
		contexts = append(contexts, ProbeCommandResultPromptContext{
			Command:                 redactedCommand,
			Args:                    redactedArgs,
			WorkDir:                 redactor.RedactPath(result.WorkDir),
			Status:                  result.Status,
			ExitCode:                result.ExitCode,
			Output:                  output,
			OutputTruncated:         result.OutputTruncated || promptOutputTruncated || outputCompacted,
			OutputCompacted:         outputCompacted,
			OutputCompactClassifier: compactClassifier,
			Error:                   redactor.RedactText(result.Error),
			DurationMs:              result.Duration.Milliseconds(),
		})
	}
	return contexts
}

func reviewProbeCommandPromptOutput(command string, args []string, output string, outputLimiter *reviewProbeResultPromptOutputLimiter, opts ProbeResultPromptContextOptions) (string, bool, string, bool) {
	if opts.CommandOutputCompactor != nil {
		if compacted, ok := opts.CommandOutputCompactor.CompactCommandOutput(reviewProbeCommandDisplay(command, args), output); ok {
			limited, truncated := outputLimiter.limit(compacted.Output)
			return limited, true, compacted.Classifier, truncated
		}
	}
	limited, truncated := outputLimiter.limit(output)
	return limited, false, "", truncated
}

func reviewProbeCommandDisplay(command string, args []string) string {
	return reviewprobe.FormatProbeCommand(command, args)
}

func redactReviewProbeSummariesForPrompt(summaries []reviewreport.ReviewProbeSummary, redactor Redactor) []reviewreport.ReviewProbeSummary {
	redactor = normalizeRedactor(redactor)
	redacted := make([]reviewreport.ReviewProbeSummary, 0, len(summaries))
	for _, summary := range summaries {
		redacted = append(redacted, reviewreport.ReviewProbeSummary{
			ProbeID:         summary.ProbeID,
			Mode:            summary.Mode,
			Status:          summary.Status,
			MutatedWorktree: summary.MutatedWorktree,
			MutatedFiles:    redactor.RedactPaths(summary.MutatedFiles),
			OutputTruncated: summary.OutputTruncated,
			Error:           redactor.RedactText(summary.Error),
			Commands:        redactReviewProbeCommandSummariesForPrompt(summary.Commands, redactor),
		})
	}
	return redacted
}

func redactReviewProbeCommandSummariesForPrompt(summaries []reviewreport.ReviewProbeCommandSummary, redactor Redactor) []reviewreport.ReviewProbeCommandSummary {
	redacted := make([]reviewreport.ReviewProbeCommandSummary, 0, len(summaries))
	for _, summary := range summaries {
		redacted = append(redacted, reviewreport.ReviewProbeCommandSummary{
			Command:         redactor.RedactText(summary.Command),
			Args:            redactor.RedactTexts(summary.Args),
			WorkDir:         redactor.RedactPath(summary.WorkDir),
			Status:          summary.Status,
			ExitCode:        summary.ExitCode,
			OutputTruncated: summary.OutputTruncated,
			Error:           redactor.RedactText(summary.Error),
			DurationMs:      summary.DurationMs,
		})
	}
	return redacted
}
