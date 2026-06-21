package modeloutput

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

var reviewModelOutputForbiddenImportRules = []importguard.Rule{
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/review/evidence",
		Message:    "internal/review/modeloutput must not import review evidence; receive fetched external docs as input",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/review/artifact",
		Message:    "internal/review/modeloutput must not import review artifacts; artifact save timing belongs to internal/review runner",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/review/modelinput",
		Message:    "internal/review/modeloutput must not import modelinput; prompt assembly and output finalization are separate boundaries",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		Message:    "internal/review/modeloutput must not import internal/agent; keep runner/model orchestration outside deterministic output finalization",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		Message:    "internal/review/modeloutput must not import internal/tui; keep terminal concerns outside model output finalization",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		Message:    "internal/review/modeloutput must not import internal/tuiagent; keep TUI adapters outside model output finalization",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/api",
		Message:    "internal/review/modeloutput must not import internal/api; keep provider payload concerns outside review model output finalization",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/commandruntime",
		Message:    "internal/review/modeloutput must not import command runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/toolruntime",
		Message:    "internal/review/modeloutput must not import tool runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/providerdiag",
		Message:    "internal/review/modeloutput must not import provider runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/providerhistory",
		Message:    "internal/review/modeloutput must not import provider runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/providerpicker",
		Message:    "internal/review/modeloutput must not import provider runtime packages",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/llmcatalog",
		Message:    "internal/review/modeloutput must not import provider/model catalog packages",
	},
	{
		ImportRoot: "github.com/charmbracelet/bubbletea",
		Message:    "internal/review/modeloutput must not import Bubble Tea directly",
	},
	{
		ImportRoot: "github.com/charmbracelet/lipgloss",
		Message:    "internal/review/modeloutput must not import Lip Gloss directly",
	},
}

func TestArchitectureBoundaries(t *testing.T) {
	importguard.AssertPackageBoundaries(t, importguard.PackageBoundaryOptions{
		PackageRoot: importguard.PackageRootFromCaller(t),
		ImportRules: reviewModelOutputForbiddenImportRules,
		ExactImportRules: []importguard.Rule{
			{
				ImportRoot: "github.com/susugadx/xelyon-cli/internal/review",
				Message:    "internal/review/modeloutput must not import parent internal/review",
			},
		},
	})
}
