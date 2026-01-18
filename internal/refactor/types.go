package refactor

import (
	"github.com/susugadx/xelyon-cli/internal/review"
)

// RefactorType represents the type of refactoring.
type RefactorType string

const (
	RefactorSplitFile     RefactorType = "split-file"     // Large file splitting
	RefactorExtractMethod RefactorType = "extract-method" // Long function extraction
	RefactorDRY           RefactorType = "dry"            // Duplicate code consolidation
	RefactorRename        RefactorType = "rename"         // Naming improvement
)

// RefactorProposal represents a refactoring suggestion.
type RefactorProposal struct {
	ID          string
	Type        RefactorType
	Description string
	FilePath    string
	LineStart   int
	LineEnd     int

	// Details for different refactor types
	FunctionName string   // For extract-method
	NewNames     []string // For split-file (new file names)

	// For actionable proposals
	Change     *review.MultiFileChange
	Confidence float64
	Actionable bool

	// Metrics
	CurrentLines int
	TargetLines  int
}

// RefactorStats contains statistics about the refactoring analysis.
type RefactorStats struct {
	FilesAnalyzed   int
	LargeFiles      int
	LongFunctions   int
	DuplicateBlocks int
	NamingIssues    int
	TotalProposals  int
	ActionableCount int
}

// RefactorReport contains the full refactoring analysis result.
type RefactorReport struct {
	Proposals []RefactorProposal
	Stats     RefactorStats
}

// RefactorConfig holds configuration for refactoring analysis.
type RefactorConfig struct {
	MaxFileLines      int  // Default: 300
	MaxFunctionLines  int  // Default: 50
	MinDuplicateLines int  // Default: 10
	UseAI             bool // Enable AI-powered analysis
	MaxAIFiles        int  // Default: 20, max files for AI analysis (0=unlimited)
}

// DefaultConfig returns the default refactoring configuration.
func DefaultConfig() RefactorConfig {
	return RefactorConfig{
		MaxFileLines:      300,
		MaxFunctionLines:  50,
		MinDuplicateLines: 10,
		UseAI:             false,
		MaxAIFiles:        20, // Limit API calls
	}
}

// RefactorOptions holds options for the /refactor command.
type RefactorOptions struct {
	Paths       []string
	Config      RefactorConfig
	TypeFilter  RefactorType // Empty means all types
	Fix         bool         // Apply fixes interactively
	AutoApprove bool         // Auto-approve all fixes
}
