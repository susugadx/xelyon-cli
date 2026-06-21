package evidence

import (
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

const (
	ReviewGenericImpactRoleSameStemTestOrSpec       = "same_stem_test_or_spec"
	ReviewGenericImpactRoleNearbyTestOrTestsDir     = "nearby_test_or_tests_dir"
	ReviewGenericImpactRoleNearbyProjectConfig      = "nearby_project_config"
	ReviewGenericImpactRoleDocsReference            = "docs_reference"
	ReviewGenericImpactRoleTextualReference         = "textual_reference"
	ReviewGenericImpactRoleChangedPathStemReference = "changed_path_stem_reference"

	reviewGenericImpactMaxTokens            = 12
	reviewGenericImpactMaxCandidatesTotal   = 40
	reviewGenericImpactMaxCandidatesPerRole = 8
	reviewGenericImpactMaxHitsPerToken      = 5
)

var (
	reviewGenericImpactDiffTokenReferenceSearches = []reviewGenericImpactReferenceSearch{
		{
			role:   ReviewGenericImpactRoleDocsReference,
			reason: "docs/readme reference to changed token",
			filter: reviewGenericImpactDocsSearchFilter,
		},
		{
			role:   ReviewGenericImpactRoleTextualReference,
			reason: "bounded token reference",
			filter: reviewGenericImpactTextualSearchFilter,
		},
	}
	reviewGenericImpactStemTokenReferenceSearch = reviewGenericImpactReferenceSearch{
		role:   ReviewGenericImpactRoleChangedPathStemReference,
		reason: "changed path stem reference",
		filter: reviewGenericImpactAllSearchFilter,
	}
	reviewGenericImpactExcludedPathParts = map[string]struct{}{
		".git":         {},
		"node_modules": {},
		"vendor":       {},
		"dist":         {},
		"build":        {},
		"coverage":     {},
	}
	reviewGenericImpactNearbyTestDirNames = []string{"test", "tests", "__tests__"}
	reviewGenericImpactDefaultIgnore      = pathmatch.NewMatcher(pathmatch.DefaultIgnorePatterns())
)

var reviewGenericImpactStopWords = map[string]struct{}{
	"and": {}, "are": {}, "case": {}, "const": {}, "else": {}, "false": {}, "for": {}, "func": {},
	"function": {}, "import": {}, "let": {}, "nil": {}, "package": {}, "return": {}, "struct": {},
	"the": {}, "true": {}, "type": {}, "var": {}, "with": {},
}
