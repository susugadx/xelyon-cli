//go:build !unix && !windows

package azure

import (
	"context"
	"os/exec"
	"time"
)

const azureAuthTokenCommandWaitDelay = 2 * time.Second

func azureAuthShellCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.WaitDelay = azureAuthTokenCommandWaitDelay
	return cmd
}
