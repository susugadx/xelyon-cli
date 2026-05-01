package agent

import (
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestConvertToHistoryCompactedItems(t *testing.T) {
	tests := []struct {
		name  string
		items []api.InputItem
	}{
		{
			name:  "empty items",
			items: []api.InputItem{},
		},
		{
			name: "single item",
			items: []api.InputItem{
				{Type: "message", Role: "user", Content: "Hello"},
			},
		},
		{
			name: "multiple items with full input item fields",
			items: []api.InputItem{
				{Type: "message", Role: "user", Content: "Hello", ID: "1", Status: "active", Data: ""},
				{Type: "message", Role: "assistant", Content: "Hi there", ID: "2", Status: "complete", Data: ""},
				{Type: "function_call", CallID: "call_1", Name: "read_file", Arguments: `{"path":"README.md"}`, ThoughtSignature: "sig_1", ThoughtParts: []map[string]any{{"text": "thinking"}}},
				{Type: "function_call_output", CallID: "call_1", Output: "result data", ID: "3", Status: "", Data: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToHistoryCompactedItems(tt.items)

			if len(result) != len(tt.items) {
				t.Errorf("convertToHistoryCompactedItems() length = %d, want %d", len(result), len(tt.items))
				return
			}

			for i, item := range tt.items {
				if !reflect.DeepEqual(result[i], item) {
					t.Errorf("item[%d] = %#v, want %#v", i, result[i], item)
				}
			}
		})
	}
}

func TestConvertFromHistoryCompactedItems(t *testing.T) {
	tests := []struct {
		name  string
		items []history.CompactedItem
	}{
		{
			name:  "empty items",
			items: []history.CompactedItem{},
		},
		{
			name: "single item",
			items: []history.CompactedItem{
				{Type: "message", Role: "user", Content: "Hello"},
			},
		},
		{
			name: "multiple items with full input item fields",
			items: []history.CompactedItem{
				{Type: "message", Role: "user", Content: "Hello", ID: "1", Status: "active", Data: ""},
				{Type: "message", Role: "assistant", Content: "Hi there", ID: "2", Status: "complete", Data: ""},
				{Type: "function_call", CallID: "call_1", Name: "read_file", Arguments: `{"path":"README.md"}`, ThoughtSignature: "sig_1", ThoughtParts: []map[string]any{{"text": "thinking"}}},
				{Type: "function_call_output", CallID: "call_1", Output: "result data", ID: "3", Status: "", Data: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertFromHistoryCompactedItems(tt.items)

			if len(result) != len(tt.items) {
				t.Errorf("convertFromHistoryCompactedItems() length = %d, want %d", len(result), len(tt.items))
				return
			}

			for i, item := range tt.items {
				if !reflect.DeepEqual(result[i], item) {
					t.Errorf("item[%d] = %#v, want %#v", i, result[i], item)
				}
			}
		})
	}
}

func TestConvertRoundTrip(t *testing.T) {
	original := []api.InputItem{
		{Type: "message", Role: "user", Content: "Hello world", ID: "msg1", Status: "active"},
		{Type: "message", Role: "assistant", Content: "Hi!", ID: "msg2", Status: "complete"},
		{Type: "function_call", CallID: "call_1", Name: "read_file", Arguments: `{"path":"README.md"}`},
		{Type: "function_call_output", CallID: "call_1", Output: "file contents"},
	}

	historyItems := convertToHistoryCompactedItems(original)
	result := convertFromHistoryCompactedItems(historyItems)

	if len(result) != len(original) {
		t.Fatalf("Round trip failed: length mismatch %d != %d", len(result), len(original))
	}

	for i := range original {
		if !reflect.DeepEqual(result[i], original[i]) {
			t.Errorf("Round trip item[%d] = %#v, want %#v", i, result[i], original[i])
		}
	}
}
