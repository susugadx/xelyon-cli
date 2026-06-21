package uistyle

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

func TestPackageBoundaries(t *testing.T) {
	rules := append(importguard.DefaultUIBoundaryRules("internal/uistyle"), importguard.Rule{
		ImportRoot: "github.com/susugadx/xelyon-cli/internal/uiruntime",
		Message:    "internal/uistyle must not import internal/uiruntime; runtime may consume low-level style, not the reverse",
	})
	importguard.AssertNoImports(t, importguard.PackageRootFromCaller(t), rules)
}
