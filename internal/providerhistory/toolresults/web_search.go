package toolresults

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const webSearchToolName = "web_search"
const webSearchQueryPreviewRunes = 160

var webSearchURLPattern = regexp.MustCompile(`https?://[^\s<>"'\]\)}]+`)

// WebSearchResultSummary は provider-facing に出せる redacted web_search source metadata。
type WebSearchResultSummary struct {
	URL    string
	Domain string
}

// WebSearchAnalysis は XELYON web_search result の provider-facing reduction 前分析。
type WebSearchAnalysis struct {
	QueryHash           string
	QueryPreview        string
	ContentHash         string
	Results             []WebSearchResultSummary
	DuplicateToolCallID string
}

func buildWebSearchReplacement(req ReplacementRequest) (Replacement, string, bool) {
	analysis, reason, ok := AnalyzeWebSearchResult(req)
	if !ok {
		return Replacement{}, reason, false
	}

	replacementText := webSearchReplacementText(analysis, req.toolCallID)
	return Replacement{
		kind:        "omit_old_web_search_result",
		text:        replacementText,
		savedBytes:  savedBytes(len(req.content), len(replacementText)),
		savedTokens: savedTokens(token.EstimateTokenCount(req.content), token.EstimateTokenCount(replacementText)),
	}, "", true
}

// AnalyzeWebSearchResult は XELYON web_search tool result を安全に要約できるか判定する。
func AnalyzeWebSearchResult(req ReplacementRequest) (WebSearchAnalysis, string, bool) {
	query, ok := webSearchQueryArgument(req.arguments)
	if !ok {
		return WebSearchAnalysis{}, "web_search_unknown_format_keep", false
	}
	if webSearchQueryIsTemporal(query) {
		return WebSearchAnalysis{}, "web_search_temporal_or_current_keep", false
	}
	if webSearchQueryIsUnsafeForMetadata(query) || webSearchContentIsUnsafeToSummarize(req.content) {
		return WebSearchAnalysis{}, "web_search_unknown_credibility_keep", false
	}

	results := parseWebSearchResultURLs(req.content)
	if len(results) == 0 {
		return WebSearchAnalysis{}, "web_search_unknown_format_keep", false
	}

	contentHash := providerHistoryToolResultHash(req.content)
	duplicate := laterDuplicateWebSearchResult(req, query, contentHash)
	if duplicate.toolCallID == "" {
		return WebSearchAnalysis{}, "web_search_citation_or_referenced_result_keep", false
	}

	return WebSearchAnalysis{
		QueryHash:           webSearchQueryHash(query),
		QueryPreview:        webSearchQueryPreview(query),
		ContentHash:         contentHash,
		Results:             results,
		DuplicateToolCallID: duplicate.toolCallID,
	}, "", true
}

func webSearchQueryArgument(arguments string) (string, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &fields); err != nil {
		return "", false
	}
	raw, ok := fields["query"]
	if !ok {
		return "", false
	}
	var query string
	if err := json.Unmarshal(raw, &query); err != nil {
		return "", false
	}
	query = strings.TrimSpace(query)
	return query, query != ""
}

func webSearchQueryIsTemporal(query string) bool {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if normalized == "" {
		return true
	}
	temporalTerms := []string{
		"latest",
		"today",
		"current",
		"breaking",
		"news",
		"price",
		"exchange rate",
		"law",
		"policy",
		"version",
		"release",
		"changelog",
		"schedule",
	}
	for _, term := range temporalTerms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}

func webSearchQueryIsUnsafeForMetadata(query string) bool {
	redacted := rawoutputs.RedactDisplaySecrets(query)
	return redacted != query && strings.TrimSpace(redacted) != strings.TrimSpace(query)
}

func webSearchContentIsUnsafeToSummarize(content string) bool {
	normalized := strings.ToLower(content)
	unsafeTerms := []string{
		"ignore previous instructions",
		"ignore all previous instructions",
		"system prompt",
		"developer message",
		"prompt injection",
		"do not summarize",
		"secret",
		"api key",
		"api_key",
		"apikey",
		"authorization",
		"bearer ",
		"password",
		"access_token",
		"refresh_token",
		"token=",
	}
	for _, term := range unsafeTerms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}

func parseWebSearchResultURLs(content string) []WebSearchResultSummary {
	rawURLs := webSearchURLPattern.FindAllString(content, -1)
	if len(rawURLs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(rawURLs))
	results := make([]WebSearchResultSummary, 0, len(rawURLs))
	for _, rawURL := range rawURLs {
		clean := cleanWebSearchResultURL(rawURL)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		results = append(results, WebSearchResultSummary{
			URL:    clean,
			Domain: webSearchResultDomain(clean),
		})
	}
	return results
}

