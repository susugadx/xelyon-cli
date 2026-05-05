# AST 基盤

`internal/ast` に Pure Go の Tree-sitter ランタイム
[`github.com/odvcencio/gotreesitter`](https://github.com/odvcencio/gotreesitter)
を使った共通 AST 基盤を持っています。

## 現在の利用箇所

- Go ファイルを CGO なしでパース
- 定義シンボルを抽出し、`search_code` の Go symbol resolver と `read_file(symbol=...)` で利用
- 行単位で `def` / `call` / `ref` / `import` / `comment` / `string` を分類
- legacy `str_replace` の Go ファイル書き込み前に構文検証を実行し、問題があれば警告を返す
- `grammar_set_core` ビルドタグで文法セットを削減

## 主なファイル

- `internal/ast/ast.go`
- `internal/ast/queries.go`
- `internal/ast/ast_test.go`

## 公開 API

- `IsSupportedFile(path string) bool`
- `ParseFile(path string) (*gotreesitter.Tree, []byte, error)`
- `ParseBytes(path string, src []byte) (*gotreesitter.Tree, []byte, error)`
- `ValidateSyntax(path string, src []byte) []SyntaxError`
- `ExtractSymbols(path string) ([]Symbol, error)`
- `ExtractSymbolsFromBytes(path string, src []byte) ([]Symbol, error)`
- `ClassifyLine(path string, src []byte, line int, targetName string) (*MatchInfo, error)`

## 現在の制約

- 対応言語は Go のみ
- `search_code` は Project Map と併用し、公開契約は `mode=auto | symbol | literal | regex` の router ベース
- `ValidateSyntax` は現時点では警告を返すだけで、書き込み自体は止めない

## 検証コマンド

```bash
go test -tags grammar_set_core ./internal/ast
go test -bench=. -benchmem ./internal/ast
CGO_ENABLED=0 go build ./...
go build -tags grammar_set_core -o xelyon .
make ci-check
```

## 補足

`ExtractSymbols` が保持する Go シグネチャ情報は、`search_code` の Go symbol resolver（内部で `InspectSymbolAuto` を使用）の
レシーバ付きメソッド指定（例: `Config.Build`, `(*Config).Build`）や auto-mode rescue の候補解決にも利用しています。

import 文中の `"fmt"` のような文字列リテラルは、通常の string と区別して
`import` と分類するように実装しています。

`ValidateSyntax` は Go ファイルの置換結果を Tree-sitter で再パースし、構文エラーがあれば行・列つきの警告を返します。
現時点では `str_replace` の書き込み前チェックで利用し、警告は返すものの書き込み自体は止めません。
