package skills

import (
	"strings"
	"testing"
)

func TestRunSkillScriptTool_ParametersIncludeArgsJSONAndLegacyArgs(t *testing.T) {
	tool := &RunSkillScriptTool{}
	params := tool.Parameters()
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("properties missing: %#v", params)
	}

	argsProp, ok := properties["args"].(map[string]interface{})
	if !ok {
		t.Fatalf("args property missing: %#v", properties)
	}
	argsDesc, _ := argsProp["description"].(string)
	if !strings.Contains(argsDesc, "Legacy simple args") {
		t.Fatalf("args description should mark legacy usage, got: %q", argsDesc)
	}

	argsJSONProp, ok := properties["args_json"].(map[string]interface{})
	if !ok {
		t.Fatalf("args_json property missing: %#v", properties)
	}
	argsJSONDesc, _ := argsJSONProp["description"].(string)
	if !strings.Contains(argsJSONDesc, "JSON array of string arguments") {
		t.Fatalf("args_json description should describe JSON argv, got: %q", argsJSONDesc)
	}
}
