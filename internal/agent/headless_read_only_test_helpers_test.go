package agent

import (
	"encoding/json"
	"fmt"
	"testing"
)

func headlessToolCallJSON(t *testing.T, tool string, args map[string]string) string {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf(`{"tool":%q,"args":%s}`, tool, argsJSON)
}
