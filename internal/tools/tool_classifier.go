package tools

// IsWriteTool はファイル変更系ツールかを判定する。
// bash は判定が複雑なため対象外。
func IsWriteTool(toolName string) bool {
	switch toolName {
	case "apply_patch", "write_file", "str_replace", "delete_file":
		return true
	}
	return false
}
