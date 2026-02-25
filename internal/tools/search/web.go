package search

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api/providers/serper"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

var green = common.Green

// ExecuteWebSearch executes web search using Serper API (with caching)
func ExecuteWebSearch(query string) string {
	if query == "" {
		return "Error: query is required"
	}

	// 確認プロンプト（--auto-approve / config で自動承認可能）
	dec := common.ConfirmWithAutoApproveDecision("web_search",
		fmt.Sprintf("Execute web search: %s", query))
	switch dec.Action {
	case common.ConfirmNo:
		return "User rejected web search"
	case common.ConfirmComment:
		return fmt.Sprintf("User feedback: %s", dec.Comment)
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
