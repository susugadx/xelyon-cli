package configdocs

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
	updated, replaced := replaceMarkerContent(content, configExampleStartMarker, configExampleEndMarker, formatConfigExample(exampleYAML))
	if !replaced {
		return "", fmt.Errorf("missing CONFIG-EXAMPLE markers")
	}
	return updated, nil
}

// hasConfigDetailsMarkers は docs/config.md が詳細ブロック更新に対応しているかを返す。
func hasConfigDetailsMarkers(content string) bool {
	return containsAll(content, configDetailsStartMarker, configDetailsEndMarker)
}

// replaceConfigDetailsBlock は docs/config.md の CONFIG-DETAILS ブロックを更新する。
func replaceConfigDetailsBlock(content, details string) string {
	updated, _ := replaceMarkerContent(content, configDetailsStartMarker, configDetailsEndMarker, details)
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
