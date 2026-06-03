package editargs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	replacementMinSavedTokens = 128
	pathSummaryMax            = 3
)

var (
	writeFileSuccessPattern              = regexp.MustCompile(`(?m)^Successfully wrote \d+ bytes \(\d+ lines?\) to (.+?)(?:\r?\n|$)`)
	strReplaceTextSuccessPattern         = regexp.MustCompile(`(?m)^Successfully replaced text in (.+?) \(lines \d+-\d+`)
	strReplaceLineRangeSuccessPattern    = regexp.MustCompile(`(?m)^Successfully replaced lines \d+-\d+ in (.+?) \(new range: \d+-\d+\)`)
	strReplaceBatchSuccessPattern        = regexp.MustCompile(`(?m)^Successfully applied \d+ edits to (.+?)(?:\r?\n|$)`)
	strReplaceSuccessResultPathPatterns  = []*regexp.Regexp{strReplaceTextSuccessPattern, strReplaceLineRangeSuccessPattern, strReplaceBatchSuccessPattern}
	applyPatchSuccessResultLinePrefixes  = []string{"Added: ", "Modified: ", "Deleted: "}
	applyPatchSuccessResultRequiredLines = map[string]struct{}{"Added: ": {}, "Modified: ": {}, "Deleted: ": {}}
)

// PayloadSummary は edit tool の古い引数 payload 計測結果を表す。
type PayloadSummary struct {
	Reason string
	Bytes  int
	Runes  int
	Tokens int
}

// ReplacementRequest は edit tool 引数の provider-facing 置換判定入力を表す。
type ReplacementRequest struct {
	ToolName          string
	Arguments         string
	ToolResultContent string
}

// Replacement は provider projection に反映する edit tool 引数置換を表す。
type Replacement struct {
	ToolName    string
	Arguments   string
	SavedBytes  int
	SavedTokens int

	applyAnthropicInput func(map[string]any) bool
}

// ApplyAnthropicInput は Anthropic tool_use.input を同じ置換内容に同期する。
func (r Replacement) ApplyAnthropicInput(input map[string]any) bool {
	if r.applyAnthropicInput == nil {
		return false
	}
	return r.applyAnthropicInput(input)
}

// IsTool は command/edit dry-run 対象の edit argument tool かどうかを返す。
func IsTool(toolName string) bool {
	switch toolName {
	case "write_file", "apply_patch", "str_replace", "delete_file":
		return true
	default:
		return false
	}
}

// Payload は edit tool 引数の candidate reason と payload サイズを計測する。
func Payload(toolName, arguments string) (PayloadSummary, string) {
	fields, err := toolCallArgumentFields(arguments)
	if err != nil {
		return PayloadSummary{}, "invalid_tool_call_arguments"
	}
	switch toolName {
	case "write_file":
		return stringFieldsPayload("write_file_content", fields, "content")
	case "apply_patch":
		return stringFieldsPayload("apply_patch_patch", fields, "patch")
	case "str_replace":
		if raw, ok := strReplaceBatchEditsArgument(fields); ok {
			return rawOrStringFieldPayload("str_replace_edits", raw)
		}
		return stringFieldsPayload("str_replace_strings", fields, "old_str", "new_str")
	case "delete_file":
		if value, ok := jsonStringArgument(fields, "path"); ok && value != "" {
			return valuesPayload("delete_file_path", []string{value}), ""
		}
		return PayloadSummary{}, "missing_edit_argument_payload"
	default:
		return PayloadSummary{}, "tool_not_in_command_edit_allowlist"
	}
}

// BuildReplacement は成功済み edit tool 結果に対応する古い引数の置換 JSON を構築する。
func BuildReplacement(req ReplacementRequest) (Replacement, bool) {
	switch req.ToolName {
	case "write_file":
		return buildWriteFileReplacement(req.Arguments, req.ToolResultContent)
	case "apply_patch":
		return buildApplyPatchReplacement(req.Arguments, req.ToolResultContent)
	case "str_replace":
		return buildStrReplaceReplacement(req.Arguments, req.ToolResultContent)
	default:
		return Replacement{}, false
	}
}

