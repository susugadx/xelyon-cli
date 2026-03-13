package ast

import (
	"fmt"
	"sync"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// goSymbolQuery は Go ファイルからシンボルを抽出する Tree-sitter クエリ。
const goSymbolQuery = `
(function_declaration
  name: (identifier) @name) @def

(method_declaration
  name: (field_identifier) @name) @def

(type_spec
  name: (type_identifier) @name
  type: [(struct_type) (interface_type)] @type_body) @def

(const_spec
  name: (identifier) @name) @def

(var_spec
  name: (identifier) @name) @def
`

var (
	goSymbolQueryOnce sync.Once
	goSymbolQueryObj  *gotreesitter.Query
	goSymbolQueryErr  error
)

func getGoSymbolQuery() (*gotreesitter.Query, error) {
	goSymbolQueryOnce.Do(func() {
		goSymbolQueryObj, goSymbolQueryErr = gotreesitter.NewQuery(goSymbolQuery, grammars.GoLanguage())
		if goSymbolQueryErr != nil {
			goSymbolQueryErr = fmt.Errorf("compile Go symbol query: %w", goSymbolQueryErr)
		}
	})
	return goSymbolQueryObj, goSymbolQueryErr
}
