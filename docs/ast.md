# AST 基盤導入 Phase 1（gotreesitter PoC）

`internal/ast` に Pure Go の Tree-sitter ランタイム
[`github.com/odvcencio/gotreesitter`](https://github.com/odvcencio/gotreesitter)
を使った共通 AST 基盤を追加しました。

## この Phase で検証すること

- Go ファイルを CGO なしでパースできること
- 定義シンボルを抽出できること
- 行単位で `def` / `call` / `ref` / `import` / `comment` / `string` を分類できること
- `grammar_set_core` ビルドタグで文法セットを削減できること
- パース速度・シンボル抽出速度・バイナリサイズ差分を測定できること

## 追加ファイル

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

## Phase 1 の制約

- 対応言語は Go のみ
- `search_code` は引き続き Project Map と併用するが、公開契約は `mode=auto | symbol | literal | regex` の router ベースへ移行済み
- まだ本番ルートでは使わず、PoC とベンチマークのための基盤追加に留める

## 検証コマンド

```bash
go test ./internal/ast/...
go test -bench=. -benchmem ./internal/ast
CGO_ENABLED=0 go build ./...
go build -o xelyon-full .
go build -tags grammar_set_core -o xelyon-core .
make ci-check
```

## 補足

`ExtractSymbols` が保持する Go シグネチャ情報は、`search_code` の Go symbol resolver（内部で `InspectSymbolAuto` を使用）の
レシーバ付きメソッド指定（例: `Config.Build`, `(*Config).Build`）や auto-mode rescue の候補解決にも利用しています。

import 文中の `"fmt"` のような文字列リテラルは、通常の string と区別して
`import` と分類するように実装しています。

`ValidateSyntax` は Go ファイルの置換結果を Tree-sitter で再パースし、構文エラーがあれば行・列つきの警告を返します。
現時点では `str_replace` の書き込み前チェックで利用し、警告は返すものの書き込み自体は止めません。
