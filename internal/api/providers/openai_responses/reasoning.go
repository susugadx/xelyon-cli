package openairesponses

// ReasoningEffortFromThinkingLevel は Thinking Level を Responses reasoning effort に変換する。
func ReasoningEffortFromThinkingLevel(level string) string {
	switch level {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "xhigh":
		return "xhigh"
	default:
		return "medium"
	}
}
