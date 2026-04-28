package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestPlanModeCheckpointRestoreSessionResponseContext_CanonicalizesAzureDisplayNameProviderConfigKey(t *testing.T) {
	checkpoint := &planModeCheckpoint{
		responseID:          "resp_azure",
		responseModel:       "corp-gpt55-deployment",
		responseProvider:    "Azure OpenAI",
		responseProviderKey: "Azure OpenAI",
	}
	session := history.NewSession("corp-gpt55-deployment")
	session.ProviderName = "azure"
	session.ProviderConfigKey = "azure"

	checkpoint.restoreSessionResponseContext(session)

	if session.ResponseProviderName != "azure" {
		t.Fatalf("session.ResponseProviderName = %q, want azure", session.ResponseProviderName)
	}
	if session.ResponseProviderConfigKey != "azure" {
		t.Fatalf("session.ResponseProviderConfigKey = %q, want azure", session.ResponseProviderConfigKey)
	}
}
