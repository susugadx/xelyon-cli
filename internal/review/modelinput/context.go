package modelinput

import (
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

const (
	maxReviewProbeResultPromptCommandOutputBytes = 8 * 1024
	maxReviewProbeResultPromptTotalOutputBytes   = 64 * 1024

	reviewProbeResultPromptOutputTruncatedMarker = "\n[probe output truncated for review prompt]"
	reviewProbeResultPromptOutputOmittedMarker   = "[probe output omitted because review prompt output budget was exhausted]"
)

// Redactor は model input assembly が使う最小 redaction 境界。
// path replacement の発見や redaction policy 自体は caller が所有する。
type Redactor interface {
	RedactText(string) string
	RedactTexts([]string) []string
	RedactPath(string) string
	RedactPaths([]string) []string
}

// ProbeResultPromptContext は Pass2 prompt 専用の probe result DTO。
// Report schema 用 summary とは別に、model 判断に必要な command output を保持する。
type ProbeResultPromptContext struct {
	ProbeID         string                            `json:"probe_id"`
	Mode            domain.ReviewProbeMode            `json:"mode"`
	Status          domain.ReviewProbeStatus          `json:"status"`
	MutatedWorktree bool                              `json:"mutated_worktree"`
	MutatedFiles    []string                          `json:"mutated_files"`
	OutputTruncated bool                              `json:"output_truncated"`
	Error           string                            `json:"error"`
	Commands        []ProbeCommandResultPromptContext `json:"commands"`
}

// ProbeCommandResultPromptContext は prompt 専用の単一 command 結果 DTO。
type ProbeCommandResultPromptContext struct {
	Command         string                   `json:"command"`
	Args            []string                 `json:"args"`
	WorkDir         string                   `json:"work_dir"`
	Status          domain.ReviewProbeStatus `json:"status"`
	ExitCode        int                      `json:"exit_code"`
	Output          string                   `json:"output"`
	OutputTruncated bool                     `json:"output_truncated"`
	Error           string                   `json:"error"`
	DurationMs      int64                    `json:"duration_ms"`
}

// BuildProbeResultPromptContexts は probe result を redacted prompt DTO へ変換する。
func BuildProbeResultPromptContexts(results []reviewprobe.ReviewProbeResult, redactor Redactor) []ProbeResultPromptContext {
	redactor = normalizeRedactor(redactor)
	outputLimiter := newReviewProbeResultPromptOutputLimiter()
	contexts := make([]ProbeResultPromptContext, 0, len(results))
	for _, result := range results {
		contexts = append(contexts, ProbeResultPromptContext{
			ProbeID:         result.ID,
			Mode:            result.Mode,
			Status:          result.Status,
			MutatedWorktree: result.MutatedWorktree,
			MutatedFiles:    redactor.RedactPaths(result.MutatedFiles),
			OutputTruncated: result.OutputTruncated,
			Error:           redactor.RedactText(result.Error),
			Commands:        buildReviewProbeCommandResultPromptContexts(result.CommandResults, redactor, outputLimiter),
		})
	}
	return contexts
}

func buildReviewProbeCommandResultPromptContexts(results []reviewprobe.ReviewProbeCommandResult, redactor Redactor, outputLimiter *reviewProbeResultPromptOutputLimiter) []ProbeCommandResultPromptContext {
	contexts := make([]ProbeCommandResultPromptContext, 0, len(results))
	for _, result := range results {
		output, promptOutputTruncated := outputLimiter.limit(redactor.RedactText(result.Output))
		contexts = append(contexts, ProbeCommandResultPromptContext{
			Command:         redactor.RedactText(result.Command),
			Args:            redactor.RedactTexts(result.Args),
			WorkDir:         redactor.RedactPath(result.WorkDir),
			Status:          result.Status,
			ExitCode:        result.ExitCode,
			Output:          output,
			OutputTruncated: result.OutputTruncated || promptOutputTruncated,
			Error:           redactor.RedactText(result.Error),
			DurationMs:      result.Duration.Milliseconds(),
		})
	}
	return contexts
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

func normalizeRedactor(redactor Redactor) Redactor {
	if redactor == nil {
		return noopRedactor{}
	}
	return redactor
}

type noopRedactor struct{}

// RedactText は nil redactor 時に text をそのまま返す。
func (noopRedactor) RedactText(text string) string {
	return text
}

// RedactTexts は nil redactor 時に text 配列をそのまま返す。
func (noopRedactor) RedactTexts(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

// RedactPath は nil redactor 時に path をそのまま返す。
func (noopRedactor) RedactPath(path string) string {
	return path
}

// RedactPaths は nil redactor 時に path 配列をそのまま返す。
func (noopRedactor) RedactPaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{}
	}
	return append([]string(nil), paths...)
}
