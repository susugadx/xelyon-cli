package plan

import (
	"context"
	"time"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

// EnhanceWithLSP はLSPを使用して依存関係を強化する。
// WriteFilesに含まれるファイルのシンボル参照を取得し、
// 参照元ファイルが他のステップで使用されていれば依存関係を追加する。
func (da *DependencyAnalyzer) EnhanceWithLSP(steps []PlanStep) []PlanStep {
	if da.lspClient == nil {
		return steps
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build: file -> stepIndex map (for quick lookup)
	fileToStepIdx := make(map[string][]int)
	for idx, step := range steps {
		for _, f := range step.ReadFiles {
			fileToStepIdx[f] = append(fileToStepIdx[f], idx)
		}
		for _, f := range step.WriteFiles {
			fileToStepIdx[f] = append(fileToStepIdx[f], idx)
		}
	}

	// For each step i, find references to its WriteFiles
	for i := range steps {
		for _, writeFile := range steps[i].WriteFiles {
			// Get files that reference writeFile via LSP
			refs, err := da.lspClient.FindReferences(ctx, writeFile, 1, 1, true)
			if err != nil {
				continue // LSP errors are not fatal
			}

			// For each reference file, check if later steps use it
			for _, ref := range refs {
				refFile := lsp.URIToFile(ref.URI)
				if refFile == writeFile {
					continue // skip self-reference
				}

				// Find later steps that use refFile
				for _, j := range fileToStepIdx[refFile] {
					// Compare by ID (not index) for safety
					if steps[j].ID > steps[i].ID && !containsInt(steps[j].DependsOn, steps[i].ID) {
						// Step j uses a file that references Step i's writeFile
						// => Step j depends on Step i
						steps[j].DependsOn = append(steps[j].DependsOn, steps[i].ID)
					}
				}
			}
		}
	}

	return steps
}