func buildWriteFileReplacement(arguments, toolResultContent string) (Replacement, bool) {
	fields, err := toolCallArgumentFields(arguments)
	if err != nil {
		return Replacement{}, false
	}
	rawPath, ok := jsonStringArgument(fields, "path")
	if !ok || rawPath == "" {
		return Replacement{}, false
	}
	path, ok := taskstate.NormalizeRepoRelativePath(rawPath)
	if !ok {
		return Replacement{}, false
	}
	content, ok := jsonStringArgument(fields, "content")
	if !ok || content == "" {
		return Replacement{}, false
	}
	if !writeFileResultSucceededForPath(toolResultContent, path) {
		return Replacement{}, false
	}

	replacementText := buildWriteFileContentPlaceholder(path)
	if len(replacementText) >= len(content) {
		return Replacement{}, false
	}
	savedTokens := savedTokenDelta(token.EstimateTokenCount(content), token.EstimateTokenCount(replacementText))
	if savedTokens < replacementMinSavedTokens {
		return Replacement{}, false
	}
	if !setJSONStringField(fields, "content", replacementText) {
		return Replacement{}, false
	}
	replacementArgs, err := json.Marshal(fields)
	if err != nil {
		return Replacement{}, false
	}

	return Replacement{
		ToolName:    "write_file",
		Arguments:   string(replacementArgs),
		SavedBytes:  len(content) - len(replacementText),
		SavedTokens: savedTokens,
		applyAnthropicInput: func(input map[string]any) bool {
			return updateWriteFileInputContent(input, path, content, replacementText)
		},
	}, true
}

func buildApplyPatchReplacement(arguments, toolResultContent string) (Replacement, bool) {
	fields, err := toolCallArgumentFields(arguments)
	if err != nil {
		return Replacement{}, false
	}
	originalFields := cloneRawFields(fields)
	patch, ok := jsonStringArgument(fields, "patch")
	if !ok || strings.TrimSpace(patch) == "" {
		return Replacement{}, false
	}
	paths, ok := applyPatchHeaderPaths(patch)
	if !ok || !applyPatchResultSucceededForPaths(toolResultContent, paths) {
		return Replacement{}, false
	}

	replacementText := buildApplyPatchPlaceholder(paths)
	if len(replacementText) >= len(patch) {
		return Replacement{}, false
	}
	savedTokens := savedTokenDelta(token.EstimateTokenCount(patch), token.EstimateTokenCount(replacementText))
	if savedTokens < replacementMinSavedTokens {
		return Replacement{}, false
	}
	if !setJSONStringField(fields, "patch", replacementText) {
		return Replacement{}, false
	}
	replacementArgs, err := json.Marshal(fields)
	if err != nil {
		return Replacement{}, false
	}

	return Replacement{
		ToolName:    "apply_patch",
		Arguments:   string(replacementArgs),
		SavedBytes:  len(patch) - len(replacementText),
		SavedTokens: savedTokens,
		applyAnthropicInput: func(input map[string]any) bool {
			if !anthropicInputMatchesToolArguments(input, originalFields) {
				return false
			}
			input["patch"] = replacementText
			return true
		},
	}, true
}

func buildStrReplaceReplacement(arguments, toolResultContent string) (Replacement, bool) {
	fields, err := toolCallArgumentFields(arguments)
	if err != nil {
		return Replacement{}, false
	}
	rawPath, ok := jsonStringArgument(fields, "path")
	if !ok || rawPath == "" {
		return Replacement{}, false
	}
	path, ok := taskstate.NormalizeRepoRelativePath(rawPath)
	if !ok || !strReplaceResultSucceededForPath(toolResultContent, path) {
		return Replacement{}, false
	}

	if raw, ok := strReplaceBatchEditsArgument(fields); ok {
		return buildStrReplaceBatchReplacement(fields, raw, path)
	}
	return buildStrReplaceSingleReplacement(fields, path)
}

func strReplaceBatchEditsArgument(fields map[string]json.RawMessage) (json.RawMessage, bool) {
	oldStr, _ := jsonStringArgument(fields, "old_str")
	if oldStr != "" {
		return nil, false
	}
	raw, ok := fields["edits"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return nil, false
	}
	var editsString string
	if err := json.Unmarshal(raw, &editsString); err == nil {
		return raw, editsString != ""
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil, false
	}
	return raw, true
}

