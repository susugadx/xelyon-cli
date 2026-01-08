package tools

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// executeWebSearch は Serper API を使って Web 検索を実行
func executeWebSearch(query string) string {
	if query == "" {
		return "Error: query is required"
	}

	green.Printf("🔍 Searching the web for: %s\n", query)

	result, err := api.WebSearch(query)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return result
}
