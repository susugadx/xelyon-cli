package configgen

import (
	"fmt"
	"strings"
)

const (
	configExampleStartMarker = "<!-- CONFIG-EXAMPLE-START -->"
	configExampleEndMarker   = "<!-- CONFIG-EXAMPLE-END -->"
	configDetailsStartMarker = "<!-- CONFIG-DETAILS-START -->"
	configDetailsEndMarker   = "<!-- CONFIG-DETAILS-END -->"
)

// ReplaceConfigExampleBlock は docs/config.md の CONFIG-EXAMPLE ブロックを更新する。
func ReplaceConfigExampleBlock(content, exampleYAML string) (string, error) {
	updated, replaced := ReplaceMarkerContent(content, configExampleStartMarker, configExampleEndMarker, FormatConfigExample(exampleYAML))
	if !replaced {
		return "", fmt.Errorf("missing CONFIG-EXAMPLE markers")
	}
	return updated, nil
}

// HasConfigDetailsMarkers は docs/config.md が詳細ブロック更新に対応しているかを返す。
func HasConfigDetailsMarkers(content string) bool {
	return containsAll(content, configDetailsStartMarker, configDetailsEndMarker)
}

// ReplaceConfigDetailsBlock は docs/config.md の CONFIG-DETAILS ブロックを更新する。
func ReplaceConfigDetailsBlock(content, details string) string {
	updated, _ := ReplaceMarkerContent(content, configDetailsStartMarker, configDetailsEndMarker, details)
	return updated
}

func containsAll(content string, needles ...string) bool {
	for _, needle := range needles {
		if needle == "" {
			continue
		}
		if !strings.Contains(content, needle) {
			return false
		}
	}
	return true
}
