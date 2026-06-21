package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func newPlanRequestTestAgent(t *testing.T, provider api.Provider, input string, out *bytes.Buffer) *Agent {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	runtime := NewAgentRuntimeWithConfig(newChatRequestTestConfig())
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(input), out, out)
	runtime.Registry = tools.DefaultRegistry.Clone()

	agent := NewAgentWithRuntime("gpt-5.4", provider, false, runtime)
	agent.setAutoApprove(true)
	return agent
}
