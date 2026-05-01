package history

import (
	"fmt"
	"strings"
)

const maxSessionIDLength = 128

func validateSessionID(sessionID string) error {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return fmt.Errorf("session ID is empty")
	}
	if len(trimmed) > maxSessionIDLength {
		return fmt.Errorf("session ID is too long")
	}
	if trimmed != sessionID {
		return fmt.Errorf("session ID must not include leading or trailing whitespace")
	}
	if sessionID == "." || sessionID == ".." {
		return fmt.Errorf("session ID is reserved")
	}
	if strings.ContainsAny(sessionID, `/\`) {
		return fmt.Errorf("session ID must not include path separators")
	}
	return nil
}
