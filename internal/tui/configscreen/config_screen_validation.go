package configscreen

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func (cs *Screen) validateConfigEditCandidate(candidate *config.Config, path, providerConfigKey string) bool {
	result := config.ValidateGeminiFunctionCallingConfig(candidate)
	if result.Valid {
		return true
	}
	for _, issue := range result.Issues {
		if issue.Severity != config.ValidationSeverityError {
			continue
		}
		if !configEditMatchesGeminiFunctionCallingIssue(issue.Field, path, providerConfigKey) {
			continue
		}
		cs.setConfigValidationIssue(issue)
		return false
	}
	return true
}

func configEditMatchesGeminiFunctionCallingIssue(field, path, providerConfigKey string) bool {
	field = strings.TrimSpace(field)
	switch path {
	case "default_model":
		return field == "default_model" || field == "provider_models.gemini.default_model"
	case "default_provider":
		return field == "default_model" || strings.HasPrefix(field, "provider_models.gemini.")
	case "provider_models":
		providerConfigKey = config.ActiveProviderConfigKey(providerConfigKey)
		return config.SameProviderRuntimeIdentity(providerConfigKey, "gemini") &&
			strings.HasPrefix(field, "provider_models."+providerConfigKey+".")
	default:
		return false
	}
}

func (cs *Screen) setConfigValidationIssue(issue config.ValidationIssue) {
	cs.saveStatus = statusFailed
	cs.saveError = configValidationIssueText(issue)
}

func configValidationIssueText(issue config.ValidationIssue) string {
	msg := fmt.Sprintf("%s: %s", issue.Field, issue.Message)
	if suggestion := strings.TrimSpace(issue.Suggestion); suggestion != "" {
		msg += " (" + suggestion + ")"
	}
	return msg
}

func geminiFunctionCallingConfigSaveError(cfg *config.Config) error {
	result := config.ValidateGeminiFunctionCallingConfig(cfg)
	if result.Valid {
		return nil
	}
	for _, issue := range result.Issues {
		if issue.Severity == config.ValidationSeverityError {
			return fmt.Errorf("%s", configValidationIssueText(issue))
		}
	}
	return nil
}

// ValidateSaveSnapshot は保存前に /config screen の保存可能性を検証する。
func ValidateSaveSnapshot(cfg *config.Config) error {
	return geminiFunctionCallingConfigSaveError(cfg)
}
