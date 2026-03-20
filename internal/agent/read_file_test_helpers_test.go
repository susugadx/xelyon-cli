package agent

import "encoding/json"

func testReadFileArgs(path string) map[string]string {
	pathsJSON, _ := json.Marshal([]string{path})
	return map[string]string{"paths": string(pathsJSON)}
}

func testReadFileRawArgs(path string) map[string]any {
	return map[string]any{"paths": []string{path}}
}
