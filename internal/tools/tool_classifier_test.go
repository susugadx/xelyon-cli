package tools

import "testing"

func TestIsWriteTool(t *testing.T) {
	writeTools := []string{
		"write_file", "str_replace", "grep_replace",
		"append_file", "prepend_file",
		"insert_after", "insert_before",
		"delete_lines", "delete_file", "move_file",
	}

	for _, name := range writeTools {
		if !IsWriteTool(name) {
			t.Errorf("IsWriteTool(%q) = false, want true", name)
		}
	}

	nonWriteTools := []string{
		"read_file", "read_files", "list_dir", "bash",
		"search_code", "search_file", "git_status",
		"git_diff", "git_log", "web_search", "format",
		"lint", "copy_file", "create_dir", "",
	}

	for _, name := range nonWriteTools {
		if IsWriteTool(name) {
			t.Errorf("IsWriteTool(%q) = true, want false", name)
		}
	}
}
