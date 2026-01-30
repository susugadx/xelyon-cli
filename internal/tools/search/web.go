package search

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api/providers/serper"
)

// ExecuteWebSearch executes web search using Serper API (with caching)
func ExecuteWebSearch(query string) string {
	if query == "" {
		return "Error: query is required"
	}

	result, cached, err := serper.WebSearchWithCache(query)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if cached {
		green.Printf("🔍 Web search (cached): %s\n", query)
	} else {
		green.Printf("🔍 Searching the web for: %s\n", query)
	}

	return result
}
