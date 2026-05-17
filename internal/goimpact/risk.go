package goimpact

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

// ClassifyRisk は inspect 結果から Go impact risk を判定する。
func ClassifyRisk(result navigation.InspectResult) string {
	if result.Symbol == nil {
		return impactplan.RiskLow
	}

	if result.Symbol.Exported || result.Symbol.Kind == "interface" || len(result.Implementations) > 0 {
		return impactplan.RiskHigh
	}

	fileCount, dirCount := ReferenceSpread(result)
	if dirCount > 1 {
		return impactplan.RiskHigh
	}
	if fileCount > 1 || IsSharedPackageSymbol(*result.Symbol) {
		return impactplan.RiskMedium
	}
	if NeedsWidening(result) {
		return impactplan.RiskMedium
	}

	return impactplan.RiskLow
}

// NeedsWidening は upstream の truncation や未取得件数により探索拡張が必要か返す。
func NeedsWidening(result navigation.InspectResult) bool {
	if result.MoreCallers || result.MoreRefs || result.UpstreamTruncated || result.UpstreamIncomplete {
		return true
	}
	return result.TotalCallers > len(result.Callers) || result.TotalRefs > len(result.Refs)
}

// ReferenceSpread は caller/ref がまたがる file 数と directory 数を返す。
func ReferenceSpread(result navigation.InspectResult) (int, int) {
	fileSeen := make(map[string]struct{})
	dirSeen := make(map[string]struct{})
	add := func(file string) {
		file = strings.TrimSpace(file)
		if file == "" {
			return
		}
		file = filepath.ToSlash(filepath.Clean(file))
		fileSeen[file] = struct{}{}
		dirSeen[filepath.ToSlash(filepath.Dir(file))] = struct{}{}
	}

	for _, ref := range result.Callers {
		add(ref.File)
	}
	for _, ref := range result.Refs {
		add(ref.File)
	}
	return len(fileSeen), len(dirSeen)
}

// IsSharedPackageSymbol は cmd 以外の共有 package にある symbol か返す。
func IsSharedPackageSymbol(symbol navigation.SymbolCandidate) bool {
	packageDir := filepath.ToSlash(strings.TrimSpace(symbol.PackageDir))
	if packageDir == "" || packageDir == "." {
		return false
	}
	return packageDir != "cmd" && !strings.HasPrefix(packageDir, "cmd/")
}
