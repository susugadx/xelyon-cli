package uitoolview

// SpinnerMessageForTool はツール名に応じたスピナーメッセージを返す。
func SpinnerMessageForTool(toolName string) string {
	switch toolName {
	case "write_file":
		return "Writing file..."
	case "str_replace":
		return "Editing file..."
	default:
		return "Preparing..."
	}
}
