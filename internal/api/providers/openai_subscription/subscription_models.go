package openaisubscription

import (
	"fmt"
	"slices"
	"strings"
)

const (
	subscriptionProviderKey              = "openai_subscription"
	subscriptionDisplayName              = "OpenAI Subscription"
	subscriptionDefaultModel             = "gpt-5.5"
	subscriptionDefaultUtilityModel      = "gpt-5.4-mini"
	subscriptionUnsupportedModelTemplate = "model %s is not supported by openai_subscription.\nSupported models: gpt-5.5, gpt-5.4, gpt-5.4-mini, gpt-5.3-codex-spark.\nUse provider openai if you need OpenAI Platform API / legacy models"
)

var subscriptionSupportedModels = []string{
	"gpt-5.5",
	"gpt-5.4",
	"gpt-5.4-mini",
	"gpt-5.3-codex-spark",
}

// SubscriptionDefaultModel は openai_subscription の既定モデルを返す。
func SubscriptionDefaultModel() string {
	return subscriptionDefaultModel
}

// SubscriptionDefaultUtilityModel は openai_subscription の utility/subagent 既定モデルを返す。
func SubscriptionDefaultUtilityModel() string {
	return subscriptionDefaultUtilityModel
}

// SubscriptionSupportedModels は openai_subscription が受け付けるモデルを安定順で返す。
func SubscriptionSupportedModels() []string {
	return append([]string(nil), subscriptionSupportedModels...)
}

// IsSubscriptionModelSupported は openai_subscription がモデルを受け付けるか返す。
func IsSubscriptionModelSupported(model string) bool {
	model = strings.TrimSpace(model)
	return slices.Contains(subscriptionSupportedModels, model)
}

// ValidateSubscriptionModel は unsupported model を user-facing error として返す。
func ValidateSubscriptionModel(model string) error {
	model = strings.TrimSpace(model)
	if IsSubscriptionModelSupported(model) {
		return nil
	}
	return fmt.Errorf(subscriptionUnsupportedModelTemplate, model)
}
