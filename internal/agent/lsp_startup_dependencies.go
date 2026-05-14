package agent

import (
	"os"
	"os/exec"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

var (
	lspStartupGetwd           = os.Getwd
	lspDetectProjectLanguages = lsp.DetectProjectLanguages
	lspGetInstallInfo         = lsp.GetInstallInfo
	lspLookPath               = exec.LookPath
)
