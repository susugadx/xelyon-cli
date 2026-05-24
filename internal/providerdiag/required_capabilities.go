package providerdiag

import (
	"fmt"
	"strings"
)

const (
	RequiredCapabilityCheckName           = "required_capability"
	RequiredCapabilityResponsesAPI        = "responses_api"
	RequiredCapabilityResponsesStreaming  = "responses_streaming"
	RequiredCapabilityChatCompletions     = "chat_completions"
	RequiredCapabilityFunctionCalling     = "function_calling"
	RequiredCapabilityImageInput          = "image_input"
	RequiredCapabilityWebSearch           = "web_search"
	RequiredCapabilityThinking            = "thinking"
	RequiredCapabilityPreviousResponseID  = "previous_response_id"
	RequiredCapabilitySessionPersistence  = "session_persistence"
	RequiredCapabilityServerCompaction    = "server_compaction"
	RequiredCapabilityLocalModelAvailable = "local_model_available"
)

type requiredCapabilityAvailabilityResolver func(CapabilitySnapshot) CapabilityAvailability

type requiredCapabilityDefinition struct {
	name    string
	resolve requiredCapabilityAvailabilityResolver
}

var requiredCapabilityDefinitions = []requiredCapabilityDefinition{
	{
		name: RequiredCapabilityResponsesAPI,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return KnownCapabilityAvailability(snapshot.ResponsesAPI)
		},
	},
	{
		name: RequiredCapabilityResponsesStreaming,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return snapshot.ResponsesStreamingAvailability
		},
	},
	{
		name: RequiredCapabilityChatCompletions,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return KnownCapabilityAvailability(snapshot.ChatCompletions)
		},
	},
	{
		name: RequiredCapabilityFunctionCalling,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return KnownCapabilityAvailability(snapshot.FunctionCalling)
		},
	},
	{
		name: RequiredCapabilityImageInput,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return snapshot.ImageInput
		},
	},
	{
		name: RequiredCapabilityWebSearch,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return snapshot.WebSearch
		},
	},
	{
		name: RequiredCapabilityThinking,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return snapshot.Thinking
		},
	},
	{
		name: RequiredCapabilityPreviousResponseID,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return KnownCapabilityAvailability(snapshot.Retention.PreviousResponseID)
		},
	},
	{
		name: RequiredCapabilitySessionPersistence,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return KnownCapabilityAvailability(snapshot.Retention.SessionPersistence)
		},
	},
	{
		name: RequiredCapabilityServerCompaction,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return KnownCapabilityAvailability(snapshot.ServerCompaction.RequestPayload)
		},
	},
	{
		name: RequiredCapabilityLocalModelAvailable,
		resolve: func(snapshot CapabilitySnapshot) CapabilityAvailability {
			return snapshot.LocalModelAvailable
		},
	},
}

// CapabilityAvailability は required capability の availability を三値で表す。
type CapabilityAvailability struct {
	Known     bool
	Available bool
}

// KnownCapabilityAvailability は availability を確定できる capability を返す。
func KnownCapabilityAvailability(available bool) CapabilityAvailability {
	return CapabilityAvailability{Known: true, Available: available}
}

// UnknownCapabilityAvailability は availability を確定できない capability を返す。
func UnknownCapabilityAvailability() CapabilityAvailability {
	return CapabilityAvailability{}
}

// ResponsesStreamingCapabilityAvailability は catalog policy から
// Responses streaming required capability の availability を返す。
func ResponsesStreamingCapabilityAvailability(responsesStreaming bool, policy CatalogPolicy) CapabilityAvailability {
	if responsesStreaming && !policy.ContextWindowKnown {
		return UnknownCapabilityAvailability()
	}
	return KnownCapabilityAvailability(responsesStreaming)
}

// RequiredCapabilityStatus は --require-capability の 1 項目の評価状態。
type RequiredCapabilityStatus string

const (
	RequiredCapabilityStatusOK                  RequiredCapabilityStatus = "ok"
	RequiredCapabilityStatusMissing             RequiredCapabilityStatus = "missing"
	RequiredCapabilityStatusUnknownName         RequiredCapabilityStatus = "unknown_name"
	RequiredCapabilityStatusUnknownAvailability RequiredCapabilityStatus = "unknown_availability"
)

// RequiredCapabilityResult は --require-capability の 1 項目の評価結果。
type RequiredCapabilityResult struct {
	Name   string
	Status RequiredCapabilityStatus
}

