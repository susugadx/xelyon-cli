package commandoutputs

import (
	"regexp"
	"strings"
)

const (
	compactSideLines                  = 20
	compactListSideEntries            = 40
	largeGenericLineThreshold         = 80
	largeGenericByteThreshold         = 16 * 1024
	failureFirstLineLimit             = 20
	failureLastLineLimit              = 40
	failureKeyErrorLineLimit          = 120
	strictFailureFirstLineLimit       = 5
	strictFailureLastLineLimit        = 20
	strictFailureKeyErrorLineLimit    = 80
	sensitiveFailureKeyErrorLineLimit = 0
	commandSummaryMaxRunes            = 120
	networkDataBearingKeepReason      = "data_bearing_network_command_output_keep"
	databaseDataBearingKeepReason     = "data_bearing_database_command_output_keep"
	observationEvidenceKeepReason     = "evidence_bearing_observation_command_output_keep"
	fileDumpEvidenceKeepReason        = "evidence_bearing_file_dump_command_output_keep"
	gitDiffEvidenceKeepReason         = "evidence_bearing_git_diff_command_output_keep"
	gitShowEvidenceKeepReason         = "evidence_bearing_git_show_command_output_keep"
	sensitiveOutputKeepReason         = "sensitive_output_artifact_forbidden"
)

var (
	exitCodePattern         = regexp.MustCompile(`\b(?:exit(?:ed)?(?:\s+with)?\s+(?:status|code)|process exited with code)\s*:?\s*(-?\d+)`)
	goTestSuccessPattern    = regexp.MustCompile(`(?m)^(?:ok|\?)\s+\S+`)
	testFailedCountPattern  = regexp.MustCompile(`(?mi)\b[1-9][0-9]*\s+failed\b`)
	testFailingCountPattern = regexp.MustCompile(`(?mi)\b[1-9][0-9]*\s+failing\b`)
	lintNonzeroCountPattern = regexp.MustCompile(`(?mi)\b[1-9][0-9]*\s+(?:errors?|issues?|problems?|warnings?)\b`)
	locationLinePattern     = regexp.MustCompile(`(?i)(?:^|\s)(?:[./\w-]+\.(?:go|ts|tsx|js|jsx|rs|py|java|c|cc|cpp|h|hpp|rb|php|cs|swift|kt|m|mm|sql|yaml|yml|json|toml|md)|[./\w-]+):[0-9]+(?::[0-9]+)?`)
	secretAssignmentRegexp  = regexp.MustCompile(`(?i)\b(access_token|refresh_token|id_token|session_token|auth_token|api_key|apikey|client_secret|private_key|password|passwd|secret|token|jwt|signature|sig)=([^\s&;]+)`)
	authHeaderRegexp        = regexp.MustCompile("(?i)(\\bauthorization\\s*[:=]\\s*)(?:([A-Za-z][A-Za-z0-9._-]*)\\s+)?([^\\s'\";]+)")
	secretHeaderRegexp      = regexp.MustCompile("(?i)\\b(x-api-key|api-key|apikey|access-token|refresh-token|id-token|session-token|auth-token|client-secret)\\s*[:=]\\s*([^\\s'\";]+)")
	cookieHeaderRegexp      = regexp.MustCompile(`(?i)\b(set-cookie|cookie)\s*[:=]\s*[^\r\n]+`)
	urlSecretQueryRegexp    = regexp.MustCompile(`(?i)([?&](?:access_token|refresh_token|id_token|session_token|auth_token|api_key|apikey|key|secret|password|passwd|token|client_secret|jwt|signature|sig)=)[^&#\s]+`)
	ansiEscapeRegexp        = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
)

// Request は command output の provider-facing replacement/compact 判定入力を表す。
type Request struct {
	command string
	content string
}

// NewRequest は command output 判定入力を組み立てる。
func NewRequest(command, content string) Request {
	return Request{command: strings.TrimSpace(command), content: content}
}

// Replacement は provider-facing projection 上で使う command output replacement/compact を表す。
type Replacement struct {
	kind        string
	reason      string
	classifier  string
	text        string
	savedBytes  int
	savedTokens int
}

// Kind は replacement/compact の分類名を返す。
func (r Replacement) Kind() string { return r.kind }

// Reason は report に記録する command output 分類 reason を返す。
func (r Replacement) Reason() string { return r.reason }

// Classifier は status breakdown 用の classifier を返す。
func (r Replacement) Classifier() string { return r.classifier }

// Text は provider-facing projection に載せる replacement/compact text を返す。
func (r Replacement) Text() string { return r.text }

// SavedBytes は元 output から削減できる byte 数を返す。
func (r Replacement) SavedBytes() int { return r.savedBytes }

// SavedTokens は元 output から削減できる概算 token 数を返す。
func (r Replacement) SavedTokens() int { return r.savedTokens }

// BuildReplacement は command output の安全な replacement/compact を構築する。
func BuildReplacement(req Request) (Replacement, string, bool) {
	decision := Decide(req)
	if replacement, ok := decision.Replacement(); ok {
		return replacement, "", true
	}
	if decision.KeepReason != "" {
		return Replacement{}, decision.KeepReason, false
	}
	return Replacement{}, "command_output_unknown_skip", false
}
