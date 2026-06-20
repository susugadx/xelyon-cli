package promptreduction

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

var promptReductionForbiddenImportRules = []importguard.Rule{
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		Message:    "internal/review/promptreduction must not import internal/agent; keep runtime orchestration outside prompt reduction",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		Message:    "internal/review/promptreduction must not import internal/tui; keep terminal concerns outside prompt reduction",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		Message:    "internal/review/promptreduction must not import internal/tuiagent; keep TUI adapters outside prompt reduction",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/api",
		Message:    "internal/review/promptreduction must not import internal/api; keep provider payload concerns outside prompt reduction",
	},
	{
		ImportRoot: "github.com/charmbracelet/bubbletea",
		Message:    "internal/review/promptreduction must not import Bubble Tea directly",
	},
	{
		ImportRoot: "github.com/charmbracelet/lipgloss",
		Message:    "internal/review/promptreduction must not import Lip Gloss directly",
	},
}

var promptReductionForbiddenFacadeSymbols = map[string]string{
	"ReviewChangeInventory":                               "internal/review/evidence owns change inventory",
	"ReviewChangedFile":                                   "internal/review/evidence owns changed file evidence",
	"ReviewCommandIndex":                                  "internal/review/report owns evidence command indexes",
	"ReviewContextFileEvidence":                           "internal/review/evidence owns context file evidence",
	"ReviewDiffEvidence":                                  "internal/review/evidence owns diff evidence",
	"ReviewEvidenceBundle":                                "internal/review/evidence owns evidence bundles",
	"ReviewEvidenceKindExternalDoc":                       "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindProbe":                             "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindProbeCommand":                      "internal/review/report owns evidence kinds",
	"ReviewEvidenceRef":                                   "internal/review/report owns evidence refs",
	"ReviewExternalDocEvidence":                           "internal/review/externaldoc owns external doc evidence",
	"ReviewExternalDocSnippetEvidence":                    "internal/review/externaldoc owns external doc snippets",
	"ReviewExternalDocSourceCredibility":                  "internal/review/externaldoc owns source credibility",
	"ReviewExternalDocSourceCredibilityOfficialCandidate": "internal/review/externaldoc owns source credibility values",
	"ReviewGenericImpactCandidate":                        "internal/review/evidence owns generic impact candidates",
	"ReviewProbeCandidateRiskNeedsProbe":                  "internal/review/probeplan owns candidate risk statuses",
	"ReviewProbeCandidateRiskStatus":                      "internal/review/probeplan owns candidate risk statuses",
	"ReviewProbeCandidateRiskUnverified":                  "internal/review/probeplan owns candidate risk statuses",
	"ReviewProbePlan":                                     "internal/review/probeplan owns probe plan schema",
	"ReviewProbeSummary":                                  "internal/review/report owns probe summaries",
	"ReviewRelatedSearchHit":                              "internal/review/evidence owns related search hits",
	"ReviewReport":                                        "internal/review/report owns reports",
	"ReviewReportCandidateRiskDismissed":                  "internal/review/report owns report candidate risk statuses",
	"ReviewReportCandidateRiskResidualRisk":               "internal/review/report owns report candidate risk statuses",
	"ReviewReportCandidateRiskUnverified":                 "internal/review/report owns report candidate risk statuses",
	"ReviewReportImpactSurfaceChecked":                    "internal/review/report owns report impact surface statuses",
	"ReviewRuleFileEvidence":                              "internal/review/evidence owns rule file evidence",
	"ReviewSaturationCheck":                               "internal/review/report owns saturation checks",
	"ReviewUntrackedFile":                                 "internal/review/evidence owns untracked file evidence",
	"ReviewWebSearchEvidence":                             "internal/review/externaldoc owns web search evidence",
	"ReviewWebSearchEvidenceQuery":                        "internal/review/externaldoc owns web search evidence queries",
	"ReviewWebSearchEvidenceResult":                       "internal/review/externaldoc owns web search evidence results",
	"TargetCurrentChanges":                                "internal/review/domain owns target kinds",
}

func TestArchitectureBoundaries(t *testing.T) {
	importguard.AssertPackageBoundaries(t, importguard.PackageBoundaryOptions{
		PackageRoot:                 importguard.PackageRootFromCaller(t),
		ImportRules:                 promptReductionForbiddenImportRules,
		ExactImportRules:            []importguard.Rule{{ImportRoot: "github.com/susugadx/xelyon-cli/internal/review", Message: "internal/review/promptreduction must not import parent internal/review"}},
		ForbidGenericPackageNames:   true,
		ForbiddenPackageNameMessage: "review prompt reduction policy must not move into generic buckets",
		ForbiddenFacadeSymbols:      promptReductionForbiddenFacadeSymbols,
		ForbidTypeAliases:           true,
		ExportedTypeAliasesOnly:     true,
		TypeAliasMessage:            "exported type alias reintroduces an owner-package facade",
	})
}
