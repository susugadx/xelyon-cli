package config

func providerConfigCandidates() []string {
	seen := map[string]bool{}
	candidates := make([]string, 0, len(ValidProviders))
	for _, provider := range ValidProviders {
		normalized := NormalizeProviderName(provider)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		candidates = append(candidates, normalized)
	}
	return candidates
}

func orderedProviderConfigCandidates(defaultProvider string) []string {
	seen := map[string]bool{}
	ordered := make([]string, 0, len(ValidProviders))

	for _, candidate := range ProviderModelLookupKeys(defaultProvider) {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		ordered = append(ordered, candidate)
	}

	for _, candidate := range providerConfigCandidates() {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		ordered = append(ordered, candidate)
	}

	return ordered
}
