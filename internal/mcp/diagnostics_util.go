package mcp

import "sort"

func sortedUnknownKeys(values map[string]string, known map[string]bool) []string {
	keys := sortedMapKeys(values)
	return sortedUnknownNames(keys, known)
}

func sortedUnknownNames(values []string, known map[string]bool) []string {
	unknown := make([]string, 0)
	for _, value := range values {
		if !known[value] {
			unknown = append(unknown, value)
		}
	}
	sort.Strings(unknown)
	return unknown
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}
