package file

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func resolveValidatedPath(out common.Output, path, emptyMessage string) (string, string) {
	if path == "" {
		return "", "Error: " + emptyMessage
	}

	absPath, err := common.ValidatePath(path)
	if err != nil {
		out.Red.Printf("🚫 Security: %v\n", err)
		return "", fmt.Sprintf("Error: %v", err)
	}
	return absPath, ""
}
