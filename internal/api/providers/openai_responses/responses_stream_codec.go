package openairesponses

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

func decodeStreamChunk(data string) (StreamChunk, error) {
	var chunk StreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return StreamChunk{}, err
	}
	return chunk, nil
}
