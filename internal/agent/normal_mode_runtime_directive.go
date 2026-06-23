package agent

import "strings"

const (
	normalModeBaseRuntimeDirective    = "Normal mode is active. Investigate as needed, implement with available tools, verify the result, then summarize changes when done."
	finalCheckFailureRuntimeDirective = "Final checks failed. Use the provided command output and diff context to fix the failure before declaring completion."
	strReplaceLoopRuntimeDirective    = "str_replace failed repeatedly because the target text was not found. Inspect the target file or symbol before retrying, and do not repeat the same old_str pattern."
)

func (a *Agent) queueRuntimeDirective(directive string) {
	if a == nil {
		return
	}
	directive = strings.TrimSpace(directive)
	if directive == "" {
		return
	}
	a.runtimeDirectives = append(a.runtimeDirectives, directive)
}

func (a *Agent) pendingRuntimeDirectives() []string {
	if a == nil || len(a.runtimeDirectives) == 0 {
		return nil
	}
	return append([]string(nil), a.runtimeDirectives...)
}

func (a *Agent) markRuntimeDirectivesDelivered(delivered []string) {
	if a == nil || len(delivered) == 0 || len(a.runtimeDirectives) == 0 {
		return
	}

	remaining := append([]string(nil), a.runtimeDirectives...)
	for _, directive := range delivered {
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}
		for i, pending := range remaining {
			if strings.TrimSpace(pending) != directive {
				continue
			}
			remaining = append(remaining[:i], remaining[i+1:]...)
			break
		}
	}
	a.runtimeDirectives = remaining
}

func appendRuntimeDirectivesToSystemPrompt(systemPrompt string, directives ...string) string {
	cleaned := make([]string, 0, len(directives))
	seen := make(map[string]struct{}, len(directives))
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}
		if _, ok := seen[directive]; ok {
			continue
		}
		seen[directive] = struct{}{}
		cleaned = append(cleaned, directive)
	}
	if len(cleaned) == 0 {
		return strings.TrimRight(systemPrompt, "\n")
	}

	var b strings.Builder
	if trimmed := strings.TrimRight(systemPrompt, "\n"); trimmed != "" {
		b.WriteString(trimmed)
		b.WriteString("\n\n")
	}
	b.WriteString("### Runtime Directives\n")
	for _, directive := range cleaned {
		b.WriteString("- ")
		b.WriteString(strings.ReplaceAll(directive, "\n", "\n  "))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
