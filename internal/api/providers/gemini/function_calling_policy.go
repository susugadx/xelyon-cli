package gemini

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

type geminiFunctionCallingPolicy struct {
	llmcatalog.GeminiFunctionCallingPolicy
}

func newGeminiFunctionCallingPolicy(cfg *config.Config, model string) geminiFunctionCallingPolicy {
	return geminiFunctionCallingPolicy{
		GeminiFunctionCallingPolicy: llmcatalog.NewGeminiFunctionCallingPolicy(model, geminiPolicyModel(cfg, model)),
	}
}
