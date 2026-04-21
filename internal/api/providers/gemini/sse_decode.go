package gemini

import (
	"encoding/json"
	"strings"
)

func parseGeminiSSEDataLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}
	return strings.TrimPrefix(line, "data: "), true
}

func decodeGeminiSSEChunk(data string) (GeminiFunctionResponse, error) {
	var chunk GeminiFunctionResponse
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return GeminiFunctionResponse{}, err
	}
	return chunk, nil
}
