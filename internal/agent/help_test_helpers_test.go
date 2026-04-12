package agent

import "github.com/susugadx/xelyon-cli/internal/tools"

// helpTestTool は /help の section builder テスト専用の最小 fake tool。
type helpTestTool struct {
	name        string
	description string
}

func (t *helpTestTool) Name() string { return t.name }

func (t *helpTestTool) Description() string { return t.description }

func (t *helpTestTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *helpTestTool) Run(_ tools.ExecutionContext, _ map[string]string) (string, *tools.FileChange, error) {
	return "", nil, nil
}
