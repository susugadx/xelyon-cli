package probe

import (
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

type hostReadOnlyCommand struct {
	command     string
	commandPath string
	args        []string
	workDir     string
}

type hostReadOnlyRequest struct {
	id             string
	mode           domain.ReviewProbeMode
	timeout        time.Duration
	maxOutputBytes int64
	commands       []hostReadOnlyCommand
}

type hostReadOnlyRuntime struct {
	request hostReadOnlyRequest
	env     []string
	sandbox probeProcessSandbox
}