// RequiredCapabilityCheck は --require-capability 全体の評価結果。
type RequiredCapabilityCheck struct {
	Results []RequiredCapabilityResult
}

// RequiredCapabilityDiagnosticOptions は provider doctor の required capability check 表示情報。
type RequiredCapabilityDiagnosticOptions struct {
	ProviderName                  string
	MissingTarget                 string
	UnknownAvailabilitySuggestion string
}

// RequiredCapabilityDiagnostic は provider doctor の check 行へ渡す評価結果。
type RequiredCapabilityDiagnostic struct {
	Requested  bool
	Satisfied  bool
	Name       string
	Message    string
	Detail     string
	Suggestion string
}

// LocalCapabilityRequest は provider doctor が local capability 判定だけで完結できるかの入力。
type LocalCapabilityRequest struct {
	Capabilities         bool
	RequiredCapabilities []string
	RunSmoke             bool
	PrintRequest         bool
}

// SupportedRequiredCapabilities は --require-capability で受け付ける名前を返す。
func SupportedRequiredCapabilities() []string {
	names := make([]string, 0, len(requiredCapabilityDefinitions))
	for _, definition := range requiredCapabilityDefinitions {
		names = append(names, definition.name)
	}
	return names
}

// SupportedRequiredCapabilitiesText は --require-capability の help/suggestion 用文字列を返す。
func SupportedRequiredCapabilitiesText() string {
	return strings.Join(SupportedRequiredCapabilities(), ", ")
}

// RequiredCapabilityFailureSuggestion は required_capability check の suggestion を返す。
func RequiredCapabilityFailureSuggestion(check RequiredCapabilityCheck, missingTarget, unknownAvailabilitySuggestion string) string {
	if check.HasUnknownCapabilityName() {
		return fmt.Sprintf("Use one of: %s", SupportedRequiredCapabilitiesText())
	}
	if check.HasUnknownAvailability() && strings.TrimSpace(unknownAvailabilitySuggestion) != "" {
		return unknownAvailabilitySuggestion
	}
	return fmt.Sprintf("Choose a %s that provides the missing capability, or remove --require-capability", missingTarget)
}

// HasRequiredCapabilityRequest は空白や空 entry を除いた要求があるか返す。
func HasRequiredCapabilityRequest(values []string) bool {
	return len(normalizeRequiredCapabilityNames(values)) > 0
}

// IsLocalCapabilityOnlyRequest は live smoke / request preview を伴わない capability request か返す。
// Provider はこの判定を使い、local-only で不要な auth/endpoint/config discovery を skip する。
func IsLocalCapabilityOnlyRequest(request LocalCapabilityRequest) bool {
	if request.RunSmoke || request.PrintRequest {
		return false
	}
	return request.Capabilities || HasRequiredCapabilityRequest(request.RequiredCapabilities)
}

// IsOnly は live smoke / request preview を伴わない capability request か返す。
func (r LocalCapabilityRequest) IsOnly() bool {
	return IsLocalCapabilityOnlyRequest(r)
}

// RequiresAuthCheck は doctor が credential/auth check を実行すべきか返す。
func (r LocalCapabilityRequest) RequiresAuthCheck() bool {
	return !r.PrintRequest && !r.IsOnly()
}

// RequiresExternalSetupCheck は doctor が endpoint/base URL/config などの外部 setup check を実行すべきか返す。
func (r LocalCapabilityRequest) RequiresExternalSetupCheck() bool {
	return !r.IsOnly()
}

// HasRequiredCapability は正規化後の required capability に capability が含まれるか返す。
func HasRequiredCapability(values []string, capability string) bool {
	capability = normalizeRequiredCapabilityName(capability)
	if capability == "" {
		return false
	}
	for _, name := range normalizeRequiredCapabilityNames(values) {
		if name == capability {
			return true
		}
	}
	return false
}

