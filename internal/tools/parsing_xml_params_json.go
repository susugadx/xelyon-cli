package tools

import (
	"encoding/json"
	"strings"
)

func parseXMLJSONParams(content string) map[string]string {
	args := make(map[string]string)
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return args
	}

	var jsonArgs map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &jsonArgs); err != nil {
		return args
	}

	for k, v := range jsonArgs {
		switch val := v.(type) {
		case string:
			args[k] = val
		default:
			if b, err := json.Marshal(v); err == nil {
				args[k] = string(b)
			}
		}
	}
	return args
}

func (xmlJSONParamsParseStrategy) Parse(content string) xmlParamsParseOutcome {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return unhandledXMLParamsOutcome()
	}
	return handledXMLParamsOutcome(parseXMLJSONParams(content))
}
