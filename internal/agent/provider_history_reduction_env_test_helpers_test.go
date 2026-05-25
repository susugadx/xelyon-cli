package agent

import (
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func unsetProviderHistoryRuntimeConfigEnv(t *testing.T) {
	t.Helper()
	unsetProviderHistoryEnvVar(t, config.ProviderHistoryReductionEnvVar)
	unsetProviderHistoryEnvVar(t, config.ProviderHistoryRehydrateContextEnvVar)
}

func unsetProviderHistoryEnvVar(t *testing.T, key string) {
	t.Helper()
	oldValue, oldSet := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
	t.Cleanup(func() {
		if oldSet {
			_ = os.Setenv(key, oldValue)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
