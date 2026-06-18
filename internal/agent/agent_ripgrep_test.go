package agent

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestCheckRipgrepAvailability_NoRg(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{Runtime: runtime}

	checkRipgrepAvailability(agent)

	if !strings.Contains(out.String(), "ripgrep (rg) not found") {
		t.Fatalf("expected ripgrep warning, got: %s", out.String())
	}
}

func TestCheckRipgrepAvailability_WithRg(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	rgPath, err := exec.LookPath("rg")
	if err != nil {
		t.Skip("ripgrep (rg) not available")
	}
	t.Setenv("PATH", filepath.Dir(rgPath))

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{Runtime: runtime}

	checkRipgrepAvailability(agent)

	if out.Len() != 0 {
		t.Fatalf("expected no output when rg exists, got: %s", out.String())
	}
}
