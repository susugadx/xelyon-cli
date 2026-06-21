package modelinput

import "github.com/susugadx/xelyon-cli/internal/review/domain"

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
