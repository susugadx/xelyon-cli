package openai

import (
	"encoding/json"
	"strings"
)

func parseResponsesSSEDataLine(line string) (data string, done bool, handled bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, false
	}

	data = strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, true
	}

	return data, false, true
}

func decodeResponsesStreamChunk(data string) (ResponsesStreamChunk, error) {
	var chunk ResponsesStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return ResponsesStreamChunk{}, err
	}
	return chunk, nil
}
