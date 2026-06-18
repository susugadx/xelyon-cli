package toolresults

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type structuredReplacementBuilder func(ReplacementRequest) (Replacement, string, bool)

var structuredReplacementBuilders = map[string]structuredReplacementBuilder{
	listDirToolName: func(req ReplacementRequest) (Replacement, string, bool) {
		return buildListDirReplacement(req.arguments, req.content)
	},
	webSearchToolName:     buildWebSearchReplacement,
	activateSkillToolName: buildActivateSkillReplacement,
}

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
	builder, ok := structuredReplacementBuilders[req.toolName]
	if !ok {
		return Replacement{}, "unsupported_structured_tool_result", false
	}
	return builder(req)
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
