package gemini

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

type geminiFunctionCallingPolicy struct {
	requestModel string
	policyModel  string
	support      llmcatalog.ModelCapabilitySupport
}

func newGeminiFunctionCallingPolicy(cfg *config.Config, model string) geminiFunctionCallingPolicy {
	return newGeminiFunctionCallingPolicyForCatalogModel(model, geminiPolicyModel(cfg, model))
}

func newGeminiFunctionCallingPolicyForCatalogModel(requestModel, catalogModel string) geminiFunctionCallingPolicy {
	policyModel := llmcatalog.CanonicalModelNameForProvider("gemini", catalogModel)
	if policyModel == "" {
		policyModel = strings.TrimSpace(requestModel)
	}
	return geminiFunctionCallingPolicy{
		requestModel: strings.TrimSpace(requestModel),
		policyModel:  policyModel,
		support:      llmcatalog.GeminiFunctionCallingSupport(policyModel),
	}
}

func (p geminiFunctionCallingPolicy) Enabled() bool {
	return !p.support.Known || p.support.Supported
}

func (p geminiFunctionCallingPolicy) UnsupportedError() error {
	if p.Enabled() {
		return nil
	}
	modelDetail := p.requestModel
	if modelDetail == "" {
		modelDetail = p.policyModel
	}
	if p.policyModel != "" && p.policyModel != modelDetail {
		modelDetail = fmt.Sprintf("%s (catalog_model=%s)", modelDetail, p.policyModel)
	}
	replacement := strings.TrimSpace(p.support.Replacement)
	if replacement == "" {
		replacement = "gemini-3.5-flash"
	}
	reason := strings.TrimSpace(p.support.Reason)
	if reason == "" {
		reason = "the selected model is not supported by XELYON's Gemini function-calling runtime"
	}
	return fmt.Errorf("Gemini function calling is not supported for %s: %s; use %s or disable tool use only for internal text-only requests", modelDetail, reason, replacement)
}
