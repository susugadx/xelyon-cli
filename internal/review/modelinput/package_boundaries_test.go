package modelinput

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

var reviewModelInputForbiddenImportRules = []importguard.Rule{
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/review/artifact",
		Message:    "internal/review/modelinput must not import review artifacts; artifact save timing belongs to internal/review runner",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		Message:    "internal/review/modelinput must not import internal/agent; keep runner/model orchestration outside deterministic input assembly",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		Message:    "internal/review/modelinput must not import internal/tui; keep terminal concerns outside model input assembly",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		Message:    "internal/review/modelinput must not import internal/tuiagent; keep TUI adapters outside model input assembly",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/api",
		Message:    "internal/review/modelinput must not import internal/api; keep provider payload concerns outside review model input assembly",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/commandruntime",
		Message:    "internal/review/modelinput must not import command runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/toolruntime",
		Message:    "internal/review/modelinput must not import tool runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/providerdiag",
		Message:    "internal/review/modelinput must not import provider runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/providerhistory",
		Message:    "internal/review/modelinput must not import provider runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/providerpicker",
		Message:    "internal/review/modelinput must not import provider runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/llmcatalog",
		Message:    "internal/review/modelinput must not import provider/model catalog packages",
	},
	{
		ImportRoot: "github.com/charmbracelet/bubbletea",
		Message:    "internal/review/modelinput must not import Bubble Tea directly",
	},
	{
		ImportRoot: "github.com/charmbracelet/lipgloss",
		Message:    "internal/review/modelinput must not import Lip Gloss directly",
	},
}

func TestArchitectureBoundaries(t *testing.T) {
	importguard.AssertPackageBoundaries(t, importguard.PackageBoundaryOptions{
		PackageRoot: importguard.PackageRootFromCaller(t),
		ImportRules: reviewModelInputForbiddenImportRules,
		ExactImportRules: []importguard.Rule{
			{
				ImportRoot: "github.com/susugadx/xelyon-cli/internal/review",
				Message:    "internal/review/modelinput must not import parent internal/review",
			},
		},
	})
}
