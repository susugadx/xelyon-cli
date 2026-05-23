package search

import (
	"context"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

const jsFamilyLSPReferenceTimeout = 5 * time.Second

func findJSFamilyReferencesWithLSP(symbol string, def genericSymbolDef, opts jsFamilyLSPReferenceOptions) (jsFamilyLSPReferenceCollection, error) {
	defPath := absoluteAffectedFilePath(def.File, opts.request, affectedFileSourceText)
	if defPath == "" {
		defPath = absoluteAffectedFilePathWithBase(def.File, invocationCWDOrGetwd(opts.request))
	}
	if defPath == "" {
		return jsFamilyLSPReferenceCollection{}, os.ErrNotExist
	}

	ctx, cancel := context.WithTimeout(context.Background(), jsFamilyLSPReferenceTimeout)
	defer cancel()
	locations, err := opts.request.LSPClient.FindReferences(ctx, defPath, def.Line, def.Character, false)
	if err != nil {
		return jsFamilyLSPReferenceCollection{}, err
	}

	return collectJSFamilyLSPReferences(symbol, def, locations, opts), nil
}

type jsFamilyLSPReferenceCollection struct {
	refs             []genericSymbolRef
	summaryRefs      []genericSymbolRef
	rawLocationCount int
}

func (collection jsFamilyLSPReferenceCollection) hasRawLocations() bool {
	return collection.rawLocationCount > 0
}

func collectJSFamilyLSPReferences(symbol string, def genericSymbolDef, locations []navigation.LSPLocation, opts jsFamilyLSPReferenceOptions) jsFamilyLSPReferenceCollection {
	collector := newJSFamilyLSPReferenceCollector(symbol, def, opts, len(locations))
	defer collector.Close()
	for _, loc := range locations {
		collector.AddLocation(loc)
	}
	collection := collector.Result()
	collection.rawLocationCount = len(locations)
	return collection
}

func jsFamilyRefFromLSPLocation(symbol string, loc navigation.LSPLocation, opts jsFamilyLSPReferenceOptions) (genericSymbolRef, bool) {
	collection := collectJSFamilyLSPReferences(symbol, genericSymbolDef{Name: symbol}, []navigation.LSPLocation{loc}, opts)
	if len(collection.refs) == 0 {
		return genericSymbolRef{}, false
	}
	return collection.refs[0], true
}
