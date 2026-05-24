package llmcatalog

import "strings"

// ModelCapabilitySupport は model capability のローカル判定結果を表す。
type ModelCapabilitySupport struct {
	Known       bool
	Supported   bool
	Reason      string
	Replacement string
}

// GeminiFunctionCallingSupport は Gemini model の function calling 対応状況を返す。
func GeminiFunctionCallingSupport(model string) ModelCapabilitySupport {
	model = CanonicalModelNameForProvider("gemini", model)
	if model == "" {
		return ModelCapabilitySupport{}
	}
	if support, ok := geminiFunctionCallingExact[model]; ok {
		return support
	}
	for _, family := range geminiFunctionCallingFamilies {
		if strings.HasPrefix(model, family.Prefix) {
			return family.Support
		}
	}
	return ModelCapabilitySupport{}
}

var geminiFunctionCallingExact = map[string]ModelCapabilitySupport{
	"gemini-3.5-flash": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-flash-lite": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-flash-lite-preview": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-pro": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-pro-preview": {
		Known:     true,
		Supported: true,
	},
	"gemini-3.1-pro-preview-customtools": {
		Known:     true,
		Supported: true,
	},
	"gemini-3-pro-preview": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.5-pro": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.5-flash": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.5-flash-lite": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.0-flash": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.0-flash-001": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.0-flash-exp": {
		Known:     true,
		Supported: true,
	},
	"gemini-2.0-flash-lite": {
		Known:       true,
		Supported:   false,
		Reason:      "Gemini 2.0 Flash-Lite is not in the Gemini function calling supported-model list",
		Replacement: "gemini-3.1-flash-lite",
	},
	"gemini-2.0-flash-lite-001": {
		Known:       true,
		Supported:   false,
		Reason:      "Gemini 2.0 Flash-Lite is not in the Gemini function calling supported-model list",
		Replacement: "gemini-3.1-flash-lite",
	},
}

type geminiFunctionCallingFamily struct {
	Prefix  string
	Support ModelCapabilitySupport
}

var geminiFunctionCallingFamilies = []geminiFunctionCallingFamily{
	{
		Prefix: "gemini-2.0-flash-lite-",
		Support: ModelCapabilitySupport{
			Known:       true,
			Supported:   false,
			Reason:      "Gemini 2.0 Flash-Lite is not in the Gemini function calling supported-model list",
			Replacement: "gemini-3.1-flash-lite",
		},
	},
	{Prefix: "gemini-3.5-flash-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-3.1-flash-lite-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-3.1-pro-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-2.5-pro-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-2.5-flash-lite-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-2.5-flash-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
	{Prefix: "gemini-2.0-flash-", Support: ModelCapabilitySupport{Known: true, Supported: true}},
}
