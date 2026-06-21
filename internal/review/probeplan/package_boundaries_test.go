package probeplan

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

var probePlanForbiddenImportRules = []importguard.Rule{
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		Message:    "internal/review/probeplan must not import internal/agent; keep runner orchestration outside probe plan schema",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		Message:    "internal/review/probeplan must not import internal/tui; keep terminal concerns outside probe plan schema",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		Message:    "internal/review/probeplan must not import internal/tuiagent; keep TUI adapters outside probe plan schema",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/api",
		Message:    "internal/review/probeplan must not import internal/api; keep provider payload concerns outside probe plan schema",
	},
	{
		ImportRoot: "github.com/charmbracelet/bubbletea",
		Message:    "internal/review/probeplan must not import Bubble Tea directly",
	},
	{
		ImportRoot: "github.com/charmbracelet/lipgloss",
		Message:    "internal/review/probeplan must not import Lip Gloss directly",
	},
}

var probePlanForbiddenFacadeSymbols = map[string]string{
	"ReviewCommandIndex":             "internal/review/report owns evidence command indexes",
	"ReviewEvidenceKindDiff":         "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindExternalDoc":  "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindFile":         "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindGitStatus":    "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindProbe":        "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindProbeCommand": "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindRuleFile":     "internal/review/report owns evidence kinds",
	"ReviewEvidenceRef":              "internal/review/report owns evidence refs",
	"ReviewGroupSeverity":            "internal/review/report owns group severity",
	"ReviewGroupSeverityCritical":    "internal/review/report owns group severity values",
	"ReviewGroupSeverityHigh":        "internal/review/report owns group severity values",
	"ReviewGroupSeverityInfo":        "internal/review/report owns group severity values",
	"ReviewGroupSeverityLow":         "internal/review/report owns group severity values",
	"ReviewGroupSeverityMedium":      "internal/review/report owns group severity values",
	"ReviewProbeHostReadOnly":        "internal/review/domain owns probe modes",
	"ReviewProbeMode":                "internal/review/domain owns probe modes",
	"ReviewProbeRepoSandbox":         "internal/review/domain owns probe modes",
	"ReviewProbeScratchOnly":         "internal/review/domain owns probe modes",
	"TargetCurrentChanges":           "internal/review/domain owns target kinds",
	"TargetKind":                     "internal/review/domain owns target kinds",
}

func TestArchitectureBoundaries(t *testing.T) {
	importguard.AssertPackageBoundaries(t, importguard.PackageBoundaryOptions{
		PackageRoot:                 importguard.PackageRootFromCaller(t),
		ImportRules:                 probePlanForbiddenImportRules,
		ExactImportRules:            []importguard.Rule{{ImportRoot: "github.com/susugadx/xelyon-cli/internal/review", Message: "internal/review/probeplan must not import parent internal/review"}},
		ForbidGenericPackageNames:   true,
		ForbiddenPackageNameMessage: "review probe plan policy must not move into generic buckets",
		ForbiddenFacadeSymbols:      probePlanForbiddenFacadeSymbols,
		ForbidTypeAliases:           true,
		ExportedTypeAliasesOnly:     true,
		TypeAliasMessage:            "exported type alias reintroduces an owner-package facade",
	})
}
