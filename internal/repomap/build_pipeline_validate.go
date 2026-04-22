package repomap

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func (pm *ProjectMap) validateBuildPreconditions() error {
	if pm == nil {
		return fmt.Errorf("project map is nil")
	}
	if !common.IsRipgrepAvailable() {
		return fmt.Errorf("ripgrep (rg) is required")
	}
	return nil
}
