package search

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type symbolResolveStatus string

const (
	symbolResolveSingle   symbolResolveStatus = "single"
	symbolResolveMultiple symbolResolveStatus = "multiple"
	symbolResolveNone     symbolResolveStatus = "none"
)

type symbolResolveResult struct {
	Output        string
	Status        symbolResolveStatus
	Bundle        *SymbolBundle
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
}

type symbolResolver interface {
	Resolve(symbol string, opts SearchOptions) symbolResolveResult
}
