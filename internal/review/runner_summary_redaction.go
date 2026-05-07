package review

func redactReviewProbeSummaries(summaries []ReviewProbeSummary, redactor reviewRunnerPromptRedactor) []ReviewProbeSummary {
	redacted := make([]ReviewProbeSummary, 0, len(summaries))
	for _, summary := range summaries {
		redacted = append(redacted, ReviewProbeSummary{
			ProbeID:         summary.ProbeID,
			Mode:            summary.Mode,
			Status:          summary.Status,
			MutatedWorktree: summary.MutatedWorktree,
			MutatedFiles:    redactor.redactPaths(summary.MutatedFiles),
			OutputTruncated: summary.OutputTruncated,
			Error:           redactor.redactText(summary.Error),
			Commands:        redactReviewProbeCommandSummaries(summary.Commands, redactor),
		})
	}
	return redacted
}

func redactReviewProbeCommandSummaries(summaries []ReviewProbeCommandSummary, redactor reviewRunnerPromptRedactor) []ReviewProbeCommandSummary {
	redacted := make([]ReviewProbeCommandSummary, 0, len(summaries))
	for _, summary := range summaries {
		redacted = append(redacted, ReviewProbeCommandSummary{
			Command:         redactor.redactText(summary.Command),
			Args:            redactor.redactTexts(summary.Args),
			WorkDir:         redactor.redactPath(summary.WorkDir),
			Status:          summary.Status,
			ExitCode:        summary.ExitCode,
			OutputTruncated: summary.OutputTruncated,
			Error:           redactor.redactText(summary.Error),
			DurationMs:      summary.DurationMs,
		})
	}
	return redacted
}