func buildStrReplaceSingleReplacement(fields map[string]json.RawMessage, path string) (Replacement, bool) {
	originalFields := cloneRawFields(fields)
	oldStr, oldOK := jsonStringArgument(fields, "old_str")
	newStr, newOK := jsonStringArgument(fields, "new_str")
	if !oldOK || !newOK || oldStr == "" {
		return Replacement{}, false
	}

	oldReplacement := buildStrReplacePlaceholder(path, "old_str", -1)
	newReplacement := buildStrReplacePlaceholder(path, "new_str", -1)
	originalBytes := len(oldStr) + len(newStr)
	replacementBytes := len(oldReplacement) + len(newReplacement)
	if replacementBytes >= originalBytes {
		return Replacement{}, false
	}
	originalTokens := token.EstimateTokenCount(oldStr) + token.EstimateTokenCount(newStr)
	replacementTokens := token.EstimateTokenCount(oldReplacement) + token.EstimateTokenCount(newReplacement)
	savedTokens := savedTokenDelta(originalTokens, replacementTokens)
	if savedTokens < replacementMinSavedTokens {
		return Replacement{}, false
	}

	if !setJSONStringField(fields, "old_str", oldReplacement) ||
		!setJSONStringField(fields, "new_str", newReplacement) {
		return Replacement{}, false
	}
	replacementArgs, err := json.Marshal(fields)
	if err != nil {
		return Replacement{}, false
	}

	return Replacement{
		ToolName:    "str_replace",
		Arguments:   string(replacementArgs),
		SavedBytes:  originalBytes - replacementBytes,
		SavedTokens: savedTokens,
		applyAnthropicInput: func(input map[string]any) bool {
			if !anthropicInputMatchesToolArguments(input, originalFields) {
				return false
			}
			input["old_str"] = oldReplacement
			input["new_str"] = newReplacement
			return true
		},
	}, true
}

func buildStrReplaceBatchReplacement(fields map[string]json.RawMessage, raw json.RawMessage, path string) (Replacement, bool) {
	originalFields := cloneRawFields(fields)
	edits, originalPayload, editsFieldWasString, ok := parseStrReplaceEditsPayload(raw)
	if !ok || len(edits) == 0 {
		return Replacement{}, false
	}

	for i := range edits {
		if _, ok := jsonStringArgument(edits[i], "old_str"); !ok {
			return Replacement{}, false
		}
		if _, ok := jsonStringArgument(edits[i], "new_str"); !ok {
			return Replacement{}, false
		}
		if !setJSONStringField(edits[i], "old_str", buildStrReplacePlaceholder(path, "old_str", i)) ||
			!setJSONStringField(edits[i], "new_str", buildStrReplacePlaceholder(path, "new_str", i)) {
			return Replacement{}, false
		}
	}

	replacementPayloadBytes, err := json.Marshal(edits)
	if err != nil {
		return Replacement{}, false
	}
	replacementPayload := string(replacementPayloadBytes)
	if len(replacementPayload) >= len(originalPayload) {
		return Replacement{}, false
	}
	savedTokens := savedTokenDelta(token.EstimateTokenCount(originalPayload), token.EstimateTokenCount(replacementPayload))
	if savedTokens < replacementMinSavedTokens {
		return Replacement{}, false
	}

	var replacementInput any
	if err := json.Unmarshal(replacementPayloadBytes, &replacementInput); err != nil {
		return Replacement{}, false
	}
	if editsFieldWasString {
		if !setJSONStringField(fields, "edits", replacementPayload) {
			return Replacement{}, false
		}
	} else {
		fields["edits"] = replacementPayloadBytes
	}
	replacementArgs, err := json.Marshal(fields)
	if err != nil {
		return Replacement{}, false
	}

	return Replacement{
		ToolName:    "str_replace",
		Arguments:   string(replacementArgs),
		SavedBytes:  len(originalPayload) - len(replacementPayload),
		SavedTokens: savedTokens,
		applyAnthropicInput: func(input map[string]any) bool {
			if !anthropicInputMatchesToolArguments(input, originalFields) {
				return false
			}
			if editsFieldWasString {
				input["edits"] = replacementPayload
			} else {
				input["edits"] = replacementInput
			}
			return true
		},
	}, true
}

func toolCallArgumentFields(arguments string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = map[string]json.RawMessage{}
	}
	return fields, nil
}

