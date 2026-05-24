package llmcatalog

import (
	"fmt"
	"strings"
)

// ModelLifecycleStage は provider catalog model の提供状態を表す。
type ModelLifecycleStage string

const (
	ModelLifecycleActive     ModelLifecycleStage = "active"
	ModelLifecycleDeprecated ModelLifecycleStage = "deprecated"
	ModelLifecycleShutdown   ModelLifecycleStage = "shutdown"
)

// ModelLifecycle は model catalog の表示・診断 policy を表す。
type ModelLifecycle struct {
	Stage            ModelLifecycleStage
	HiddenFromPicker bool
	ShutdownDate     string
	Replacement      string
	Reason           string
}

func (l ModelLifecycle) pickerVisible() bool {
	return l.Stage == ModelLifecycleActive && !l.HiddenFromPicker
}

// ShouldWarn は doctor/status で注意喚起すべき model lifecycle か返す。
func (l ModelLifecycle) ShouldWarn() bool {
	return l.Stage == ModelLifecycleDeprecated ||
		l.Stage == ModelLifecycleShutdown ||
		strings.TrimSpace(l.Reason) != "" ||
		strings.TrimSpace(l.Replacement) != ""
}

// DiagnosticDetail は lifecycle warning の detail 文字列を返す。
func (l ModelLifecycle) DiagnosticDetail(model string) string {
	parts := []string{
		fmt.Sprintf("model=%s", strings.TrimSpace(model)),
		fmt.Sprintf("stage=%s", l.Stage),
	}
	if l.HiddenFromPicker {
		parts = append(parts, "picker=hidden")
	}
	if shutdownDate := strings.TrimSpace(l.ShutdownDate); shutdownDate != "" {
		parts = append(parts, "shutdown_date="+shutdownDate)
	}
	if replacement := strings.TrimSpace(l.Replacement); replacement != "" {
		parts = append(parts, "replacement="+replacement)
	}
	if reason := strings.TrimSpace(l.Reason); reason != "" {
		parts = append(parts, "reason="+reason)
	}
	return strings.Join(parts, ", ")
}

// DiagnosticSuggestion は lifecycle warning の suggestion 文字列を返す。
func (l ModelLifecycle) DiagnosticSuggestion() string {
	replacement := strings.TrimSpace(l.Replacement)
	switch {
	case replacement != "":
		return "Use " + replacement + " for new Gemini configurations"
	case l.Stage == ModelLifecycleShutdown:
		return "Switch to a current Gemini model before running live requests"
	case l.Stage == ModelLifecycleDeprecated:
		return "Plan a migration to a current Gemini model before the shutdown date"
	default:
		return "Use a picker-recommended Gemini model unless this model is intentional"
	}
}

// ModelLifecycleForProvider は provider/model の lifecycle metadata を返す。
func ModelLifecycleForProvider(provider, model string) (ModelLifecycle, bool) {
	provider = CanonicalProviderKey(provider)
	model = normalizeModelName(model)
	if provider == "" || model == "" {
		return ModelLifecycle{}, false
	}
	if provider != "gemini" {
		return ModelLifecycle{}, false
	}
	if lifecycle, ok := geminiModelLifecycle[model]; ok {
		return lifecycle, true
	}
	return geminiFamilyLifecycle(model)
}

func pickerVisibleModelForProvider(provider, model string) bool {
	if lifecycle, ok := ModelLifecycleForProvider(provider, model); ok {
		return lifecycle.pickerVisible()
	}
	return true
}

var geminiModelLifecycle = map[string]ModelLifecycle{
	"gemini-3.5-flash": {
		Stage: ModelLifecycleActive,
	},
	"gemini-3.1-flash-lite": {
		Stage: ModelLifecycleActive,
	},
	"gemini-3.1-pro-preview-customtools": {
		Stage: ModelLifecycleActive,
	},
	"gemini-2.5-pro": {
		Stage: ModelLifecycleActive,
	},
	"gemini-2.5-flash": {
		Stage: ModelLifecycleActive,
	},
	"gemini-3.1-pro": {
		Stage:            ModelLifecycleActive,
		HiddenFromPicker: true,
		Replacement:      "gemini-3.1-pro-preview-customtools",
		Reason:           "plain 3.1 Pro is supported but not recommended for the picker",
	},
	"gemini-3.1-pro-preview": {
		Stage:            ModelLifecycleActive,
		HiddenFromPicker: true,
		Replacement:      "gemini-3.1-pro-preview-customtools",
		Reason:           "customtools variant is preferred for XELYON tool use",
	},
	"gemini-3-pro-preview": {
		Stage:            ModelLifecycleShutdown,
		HiddenFromPicker: true,
		ShutdownDate:     "2026-03-09",
		Replacement:      "gemini-3.1-pro-preview-customtools",
	},
	"gemini-2.0-flash": {
		Stage:            ModelLifecycleDeprecated,
		HiddenFromPicker: true,
		ShutdownDate:     "2026-06-01",
		Replacement:      "gemini-3.5-flash",
	},
	"gemini-2.0-flash-001": {
		Stage:            ModelLifecycleDeprecated,
		HiddenFromPicker: true,
		ShutdownDate:     "2026-06-01",
		Replacement:      "gemini-3.5-flash",
	},
	"gemini-2.0-flash-exp": {
		Stage:            ModelLifecycleShutdown,
		HiddenFromPicker: true,
		ShutdownDate:     "2025-12-09",
		Replacement:      "gemini-3.5-flash",
	},
	"gemini-2.0-flash-lite": {
		Stage:            ModelLifecycleDeprecated,
		HiddenFromPicker: true,
		ShutdownDate:     "2026-06-01",
		Replacement:      "gemini-3.1-flash-lite",
	},
	"gemini-2.0-flash-lite-001": {
		Stage:            ModelLifecycleDeprecated,
		HiddenFromPicker: true,
		ShutdownDate:     "2026-06-01",
		Replacement:      "gemini-3.1-flash-lite",
	},
	"gemini-1.5-pro": {
		Stage:            ModelLifecycleShutdown,
		HiddenFromPicker: true,
		ShutdownDate:     "2025-09-29",
		Replacement:      "gemini-3.1-pro-preview-customtools",
	},
	"gemini-1.5-flash": {
		Stage:            ModelLifecycleShutdown,
		HiddenFromPicker: true,
		ShutdownDate:     "2025-09-29",
		Replacement:      "gemini-3.5-flash",
	},
	"gemini-3.1-flash-lite-preview": {
		Stage:            ModelLifecycleDeprecated,
		HiddenFromPicker: true,
		ShutdownDate:     "2026-05-25",
		Replacement:      "gemini-3.1-flash-lite",
	},
}

type geminiLifecycleFamily struct {
	Prefix    string
	Canonical string
}

var geminiLifecycleFamilies = []geminiLifecycleFamily{
	{Prefix: "gemini-1.5-pro", Canonical: "gemini-1.5-pro"},
	{Prefix: "gemini-1.5-flash", Canonical: "gemini-1.5-flash"},
}

func geminiFamilyLifecycle(model string) (ModelLifecycle, bool) {
	for _, family := range geminiLifecycleFamilies {
		if strings.HasPrefix(model, family.Prefix+"-") {
			lifecycle, ok := geminiModelLifecycle[family.Canonical]
			return lifecycle, ok
		}
	}
	return ModelLifecycle{}, false
}
