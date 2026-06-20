package plan

import "encoding/json"

func decodeRawJSONString(data []byte, target *string) error {
	return json.Unmarshal(data, target)
}
