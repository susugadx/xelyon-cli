package repomap

import "github.com/susugadx/xelyon-cli/internal/ast"

type projectMapASTScanner struct{}

func newProjectMapASTScanner() *projectMapASTScanner {
	return &projectMapASTScanner{}
}

func (s *projectMapASTScanner) supports(path string) bool {
	return ast.IsSupportedFile(path)
}

func (s *projectMapASTScanner) scan(absPath string) ([]Symbol, error) {
	astSymbols, err := ast.ExtractSymbols(absPath)
	if err != nil {
		return nil, err
	}

	repoSymbols := make([]Symbol, 0, len(astSymbols))
	for _, symbol := range astSymbols {
		repoSymbols = append(repoSymbols, Symbol{
			Name:      symbol.Name,
			Kind:      string(symbol.Kind),
			Line:      symbol.Line,
			EndLine:   symbol.EndLine,
			Signature: symbol.Signature,
			Exported:  symbol.Exported,
		})
	}
	return repoSymbols, nil
}
