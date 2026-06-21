package configedit

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil/importguard"
)

func TestPackageBoundaries(t *testing.T) {
	importguard.AssertNoImports(t, importguard.PackageRootFromCaller(t), importguard.DefaultUIBoundaryRules("internal/configedit"))
}
