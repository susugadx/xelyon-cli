package domain

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

var reviewDomainForbiddenImportRules = []importguard.Rule{
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		Message:    "internal/review/domain must not import internal/agent; keep runner orchestration outside domain types",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		Message:    "internal/review/domain must not import internal/tui; keep terminal concerns outside domain types",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/api",
		Message:    "internal/review/domain must not import internal/api; keep provider payload concerns outside domain types",
	},
	{
		ImportRoot: "github.com/charmbracelet/bubbletea",
		Message:    "internal/review/domain must not import Bubble Tea directly",
	},
	{
		ImportRoot: "github.com/charmbracelet/lipgloss",
		Message:    "internal/review/domain must not import Lip Gloss directly",
	},
}

func TestArchitectureBoundaries(t *testing.T) {
	importguard.AssertPackageBoundaries(t, importguard.PackageBoundaryOptions{
		PackageRoot: importguard.PackageRootFromCaller(t),
		ImportRules: reviewDomainForbiddenImportRules,
	})
}
