package tools

import (
	"io"
	"testing"
)

// xmlTestTool は XML rescue テスト用のダミーツール。
// Run を実行するテストではないため、実行結果は空で返す。
type xmlTestTool struct {
	name string
}

func (t *xmlTestTool) Name() string                       { return t.name }
func (t *xmlTestTool) Description() string                { return "test tool" }
func (t *xmlTestTool) Parameters() map[string]interface{} { return nil }
func (t *xmlTestTool) Run(_ ExecutionContext, args map[string]string) (string, *FileChange, error) {
	return "", nil, nil
}

func parseToolCallsForXMLTest(t *testing.T, input string) []*ToolCall {
	t.Helper()
	return ParseToolCallsWithRegistry(input, newXMLTestRegistry(t), io.Discard)
}

// newXMLTestRegistry は XML rescue テスト専用レジストリを返す。
// DefaultRegistry への副作用を避けるため clone を使う。
func newXMLTestRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := DefaultRegistry.Clone()
	for _, name := range []string{"read_file", "list_dir", "bash", "write_file", "str_replace"} {
		if !registry.HasTool(name) {
			registry.Register(&xmlTestTool{name: name})
		}
	}
	return registry
}
