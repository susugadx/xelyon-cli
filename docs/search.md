# Search optimization and structured impact

XELYON の検索最適化は、単なる grep や symbol search ではありません。モデルに検索を任せきるのではなく、runtime が調査の入口、検索経路、読むべき証拠、診断情報をまとめて扱います。

## Overview

- `gather_context` は既定の調査入口です。自然言語、symbol-like query、path、line range、locator ID を受け取り、適切な経路へ routing します。
- `search_code(intent=impact)` は shared-change impact analysis の低レベル入口です。1つの起点 symbol から、definition / caller / reference / test / implementation などを構造化します。
- `SemanticEvidence` は LSP / AST / fallback の結果を共通 evidence に変換する内部 contract です。
- `SymbolBundle` は調査結果の構造化 bundle です。identity、definition、sections、risk、RecommendedReads、diagnostics を持ちます。
- `RecommendedReads` は runtime が選ぶ「先に読むべき証拠」です。
- `gather_context` は `RecommendedReads` を compact read で prefetch し、diagnostics に応じて件数を制御します。

## Architecture

```text
gather_context
  -> search_code(intent=impact)
  -> language resolver
  -> LSP / AST / fallback
  -> SemanticEvidence
  -> SymbolBundle
  -> RecommendedReads
  -> diagnostics-aware prefetch
```

## Runtime-owned investigation

XELYON は調査時に、モデルへ低レベル検索ツールを何度も自由実行させるだけの設計にはしていません。runtime が evidence を束ね、definition、caller、test、reference を同じ bundle として返すことで、編集前に読むべき文脈を安定して揃えます。

この設計の狙いは、検索往復と読み過ぎを減らしつつ、調査の証拠を diagnostics と一緒に見える形で返すことです。`/review` の evidence / probe / report フローとも相性がありますが、検索最適化は review 専用ではなく、通常の実装前調査にも使われます。

## Supported targets

現時点で structured impact の深い経路として扱う target は次の範囲です。

| language / target | file_filter / pattern | structured impact depth | notes |
| --- | --- | --- | --- |
| Go | `go`, `*.go`, `**/*.go`, direct `.go` path | definition, callers, references, tests, implementations | LSP / Go navigation を優先 |
| TypeScript | `ts`, `*.ts`, direct `.ts` path | definition, imports, callers, type refs, references, tests | `.tsx` は含めず対象を絞る |
| TypeScript declaration | `*.d.ts`, direct `.d.ts` path | declaration definition, type refs, imports, references | paired implementation は別 target として扱う |
| TSX | `tsx`, `*.tsx`, direct `.tsx` path | definition, JSX usage / callers, type refs, references, tests | React/JSX component 調査向け |
| JavaScript | `js`, `*.js`, direct `.js` path | definition, imports, callers, references, tests | `.mjs` / `.cjs` は structured impact 対象外 |
| JSX | `jsx`, `*.jsx`, direct `.jsx` path | definition, JSX usage / callers, references, tests | JavaScript family の JSX target |

Python、Rust、Java、C# などは LSP サーバー設定を持てますが、上記と同じ structured impact 深度があるとは扱いません。

## file_filter contract

- `file_filter=go`, `*.go`, `**/*.go`, direct `.go` path は Go structured impact を狙います。
- `file_filter=ts`, `*.ts`, direct `.ts` path は TypeScript `.ts` structured impact を狙います。
- `*.d.ts` と direct `.d.ts` path は TypeScript declaration structured impact を狙います。
- `file_filter=tsx`, `*.tsx`, direct `.tsx` path は TSX structured impact を狙います。
- `file_filter=js`, `*.js`, direct `.js` path は JavaScript `.js` structured impact を狙います。
- `file_filter=jsx`, `*.jsx`, direct `.jsx` path は JSX structured impact を狙います。
- `file_filter=typescript` は `.ts` と `.tsx` を広く探す fallback scope です。structured impact target ではありません。
- `file_filter=javascript` は `.js` / `.jsx` / `.mjs` / `.cjs` を広く探す fallback scope です。structured impact target ではありません。

targeted structured impact が必要な場合は、`ts` / `tsx` / `js` / `jsx` / direct path / structured 対象 glob を使ってください。

## Diagnostics

Structured impact は結果に diagnostics summary を付けます。

- `resolved_by`: `lsp`, `ast`, `mixed`, `fallback` のどの経路で解決したか。
- `confidence`: `high`, `medium`, `low` の信頼度。
- `fallback_reason`: fallback や mixed になった理由。
- `incomplete`: evidence が不完全である可能性。
- `truncated`: 結果が上限で切られたこと。
- `budget_limit_hit`: 収集 budget に到達したこと。

詳細 metadata では raw / accepted / dropped refs などの内部診断も保持します。通常は summary を見れば、LSP-first の結果なのか、AST/fallback を含む保守的な結果なのかを判断できます。

## Prefetch policy

`gather_context` は structured impact の `RecommendedReads` を compact read で prefetch します。件数は diagnostics によって制御されます。

| diagnostics | prefetch behavior |
| --- | --- |
| high confidence | 最大 3 件 |
| medium confidence | 最大 2 件 |
| low confidence | 最大 1 件 |
| `resolved_by=fallback` or `resolved_by=mixed` | 最大 1 件 |
| `incomplete=true`, `truncated=true`, `budget_limit_hit=true` | 最大 1 件 |
| ambiguous structured impact | speculative prefetch しない |

制限された場合は `Prefetch limited: ...`、曖昧で skip した場合は `Prefetch skipped: ambiguous structured impact` が discovery note に出ます。

## LSP-first, AST classification, shallow fallback

- LSP は semantic references と definition の第一候補です。
- AST は import / export / call / type ref / test / JSX usage などの分類、snippet、近傍テストの判定を補助します。
- fallback は LSP が使えない場合や不完全な場合の lightweight evidence extractor です。

fallback は自前 LSP や完全な TypeScript resolver ではありません。module graph、`tsconfig` paths、dynamic CommonJS、bundler alias の完全解決を目的にしていません。

## Limitations

- 全言語で semantic impact が完全対応しているわけではありません。
- LSP サーバーを設定できる言語と、structured impact の深い bundle を返せる言語は別です。
- TypeScript / JavaScript fallback は module graph や `tsconfig` path mapping を完全解決しません。
- `.mjs` / `.cjs` は `file_filter=javascript` の fallback scope には含まれますが、JavaScript structured impact target ではありません。
- ambiguous result では候補を返しますが、誤読を避けるため speculative prefetch は行いません。

## Examples

```text
# Go shared-change impact
gather_context query="Build" file_filter=go

# TypeScript function
gather_context query="buildUser" file_filter=ts

# TypeScript declaration
gather_context query="BuildOptions" file_filter="**/*.d.ts"

# TSX component
gather_context query="Button" file_filter=tsx

# JavaScript function
gather_context query="buildUser" file_filter=js

# JSX component
gather_context query="Button" file_filter=jsx
```

`search_code(intent=impact)` は low-level expert override です。通常の調査では `gather_context` を使い、明示的に search route や raw search output を制御したい場合だけ `search_code` を直接使います。
