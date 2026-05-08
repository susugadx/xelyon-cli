package review

import "unicode/utf8"

const (
	maxReviewProbeResultPromptCommandOutputBytes = 8 * 1024
	maxReviewProbeResultPromptTotalOutputBytes   = 64 * 1024

	reviewProbeResultPromptOutputTruncatedMarker = "\n[probe output truncated for review prompt]"
	reviewProbeResultPromptOutputOmittedMarker   = "[probe output omitted because review prompt output budget was exhausted]"
)

// reviewProbeResultPromptContext は Pass2 prompt 専用の probe result DTO。
// Report schema 用 summary とは別に、model 判断に必要な command output を保持する。
type reviewProbeResultPromptContext struct {
	ProbeID         string                                  `json:"probe_id"`
	Mode            ReviewProbeMode                         `json:"mode"`
	Status          ReviewProbeStatus                       `json:"status"`
	MutatedWorktree bool                                    `json:"mutated_worktree"`
	MutatedFiles    []string                                `json:"mutated_files"`
	OutputTruncated bool                                    `json:"output_truncated"`
	Error           string                                  `json:"error"`
	Commands        []reviewProbeCommandResultPromptContext `json:"commands"`
}

type reviewProbeCommandResultPromptContext struct {
	Command         string            `json:"command"`
	Args            []string          `json:"args"`
	WorkDir         string            `json:"work_dir"`
	Status          ReviewProbeStatus `json:"status"`
	ExitCode        int               `json:"exit_code"`
	Output          string            `json:"output"`
	OutputTruncated bool              `json:"output_truncated"`
	Error           string            `json:"error"`
	DurationMs      int64             `json:"duration_ms"`
}

func buildReviewProbeResultPromptContexts(results []ReviewProbeResult, redactor reviewRunnerPromptRedactor) []reviewProbeResultPromptContext {
	outputLimiter := newReviewProbeResultPromptOutputLimiter()
	contexts := make([]reviewProbeResultPromptContext, 0, len(results))
	for _, result := range results {
		contexts = append(contexts, reviewProbeResultPromptContext{
			ProbeID:         result.ID,
			Mode:            result.Mode,
			Status:          result.Status,
			MutatedWorktree: result.MutatedWorktree,
			MutatedFiles:    redactor.redactPaths(result.MutatedFiles),
			OutputTruncated: result.OutputTruncated,
			Error:           redactor.redactText(result.Error),
			Commands:        buildReviewProbeCommandResultPromptContexts(result.CommandResults, redactor, outputLimiter),
		})
	}
	return contexts
}

func buildReviewProbeCommandResultPromptContexts(results []ReviewProbeCommandResult, redactor reviewRunnerPromptRedactor, outputLimiter *reviewProbeResultPromptOutputLimiter) []reviewProbeCommandResultPromptContext {
	contexts := make([]reviewProbeCommandResultPromptContext, 0, len(results))
	for _, result := range results {
		output, promptOutputTruncated := outputLimiter.limit(redactor.redactText(result.Output))
		contexts = append(contexts, reviewProbeCommandResultPromptContext{
			Command:         redactor.redactText(result.Command),
			Args:            redactor.redactTexts(result.Args),
			WorkDir:         redactor.redactPath(result.WorkDir),
			Status:          result.Status,
			ExitCode:        result.ExitCode,
			Output:          output,
			OutputTruncated: result.OutputTruncated || promptOutputTruncated,
			Error:           redactor.redactText(result.Error),
			DurationMs:      result.Duration.Milliseconds(),
		})
	}
	return contexts
}

func redactReviewProbeSummariesForPrompt(summaries []ReviewProbeSummary, redactor reviewRunnerPromptRedactor) []ReviewProbeSummary {
	return redactReviewProbeSummaries(summaries, redactor)
}

type reviewProbeResultPromptOutputLimiter struct {
	remainingBytes int
}

func newReviewProbeResultPromptOutputLimiter() *reviewProbeResultPromptOutputLimiter {
	return &reviewProbeResultPromptOutputLimiter{remainingBytes: maxReviewProbeResultPromptTotalOutputBytes}
}

func (l *reviewProbeResultPromptOutputLimiter) limit(output string) (string, bool) {
	if output == "" {
		return "", false
	}
	if l == nil {
		return output, false
	}
	if l.remainingBytes <= 0 {
		return reviewProbeResultPromptOutputOmittedMarker, true
	}

	limit := min(maxReviewProbeResultPromptCommandOutputBytes, l.remainingBytes)
	if len(output) <= limit {
		l.remainingBytes -= len(output)
		return output, false
	}

	truncated := truncateReviewProbeResultPromptOutput(output, limit)
	l.remainingBytes -= len(truncated)
	return truncated + reviewProbeResultPromptOutputTruncatedMarker, true
}

func truncateReviewProbeResultPromptOutput(output string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(output) <= limit {
		return output
	}

	truncated := output[:limit]
	for !utf8.ValidString(truncated) {
		_, size := utf8.DecodeLastRuneInString(truncated)
		if size <= 0 || size > len(truncated) {
			return ""
		}
		truncated = truncated[:len(truncated)-size]
	}
	return truncated
}
