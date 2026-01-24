package plan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// HashToolCalls は toolCalls セットのハッシュを生成（ループ検知用）
func HashToolCalls(toolCalls []*tools.ToolCall) string {
	var parts []string
	for _, tc := range toolCalls {
		argsStr := fmt.Sprintf("%v", tc.Args)
		parts = append(parts, tc.Tool+":"+argsStr)
	}
	sort.Strings(parts) // 順序に依存しないようソート
	return strings.Join(parts, "|")
}
