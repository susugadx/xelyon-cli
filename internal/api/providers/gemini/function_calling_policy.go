package gemini

import (
	"fmt"
	"strings"

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

func (p geminiFunctionCallingPolicy) UnsupportedError() error {
	if p.Enabled() {
		return nil
	}
	modelDetail := p.RequestModel()
	if modelDetail == "" {
		modelDetail = p.PolicyModel()
	}
	if p.PolicyModel() != "" && p.PolicyModel() != modelDetail {
		modelDetail = fmt.Sprintf("%s (catalog_model=%s)", modelDetail, p.PolicyModel())
	}
	support := p.Support()
	replacement := strings.TrimSpace(support.Replacement)
	if replacement == "" {
		replacement = "gemini-3.5-flash"
	}
	reason := strings.TrimSpace(support.Reason)
	if reason == "" {
		reason = "the selected model is not supported by XELYON's Gemini function-calling runtime"
	}
	return fmt.Errorf("Gemini function calling is not supported for %s: %s; use %s or disable tool use only for internal text-only requests", modelDetail, reason, replacement)
}
