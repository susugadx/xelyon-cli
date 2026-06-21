package file

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/file/listtool"
	"github.com/susugadx/xelyon-cli/internal/tools/file/mutation"
	"github.com/susugadx/xelyon-cli/internal/tools/file/readtool"
)

type metadataTool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
}

func TestToolMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tool          metadataTool
		wantName      string
		wantProps     []string
		wantRequired  []string
		wantNoRequire bool
	}{
		{
			name:          "read_file",
			tool:          &readtool.ReadFileTool{},
			wantName:      "read_file",
			wantProps:     []string{"paths", "targets", "detail"},
			wantNoRequire: true,
		},
		{
			name:         "write_file",
			tool:         &mutation.WriteFileTool{},
			wantName:     "write_file",
			wantProps:    []string{"path", "content"},
			wantRequired: []string{"path", "content"},
		},
		{
			name:         "str_replace",
			tool:         &mutation.StrReplaceTool{},
			wantName:     "str_replace",
			wantProps:    []string{"path", "old_str", "new_str"},
			wantRequired: []string{"path"},
		},
		{
			name:         "delete_file",
			tool:         &mutation.DeleteFileTool{},
			wantName:     "delete_file",
			wantProps:    []string{"path"},
			wantRequired: []string{"path"},
		},
		{
			name:         "list_dir",
			tool:         &listtool.ListDirTool{},
			wantName:     "list_dir",
			wantProps:    []string{"path"},
			wantRequired: []string{"path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tool.Name(); got != tt.wantName {
				t.Fatalf("Name() = %q, want %q", got, tt.wantName)
			}
			if tt.tool.Description() == "" {
				t.Fatal("Description() should not be empty")
			}

			params := tt.tool.Parameters()
			props, ok := params["properties"].(map[string]interface{})
			if !ok {
				t.Fatal("Parameters() should have properties map")
			}
			for _, key := range tt.wantProps {
				if _, ok := props[key]; !ok {
					t.Fatalf("Parameters() should include property %q", key)
				}
			}

			if tt.wantNoRequire {
				if _, ok := params["required"]; ok {
					t.Fatal("Parameters() should not include required array")
				}
				return
			}

			required, ok := params["required"].([]string)
			if !ok {
				t.Fatal("Parameters() should have required array")
			}
			requiredSet := make(map[string]bool, len(required))
			for _, key := range required {
				requiredSet[key] = true
			}
			for _, key := range tt.wantRequired {
				if !requiredSet[key] {
					t.Fatalf("Parameters() required should include %q", key)
				}
			}
		})
	}
}
