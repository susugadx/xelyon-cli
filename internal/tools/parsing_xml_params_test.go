package tools

import "testing"

func TestParseXMLParams(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "with args wrapper",
			content: "<args>\n  <path>main.go</path>\n</args>",
			want:    map[string]string{"path": "main.go"},
		},
		{
			name:    "without args wrapper",
			content: "<path>main.go</path>\n<pattern>func main</pattern>",
			want:    map[string]string{"path": "main.go", "pattern": "func main"},
		},
		{
			name:    "single param",
			content: "<command>ls -la</command>",
			want:    map[string]string{"command": "ls -la"},
		},
		{
			name:    "empty content",
			content: "",
			want:    map[string]string{},
		},
		{
			name:    "JSON inside XML",
			content: `{"path": "main.go"}`,
			want:    map[string]string{"path": "main.go"},
		},
		{
			name:    "JSON with multiple keys",
			content: `{"pattern": "func main", "path": "."}`,
			want:    map[string]string{"pattern": "func main", "path": "."},
		},
		{
			name:    "tag params take precedence over trailing JSON-like text",
			content: "<path>main.go</path>\n{\"path\": \"fallback.go\"}",
			want:    map[string]string{"path": "main.go"},
		},
		{
			name:    "empty JSON object",
			content: `{}`,
			want:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseXMLParams(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("parseXMLParams() returned %d params, want %d: got=%v", len(got), len(tt.want), got)
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseXMLParams()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestXMLTagParamsParseStrategy_Parse_HandledContract(t *testing.T) {
	strategy := xmlTagParamsParseStrategy{}

	handled := strategy.Parse("<path>main.go</path>")
	if !handled.handled {
		t.Fatal("Parse(tag) handled = false, want true")
	}
	if handled.args["path"] != "main.go" {
		t.Fatalf("Parse(tag) args[path] = %q, want main.go", handled.args["path"])
	}

	unhandled := strategy.Parse(`{"path":"main.go"}`)
	if unhandled.handled {
		t.Fatal("Parse(json) handled = true, want false")
	}
}

func TestXMLJSONParamsParseStrategy_Parse_HandledContract(t *testing.T) {
	strategy := xmlJSONParamsParseStrategy{}

	unhandled := strategy.Parse("<path>main.go</path>")
	if unhandled.handled {
		t.Fatal("Parse(tag) handled = true, want false")
	}

	handled := strategy.Parse(`{"path":"main.go"}`)
	if !handled.handled {
		t.Fatal("Parse(json) handled = false, want true")
	}
	if handled.args["path"] != "main.go" {
		t.Fatalf("Parse(json) args[path] = %q, want main.go", handled.args["path"])
	}
}

func TestXMLJSONParamsParseStrategy_Parse_InvalidJSONHandled(t *testing.T) {
	strategy := xmlJSONParamsParseStrategy{}
	outcome := strategy.Parse(`{"path":`)
	if !outcome.handled {
		t.Fatal("Parse(invalid json) handled = false, want true")
	}
	if len(outcome.args) != 0 {
		t.Fatalf("Parse(invalid json) args len = %d, want 0", len(outcome.args))
	}
}
