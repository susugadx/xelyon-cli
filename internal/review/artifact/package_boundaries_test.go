package artifact

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

var reviewArtifactForbiddenImportRules = []importguard.Rule{
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		Message:    "internal/review/artifact must not import internal/agent; keep runner orchestration in internal/review",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		Message:    "internal/review/artifact must not import internal/tui; keep terminal concerns outside artifact persistence",
	},
	{
		ImportRoot: "github.com/charmbracelet/bubbletea",
		Message:    "internal/review/artifact must not import Bubble Tea directly",
	},
	{
		ImportRoot: "github.com/charmbracelet/lipgloss",
		Message:    "internal/review/artifact must not import Lip Gloss directly",
	},
}

func TestArchitectureBoundaries(t *testing.T) {
	importguard.AssertPackageBoundaries(t, importguard.PackageBoundaryOptions{
		PackageRoot: importguard.PackageRootFromCaller(t),
		ImportRules: reviewArtifactForbiddenImportRules,
	})
}
