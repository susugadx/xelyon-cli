package ui

import (
	"strings"
	"testing"
)

func TestNewTable(t *testing.T) {
	table := NewTable()
	if table == nil {
		t.Fatal("NewTable returned nil")
	}
	if table.Headers != nil {
		t.Error("Headers should be nil initially")
	}
	if table.Rows != nil {
		t.Error("Rows should be nil initially")
	}
}

func TestTable_SetHeaders(t *testing.T) {
	table := NewTable().SetHeaders("Name", "Value")
	if len(table.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(table.Headers))
	}
	if table.Headers[0] != "Name" || table.Headers[1] != "Value" {
		t.Error("Headers not set correctly")
	}
}

func TestTable_AddRow(t *testing.T) {
	table := NewTable().AddRow("foo", "bar").AddRow("baz", "qux")
	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.Rows))
	}
	if table.Rows[0][0] != "foo" || table.Rows[0][1] != "bar" {
		t.Error("First row not set correctly")
	}
}

func TestTable_Render_Empty(t *testing.T) {
	table := NewTable()
	result := table.Render()
	if result != "" {
		t.Errorf("Expected empty string for empty table, got %q", result)
	}
}

func TestTable_Render_Simple(t *testing.T) {
	table := NewTable().
		AddRow("Provider", "OpenAI").
		AddRow("Model", "gpt-5.2-codex")

	result := table.Render()

	// テーブルの基本構造を検証
	if !strings.Contains(result, "┌") {
		t.Error("Missing top-left corner")
	}
	if !strings.Contains(result, "┐") {
		t.Error("Missing top-right corner")
	}
	if !strings.Contains(result, "└") {
		t.Error("Missing bottom-left corner")
	}
	if !strings.Contains(result, "┘") {
		t.Error("Missing bottom-right corner")
	}
	if !strings.Contains(result, "│") {
		t.Error("Missing vertical lines")
	}
	if !strings.Contains(result, "Provider") {
		t.Error("Missing cell content 'Provider'")
	}
	if !strings.Contains(result, "OpenAI") {
		t.Error("Missing cell content 'OpenAI'")
	}
}

func TestTable_Render_WithHeaders(t *testing.T) {
	table := NewTable().
		SetHeaders("Key", "Value").
		AddRow("Provider", "OpenAI").
		AddRow("Model", "gpt-5.2")

	result := table.Render()

	if !strings.Contains(result, "Key") {
		t.Error("Missing header 'Key'")
	}
	if !strings.Contains(result, "Value") {
		t.Error("Missing header 'Value'")
	}
	// ヘッダーと本体の区切り線
	if !strings.Contains(result, "├") {
		t.Error("Missing header separator")
	}
}

func TestTable_RenderCompact(t *testing.T) {
	table := NewTable().
		AddRow("A", "1").
		AddRow("B", "2").
		AddRow("C", "3")

	result := table.RenderCompact()

	// コンパクト形式では行間の区切り線がないことを確認
	lines := strings.Split(result, "\n")
	separatorCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "├") {
			separatorCount++
		}
	}
	// ヘッダーがないので区切り線は0個
	if separatorCount != 0 {
		t.Errorf("Expected 0 separators in compact mode without headers, got %d", separatorCount)
	}
}

func TestStringWidth_ASCII(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello", 5},
		{"", 0},
		{"a", 1},
		{"Hello World", 11},
	}

	for _, tt := range tests {
		result := StringWidth(tt.input)
		if result != tt.expected {
			t.Errorf("StringWidth(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestStringWidth_Japanese(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"日本語", 6},    // 全角3文字 = 6
		{"あ", 2},       // 全角1文字 = 2
		{"テスト", 6},    // 全角3文字 = 6
		{"Hello日本語", 11}, // 5 + 6 = 11
	}

	for _, tt := range tests {
		result := StringWidth(tt.input)
		if result != tt.expected {
			t.Errorf("StringWidth(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"abc", 5, "abc  "},
		{"abc", 3, "abc"},
		{"abc", 2, "abc"},
		{"", 3, "   "},
	}

	for _, tt := range tests {
		result := PadRight(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("PadRight(%q, %d) = %q, expected %q", tt.input, tt.width, result, tt.expected)
		}
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"abc", 5, "  abc"},
		{"abc", 3, "abc"},
		{"abc", 2, "abc"},
	}

	for _, tt := range tests {
		result := PadLeft(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("PadLeft(%q, %d) = %q, expected %q", tt.input, tt.width, result, tt.expected)
		}
	}
}

func TestPadCenter(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"abc", 7, "  abc  "},
		{"abc", 6, " abc  "},
		{"abc", 3, "abc"},
	}

	for _, tt := range tests {
		result := PadCenter(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("PadCenter(%q, %d) = %q, expected %q", tt.input, tt.width, result, tt.expected)
		}
	}
}

func TestTruncateWidth(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"", 5, ""},
		{"hi", 0, ""},
	}

	for _, tt := range tests {
		result := TruncateWidth(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("TruncateWidth(%q, %d) = %q, expected %q", tt.input, tt.width, result, tt.expected)
		}
	}
}

func TestKeyValueTable(t *testing.T) {
	result := KeyValueTable("Provider", "OpenAI", "Model", "gpt-5.2")

	if !strings.Contains(result, "Provider") {
		t.Error("Missing key 'Provider'")
	}
	if !strings.Contains(result, "OpenAI") {
		t.Error("Missing value 'OpenAI'")
	}
	if !strings.Contains(result, "Model") {
		t.Error("Missing key 'Model'")
	}
}

func TestKeyValueTable_Empty(t *testing.T) {
	result := KeyValueTable()
	if result != "" {
		t.Errorf("Expected empty string for no pairs, got %q", result)
	}
}

func TestKeyValueTable_OddPairs(t *testing.T) {
	result := KeyValueTable("key1", "val1", "key2")
	if result != "" {
		t.Errorf("Expected empty string for odd number of arguments, got %q", result)
	}
}

func TestSimpleTable(t *testing.T) {
	headers := []string{"Name", "Age"}
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "25"},
	}

	result := SimpleTable(headers, rows)

	if !strings.Contains(result, "Name") {
		t.Error("Missing header 'Name'")
	}
	if !strings.Contains(result, "Alice") {
		t.Error("Missing data 'Alice'")
	}
}

func TestCountRunes(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello", 5},
		{"日本語", 3},
		{"Hello日本語", 8},
		{"", 0},
	}

	for _, tt := range tests {
		result := CountRunes(tt.input)
		if result != tt.expected {
			t.Errorf("CountRunes(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestTable_JapaneseContent(t *testing.T) {
	table := NewTable().
		AddRow("プロバイダー", "OpenAI").
		AddRow("モデル", "gpt-5.2-codex")

	result := table.Render()

	// 日本語コンテンツが含まれていることを確認
	if !strings.Contains(result, "プロバイダー") {
		t.Error("Missing Japanese content 'プロバイダー'")
	}
	if !strings.Contains(result, "モデル") {
		t.Error("Missing Japanese content 'モデル'")
	}

	// テーブルの罫線が正しく描画されていることを確認
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "│") {
			// 各行が │ で終わることを確認
			if !strings.HasSuffix(line, "│") {
				t.Errorf("Row does not end with │: %q", line)
			}
		}
	}
}