func stringFieldsPayload(reason string, fields map[string]json.RawMessage, keys ...string) (PayloadSummary, string) {
	var values []string
	for _, key := range keys {
		value, ok := jsonStringArgument(fields, key)
		if !ok {
			continue
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return PayloadSummary{}, "missing_edit_argument_payload"
	}
	return valuesPayload(reason, values), ""
}

func rawOrStringFieldPayload(reason string, raw json.RawMessage) (PayloadSummary, string) {
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return valuesPayload(reason, []string{value}), ""
	}
	rawValue := strings.TrimSpace(string(raw))
	if rawValue == "" {
		return PayloadSummary{}, "missing_edit_argument_payload"
	}
	return valuesPayload(reason, []string{rawValue}), ""
}

func valuesPayload(reason string, values []string) PayloadSummary {
	summary := PayloadSummary{Reason: reason}
	for _, value := range values {
		summary.Bytes += len(value)
		summary.Runes += utf8.RuneCountInString(value)
		summary.Tokens += token.EstimateTokenCount(value)
	}
	return summary
}

func jsonStringArgument(fields map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := fields[key]
	if !ok {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func setJSONStringField(fields map[string]json.RawMessage, key, value string) bool {
	encoded, err := json.Marshal(value)
	if err != nil {
		return false
	}
	fields[key] = encoded
	return true
}

func writeFileResultSucceededForPath(content, path string) bool {
	resultPath, ok := writeFileSuccessResultPath(content)
	return ok && resultPath == path
}

func writeFileSuccessResultPath(content string) (string, bool) {
	matches := writeFileSuccessPattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", false
	}
	return taskstate.NormalizeRepoRelativePath(matches[1])
}

func updateWriteFileInputContent(input map[string]any, path, originalContent, replacementText string) bool {
	if len(input) == 0 {
		return false
	}
	pathValue, ok := input["path"].(string)
	if !ok || strings.TrimSpace(pathValue) == "" {
		return false
	}
	inputPath, ok := taskstate.NormalizeRepoRelativePath(pathValue)
	if !ok || inputPath != path {
		return false
	}
	contentValue, ok := input["content"].(string)
	if !ok || contentValue != originalContent {
		return false
	}
	input["content"] = replacementText
	return true
}

func applyPatchHeaderPaths(patch string) ([]string, bool) {
	paths := make(map[string]struct{})
	lines := normalizedLines(patch)
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "*** Update File: ") {
			path, ok := taskstate.NormalizeRepoRelativePath(strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: ")))
			if !ok {
				return nil, false
			}
			if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "*** Move to: ") {
				movePath, ok := taskstate.NormalizeRepoRelativePath(strings.TrimSpace(strings.TrimPrefix(lines[i+1], "*** Move to: ")))
				if !ok {
					return nil, false
				}
				path = movePath
			}
			paths[path] = struct{}{}
			continue
		}
		for _, prefix := range []string{"*** Add File: ", "*** Delete File: "} {
			if strings.HasPrefix(line, prefix) {
				path, ok := taskstate.NormalizeRepoRelativePath(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
				if !ok {
					return nil, false
				}
				paths[path] = struct{}{}
				break
			}
		}
	}
	return sortedPathSet(paths)
}

func applyPatchResultSucceededForPaths(content string, paths []string) bool {
	if !strings.Contains(content, "✓ Patch applied successfully.") {
		return false
	}
	resultPaths, ok := applyPatchSuccessResultPaths(content)
	return ok && stringSlicesEqual(resultPaths, paths)
}

func applyPatchSuccessResultPaths(content string) ([]string, bool) {
	paths := make(map[string]struct{})
	seen := make(map[string]bool, len(applyPatchSuccessResultRequiredLines))
	for prefix := range applyPatchSuccessResultRequiredLines {
		seen[prefix] = false
	}

	for _, line := range normalizedLines(content) {
		trimmed := strings.TrimSpace(line)
		for _, prefix := range applyPatchSuccessResultLinePrefixes {
			if !strings.HasPrefix(trimmed, prefix) {
				continue
			}
			if seen[prefix] {
				return nil, false
			}
			seen[prefix] = true
			if !collectResultPaths(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), paths) {
				return nil, false
			}
			break
		}
	}
	for _, found := range seen {
		if !found {
			return nil, false
		}
	}
	return sortedPathSet(paths)
}

