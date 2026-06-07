package toolresults

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const listDirToolName = "list_dir"

var (
	listDirSummaryLinePattern = regexp.MustCompile(`^summary: depth=([0-9]+), dirs=([0-9]+), files=([0-9]+)$`)
	windowsAbsPathPattern     = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
)

// ReplacementRequest は structured tool result replacement の入力を表す。
// zero value は invalid として扱われ、builder は replacement を返さない。
type ReplacementRequest struct {
	toolName     string
	arguments    string
	content      string
	toolCallID   string
	historyIndex int
	messages     []api.Message
}

// NewReplacementRequest は tool result replacement の入力を組み立てる。
func NewReplacementRequest(toolName, arguments, content string) ReplacementRequest {
	return ReplacementRequest{
		toolName:  strings.TrimSpace(toolName),
		arguments: arguments,
		content:   content,
	}
}

// NewReplacementRequestWithMessages は history context が必要な structured replacement 入力を組み立てる。
func NewReplacementRequestWithMessages(toolName, arguments, content, toolCallID string, historyIndex int, messages []api.Message) ReplacementRequest {
	req := NewReplacementRequest(toolName, arguments, content)
	req.toolCallID = strings.TrimSpace(toolCallID)
	req.historyIndex = historyIndex
	req.messages = api.CloneMessages(messages)
	return req
}

// Replacement は provider-facing projection 上へ適用できる structured replacement を表す。
type Replacement struct {
	kind        string
	text        string
	savedBytes  int
	savedTokens int
}

// Kind は replacement の分類名を返す。
func (r Replacement) Kind() string { return r.kind }

// Text は provider-facing projection に載せる replacement text を返す。
func (r Replacement) Text() string { return r.text }

// SavedBytes は元 content から削減できる byte 数を返す。
func (r Replacement) SavedBytes() int { return r.savedBytes }

// SavedTokens は元 content から削減できる概算 token 数を返す。
func (r Replacement) SavedTokens() int { return r.savedTokens }

// BuildStructuredReplacement は evidence pointer を持たない structured tool result replacement を作る。
func BuildStructuredReplacement(req ReplacementRequest) (Replacement, string, bool) {
	switch req.toolName {
	case listDirToolName:
		return buildListDirReplacement(req.arguments, req.content)
	case webSearchToolName:
		return buildWebSearchReplacement(req)
	case activateSkillToolName:
		return buildActivateSkillReplacement(req)
	default:
		return Replacement{}, "unsupported_structured_tool_result", false
	}
}

func buildListDirReplacement(arguments, content string) (Replacement, string, bool) {
	argumentPath, ok := listDirPathArgument(arguments)
	if !ok {
		return Replacement{}, "missing_list_dir_path_argument", false
	}
	normalizedPath, ok := normalizeListDirReplacementPath(argumentPath)
	if !ok {
		return Replacement{}, "unsafe_list_dir_path", false
	}

	summary, reason, ok := parseListDirResultSummary(content)
	if !ok {
		return Replacement{}, reason, false
	}

	replacementText := fmt.Sprintf(
		"[omitted old list_dir result; path=%s; entries=%d; depth=%d]",
		normalizedPath,
		summary.entries(),
		summary.depth,
	)
	return Replacement{
		kind:        "omit_old_list_dir_result",
		text:        replacementText,
		savedBytes:  savedBytes(len(content), len(replacementText)),
		savedTokens: savedTokens(token.EstimateTokenCount(content), token.EstimateTokenCount(replacementText)),
	}, "", true
}

func listDirPathArgument(arguments string) (string, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &fields); err != nil {
		return "", false
	}
	raw, ok := fields["path"]
	if !ok {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

func normalizeListDirReplacementPath(value string) (string, bool) {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`\"'")
	value = strings.ReplaceAll(value, "\\", "/")
	if value == "" || strings.Contains(value, "\x00") || strings.ContainsAny(value, "\n\r") {
		return "", false
	}
	if strings.Contains(value, "://") || strings.ContainsAny(value, "*?") {
		return "", false
	}
	if strings.HasPrefix(value, "locator:") || strings.HasPrefix(value, "L") && isDigits(value[1:]) {
		return "", false
	}
	if strings.HasPrefix(value, "/") || windowsAbsPathPattern.MatchString(value) {
		return "", false
	}

	cleaned := path.Clean(value)
	if cleaned == "." {
		return ".", true
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", false
	}
	if windowsAbsPathPattern.MatchString(cleaned) {
		return "", false
	}
	return cleaned, true
}

type listDirSummary struct {
	depth int
	dirs  int
	files int
}

func (s listDirSummary) entries() int {
	return s.dirs + s.files
}

func parseListDirResultSummary(content string) (listDirSummary, string, bool) {
	lines := strings.Split(content, "\n")
	first := firstNonEmptyLine(lines)
	if strings.HasPrefix(first, "Error:") || !strings.HasPrefix(first, "📂 ") {
		return listDirSummary{}, "list_dir_result_not_success", false
	}
	if listDirResultContainsFailurePhrase(content) {
		return listDirSummary{}, "list_dir_result_not_success", false
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "summary:") {
			continue
		}
		summary, ok := parseListDirSummaryLine(line)
		if !ok {
			return listDirSummary{}, "list_dir_summary_unparseable", false
		}
		return summary, "", true
	}
	return listDirSummary{}, "list_dir_summary_unparseable", false
}

func firstNonEmptyLine(lines []string) string {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func parseListDirSummaryLine(line string) (listDirSummary, bool) {
	matches := listDirSummaryLinePattern.FindStringSubmatch(line)
	if len(matches) != 4 {
		return listDirSummary{}, false
	}
	depth, ok := atoiNonNegative(matches[1])
	if !ok {
		return listDirSummary{}, false
	}
	dirs, ok := atoiNonNegative(matches[2])
	if !ok {
		return listDirSummary{}, false
	}
	files, ok := atoiNonNegative(matches[3])
	if !ok {
		return listDirSummary{}, false
	}
	return listDirSummary{depth: depth, dirs: dirs, files: files}, true
}

func atoiNonNegative(value string) (int, bool) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func listDirResultContainsFailurePhrase(content string) bool {
	lower := strings.ToLower(content)
	for _, phrase := range []string{
		"error:",
		"failed",
		"denied",
		"cancelled",
		"permission denied",
		"unsafe path",
		"invalid path",
		"not a directory",
		"path is empty",
		"symlink escape",
		"outside",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func savedBytes(originalBytes, replacementBytes int) int {
	if originalBytes <= replacementBytes {
		return 0
	}
	return originalBytes - replacementBytes
}

func savedTokens(originalTokens, replacementTokens int) int {
	if originalTokens <= replacementTokens {
		return 0
	}
	return originalTokens - replacementTokens
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
