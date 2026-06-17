# Scripts Generation Package Boundary Refactor

この文書は `scripts/` 生成系リファクタリングの内部実装仕様である。
公開 docs ではなく、owner 境界、検証契約、handoff の source of truth として使う。

## Purpose

`scripts/` 配下の生成系を、外部挙動を変えずに owner package 単位で整理する。

対象は generator entrypoint、`scripts/internal/*` の生成 owner、`scripts/config_sections.go` の互換 shim、generated surface、関連テストである。

commit / push / PR はこの作業に含めない。ユーザーの明示指示があるまで行わない。

## Global Contracts

- behavior-preserving refactor とする。
- config key、default、migration、validation、MCP runtime behavior は変更しない。
- generated output は byte-for-byte 維持する。
- generated file を手編集して契約を合わせたことにしない。
- `scripts/config_sections.go` の `go run scripts/config_sections.go scripts/gen-config-*.go` 互換契約は維持する。
- Makefile target 名、generator command contract、docs marker 文字列は変更しない。
- `utils` / `helpers` / `common` に domain policy を逃がさない。
- 公開関数・公開型には日本語 godoc コメントを付ける。

## Package Owners

- `scripts/internal/configmeta`: config section/category/field path/example policy の canonical metadata。
- `scripts/internal/configregistry`: `internal/config/registry_generated.go` の registry entry build と Go source render。
- `scripts/internal/configdocs`: `docs/config.md` の marker replacement、config struct parse、details render/update。
- `scripts/internal/configexample`: `config.yaml.example` の generation/filter/comment/format policy。
- `scripts/internal/scriptio`: generator entrypoint 共通の output path、optional read、error exit。
- `scripts/internal/commanddocs`: slash command docs の existing heading detection、missing detection、skeleton append。
- `scripts/internal/helpgen`: `internal/agent/help_generated.go` の generated source composition。

旧 `scripts/internal/configgen` package は残さない。薄い wrapper / forwarding package で互換を取らず、callers を新 owner package に直接寄せる。

## Dependency Direction

- `configregistry`、`configdocs`、`configexample` は `configmeta` に依存してよい。
- `configdocs` は docs に埋め込む example formatting のために `configexample.FormatExampleOutput` だけを使ってよい。
- `scriptio` は domain package に依存しない。
- entrypoint は orchestration、fixed repo file read/write、exit だけを持つ。
- import cycle 回避のためだけの interface / wrapper は追加しない。

## Generated Surface

`make gen-all` 後、以下は refactor による内容差分が出てはいけない。

- `config.yaml.example`
- `docs/config.md`
- `docs/commands.md`
- `internal/config/registry_generated.go`
- `internal/agent/help_generated.go`

差分が出た場合は generator drift、stale generated 更新、予期しない behavior change を切り分ける。

## Test Boundary

- `configmeta` tests は metadata consistency、category/order、field path、example policy を固定する。
- `configregistry` tests は field type mapping、category fields、registry source、MCP metadata preservation を固定する。
- `configdocs` tests は marker replacement、AST parse/render、details marker present/absent、MCP details render を固定する。
- `configexample` tests は internal field filtering、comment injection、example output、provider model explicit zero、example policy override を固定する。
- `scriptio` tests は output path argument と optional file read を固定する。
- `commanddocs` / `helpgen` tests は既存 owner contract を維持する。

## Verification

Required focused commands:

```sh
go test ./scripts/internal/...
go test ./internal/config ./scripts/internal/configmeta ./scripts/internal/configregistry ./scripts/internal/configdocs ./scripts/internal/configexample ./scripts/internal/scriptio
```

Required full gate:

```sh
make gen-all
go test ./...
make ci-check
```

追加 self-review:

```sh
rg "scripts/internal/configgen|package configgen" scripts docs/dev/scripts-refactor-master-plan.md
git diff --check
git diff -- config.yaml.example docs/config.md docs/commands.md internal/config/registry_generated.go internal/agent/help_generated.go
```

## Final-A / Final-B Gate

Final-A では generated surface、MCP metadata、docs/config marker、command docs skeleton、entrypoint file I/O の drift を確認する。

Final-B では production diff と test diff の両方について、次を確認する。

- package split が owner と依存方向を明確にしているか。
- export は必要最小限か。
- 旧 package wrapper、alias、forwarding helper が残っていないか。
- tests が旧 owner に残っていないか。
- config/runtime/user-visible behavior を変えていないか。
- generated file を手編集で合わせていないか。

残る debt は package / file / symbol 単位で報告する。