func cleanWebSearchResultURL(rawURL string) string {
	candidate := strings.TrimSpace(rawURL)
	candidate = strings.TrimRight(candidate, ".,;:!?")
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	return parsed.String()
}

func webSearchResultDomain(resultURL string) string {
	parsed, err := url.Parse(resultURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func webSearchQueryHash(query string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(query)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func webSearchQueryPreview(query string) string {
	return rawoutputs.SanitizeDisplayPreview(query, webSearchQueryPreviewRunes)
}

type duplicateWebSearchResult struct {
	toolCallID string
}

func laterDuplicateWebSearchResult(req ReplacementRequest, query, contentHash string) duplicateWebSearchResult {
	if req.historyIndex < 0 || req.historyIndex >= len(req.messages)-1 {
		return duplicateWebSearchResult{}
	}
	for i := req.historyIndex + 1; i < len(req.messages); i++ {
		msg := req.messages[i]
		if msg.Role != "tool" {
			continue
		}
		toolName := strings.TrimSpace(msg.ToolName)
		if toolName == "" {
			toolName = toolNameForToolResultAt(req.messages, i)
		}
		if toolName != webSearchToolName {
			continue
		}
		args := toolArgumentsForToolResultAt(req.messages, i)
		laterQuery, ok := webSearchQueryArgument(args)
		if !ok || laterQuery != query {
			continue
		}
		if providerHistoryToolResultHash(msg.Content) != contentHash {
			continue
		}
		return duplicateWebSearchResult{toolCallID: msg.ToolCallID}
	}
	return duplicateWebSearchResult{}
}

func webSearchReplacementText(analysis WebSearchAnalysis, toolCallID string) string {
	const maxSelectedSources = 3
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"[compacted old web_search result; query_hash=%s; query_preview=%q; results=%d; selected_urls=%d; content_hash=%s; raw_tool_call_id=%s; duplicate_of=%s]\n",
		analysis.QueryHash,
		analysis.QueryPreview,
		len(analysis.Results),
		min(len(analysis.Results), maxSelectedSources),
		analysis.ContentHash,
		singleLine(toolCallID),
		singleLine(analysis.DuplicateToolCallID),
	)
	b.WriteString("selected_sources:\n")
	limit := min(len(analysis.Results), maxSelectedSources)
	for _, result := range analysis.Results[:limit] {
		fmt.Fprintf(&b, "- %s (domain=%s)\n", result.URL, singleLine(result.Domain))
	}
	if omitted := len(analysis.Results) - limit; omitted > 0 {
		fmt.Fprintf(&b, "- +%d omitted duplicate raw sources\n", omitted)
	}
	b.WriteString("notes:\n")
	b.WriteString("- raw duplicate result is preserved later in history/audit\n")
	b.WriteString("- source credibility is not upgraded by this compact summary")
	return b.String()
}

func providerHistoryToolResultHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}

func toolArgumentsForToolResultAt(messages []api.Message, toolResultIndex int) string {
	toolCallID := strings.TrimSpace(messages[toolResultIndex].ToolCallID)
	if toolCallID == "" {
		return ""
	}
	assistantIndex := contiguousAssistantIndexForToolResult(messages, toolResultIndex)
	if assistantIndex < 0 {
		return ""
	}
	for _, toolCall := range messages[assistantIndex].ToolCalls {
		if toolCall.ID == toolCallID {
			return toolCall.Function.Arguments
		}
	}
	return ""
}

func toolNameForToolResultAt(messages []api.Message, toolResultIndex int) string {
	toolCallID := strings.TrimSpace(messages[toolResultIndex].ToolCallID)
	if toolCallID == "" {
		return ""
	}
	assistantIndex := contiguousAssistantIndexForToolResult(messages, toolResultIndex)
	if assistantIndex < 0 {
		return ""
	}
	for _, toolCall := range messages[assistantIndex].ToolCalls {
		if toolCall.ID == toolCallID {
			return strings.TrimSpace(toolCall.Function.Name)
		}
	}
	return ""
}

func contiguousAssistantIndexForToolResult(messages []api.Message, toolResultIndex int) int {
	for i := toolResultIndex - 1; i >= 0; i-- {
		switch messages[i].Role {
		case "tool":
			continue
		case "assistant":
			return i
		default:
			return -1
		}
	}
	return -1
}

func singleLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
