package gathercontext

import (
	"path/filepath"

	"github.com/susugadx/xelyon-cli/internal/filequery"
)

type requestQueryShape struct {
	rewriteProtected        bool
	singleEntry             bool
	explicitDirectoryMarker bool
}

func classifyRequestQueryShape(req request) requestQueryShape {
	if req.quotedPattern {
		return requestQueryShape{
			rewriteProtected: true,
			singleEntry:      true,
		}
	}

	input, ok := filequery.ParseInput(req.query)
	if !ok || len(input.Entries) == 0 {
		return requestQueryShape{singleEntry: true}
	}

	shape := requestQueryShape{
		singleEntry:             len(input.Entries) == 1,
		explicitDirectoryMarker: inputHasExplicitDirectoryMarkerSyntax(input),
	}
	shape.rewriteProtected = inputHasHardExplicitPathSyntax(input) || inputHasDirectReadBatchSyntax(input)
	return shape
}

func inputHasHardExplicitPathSyntax(input filequery.Input) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		if !entryHasHardExplicitPathSyntax(entry) {
			return false
		}
	}
	return true
}

func entryHasHardExplicitPathSyntax(entry filequery.Entry) bool {
	return filepath.IsAbs(entry.CleanedPath) ||
		filequery.HasWindowsPathPrefix(entry.RawPath) ||
		entry.ExplicitRelative ||
		entry.StartLine > 0 ||
		entry.EndLine > 0
}

func inputHasExplicitDirectoryMarkerSyntax(input filequery.Input) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		if !filequery.HasExplicitDirectoryMarker(entry.RawPath) {
			return false
		}
	}
	return true
}

func inputHasDirectReadBatchSyntax(input filequery.Input) bool {
	if len(input.Entries) < 2 {
		return false
	}
	for _, entry := range input.Entries {
		if !filequery.EntryHasStrongDirectIntent(entry) {
			return false
		}
	}
	return true
}
