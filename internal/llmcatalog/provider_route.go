package llmcatalog

import "strings"

// ProviderRoute は provider key と model/catalog_model から解決した実行時 family を表す。
type ProviderRoute struct {
	ProviderKey        string
	RuntimeFamily      string
	PromptFamily       string
	EditToolFamily     string
	CapabilityFamily   string
	ModelCatalogFamily string
	PricingFamily      string
	DoctorPolicyFamily string
}

// ResolveProviderRoute は provider key を維持したまま、model ごとの routed family を返す。
func ResolveProviderRoute(provider, model, catalogModel string) ProviderRoute {
	key := CanonicalProviderKey(provider)
	entry, ok := ProviderDescriptorFor(key)
	if !ok {
		return unknownProviderRoute(key)
	}

	route := routeFromDescriptor(entry)
	switch key {
	case "bedrock":
		return resolveBedrockRoute(route, model, catalogModel)
	case "openrouter":
		return resolveOpenRouterRoute(route, model)
	default:
		return route
	}
}

func unknownProviderRoute(key string) ProviderRoute {
	if key == "google" {
		return ProviderRoute{
			ProviderKey:        key,
			RuntimeFamily:      key,
			PromptFamily:       "",
			EditToolFamily:     "apply_patch",
			CapabilityFamily:   key,
			ModelCatalogFamily: key,
			PricingFamily:      key,
			DoctorPolicyFamily: key,
		}
	}
	return ProviderRoute{
		ProviderKey:        key,
		RuntimeFamily:      key,
		PromptFamily:       key,
		EditToolFamily:     "legacy",
		CapabilityFamily:   key,
		ModelCatalogFamily: key,
		PricingFamily:      key,
		DoctorPolicyFamily: key,
	}
}

// DefaultModelForProvider は provider descriptor 上の request default model を返す。
func DefaultModelForProvider(provider string) string {
	entry, ok := ProviderDescriptorFor(provider)
	if !ok {
		return ""
	}
	return strings.TrimSpace(entry.ModelDefaults.DefaultModel)
}

func routeFromDescriptor(entry ProviderDescriptor) ProviderRoute {
	key := CanonicalProviderKey(entry.Key)
	if key == "" {
		key = NormalizeProviderKey(entry.Key)
	}
	route := ProviderRoute{
		ProviderKey:        key,
		RuntimeFamily:      familyOrDefault(entry.RuntimeFamily, key),
		PromptFamily:       strings.TrimSpace(entry.PromptFamily),
		EditToolFamily:     familyOrDefault(entry.EditToolFamily, "legacy"),
		CapabilityFamily:   familyOrDefault(entry.CapabilityFamily, key),
		ModelCatalogFamily: familyOrDefault(entry.ModelCatalogFamily, key),
		PricingFamily:      familyOrDefault(entry.PricingFamily, key),
		DoctorPolicyFamily: familyOrDefault(entry.DoctorPolicyFamily, key),
	}
	return route
}

func resolveBedrockRoute(route ProviderRoute, model, catalogModel string) ProviderRoute {
	if BedrockModelFamilyFor(model, catalogModel) != BedrockModelFamilyClaude {
		return route
	}
	route.RuntimeFamily = "bedrock_claude"
	route.PromptFamily = "claude"
	return route
}

func resolveOpenRouterRoute(route ProviderRoute, model string) ProviderRoute {
	owner, _, ok := splitRoutedModelID(model)
	if !ok {
		return route
	}
	switch owner {
	case "openai":
		route.EditToolFamily = "apply_patch"
	case "google", "gemini":
		route.EditToolFamily = "apply_patch"
	default:
		route.EditToolFamily = "legacy"
	}
	return route
}

func familyOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}
