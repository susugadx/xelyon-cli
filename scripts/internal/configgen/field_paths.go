package configgen

import "sort"

// CanonicalFieldPath は section と field から生成コードで使う正規化パスを返す。
func CanonicalFieldPath(sectionName, fieldName string) string {
	if fieldName == sectionName {
		return sectionName
	}
	return sectionName + "." + fieldName
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
