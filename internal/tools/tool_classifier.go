package tools

// IsWriteTool はファイル変更系ツールかを判定する。
// bash は判定が複雑なため対象外。
func IsWriteTool(toolName string) bool {
	switch toolName {
	case "write_file", "str_replace",
		"append_file", "prepend_file",
		"insert_after", "insert_before",
		"delete_lines", "delete_file", "move_file":
		return true
	}
	return false
}
