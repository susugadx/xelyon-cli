package search

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func setupSearchTestMocks(t *testing.T) {
	t.Helper()
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
}
