package search

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api/providers/serper"
)

// ExecuteWebSearch executes web search using Serper API
func ExecuteWebSearch(query string) string {
	if query == "" {
		return "Error: query is required"
	}

	green.Printf("🔍 Searching the web for: %s\n", query)

	result, err := serper.WebSearch(query)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return result
}
