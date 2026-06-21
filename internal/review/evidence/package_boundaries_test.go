package evidence

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

var reviewEvidenceForbiddenImportRules = []importguard.Rule{
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		Message:    "internal/review/evidence must not import internal/agent; keep runner orchestration in internal/review",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		Message:    "internal/review/evidence must not import internal/tui; keep terminal concerns outside evidence collection",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/api",
		Message:    "internal/review/evidence must not import internal/api; keep provider payload concerns outside evidence collection",
	},
	{
		ImportRoot: "github.com/charmbracelet/bubbletea",
		Message:    "internal/review/evidence must not import Bubble Tea directly",
	},
	{
		ImportRoot: "github.com/charmbracelet/lipgloss",
		Message:    "internal/review/evidence must not import Lip Gloss directly",
	},
}

var reviewEvidenceForbiddenFacadeSymbols = map[string]string{
	"BuildReviewAnalysisEvidenceInput": "internal/review/modelinput owns evidence-to-analysis model input conversion",
	"BuildReviewEvidenceModelInput":    "internal/review/modelinput owns evidence model input DTO construction",
	"BuildReviewPressureSignalInputs":  "internal/review/modelinput owns pressure-signal input conversion",
	"NewHTTPReviewExternalDocFetcher":  "internal/review/externaldoc owns HTTP external doc fetchers",
	"RenderReviewEvidenceJSON":         "internal/review/modelinput owns evidence JSON rendering",
	"RenderReviewEvidenceMarkdown":     "internal/review/modelinput owns evidence Markdown rendering",
	"ReviewExternalDocFetchRequest":    "internal/review/externaldoc owns external doc fetch requests",
	"ReviewExternalDocFetcher":         "internal/review/externaldoc owns external doc fetchers",
	"ReviewExternalDocFocusTerm":       "internal/review/externaldoc owns external doc focus terms",
	"ReviewExternalDocSnippetEvidence": "internal/review/externaldoc owns external doc snippets",
	"ReviewExternalDocEvidence":        "internal/review/externaldoc owns external doc evidence",
	"ReviewExternalSupportSummary":     "internal/review/externaldoc owns external support summaries",
	"ReviewPressureSignalInput":        "internal/review/analysis owns pressure signals",
	"ReviewWebSearchEvidence":          "internal/review/externaldoc owns web search evidence",
	"ReviewWebSearchEvidenceQuery":     "internal/review/externaldoc owns web search evidence queries",
	"ReviewWebSearchEvidenceResult":    "internal/review/externaldoc owns web search evidence results",
	"ReviewWebSearchQueryResult":       "internal/review/externaldoc owns web search query results",
	"HTTPReviewExternalDocFetcher":     "internal/review/externaldoc owns HTTP external doc fetchers",
	"TargetCurrentChanges":             "internal/review/domain owns target kinds",
	"TargetKind":                       "internal/review/domain owns target kinds",
}

func TestArchitectureBoundaries(t *testing.T) {
	importguard.AssertPackageBoundaries(t, importguard.PackageBoundaryOptions{
		PackageRoot: importguard.PackageRootFromCaller(t),
		ImportRules: reviewEvidenceForbiddenImportRules,
		ExactImportRules: []importguard.Rule{
			{
				ImportRoot: "github.com/susugadx/xelyon-cli/internal/review",
				Message:    "internal/review/evidence must not import parent internal/review",
			},
		},
		ForbiddenFacadeSymbols:  reviewEvidenceForbiddenFacadeSymbols,
		ForbidTypeAliases:       true,
		ExportedTypeAliasesOnly: true,
		TypeAliasMessage:        "exported type alias reintroduces an owner-package facade",
	})
}
