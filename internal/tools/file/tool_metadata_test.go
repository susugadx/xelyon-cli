package file

import (
	"testing"
)

// --- ReadFileTool ---

func TestReadFileTool_Name(t *testing.T) {
	tool := &ReadFileTool{}
	if got := tool.Name(); got != "read_file" {
		t.Errorf("ReadFileTool.Name() = %q, want %q", got, "read_file")
	}
}

func TestReadFileTool_Description(t *testing.T) {
	tool := &ReadFileTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("ReadFileTool.Description() should not be empty")
	}
}

func TestReadFileTool_Parameters(t *testing.T) {
	tool := &ReadFileTool{}
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("ReadFileTool.Parameters() should have properties map")
	}
	if _, ok := props["paths"]; !ok {
		t.Error("ReadFileTool.Parameters() should have 'paths' key in properties")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("ReadFileTool.Parameters() should have required array")
	}
	found := false
	for _, r := range required {
		if r == "paths" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ReadFileTool.Parameters() required should include 'paths'")
	}
}

// --- WriteFileTool ---

func TestWriteFileTool_Name(t *testing.T) {
	tool := &WriteFileTool{}
	if got := tool.Name(); got != "write_file" {
		t.Errorf("WriteFileTool.Name() = %q, want %q", got, "write_file")
	}
}

func TestWriteFileTool_Description(t *testing.T) {
	tool := &WriteFileTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("WriteFileTool.Description() should not be empty")
	}
}

func TestWriteFileTool_Parameters(t *testing.T) {
	tool := &WriteFileTool{}
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("WriteFileTool.Parameters() should have properties map")
	}
	for _, key := range []string{"path", "content"} {
		if _, ok := props[key]; !ok {
			t.Errorf("WriteFileTool.Parameters() should have '%s' key in properties", key)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("WriteFileTool.Parameters() should have required array")
	}
	requiredSet := make(map[string]bool, len(required))
	for _, r := range required {
		requiredSet[r] = true
	}
	for _, key := range []string{"path", "content"} {
		if !requiredSet[key] {
			t.Errorf("WriteFileTool.Parameters() required should include '%s'", key)
		}
	}
}

// --- StrReplaceTool ---

func TestStrReplaceTool_Name(t *testing.T) {
	tool := &StrReplaceTool{}
	if got := tool.Name(); got != "str_replace" {
		t.Errorf("StrReplaceTool.Name() = %q, want %q", got, "str_replace")
	}
}

func TestStrReplaceTool_Description(t *testing.T) {
	tool := &StrReplaceTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("StrReplaceTool.Description() should not be empty")
	}
}

func TestStrReplaceTool_Parameters(t *testing.T) {
	tool := &StrReplaceTool{}
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("StrReplaceTool.Parameters() should have properties map")
	}
	for _, key := range []string{"path", "old_str", "new_str"} {
		if _, ok := props[key]; !ok {
			t.Errorf("StrReplaceTool.Parameters() should have '%s' key in properties", key)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("StrReplaceTool.Parameters() should have required array")
	}
	requiredSet := make(map[string]bool, len(required))
	for _, r := range required {
		requiredSet[r] = true
	}
	if !requiredSet["path"] {
		t.Error("StrReplaceTool.Parameters() required should include 'path'")
	}
}

// --- DeleteFileTool ---

func TestDeleteFileTool_Name(t *testing.T) {
	tool := &DeleteFileTool{}
	if got := tool.Name(); got != "delete_file" {
		t.Errorf("DeleteFileTool.Name() = %q, want %q", got, "delete_file")
	}
}

func TestDeleteFileTool_Description(t *testing.T) {
	tool := &DeleteFileTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("DeleteFileTool.Description() should not be empty")
	}
}

func TestDeleteFileTool_Parameters(t *testing.T) {
	tool := &DeleteFileTool{}
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("DeleteFileTool.Parameters() should have properties map")
	}
	if _, ok := props["path"]; !ok {
		t.Error("DeleteFileTool.Parameters() should have 'path' key in properties")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("DeleteFileTool.Parameters() should have required array")
	}
	found := false
	for _, r := range required {
		if r == "path" {
			found = true
			break
		}
	}
	if !found {
		t.Error("DeleteFileTool.Parameters() required should include 'path'")
	}
}

// --- ListDirTool ---

func TestListDirTool_Name(t *testing.T) {
	tool := &ListDirTool{}
	if got := tool.Name(); got != "list_dir" {
		t.Errorf("ListDirTool.Name() = %q, want %q", got, "list_dir")
	}
}

func TestListDirTool_Description(t *testing.T) {
	tool := &ListDirTool{}
	desc := tool.Description()
	if desc == "" {
		t.Error("ListDirTool.Description() should not be empty")
	}
}

func TestListDirTool_Parameters(t *testing.T) {
	tool := &ListDirTool{}
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("ListDirTool.Parameters() should have properties map")
	}
	if _, ok := props["path"]; !ok {
		t.Error("ListDirTool.Parameters() should have 'path' key in properties")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("ListDirTool.Parameters() should have required array")
	}
	found := false
	for _, r := range required {
		if r == "path" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListDirTool.Parameters() required should include 'path'")
	}
}
