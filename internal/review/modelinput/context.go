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

// ProbeResultPromptContextOptions は probe result prompt DTO 生成時の provider-facing 最適化を表す。
type ProbeResultPromptContextOptions struct {
	CommandOutputCompactor      CommandOutputCompactor
	AbsorbedProbeResults        map[string]ProbeResultAbsorptionSummary
	AbsorptionCandidateProbeIDs map[string]struct{}
	AbsorbedProbeCommands       map[ProbeCommandResultKey]ProbeResultAbsorptionSummary
	AbsorptionCandidateCommands map[ProbeCommandResultKey]struct{}
}

// ProbeCommandResultKey は probe result 内の command result を安定参照する key。
type ProbeCommandResultKey struct {
	ProbeID      string
	CommandIndex int
}

// CommandOutputCompactor は review prompt に載せる command output compact 境界。
type CommandOutputCompactor interface {
	CompactCommandOutput(command, output string) (CommandOutputCompactResult, bool)
}

// ProbeResultAbsorptionSummary は latest report に吸収済みの probe result context を表す。
type ProbeResultAbsorptionSummary struct {
	Summary        string
	AbsorbedBy     []string
	RawArtifactRef string
}

// CommandOutputCompactResult は provider-facing command output compact 結果。
type CommandOutputCompactResult struct {
	Output     string
	Classifier string
}

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
	ProbeID           string                            `json:"probe_id"`
	Mode              domain.ReviewProbeMode            `json:"mode"`
	Status            domain.ReviewProbeStatus          `json:"status"`
	MutatedWorktree   bool                              `json:"mutated_worktree"`
	MutatedFiles      []string                          `json:"mutated_files"`
	OutputTruncated   bool                              `json:"output_truncated"`
	Error             string                            `json:"error"`
	Absorbed          bool                              `json:"absorbed,omitempty"`
	AbsorptionSummary string                            `json:"absorption_summary,omitempty"`
	AbsorbedBy        []string                          `json:"absorbed_by,omitempty"`
	RawArtifactRef    string                            `json:"raw_artifact_ref,omitempty"`
	Commands          []ProbeCommandResultPromptContext `json:"commands"`
}

// ProbeCommandResultPromptContext は prompt 専用の単一 command 結果 DTO。
type ProbeCommandResultPromptContext struct {
	Command                 string                   `json:"command"`
	Args                    []string                 `json:"args"`
	WorkDir                 string                   `json:"work_dir"`
	Status                  domain.ReviewProbeStatus `json:"status"`
	ExitCode                int                      `json:"exit_code"`
	Output                  string                   `json:"output"`
	OutputTruncated         bool                     `json:"output_truncated"`
	OutputCompacted         bool                     `json:"output_compacted,omitempty"`
	OutputCompactClassifier string                   `json:"output_compact_classifier,omitempty"`
	Error                   string                   `json:"error"`
	DurationMs              int64                    `json:"duration_ms"`
	Absorbed                bool                     `json:"absorbed,omitempty"`
	AbsorptionSummary       string                   `json:"absorption_summary,omitempty"`
	AbsorbedBy              []string                 `json:"absorbed_by,omitempty"`
	RawArtifactRef          string                   `json:"raw_artifact_ref,omitempty"`
}

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
