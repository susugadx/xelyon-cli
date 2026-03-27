package agent

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	tmpHome, err := os.MkdirTemp("", "xelyon-agent-test-home-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpHome)

	originalHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tmpHome); err != nil {
		panic(err)
	}
	defer func() {
		if originalHome == "" {
			_ = os.Unsetenv("HOME")
			return
		}
		_ = os.Setenv("HOME", originalHome)
	}()

	originalDisableMCP := os.Getenv("XELYON_DISABLE_MCP")
	if err := os.Setenv("XELYON_DISABLE_MCP", "1"); err != nil {
		panic(err)
	}
	defer func() {
		if originalDisableMCP == "" {
			_ = os.Unsetenv("XELYON_DISABLE_MCP")
			return
		}
		_ = os.Setenv("XELYON_DISABLE_MCP", originalDisableMCP)
	}()

	originalDisableLSPWarmup := os.Getenv("XELYON_DISABLE_LSP_WARMUP")
	if err := os.Setenv("XELYON_DISABLE_LSP_WARMUP", "1"); err != nil {
		panic(err)
	}
	defer func() {
		if originalDisableLSPWarmup == "" {
			_ = os.Unsetenv("XELYON_DISABLE_LSP_WARMUP")
			return
		}
		_ = os.Setenv("XELYON_DISABLE_LSP_WARMUP", originalDisableLSPWarmup)
	}()

	os.Exit(m.Run())
}
