package review

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

var reviewForbiddenImportRules = []importguard.Rule{
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		Message:    "internal/review packages must not import internal/agent; keep agent adapters outside review orchestration",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		Message:    "internal/review packages must not import internal/tui; keep terminal concerns outside review orchestration",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		Message:    "internal/review packages must not import internal/tuiagent; keep TUI adapters outside review orchestration",
	},
	{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/api",
		Message:    "internal/review packages must not import internal/api; keep provider payloads behind ReviewModel",
	},
}

var reviewRootForbiddenForwarders = map[string]string{
	"BuildReviewProbeRequestsFromPlan":       "internal/review/probe owns probe runtime request conversion",
	"BuildReviewProbeSummaries":              "internal/review/probe owns probe summary construction",
	"ComputeReviewReportComputedSummary":     "internal/review/report owns report computed summaries",
	"DecodeReviewProbePlanJSON":              "internal/review/probeplan owns probe plan decoding",
	"DecodeReviewReportJSON":                 "internal/review/report owns report decoding",
	"DecodeReviewSaturationCheckJSON":        "internal/review/report owns saturation check decoding",
	"KnownReviewProbeKind":                   "internal/review/probeplan owns probe plan enums",
	"NewBufferedReviewRunArtifactWriter":     "internal/review/artifact owns artifact writers",
	"NewProbeRunner":                         "internal/review/probe owns probe execution",
	"NewReviewEvidenceBuilder":               "internal/review/evidence owns evidence collection",
	"NewReviewRunDirectoryArtifactWriter":    "internal/review/artifact owns artifact writers",
	"NewReviewWebSearchEvidenceCollector":    "internal/review/evidence owns web-search evidence collection",
	"ValidateReviewProbePlan":                "internal/review/probeplan owns probe plan validation",
	"ValidateReviewReport":                   "internal/review/report owns report validation",
	"ValidateReviewReportExternalDocSupport": "internal/review/report owns report validation",
	"ValidateReviewSaturationCheck":          "internal/review/report owns saturation validation",
	"WithReviewWebSearchEvidenceProvider":    "internal/review/evidence owns evidence builder options",
}

func TestArchitectureBoundaries(t *testing.T) {
	importguard.AssertPackageBoundaries(t, reviewPackageBoundaryOptions(t))
}

func TestReviewForbiddenImportRulesCoverSubpackages(t *testing.T) {
	opts := reviewPackageBoundaryOptions(t)
	if opts.ImportFiles != importguard.PackageBoundaryProductionGoFiles {
		t.Fatalf("review forbidden imports must cover subpackages, got %q", opts.ImportFiles)
	}
	if opts.FacadeSymbolFiles != importguard.PackageBoundaryRootProductionGoFiles {
		t.Fatalf("review facade guard must stay root-only, got %q", opts.FacadeSymbolFiles)
	}
}

func reviewPackageBoundaryOptions(t testing.TB) importguard.PackageBoundaryOptions {
	t.Helper()
	return importguard.PackageBoundaryOptions{
		PackageRoot:                 importguard.PackageRootFromCaller(t),
		ImportRules:                 reviewForbiddenImportRules,
		ImportFiles:                 importguard.PackageBoundaryProductionGoFiles,
		ForbidGenericPackageNames:   true,
		ForbiddenPackageNameMessage: "review domain policy must not move into generic buckets",
		ForbiddenFacadeSymbols:      reviewRootForbiddenForwarders,
		ForbidTypeAliases:           true,
		TypeAliasMessage:            "root internal/review must not be a facade over owner packages",
		FacadeSymbolFiles:           importguard.PackageBoundaryRootProductionGoFiles,
	}
}
