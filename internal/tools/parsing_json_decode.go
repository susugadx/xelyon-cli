package tools

import "encoding/json"

type jsonToolCallDecoder struct {
	logger *parseDebugLogger
}

func (d *jsonToolCallDecoder) Decode(candidate jsonToolCallCandidate) (*ToolCall, bool) {
	return decodeToolCallJSON(candidate.json, d.logger)
}

func decodeToolCallJSON(jsonStr string, logger *parseDebugLogger) (*ToolCall, bool) {
	var toolCall ToolCall
	if !unmarshalToolCallJSONWithRepair(jsonStr, &toolCall, logger) {
		return nil, false
	}

	if toolCall.Tool == "" {
		logger.LogEvent(newParseDebugSkipEmptyToolFieldEvent())
		return nil, false
	}

	toolCall.NormalizeArgs()
	return &toolCall, true
}

func unmarshalToolCallJSONWithRepair(jsonStr string, toolCall *ToolCall, logger *parseDebugLogger) bool {
	err := json.Unmarshal([]byte(jsonStr), toolCall)
	if err == nil {
		return true
	}

	repaired := repairJSONStringValues(jsonStr)
	if repaired != jsonStr {
		if err2 := json.Unmarshal([]byte(repaired), toolCall); err2 == nil {
			logger.LogEvent(newParseDebugJSONRepairedEvent())
			return true
		}
	}

	logger.LogEvent(newParseDebugJSONParseErrorEvent(err))
	return false
}
