package search

import (
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools/search/internal/bundlekeys"
)

func stableGoSymbolBundleKey(packageDir, receiverNorm, name, kind, signature string) string {
	return bundlekeys.StableGoSymbolKey(packageDir, receiverNorm, name, kind, signature)
}

func canonicalGoSymbolBundleKey(symbol navigation.SymbolCandidate) string {
	return bundlekeys.CanonicalGoSymbolKey(symbol)
}

func canonicalSymbolBundleKey(lang, file string, line int, displayName string) string {
	return bundlekeys.CanonicalSymbolKey(lang, file, line, displayName)
}

func attachBundleRoute(bundle *SymbolBundle, route searchRouteTrace) *SymbolBundle {
	if bundle == nil {
		return nil
	}
	bundle.Debug.Route = route
	if bundle.Debug.MatchedPatterns == nil && route.SymbolCandidates != nil {
		bundle.Debug.MatchedPatterns = append([]string(nil), route.SymbolCandidates...)
	}
	return bundle
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
