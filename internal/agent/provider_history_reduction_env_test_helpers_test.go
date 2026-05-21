package agent

import (
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func unsetProviderHistoryReductionEnv(t *testing.T) {
	t.Helper()
	oldValue, oldSet := os.LookupEnv(config.ProviderHistoryReductionEnvVar)
	if err := os.Unsetenv(config.ProviderHistoryReductionEnvVar); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
	t.Cleanup(func() {
		if oldSet {
			_ = os.Setenv(config.ProviderHistoryReductionEnvVar, oldValue)
		} else {
			_ = os.Unsetenv(config.ProviderHistoryReductionEnvVar)
		}
	})
}
