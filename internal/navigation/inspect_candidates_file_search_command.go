package navigation

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const goFileSearchTimeout = 10 * time.Second

func runGoFileSearch(searchPath string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), goFileSearchTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, common.RipgrepPath(), goFileSearchArgs(searchPath)...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return "", false
	}
	return stdout.String(), true
}

func goFileSearchArgs(searchPath string) []string {
	return []string{
		"--files",
		"--type", "go",
		"--glob", "!vendor/",
		searchPath,
	}
}