// EvaluateRequiredCapabilities は capability snapshot が要求を満たすか判定する。
func EvaluateRequiredCapabilities(snapshot CapabilitySnapshot, values []string) RequiredCapabilityCheck {
	names := normalizeRequiredCapabilityNames(values)
	results := make([]RequiredCapabilityResult, 0, len(names))
	for _, name := range names {
		result := RequiredCapabilityResult{Name: name}
		definition, ok := requiredCapabilityDefinitionFor(name)
		availability := CapabilityAvailability{}
		if ok {
			availability = definition.resolve(snapshot)
		}
		switch {
		case !ok:
			result.Status = RequiredCapabilityStatusUnknownName
		case !availability.Known:
			result.Status = RequiredCapabilityStatusUnknownAvailability
		case availability.Available:
			result.Status = RequiredCapabilityStatusOK
		default:
			result.Status = RequiredCapabilityStatusMissing
		}
		results = append(results, result)
	}
	return RequiredCapabilityCheck{Results: results}
}

// NewRequiredCapabilityDiagnostic は snapshot と要求から provider doctor の check 行を組み立てる。
func NewRequiredCapabilityDiagnostic(snapshot CapabilitySnapshot, values []string, options RequiredCapabilityDiagnosticOptions) RequiredCapabilityDiagnostic {
	check := EvaluateRequiredCapabilities(snapshot, values)
	if !check.Any() {
		return RequiredCapabilityDiagnostic{}
	}

	providerName := strings.TrimSpace(options.ProviderName)
	if providerName != "" {
		providerName += " "
	}
	diagnostic := RequiredCapabilityDiagnostic{
		Requested: true,
		Satisfied: check.Satisfied(),
		Name:      RequiredCapabilityCheckName,
		Detail:    check.Detail(),
	}
	if diagnostic.Satisfied {
		diagnostic.Message = fmt.Sprintf("required %scapabilities are available", providerName)
		return diagnostic
	}
	diagnostic.Message = fmt.Sprintf("required %scapabilities are missing", providerName)
	diagnostic.Suggestion = RequiredCapabilityFailureSuggestion(
		check,
		options.MissingTarget,
		options.UnknownAvailabilitySuggestion,
	)
	return diagnostic
}

func requiredCapabilityDefinitionFor(name string) (requiredCapabilityDefinition, bool) {
	for _, definition := range requiredCapabilityDefinitions {
		if definition.name == name {
			return definition, true
		}
	}
	return requiredCapabilityDefinition{}, false
}

func normalizeRequiredCapabilityNames(values []string) []string {
	seen := map[string]bool{}
	names := make([]string, 0, len(values))
	for _, value := range values {
		for _, raw := range strings.Split(value, ",") {
			name := normalizeRequiredCapabilityName(raw)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

func normalizeRequiredCapabilityName(value string) string {
	name := strings.ToLower(strings.TrimSpace(value))
	return strings.ReplaceAll(name, "-", "_")
}

// Any は評価対象の要求が 1 つ以上あるか返す。
func (c RequiredCapabilityCheck) Any() bool {
	return len(c.Results) > 0
}

// Satisfied はすべての要求が既知かつ available か返す。
func (c RequiredCapabilityCheck) Satisfied() bool {
	if len(c.Results) == 0 {
		return true
	}
	for _, result := range c.Results {
		if result.Status != RequiredCapabilityStatusOK {
			return false
		}
	}
	return true
}

// HasUnknown は未知の capability 名または未確定の availability が含まれるか返す。
func (c RequiredCapabilityCheck) HasUnknown() bool {
	return c.HasUnknownCapabilityName() || c.HasUnknownAvailability()
}

// HasUnknownCapabilityName は未対応の capability 名が含まれるか返す。
func (c RequiredCapabilityCheck) HasUnknownCapabilityName() bool {
	for _, result := range c.Results {
		if result.Status == RequiredCapabilityStatusUnknownName {
			return true
		}
	}
	return false
}

// HasUnknownAvailability は capability 名は既知だが availability が未確定の項目を含むか返す。
func (c RequiredCapabilityCheck) HasUnknownAvailability() bool {
	for _, result := range c.Results {
		if result.Status == RequiredCapabilityStatusUnknownAvailability {
			return true
		}
	}
	return false
}

// Detail は doctor check detail 用の評価結果を返す。
func (c RequiredCapabilityCheck) Detail() string {
	parts := make([]string, 0, len(c.Results))
	for _, result := range c.Results {
		parts = append(parts, fmt.Sprintf("%s=%s", result.Name, result.detailStatus()))
	}
	return strings.Join(parts, ", ")
}

func (r RequiredCapabilityResult) detailStatus() string {
	switch r.Status {
	case RequiredCapabilityStatusOK:
		return "ok"
	case RequiredCapabilityStatusMissing:
		return "missing"
	default:
		return "unknown"
	}
}
