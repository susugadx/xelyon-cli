package tools

import "testing"

func TestParseToolCalls_XMLRescue_UnknownToolIgnored(t *testing.T) {
	// "unknown_tool" は registry に未登録なのでスキップされる。
	input := `<unknown_tool>
<param1>value1</param1>
</unknown_tool>`

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls, want 0 (unknown tool)", len(result))
	}
}

func TestParseToolCalls_XMLRescue_InCodeBlockIgnored(t *testing.T) {
	input := "Example:\n```\n<read_file>\n<path>test.go</path>\n</read_file>\n```"

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls, want 0 (in code block)", len(result))
	}
}
