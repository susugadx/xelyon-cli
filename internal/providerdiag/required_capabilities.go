package providerdiag

import (
	"fmt"
	"strings"
)

const (
	RequiredCapabilityCheckName          = "required_capability"
	RequiredCapabilityResponsesAPI       = "responses_api"
	RequiredCapabilityResponsesStreaming = "responses_streaming"
	RequiredCapabilityChatCompletions    = "chat_completions"
	RequiredCapabilityFunctionCalling    = "function_calling"
	RequiredCapabilityImageInput         = "image_input"
	RequiredCapabilityPreviousResponseID = "previous_response_id"
	RequiredCapabilitySessionPersistence = "session_persistence"
	RequiredCapabilityServerCompaction   = "server_compaction"
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
			return KnownCapabilityAvailability(snapshot.ImageInput)
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
			name := strings.ToLower(strings.TrimSpace(raw))
			name = strings.ReplaceAll(name, "-", "_")
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
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
