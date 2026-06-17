package gathercontext

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func registerPrefetchLocator(reg *locator.Registry, item search.SymbolBundleItem) string {
	if reg == nil {
		return ""
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = strings.TrimSpace(item.Kind)
	}
	return reg.Register(locator.Location{
		FilePath:     item.File,
		ResolvedPath: item.ResolvedPath,
		Line:         item.Line,
		EndLine:      item.EndLine,
		Name:         name,
	})
}