func collectResultPaths(value string, paths map[string]struct{}) bool {
	if value == "(none)" {
		return true
	}
	if value == "" {
		return false
	}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		rawPath := strings.TrimSpace(part)
		if rawPath == "" || rawPath == "(none)" {
			return false
		}
		path, ok := taskstate.NormalizeRepoRelativePath(rawPath)
		if !ok {
			return false
		}
		paths[path] = struct{}{}
	}
	return true
}

func strReplaceResultSucceededForPath(content, path string) bool {
	resultPath, ok := strReplaceSuccessResultPath(content)
	return ok && resultPath == path
}

func strReplaceSuccessResultPath(content string) (string, bool) {
	var matchedPaths []string
	for _, pattern := range strReplaceSuccessResultPathPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			matchedPaths = append(matchedPaths, strings.TrimSpace(match[1]))
		}
	}
	if len(matchedPaths) != 1 {
		return "", false
	}
	return taskstate.NormalizeRepoRelativePath(matchedPaths[0])
}

func parseStrReplaceEditsPayload(raw json.RawMessage) ([]map[string]json.RawMessage, string, bool, bool) {
	originalPayload := strings.TrimSpace(string(raw))
	editsPayload := originalPayload
	editsFieldWasString := false
	var editsString string
	if err := json.Unmarshal(raw, &editsString); err == nil {
		editsFieldWasString = true
		originalPayload = editsString
		editsPayload = strings.TrimSpace(editsString)
	}
	if strings.TrimSpace(originalPayload) == "" || editsPayload == "" {
		return nil, "", false, false
	}

	var edits []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(editsPayload), &edits); err != nil {
		return nil, "", false, false
	}
	return edits, originalPayload, editsFieldWasString, true
}

func anthropicInputMatchesToolArguments(input map[string]any, fields map[string]json.RawMessage) bool {
	if len(input) == 0 || len(fields) == 0 {
		return false
	}
	for key, raw := range fields {
		actual, ok := input[key]
		if !ok || !jsonRawMatchesValue(raw, actual) {
			return false
		}
	}
	return true
}

func jsonRawMatchesValue(raw json.RawMessage, value any) bool {
	var expected any
	if err := json.Unmarshal(raw, &expected); err != nil {
		return false
	}
	actualBytes, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var actual any
	if err := json.Unmarshal(actualBytes, &actual); err != nil {
		return false
	}
	return reflect.DeepEqual(expected, actual)
}

func cloneRawFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	if len(fields) == 0 {
		return nil
	}
	cloned := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		cloned[key] = append(json.RawMessage(nil), value...)
	}
	return cloned
}

func buildWriteFileContentPlaceholder(path string) string {
	return fmt.Sprintf("[omitted old write_file.content; path=%s]", singleLine(path))
}

func buildApplyPatchPlaceholder(paths []string) string {
	return fmt.Sprintf("[omitted old apply_patch.patch; files=%s]", pathSummary(paths))
}

func buildStrReplacePlaceholder(path, field string, editIndex int) string {
	path = singleLine(path)
	if editIndex >= 0 {
		return fmt.Sprintf("[omitted old str_replace.edits[%d].%s; path=%s]", editIndex, field, path)
	}
	return fmt.Sprintf("[omitted old str_replace.%s; path=%s]", field, path)
}

func pathSummary(paths []string) string {
	if len(paths) == 0 {
		return "unknown"
	}
	limit := len(paths)
	if limit > pathSummaryMax {
		limit = pathSummaryMax
	}
	parts := make([]string, 0, limit+1)
	for _, path := range paths[:limit] {
		parts = append(parts, singleLine(path))
	}
	if remaining := len(paths) - limit; remaining > 0 {
		parts = append(parts, fmt.Sprintf("+%d more", remaining))
	}
	return strings.Join(parts, ", ")
}

func sortedPathSet(paths map[string]struct{}) ([]string, bool) {
	if len(paths) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(paths))
	for path := range paths {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, true
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func normalizedLines(value string) []string {
	return strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
}

func singleLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func savedTokenDelta(originalTokens, replacementTokens int) int {
	if originalTokens <= replacementTokens {
		return 0
	}
	return originalTokens - replacementTokens
}
