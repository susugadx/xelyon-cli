# Provider History Reduction 追加最適化 Master Plan

この文書は、dogfood 前に追加したい provider-facing history reduction / compact / policy の内部仕様メモである。公開用 docs ではない。

今後の設計相談で追記し、最後に Codex Goal へ渡して複数の追加最適化をまとめて実装するための master plan として使う。

## 現在実装済みの前提

- provider-facing projection 上の old tool result replacement
  - `read_file`
  - `search_code`
  - `gather_context`
- safe successful test/build/lint command output replacement
- old successful `write_file.content` replacement
- old successful `apply_patch.patch` replacement
- old successful `str_replace` `old_str` / `new_str` / `edits` replacement
- `str_replace` batch `edits` の success count mismatch guard
- provider-facing report / token accounting 改善
- replacement が実際に起きた場合の Responses `previous_response_id` chain disable
- raw `Agent.History` / `Session.Messages` / audit / change records / persisted JSONL を保持する契約
- rehydrated evidence は request-local active context で、history には保存しない契約

## 全体の最重要契約

すべての最適化で以下を守る。

- raw `Agent.History` を変更しない
- `Session.Messages` を変更しない
- audit / change records を変更しない
- persisted JSONL を変更しない
- provider-facing projection 上だけを軽量化する
- request-local active context は history に保存しない
- latest tool suffix は置換しない
- no later assistant の直近 tool result は置換しない
- failed / denied / cancelled / error / unsafe result は安全側で置換しない
- dry-run mode では payload を変更しない
- apply mode で実 replacement が 1 件でも起きた場合だけ Responses `previous_response_id` chain を disable する
- no replacement なら Responses chain を保持する
- placeholder / compact result が元 payload より小さくない場合は置換しない
- min saved tokens threshold を満たさない場合は置換しない

## 実装候補の優先順位

1. `list_dir` old successful tool result replacement
2. successful command output replacement / compact
3. git diff / git show / git status output compact
4. failure command output compact
5. `delete_file` actual replacement の要否判断
6. mode policy / default policy
7. review command cost optimization
8. `/config` / docs / generated config metadata 公開方針
9. default ON 判断基準
10. Goal handoff policy

## 確定設計: tool result content replacement の 2 系統化

現在の `providerhistory` 実装では、`read_file` / `search_code` / `gather_context` の content replacement は evidence pointer 必須の flow として実装されている。

`list_dir` は content replacement ではあるが、evidence pointer 必須ではなく、rehydrate_context 対象でもない。そのため、`list_dir` を単純に `isReductionCandidateTool` に追加する実装は禁止する。

provider-facing tool result content replacement は、以下の 2 系統として扱う。

### evidence-backed tool result replacement

対象:

- `read_file`
- `search_code`
- `gather_context`

性質:

- evidence pointer 必須
- rehydrate_context 対象
- active context transport が必要な場合がある
- replacement text は evidence pointer summary を含む
- `AppliedEvidencePointers` / `BuildRehydratePlan` の対象

### structured tool result replacement

対象:

- `list_dir`

性質:

- evidence pointer 不要
- rehydrate_context 対象外
- assistant tool call arguments と tool result content の parse で安全判定する
- replacement text は path / entries / depth を含む
- `AppliedEvidencePointers` / `BuildRehydratePlan` の対象外

design note:

- `read_file` / `search_code` / `gather_context`: rehydratable evidence
- `list_dir`: re-runnable observation

### helper / predicate 方針

既存の `isReductionCandidateTool` をそのまま拡張して `list_dir` を混ぜない。

実際の関数名は既存命名に合わせてよいが、意味は以下に分ける。

```go
isEvidenceBackedReductionTool(toolName string) bool
isStructuredToolResultReductionTool(toolName string) bool
isToolResultReductionCandidateTool(toolName string) bool
```

期待:

- `isEvidenceBackedReductionTool` は `read_file` / `search_code` / `gather_context` のみ true
- `isStructuredToolResultReductionTool` は現行追加対象では `list_dir` のみ true
- `isToolResultReductionCandidateTool` は上 2 つの union
- `AppliedEvidencePointers` / `BuildRehydratePlan` は `isEvidenceBackedReductionTool` のみを対象にする
- `list_dir` は rehydrate evidence に混ぜない

### `ReductionCandidate` と raw arguments

`ReductionCandidate` に raw assistant tool arguments は追加しない。

理由:

- report / candidate に重い raw payload を持ち込まない
- provider-facing で削りたい情報を report 側へ再保持しない
- raw history 保持契約と provider-facing projection の責務を混ぜない

代わりに、apply 時に `original` messages から tool result linkage を再解決して `linkage.Ref.arguments` を読む。

そのため、必要なら `applyProviderHistoryReduction` の signature を以下のように変更する。

```go
applyProviderHistoryReduction(report *ProjectionReport, original []api.Message, projection []api.Message, policy Policy)
```

`Project` の apply path では `original` clone がすでにあるため、それを渡す。

### active context transport gating

active context transport gating は content candidate 全体への early return にしない。per-candidate にする。

- evidence-backed candidate は、`EvidenceReductionRequiresActiveContext == true` かつ `ActiveContextTransportAvailable == false` の場合 keep する
  - keep reason: `active_context_transport_unsupported`
- structured candidate の `list_dir` は active context transport availability に依存しない
- `list_dir` は safety check と threshold を満たせば apply mode で置換できる

### evidence pointer 判定の適用範囲

以下は evidence-backed candidate にだけ適用する。

- `ambiguous_evidence_pointer`
- `missing_evidence_pointer`
- evidence pointer summary replacement
- applied evidence pointer tracking

`list_dir` には適用しない。

### structured tool result replacement builder

`list_dir` parsing / placeholder build を `reduction_apply.go` に直書きしない。

できれば以下の subpackage を追加する。

```text
internal/providerhistory/toolresults
```

この builder は `list_dir` に対応する。

想定 API は既存命名に合わせてよいが、方向性は以下。

```go
type ReplacementRequest struct {
    ToolName  string
    Arguments string
    Content   string
}

type Replacement struct {
    Kind        string
    Text        string
    SavedBytes  int
    SavedTokens int
}

func BuildStructuredReplacement(req ReplacementRequest) (Replacement, string, bool)
```

戻り値の `string` は keep reason などに使える failure reason として扱ってよい。既存設計に合わせて別形にしてもよい。

### content replacement threshold

content replacement に明確な min saved tokens threshold を導入する。

対象:

- `read_file`
- `search_code`
- `gather_context`
- `list_dir`

方針:

```go
const providerHistoryContentReplacementMinSavedTokens = 128
```

または既存命名に合わせた helper を用意する。

apply mode では、replacement が元 content より小さく、かつ saved tokens が threshold 以上の場合のみ置換する。

threshold 未満の場合:

```text
keep reason: replacement_below_min_saved_tokens
```

dry-run の estimated savings も、apply eligibility とズレないようにする。threshold 未満の候補を savings に過大計上しない。

## 1. `list_dir` old successful tool result replacement

### 目的

古い成功済み `list_dir` tool result を provider-facing projection 上だけで placeholder 化し、raw history を保持したまま provider request の入力トークンを削減する。

### 非目的

以下は今回の対象外。

- `list_dir` tool runtime behavior の変更
- raw history / session / audit / persisted JSONL の変更
- latest tool suffix の置換
- no later assistant の直近 tool result の置換
- failure / denied / cancelled / error result の置換
- generic command output compact
- git diff compact
- failure command compact
- mode policy / default policy
- default ON
- `/config` 公開

### 現在確認できた `list_dir` result format

確認元:

- `internal/tools/file/list_dir.go`
- `internal/tools/file/list_dir_render.go`
- `internal/tools/file/list_dir_runtime.go`
- `internal/tools/file/list_dir_depth_test.go`

現行 `list_dir` は `executeListDirWithRuntimeMode` で request を解決し、`summarizeListDir` と `renderListDirSummary` により文字列を返す。

成功結果の基本形式:

```text
📂 <absPath>
summary: depth=<depth>, dirs=<totalDirs>, files=<totalFiles>
dirs: <dir>/, <dir>/, ...
files: <file> (<bytes> bytes), <file> (<bytes> bytes), ...
subtrees: <n> shown
- <relPath>/ -> dirs=<n>, files=<n>
  dirs: ...
  files: ...
```

depth が 2 以上の場合は `subtrees:` と `- <relPath> -> dirs=<n>, files=<n>` が出る。多すぎる entries は `(+<n> more)` で compact される。

現行 error result の例:

```text
Error: <os/stat/validation error>
Error: <absPath> is not a directory
Error: failed to read directory
```

注意:

- 成功 result 先頭の path は現状 `<absPath>` である。
- provider-facing placeholder に `<absPath>` を入れるとローカル絶対パスを provider に送るため、placeholder の `path` は assistant tool call arguments の `path` を優先する。
- arguments 側の `path` が空、invalid、repo-relative safe path に正規化できない場合は replacement しない。
- result content から path は推定しない。成功 result 先頭の absolute path は safety check の成功形式確認にだけ使い、provider-facing placeholder には絶対に使わない。

### 安全条件

`list_dir` replacement は以下をすべて満たす場合のみ行う。

- tool result が `list_dir` に対応している
- old tool result である
- tool result の後に assistant message が存在する
- latest tool suffix に含まれない
- tool result が成功結果である
- content が現行 `list_dir` 成功形式に一致する
- first non-empty line が `📂 ` header で始まる
- `summary: depth=<depth>, dirs=<dirs>, files=<files>` を parse できる
- error / failed / denied / cancelled / permission denied / unsafe path / invalid path / not a directory 系の内容ではない
- path を assistant tool call arguments から取得できる
- path は repo-relative safe path として正規化できる
- path を安全に取得または正規化できない場合は replacement しない
- placeholder が元 content より小さい
- content replacement threshold 以上の saved tokens がある
- dry-run mode では payload を変更せず candidate / report のみ出す
- apply mode では provider-facing projection 上だけ変更する

success 判定:

- content が現行 `list_dir` 成功形式に一致する
- first non-empty line が `📂 ` header で始まる
- `summary: depth=<depth>, dirs=<dirs>, files=<files>` を parse できる

failure 判定:

- first non-empty line が `Error:` で始まる
- known failure phrase を含む
- `failed`
- `denied`
- `cancelled`
- `permission denied`
- `unsafe path`
- `invalid path`
- `not a directory`
- `path is empty`
- `symlink escape`
- `outside`

success 判定を満たさない場合は replacement しない。上記 failure phrase は初期案であり、実装前に既存 error wording を再確認する。

### placeholder 形式

推奨形式:

```text
[omitted old list_dir result; path=<path>; entries=<dirs+files>; depth=<depth>]
```

`mode=<mode>` は採用しない。現行 output の `summary: depth=<depth>, dirs=<dirs>, files=<files>` に合わせて `depth=<depth>` を使う。

path または summary が不明な場合は replacement しない。

### path / entries / depth の推定

実装前に現在の `list_dir` tool result format と tool arguments を再確認する。

仕様方針:

- placeholder の path は assistant tool call arguments の `path` から取る
- arguments の `path` は `list_dir` 専用 helper で repo-relative safe path として正規化する
- repo-relative safe path に正規化できない場合は replacement しない
- result content の absolute path は path 推定に使わない
- `summary: depth=<depth>, dirs=<dirs>, files=<files>` を parse できる場合だけ replacement する
- `entries=<dirs+files>` として扱う
- `depth=<depth>` を placeholder に入れる
- `summary:` がない、または parse できない場合は replacement しない
- 推定できない値を無理に埋めない

```text
[omitted old list_dir result; path=<path>; entries=<dirs+files>; depth=<depth>]
```

`summary:` が parse できない `list_dir` result は replacement しない。この仕様で確定する。

### `list_dir` path normalization

`taskstate.NormalizeRepoRelativePath` は `"."` を reject するため、`list_dir` 用 path normalization helper を用意する。

`taskstate.NormalizeRepoRelativePath` 自体は変更しない。他の evidence path / edit arg path に影響させないため、`list_dir` 専用 helper で `"."` だけ明示的に許容する。

期待挙動:

```text
"." -> "."
"./" -> "."
"./internal" -> "internal"
"internal/providerhistory" -> "internal/providerhistory"
"../outside" -> reject
"/abs/path" -> reject
"" -> reject
```

result content の absolute path は placeholder path に絶対に使わない。

### rehydrate_context との関係

`list_dir` は rehydrate_context evidence 対象に含めない。

理由:

- `list_dir` は `read_file` / `search_code` / `gather_context` より再取得コストが低い
- 古い directory listing が必要なら再度 `list_dir` すればよい
- directory listing は古くなりやすく、rehydrate で戻すより再取得の方が安全な場合がある

将来的には evidence ではなく re-runnable observation として、再取得方針や report 表示を統合する余地は残す。

### report / status

既存の content replacement report に自然に反映する。

確認項目:

- candidate count
- replaced count
- saved bytes
- approx saved tokens
- replacement status
- total provider-facing savings
- Responses chain disabled

現状の report owner:

- `internal/providerhistory/reduction.go`
- `internal/providerhistory/reduction_detect.go`
- `internal/providerhistory/reduction_apply.go`
- `internal/providerhistory/reduction_report.go`
- `internal/agent/provider_history_reduction_status.go`

初期方針:

- `list_dir` 専用 field は増やさず、tool breakdown が必要な場合も generic content replacement breakdown として扱う
- `ReductionCandidate.ToolName == "list_dir"` として既存 `Candidates` / `Kept` に載せる
- `ContentReplacementSavedBytes` / `ApproxContentReplacementSavedTokens` / `EstimatedSavedBytes` / `ApproxSavedTokens` に既存集計で反映する
- 既存 status 表示では `content_replacements` に含める
- dogfood 観測用に status へ content replacement tool breakdown を追加する

status 例:

```text
content_replacement_tools=read_file:2, list_dir:1
```

apply mode では実際に `ReplacementApplied` になった content candidates を数える。

dry-run mode では replacement eligibility を満たす content candidates を数えるか、既存 report の意味に合わせて candidate counts として扱う。

実装上、必要なら `ProjectionReport` に以下のような map を追加してもよい。

```go
ContentReplacementToolCounts map[string]int
ContentCandidateReasonCounts map[string]int
```

ただし、report struct を増やしすぎない方が自然なら、status formatting 時に `Candidates` から集計してもよい。

### Responses `previous_response_id` chain

apply mode で `list_dir` replacement が 1 件でも実際に起きた場合、既存契約どおり Responses `previous_response_id` chain を disabled にする。

dry-run で candidate があるだけの場合は chain disabled にしない。

no replacement の場合は chain を保持する。

### 実装候補ファイル

最新ソースから確認した候補。

`list_dir` result format / runtime owner:

- `internal/tools/file/list_dir_tool.go`
- `internal/tools/file/list_dir.go`
- `internal/tools/file/list_dir_runtime.go`
- `internal/tools/file/list_dir_render.go`
- `internal/tools/file/list_dir_summary.go`
- `internal/tools/file/list_dir_summary_entries.go`
- `internal/tools/file/list_dir_summary_subtrees.go`
- `internal/tools/file/list_dir_visible_tree.go`
- `internal/tools/file/list_dir_depth_test.go`
- `internal/tools/file/list_dir_execute_test.go`
- `internal/tools/file/list_dir_render_test.go`
- `internal/tools/file/list_dir_runtime_test.go`
- `internal/tools/file/list_dir_summary_test.go`
- `internal/tools/file/list_dir_summary_subtrees_test.go`
- `internal/tools/file/list_dir_visible_tree_test.go`
- `internal/tools/file/direct_query_execute.go`

provider-facing content reduction owner:

- `internal/providerhistory/reduction.go`
- `internal/providerhistory/reduction_detect.go`
- `internal/providerhistory/reduction_apply.go`
- `internal/providerhistory/reduction_report.go`
- `internal/providerhistory/rehydrate_plan.go`
- `internal/providerhistory/projection.go`
- `internal/providerhistory/projection_test.go`
- `internal/providerhistory/package_boundaries_test.go`
- `internal/providerhistory/toolresults` を新設候補にする

command / edit replacement owner。`list_dir` では直接使わない想定だが、threshold / reporting の既存挙動確認に使う。

- `internal/providerhistory/command_edit_dry_run.go`
- `internal/providerhistory/edit_arg_replacement.go`
- `internal/providerhistory/editargs/editargs.go`

agent request / status adapter:

- `internal/agent/provider_history_projection.go`
- `internal/agent/provider_history_request_apply_test.go`
- `internal/agent/provider_history_reduction_apply_lifecycle_test.go`
- `internal/agent/provider_history_reduction_phase5d_openai_test.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/agent/provider_history_reduction_status_test.go`
- `internal/agent/provider_history_rehydrate_context.go`
- `internal/agent/provider_history_rehydrate_context_test.go`
- `internal/agent/provider_history_rehydrate_plan.go`
- `internal/agent/provider_history_rehydrate_plan_test.go`

task / path / evidence owner:

- `internal/taskstate/path_policy.go`
- `internal/taskstate/path_policy_test.go`
- `internal/taskstate/rehydrate_plan.go`
- `internal/taskstate/rehydrate_plan_test.go`
- `internal/tools/common/validation.go`

### 実装設計メモ

実装方針は、`read_file` / `search_code` / `gather_context` と同じ content replacement report に載せつつ、evidence pointer 必須 path とは内部的に分ける。

候補設計:

- `isReductionCandidateTool("list_dir")` を単純に追加するだけにはしない
- helper / predicate は evidence-backed / structured / union に分ける
- `list_dir` は evidence pointer を必要としないため、`applyProviderHistoryReduction` の evidence pointer 必須 flow とは別の replacement builder を持つ
- report 上は既存 content replacement の `Candidates` / `Kept` に載せてよい
- `providerHistoryToolResultLinkage.Ref.arguments` を使って path を読む
- result content の `summary: depth=<depth>, dirs=<dirs>, files=<files>` から entries / depth を読む
- result 先頭の absolute path は provider-facing placeholder に絶対に使わない
- detection 時に `list_dir` 用 candidate reason を入れる
  - 例: `old_successful_list_dir_result`
- apply 時に `buildListDirReplacement(candidate, ref.arguments, content)` のような owner で安全判定する
- builder は assistant tool call arguments の `path` と result content の summary を使う
- builder は `internal/providerhistory/toolresults` に置く候補を第一候補にする
- `ReductionCandidate` に raw arguments は追加しない
- apply 時は `original` messages から tool result linkage を再解決して `linkage.Ref.arguments` を読む
- active context transport gating は per-candidate にし、`list_dir` には適用しない
- evidence pointer ambiguity / missing pointer 判定は `list_dir` に適用しない
- replacement failure は `KeptReason` に具体 reason を入れる
  - `list_dir_result_not_success`
  - `missing_list_dir_path_argument`
  - `unsafe_list_dir_path`
  - `list_dir_summary_unparseable`
  - `replacement_not_smaller`
  - `replacement_below_min_saved_tokens`

命名・配置の実装判断は、末尾の「実装者に委ねる命名・配置判断」に従う。

### テスト方針

既存テストに加えて、以下を必須にする。

#### architecture / gating

- `list_dir` replacement は active context transport unavailable でも、safety check を満たせば apply される
- active context transport unavailable の場合でも、evidence-backed `read_file` candidate は keep される
- 同じ history 内で `read_file` は keep、`list_dir` は replace になる partial apply を確認する
- `list_dir` replacement は `AppliedEvidencePointers` に含まれない
- `BuildRehydratePlan` は `list_dir` replacement だけでは empty plan を返す
- `AppliedEvidencePointers` / `BuildRehydratePlan` は `isEvidenceBackedReductionTool` 相当の predicate だけを見る

#### path

- `path="."` が placeholder path `"."` になる
- `path="./"` が placeholder path `"."` になる
- `path="./internal"` が `"internal"` になる
- `path="internal/providerhistory"` がそのまま使われる
- `path="../outside"` は置換されない
- absolute path は置換されない
- result header の absolute path は placeholder に使われない

#### success parse

- `📂 <absPath>` + `summary: depth=2, dirs=3, files=4` で `entries=7; depth=2` になる
- `summary:` がない場合は置換されない
- `summary:` が malformed の場合は置換されない
- `Error:` result は置換されない
- `Error: <absPath> is not a directory` は置換されない
- `Error: failed to read directory` は置換されない
- latest `list_dir` result は置換されない
- no later assistant の `list_dir` result は置換されない

#### threshold / report

- threshold 未満では置換されず、Responses chain disabled にならない
- threshold 以上では置換され、Responses chain disabled になる
- dry-run では payload を変えず、eligible savings / report だけ出る
- dry-run で threshold 未満の候補を savings に過大計上しない
- content replacement saved bytes / tokens に `list_dir` が反映される
- status に `content_replacement_tools=list_dir:1` が出る
- placeholder が元 content より小さくない場合は置換されない

#### raw preservation

- raw `Agent.History` は保持される
- `Session.Messages` は保持される
- persisted JSONL / audit / change records に影響しない範囲で既存契約を壊さない

## 2. successful command output replacement / compact

### 目的

古い成功済み command output を provider-facing projection 上だけで安全に placeholder または compact 化し、raw history を保持したまま provider request の入力トークンを削減する。

### 非目的

以下は今回の対象外。

- raw `Agent.History` の変更
- `Session.Messages` の変更
- audit / change records / persisted JSONL の変更
- command runtime behavior の変更
- latest tool suffix の置換
- no later assistant の直近 command result の置換
- failed command output compact
- git diff / git show / git status の具体 compact 実装
- secret / env dump / credential を含む可能性がある output の first/last compact
- package install / network fetch / deploy / publish 系 output の first/last compact
- database query / dump output の first/last compact
- 分類不能な command output の full omission / safe placeholder / first/last compact

補足:

- secret / env / config / auth / package / network / install / deploy / database 系は successful command output optimization 全体の対象外ではない
- それらは generic first/last compact の対象外であり、高信頼に分類できる場合は safe placeholder の対象にする
- raw output の first/last lines は provider-facing に残さない

### 現在確認できた command output representation

確認元:

- `internal/providerhistory/command_edit_dry_run.go`
- `internal/providerhistory/reduction.go`
- `internal/providerhistory/reduction_report.go`
- `internal/providerhistory/projection.go`
- `internal/tools/dev/bash_exec.go`
- `internal/tools/execute.go`
- `internal/toolruntime/history.go`
- `internal/agent/tool_result_history.go`

現行 provider history reduction は、tool name が `bash` または `command` の tool result を command output として扱う。

assistant tool call arguments は JSON string で、command text は `command` field から読む。

```json
{"command":"go test ./..."}
```

tool result の provider-facing content は command stdout/stderr を整形した text であり、失敗時は `Error:` prefix を含む場合がある。

```text
Error: exit status 2
Output: ...
```

現行 runtime の `tools.ExecutionResult` には `StartedAt` / `Duration` があるが、provider history projection の入力である `api.Message` には duration metadata は保存されていない。そのため、この章の placeholder / compact は assistant tool call arguments と tool result content だけで組み立てる。duration を入れるのは、projection input から安全に参照できる metadata が導入されている場合だけにする。

既存実装済みの safe successful test/build/lint command output replacement は、この章の `validation success` classifier に統合する。

### 全体方針

successful command output は command output classifier で以下に分類する。

1. validation success
2. observation success
3. file dump success
4. git state success
5. sensitive/config/auth success
6. package/network/install/deploy success
7. database output success
8. unknown success

分類ごとに `placeholder` / `compact` / `skip` を決める。

分類ごとの action:

- validation success: `placeholder`
- observation success: `first/last compact`
- file dump success: `first/last compact`
- git state success: `git-specific owner に dispatch`
- sensitive/config/auth success: `safe placeholder`, never first/last compact
- package/network/install/deploy success: `safe placeholder`, never first/last compact
- database output success: classifier confidence が高い場合のみ `safe placeholder`, never first/last compact
- unknown success: `skip`

共通条件:

- tool result が `bash` または `command` に対応している
- assistant tool call arguments から `command` を取得できる
- old command result である
- tool result の後に assistant message が存在する
- latest tool suffix に含まれない
- no later assistant の直近 command result ではない
- command result が successful と判定できる
- failed / denied / cancelled / interrupted / partial / unsafe / nonzero exit ではない
- replacement / compact text が元 output より小さい
- saved tokens が command output replacement threshold 以上
- dry-run mode では payload を変更せず candidate / report のみ出す
- apply mode では provider-facing projection 上だけ変更する

command output replacement threshold:

- source of truth は providerhistory の command output policy に置く
- 現行の `providerHistoryCommandReplacementMinSavedTokens = 128` と同じ 128 saved tokens を採用する
- validation placeholder / observation compact / file dump compact / safe placeholder のすべてに共通適用する

success 判定:

- tool result が `Error:` で始まらない
- command result が nonzero exit pattern を含まない
- interrupted / partial / cancelled / killed 系 phrase を含まない
- classifier ごとの success evidence を満たす

shell composition:

- `&&` / `||` / `;` / pipe / redirect / command substitution / newline を含む command は、全 command parts を安全に分類できる場合だけ対象にする
- 1 つでも generic first/last compact から除外する family または unknown unsafe part が混ざる場合は、first/last compact しない
- 該当 family が高信頼に分類できる場合は safe placeholder 方針へ送り、分類できない場合は unknown success として skip する
- validation placeholder は単一 validation command、または全 parts が validation success として安全に分類できる command に限定する

execution approval 用の `config.ClassifyShellCommand` は参考情報として使ってよいが、provider history reduction の source of truth にはしない。実行承認の安全分類と、provider-facing output を削ってよいかの分類は責務が違う。

### 1. validation success

#### 対象

テスト・ビルド・lint・typecheck・fmt check など、成功した事実が主な価値であり、古い詳細ログの再読価値が低い command。

例:

```sh
go test ./...
go test ./internal/...
go build ./...
go vet ./...
npm test
npm run test
npm run lint
npm run typecheck
npm run build
pnpm test
pnpm lint
yarn test
yarn lint
cargo test
cargo build
cargo clippy
cargo check
pytest
ruff check
mypy
tsc --noEmit
```

#### replacement 方針

古い成功済み validation output は placeholder 化してよい。

placeholder 例:

```text
[omitted old successful validation command output; command="go test ./..."; exit=0; classifier=validation]
```

placeholder には以下を含める。

- command
- exit code
- classifier: `validation`

安全に取れる場合だけ以下を含める。

- summary line
- duration

summary line は、成功を示す whitelisted one-line summary に限定する。

例:

- `ok <package> <duration>`
- `test result: ok`
- `build succeeded`
- `compiled successfully`
- `lint clean`
- `0 errors`
- `0 problems`
- `Process exited with code 0`

summary line は 1 行に正規化し、quote / tab / newline を provider-facing placeholder に残さない。安全な summary が取れない場合は summary を省略する。

output 本文を provider-facing に残す必要はない。

#### classifier detail

validation success は command family と output success evidence の両方で判定する。

- command が validation family である
- output に failure / warning-as-issue / interrupted / partial / nonzero evidence がない
- output に success evidence がある

success evidence がない validation command output は unknown success として扱い、validation placeholder にはしない。

### 2. observation success

#### 対象

output 自体が調査結果・探索結果である command。

例:

```sh
rg "ProviderHistory"
grep -R "ProviderHistory" internal
find internal -name "*.go"
ls -la
wc -l ...
tree ...
```

#### compact 方針

full omission しない。

古い成功済み output が大きい場合だけ compact する。

large observation threshold:

- original line count が 80 lines 以上、または original bytes が 16 KiB 以上
- compact 後の saved tokens が command output replacement threshold 以上

compact には最低限以下を残す。

- command
- classifier: `observation`
- total lines
- total bytes
- first 20 lines
- last 20 lines
- omitted line count

compact 例:

```text
[compacted old successful observation command output; command="rg ProviderHistory"; classifier=observation; lines=240; bytes=18320]
<first 20 lines>
...
[omitted 200 lines]
...
<last 20 lines>
```

20 行より少ない side がある場合も、同じ line を重複して残さない。omitted line count が 0 になる output は compact しない。

observation command でも、command family または output が sensitive / config / auth / package / network / install / deploy / database 系に該当する場合は generic first/last compact から除外し、該当 family の safe placeholder 方針へ送る。

### 3. file dump success

#### 対象

file content を直接出力する command。

例:

```sh
cat file.go
sed -n '1,200p' file.go
head -200 file.go
tail -200 file.go
bat file.go
```

#### compact 方針

full omission しない。

`read_file` と違って evidence pointer がないため、placeholder 化もしない。

巨大な古い file dump output は observation compact と同じ first/last compact にする。

large file dump threshold:

- original line count が 80 lines 以上、または original bytes が 16 KiB 以上
- compact 後の saved tokens が command output replacement threshold 以上

compact には以下を残す。

- command
- classifier: `file_dump`
- total lines
- total bytes
- first 20 lines
- last 20 lines
- omitted line count

file dump target path または command が secret / env / credential / config dump 系に該当する場合は generic first/last compact から除外する。

この場合、高信頼に sensitive / config / auth family と分類できるなら safe placeholder 方針へ送り、分類できないなら unknown success として skip する。

### 4. git state success

#### 対象

```sh
git status
git diff
git show
git log
git branch
```

#### 方針

successful command classifier では git family として検出するが、具体 compact は `## 3. git diff / git show / git status output compact` の owner に委譲する。

この章で決めること:

- git 系 output を unknown success として扱わない
- git 系 output を validation success として placeholder 化しない
- git 系 output は git-specific compact path に dispatch する
- git compact owner が利用できない場合は skip する

### 5. sensitive/config/auth success

#### 対象

環境変数、設定、認証、system info を出す command。

例:

```sh
env
printenv
set
op read
gh auth status
npm config list
git config --list
docker info
kubectl config view
```

#### 方針

sensitive / config / auth success は generic first/last compact から除外する。

古い成功済み output で安全条件を満たす場合は、raw output の行を残さない safe placeholder 化を行う。

placeholder 例:

```text
[omitted old successful sensitive command output; command_family=env_config; exit=0]
```

never first/last compact:

- secret / credential / token / local environment を含む可能性がある
- provider-facing projection で first / last lines を残すと、部分的に secret を送るリスクがある
- `env` / `printenv` / auth / config / system info の raw output 行は provider-facing placeholder に絶対に残さない

safe placeholder の条件:

- exit code 0
- old result
- latest tool suffix ではない
- no later assistant ではない
- command family を高信頼に分類できる
- placeholder が元 output より小さい
- saved tokens が command output replacement threshold 以上
- raw `Agent.History` / `Session.Messages` / audit / persisted JSONL は保持する

### 6. package/network/install/deploy success

#### 対象

依存 install、network fetch、deploy、publish、curl / wget など。

例:

```sh
npm install
pnpm install
cargo fetch
go mod download
curl ...
wget ...
gh release ...
npm publish
vercel deploy
firebase deploy
```

#### 方針

package / network / install / deploy success は generic first/last compact から除外する。

古い成功済み output で安全条件を満たす場合は、raw output の行を残さない safe placeholder 化を行う。

placeholder 例:

```text
[omitted old successful side-effect command output; command_family=package_install; exit=0]
[omitted old successful external-state command output; command_family=network_fetch; exit=0]
```

never first/last compact:

- side effect や network state が絡む
- output に重要な warning / URL / token / auth / publish result が含まれる可能性がある
- 成功 output でも後続判断に必要な場合がある
- package / network / install / deploy 系の raw output 行は provider-facing placeholder に残さない

safe placeholder の条件:

- exit code 0
- old result
- latest tool suffix ではない
- no later assistant ではない
- command family を高信頼に分類できる
- placeholder が元 output より小さい
- saved tokens が command output replacement threshold 以上
- raw `Agent.History` / `Session.Messages` / audit / persisted JSONL は保持する

### 7. database output success

#### 対象

database query / dump / migration status など、DB 内容や外部状態を出す command。

例:

```sh
psql ...
mysql ...
sqlite3 ...
mongosh ...
pg_dump ...
mysqldump ...
prisma db execute ...
```

#### 方針

database output success は generic first/last compact から除外する。

classifier confidence が高く、古い成功済み output で安全条件を満たす場合だけ、raw output の行を残さない safe placeholder 化を行う。

placeholder 例:

```text
[omitted old successful database command output; command_family=database; exit=0]
```

never first/last compact:

- output に application data、PII、secret、connection info、large dump が含まれる可能性がある
- query / dump output は first / last lines だけでも provider-facing に残すリスクがある
- database 系の raw output 行は provider-facing placeholder に残さない

safe placeholder の条件:

- exit code 0
- old result
- latest tool suffix ではない
- no later assistant ではない
- command family を高信頼に分類できる
- placeholder が元 output より小さい
- saved tokens が command output replacement threshold 以上
- raw `Agent.History` / `Session.Messages` / audit / persisted JSONL は保持する

classifier confidence が低い database-like command output は unknown success として扱い、safe placeholder もしない。

### 8. unknown success

#### 対象

上記分類に入らない successful command output。

#### 方針

基本は skip。

full omission はしない。

first/last compact も行わない。

safe placeholder もしない。

理由:

- command family を高信頼に分類できない
- raw output を残してよいか、全削除してよいかの安全条件を定義できない
- 分類不能な successful output を provider-facing optimization の対象にすると、後続判断に必要な内容を誤って消す可能性がある

### generic first/last compact から除外する command family

以下の command family は、successful でも generic first/last compact しない。

- package install / dependency download
- network fetch
- deploy / publish
- auth / credential
- env / config dump
- docker / kubernetes / cloud status / config
- database dump / query output
- custom script output that cannot be classified safely

このリストは provider-facing optimization そのものからの除外ではない。

family を高信頼に分類でき、safe placeholder 条件を満たす sensitive / package / network / install / deploy / database 系 output は、first/last compact ではなく safe placeholder にする。

unknown success は分類できないため、safe placeholder もしない。

generic first/last compact から除外された command output は `KeptReason` または placeholder kind / classifier breakdown で理由を残す。

例:

- `command_output_safe_placeholder_sensitive`
- `command_output_safe_placeholder_package_install`
- `command_output_safe_placeholder_network_fetch`
- `command_output_safe_placeholder_database`
- `command_output_unknown_skip`

### safe placeholder 共通条件

safe placeholder は以下をすべて満たす場合のみ行う。

- exit code 0
- old result
- latest tool suffix ではない
- no later assistant ではない
- command family を高信頼に分類できる
- placeholder が元 output より小さい
- saved tokens が command output replacement threshold 以上
- raw `Agent.History` / `Session.Messages` / audit / persisted JSONL は保持する

safe placeholder には command family / exit code / omitted reason だけを入れる。raw output の行は残さない。

placeholder 例:

```text
[omitted old successful sensitive command output; command_family=env_config; exit=0]
[omitted old successful side-effect command output; command_family=package_install; exit=0]
[omitted old successful external-state command output; command_family=network_fetch; exit=0]
```

secret / env / config / auth / package / network / install / deploy / database 系では、raw output の first / last lines を provider-facing に絶対に残さない。

### active context / rehydrate_context

successful command output compact は rehydrate_context evidence 対象外。

理由:

- command output は `read_file` / `search_code` / `gather_context` のような evidence pointer 管理対象ではない
- 古い command output は再実行できるものと再実行できないものが混ざる
- rehydrate ではなく compact text を provider-facing に残す

successful command output compact は active context transport availability に依存しない。

### report / status

report / status には command output compact の分類が見えるようにする。

最低限:

- command output candidates
- command output replaced / compacted count
- saved bytes
- approx saved tokens
- classifier breakdown

status 例:

```text
command_output_replacements=4
command_output_tools=validation:2, sensitive:1, package_install:1
```

既存の safe successful test/build/lint command output replacement は validation success classifier に統合する。

dry-run mode では replacement / compact eligibility を満たす command output だけを savings に計上する。skip される command output、unknown success、threshold 未満の command output を estimated savings に過大計上しない。

### Responses `previous_response_id` chain

apply mode で successful command output replacement / compact が 1 件でも実際に起きた場合、既存契約どおり Responses `previous_response_id` chain を disabled にする。

dry-run で candidate があるだけの場合は chain disabled にしない。

no replacement の場合は chain を保持する。

### テスト方針

最低限、以下をテストする。

#### validation success

- old successful `go test ./...` output は placeholder 化される
- old successful `go build ./...` output は placeholder 化される
- old successful lint / typecheck output は placeholder 化される
- latest validation output は置換されない
- no later assistant の validation output は置換されない
- threshold 未満の validation output は置換されない

#### observation success

- old large `rg` output は first/last compact される
- old large `find` output は first/last compact される
- old large `ls` output は first/last compact される
- compact text は command / total lines / omitted count を含む
- small observation output は置換されない

#### git state success

- `git diff` は successful command classifier で unknown 扱いされない
- git compact owner が利用できない場合は skip される
- git output を validation placeholder にしない

#### file dump success

- old large `cat file` output は full omission されない
- old large `cat file` output は file_dump compact される
- compact text は first/last lines を残す

#### safe placeholder / generic first/last compact exclusion

- `npm install` success output は first/last compact されず、safe placeholder 化される
- `curl` success output は first/last compact されず、safe placeholder 化される
- `env` / `printenv` output は first/last compact されず、safe placeholder 化される
- `gh auth status` output は first/last compact されず、safe placeholder 化される
- `docker info` / `kubectl config view` output は first/last compact されず、safe placeholder 化される
- database command output は classifier confidence が高い場合だけ safe placeholder 化される
- sensitive / config / auth / package / network / install / deploy / database 系 placeholder は raw output の first / last lines を含まない

#### unknown success

- unknown command success output は基本 skip
- huge unknown success output でも safe placeholder しない
- unknown success は first/last compact されない
- unknown success は full omission されない

#### raw preservation

- raw `Agent.History` は保持される
- `Session.Messages` は保持される
- persisted JSONL / audit / change records は変更されない
- apply mode で実 compact が起きた場合だけ Responses chain disabled になる
- dry-run mode では payload を変更しない

### 実装候補ファイル

最新ソースから確認した候補。

providerhistory command output reduction owner:

- `internal/providerhistory/command_edit_dry_run.go`
- `internal/providerhistory/reduction.go`
- `internal/providerhistory/reduction_report.go`
- `internal/providerhistory/projection.go`
- `internal/providerhistory/projection_test.go`
- `internal/providerhistory/commandoutputs` を command output classifier / builder / compactor 用 subpackage の新設候補にする

agent request / Responses chain / status owner:

- `internal/agent/provider_history_projection.go`
- `internal/agent/provider_history_reduction.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/agent/provider_history_reduction_status_test.go`
- `internal/agent/provider_history_command_edit_dry_run_test.go`
- `internal/agent/provider_history_command_edit_test_helpers_test.go`
- `internal/agent/provider_history_reduction_phase5d_openai_test.go`
- `internal/agent/provider_history_request_apply_test.go`

command execution / result formatting owner。runtime behavior は変更しないが、command representation と success / error wording の確認に使う。

- `internal/tools/dev/bash_exec.go`
- `internal/tools/dev/bash_guard.go`
- `internal/tools/dev/bash_safe_command.go`
- `internal/tools/dev/bash_safety.go`
- `internal/tools/execute.go`
- `internal/tools/context.go`
- `internal/toolruntime/history.go`
- `internal/agent/tool_result_history.go`

command parsing / shell classification reference:

- `internal/commandruntime/parse.go`
- `internal/commandruntime/parse_quote_policy.go`
- `internal/config/execution.go`
- `internal/config/execution_test.go`

## 3. git diff / git show / git status output compact

### 目的

古い git state / diff 系 command output を provider-facing projection 上だけで安全に compact 化し、raw history を保持したまま provider request の入力トークンを削減する。

git output は単なる successful command output ではなく、repo state / diff evidence である。そのため、section 2 の successful command output classifier では git family として検出し、この section の git-specific compact owner に dispatch する。

### 非目的

以下は今回の対象外。

- raw `Agent.History` の変更
- `Session.Messages` の変更
- audit / change records / persisted JSONL の変更
- git command runtime behavior の変更
- latest tool suffix の置換
- no later assistant の直近 git result の置換
- git command の再実行
- diff の semantic review
- diff の正しさ判定
- failed git command の generic failure compact
- git output の safe placeholder full omission

特に、`git diff` / patch output を `[omitted ...]` だけに潰すことは禁止する。diff は後続の説明・レビュー・修正判断の根拠になるため、compact しても最低限の file / hunk / changed-line context を残す。

### 対象 command family

git-specific compact owner は、以下を扱う。

#### repo status

```sh
git status
git status --short
git status --porcelain
git status --porcelain=v1
git status --porcelain=v2
```

#### diff / patch

```sh
git diff
git diff --staged
git diff --cached
git diff HEAD
git diff <rev>
git diff <rev>..<rev>
git diff --stat
git diff --name-only
git diff --name-status
git diff --numstat
git diff --exit-code
```

#### show

```sh
git show
git show <rev>
git show --stat <rev>
git show --name-only <rev>
git show --name-status <rev>
```

#### log / branch / file list

```sh
git log
git log --oneline
git log --stat
git log -p
git branch
git branch -a
git ls-files
```

#### explicit non-targets

`git grep` は git command だが、repo state / diff ではなく observation output なので section 2 の observation compact に任せる。

`git config --list` は git command だが config / sensitive family なので section 2 の sensitive safe placeholder に任せる。

`git remote -v` は URL / remote state を含むため、git-specific first/last compact の対象にしない。高信頼に external-state / config family と分類できる場合は safe placeholder、そうでなければ skip する。

### 共通安全条件

git-specific compact は以下をすべて満たす場合のみ行う。

- tool result が `bash` または `command` に対応している
- assistant tool call arguments から `command` を取得できる
- command family を git state / diff family として高信頼に分類できる
- old command result である
- tool result の後に assistant message が存在する
- latest tool suffix に含まれない
- no later assistant の直近 command result ではない
- output が parse 可能な git state / diff / log / show 形式である
- compact text が元 output より小さい
- saved tokens が command output replacement threshold 以上
- dry-run mode では payload を変更せず candidate / report のみ出す
- apply mode では provider-facing projection 上だけ変更する

parse できない git output は skip する。parse できない diff を generic first/last compact に落とさない。git diff は file / hunk 構造を失うと誤解を生むため、parse 不能なら安全側で保持する。

ANSI color codes は classification / parse 前に除去してよい。provider-facing compact output には ANSI escape sequence を残さない。

### semantic success / `git diff --exit-code`

`git diff --exit-code` は差分がある場合に exit code 1 になるが、これは必ずしも実行失敗ではなく repo-state observation である。

方針:

- command が `git diff --exit-code` 系である
- output に parse 可能な diff が含まれる
- error / fatal / permission / repository corruption などの実エラーがない

この条件を満たす場合、exit status 1 でも git-specific diff compact の対象にしてよい。

ただし、以下は対象外。

- `fatal:`
- `error:`
- `not a git repository`
- ambiguous revision
- permission denied
- command killed / cancelled / timeout

これらは section 4 の failure command output compact で扱う。

### compact action

git-specific owner は command family ごとに以下の action を取る。

- `git status`: status compact
- `git diff`: diff compact
- `git show`: show metadata compact + diff compact
- `git log`: log compact
- `git branch`: branch compact
- `git ls-files`: file-list compact
- `git diff --stat` / `--name-only` / `--name-status` / `--numstat`: summary/list compact

full omission / safe placeholder はしない。

### 1. git status compact

#### 対象

`git status` 系 output。

#### 方針

小さい status output は compact しない。大きい status output のみ compact する。

large status threshold:

- original line count が 80 lines 以上、または original bytes が 16 KiB 以上
- compact 後の saved tokens が command output replacement threshold 以上

#### compact に残す情報

- command
- classifier: `git_status`
- branch / detached HEAD が分かる場合は branch
- ahead / behind が分かる場合は counts
- staged count
- unstaged count
- untracked count
- conflicted count
- ignored count が分かる場合は count
- category ごとの first 20 paths
- category ごとの omitted count

compact 例:

```text
[compacted old git status output; command="git status --short"; classifier=git_status; branch=feature/x; staged=3; unstaged=8; untracked=120]
staged:
  M internal/foo.go
  A internal/bar.go
  [omitted 1 staged path]
unstaged:
  M internal/baz.go
  [omitted 7 unstaged paths]
untracked:
  docs/a.md
  docs/b.md
  [omitted 118 untracked paths]
```

path lines は raw status output から取得するが、ANSI escape sequence は除去する。

### 2. git diff compact

#### 対象

patch output を含む `git diff` 系 command。

#### 方針

`git diff` は full omission しない。diff が大きい場合に semantic compact する。

large diff threshold:

- original line count が 120 lines 以上、または original bytes が 24 KiB 以上
- compact 後の saved tokens が command output replacement threshold 以上

小さい diff は compact しない。小さい diff は後続判断で全文が有用なため、そのまま provider-facing に残す。

#### parse 対象

unified diff の代表的な要素を parse する。

- `diff --git a/<path> b/<path>`
- `index ...`
- `new file mode`
- `deleted file mode`
- `similarity index`
- `rename from`
- `rename to`
- `--- a/<path>`
- `+++ b/<path>`
- `@@ -old,+new @@ optional context`
- `Binary files ... differ`
- `GIT binary patch`

#### compact に残す情報

全体 header:

- command
- classifier: `git_diff`
- files changed count
- hunk count
- additions count
- deletions count
- binary file count
- omitted file count

file ごと:

- path
- change kind: modified / added / deleted / renamed / copied / binary / mode-only / unknown
- additions / deletions count
- hunk count
- file header information
- hunk headers
- changed line samples
- omitted changed line count

#### changed line sample policy

diff は first/last of raw output ではなく、file / hunk 構造を保って compact する。

固定 caps:

- max files shown: 20
- max hunks per file: 8
- max changed sample lines per hunk: 8
- max total changed sample lines: 160

changed sample line は `+` / `-` lines を対象にする。`+++` / `---` file header は changed sample には数えない。

context lines は原則残さない。ただし hunk header に trailing context がある場合はそのまま残す。

compact 例:

```text
[compacted old git diff output; command="git diff"; classifier=git_diff; files=12; hunks=31; +342 -128]
file: internal/providerhistory/reduction_apply.go (modified; hunks=4; +42 -18)
  @@ -120,15 +120,31 @@ func applyProviderHistoryReduction(...)
    - old changed line sample
    + new changed line sample
    + another changed line sample
    [omitted 21 changed lines in hunk]
  @@ -240,8 +256,14 @@ func buildReplacement(...)
    - old sample
    + new sample
    [omitted 10 changed lines in hunk]
  [omitted 2 hunks in file]

file: internal/providerhistory/toolresults/list_dir.go (added; hunks=3; +180 -0)
  @@ -0,0 +1,80 @@
    + package toolresults
    + ...
    [omitted 72 changed lines in hunk]

[omitted 9 files]
```

#### binary diff

Binary diff は raw binary patch を provider-facing に残さない。

compact 例:

```text
[compacted old git diff output; command="git diff"; classifier=git_diff; files=1; binary=1]
file: assets/logo.png (binary; changed)
```

`GIT binary patch` body は省略する。

#### rename / copy

rename / copy は path transition を残す。

```text
file: internal/old.go -> internal/new.go (renamed; similarity=92%; hunks=1; +4 -4)
```

#### mode-only diff

mode-only change は path と mode change を残し、raw body は残さない。

```text
file: scripts/run.sh (mode-only; old=100644; new=100755)
```

#### diff stat / name-only / name-status / numstat

`git diff --stat`, `--name-only`, `--name-status`, `--numstat` は patch ではないが git diff family として扱う。

小さい output は compact しない。大きい output は list-style compact する。

list-style compact に残す情報:

- command
- classifier: `git_diff_summary`
- total entries
- first 40 entries
- last 40 entries
- omitted count

### 3. git show compact

#### 対象

`git show` 系 output。

#### 方針

`git show` は commit metadata と diff body を分けて compact する。

残す metadata:

- commit hash
- subject
- date if present
- parent hash if present
- file summary if present

author email は provider-facing compact output に必須ではないため、原則残さない。author line を残す場合は既存 raw output に存在していた情報から最小限にする。安全側では author name / email を omit してよい。

patch body がある場合は `git diff compact` と同じルールで compact する。

compact 例:

```text
[compacted old git show output; command="git show abc123"; classifier=git_show; commit=abc123; subject="provider履歴の圧縮を改善"; files=5; hunks=12; +220 -80]
diff:
file: internal/providerhistory/reduction.go (modified; hunks=2; +20 -10)
  @@ -42,7 +42,9 @@ ...
    - old sample
    + new sample
    [omitted 18 changed lines in hunk]
```

`git show --stat` / `--name-only` / `--name-status` は list-style compact として扱う。

### 4. git log compact

#### 対象

`git log` 系 output。

#### 方針

small log output は compact しない。large log output は commit-list compact する。

large log threshold:

- original line count が 120 lines 以上、または original bytes が 24 KiB 以上
- compact 後の saved tokens が command output replacement threshold 以上

commit-list compact に残す情報:

- command
- classifier: `git_log`
- total commits if parse できる場合
- first 30 commit summaries
- last 30 commit summaries
- omitted commit count

commit summary:

- short hash
- subject
- date if one-line format から安全に取れる場合

commit body は残さない。patch 付き `git log -p` は、commit metadata と diff compact を組み合わせるが、max commits shown を 10 に制限する。

### 5. git branch compact

#### 対象

`git branch` / `git branch -a` output。

#### 方針

small branch output は compact しない。large branch output は branch-list compact する。

残す情報:

- command
- classifier: `git_branch`
- current branch
- total branches
- first 40 branches
- last 40 branches
- omitted count

### 6. git ls-files compact

#### 対象

`git ls-files` output。

#### 方針

`git ls-files` は repo file-list snapshot として扱う。large output は list-style compact する。

残す情報:

- command
- classifier: `git_file_list`
- total entries
- first 40 entries
- last 40 entries
- omitted count

### classifier / dispatch

section 2 の successful command output classifier は git family を検出したら、この git-specific owner に dispatch する。

dispatch rules:

- `git status*` -> `git_status`
- `git diff*` -> `git_diff` or `git_diff_summary`
- `git show*` -> `git_show`
- `git log*` -> `git_log`
- `git branch*` -> `git_branch`
- `git ls-files*` -> `git_file_list`
- `git grep*` -> section 2 observation compact
- `git config*` -> section 2 sensitive / config safe placeholder
- `git remote -v` -> section 2 external-state / config safe placeholder or skip

shell composition:

- `git diff --stat && git status --short` のように全 parts が git-specific owner で分類できる場合は、それぞれ compact して combined compact output を作ってよい
- git command と sensitive / network / unknown command が混在する場合は、安全側で skip する
- pipe / redirect を含む git command は、全 parts を安全に分類できる場合のみ対象にする
- `git diff | cat` のように diff output の形が維持されるだけの command は対象にしてよいが、parse できない場合は skip する

### report / status

report / status には git-specific compact の分類が見えるようにする。

最低限:

- git output candidates
- git output compacted count
- saved bytes
- approx saved tokens
- classifier breakdown

status 例:

```text
git_output_replacements=3
git_output_tools=git_diff:1, git_status:1, git_show:1
```

section 2 の command output report に含める場合でも、git-specific breakdown は dogfood 観測できるようにする。

### Responses `previous_response_id` chain

apply mode で git-specific compact が 1 件でも実際に起きた場合、既存契約どおり Responses `previous_response_id` chain を disabled にする。

dry-run で candidate があるだけの場合は chain disabled にしない。

no replacement の場合は chain を保持する。

### active context / rehydrate_context

git-specific compact は rehydrate_context evidence 対象外。

理由:

- git command output は `read_file` / `search_code` / `gather_context` の evidence pointer 管理対象ではない
- repo state は時間とともに変わる
- 古い git output は rehydrate ではなく compact text として残す
- 必要なら agent は git command を再実行できる

git-specific compact は active context transport availability に依存しない。

### テスト方針

最低限、以下をテストする。

#### status

- old large `git status --short` output は status compact される
- compact text は staged / unstaged / untracked counts を含む
- small `git status` output は compact されない
- latest `git status` output は compact されない
- no later assistant の `git status` output は compact されない

#### diff

- old large `git diff` output は semantic diff compact される
- compact text は file path / hunk header / changed sample lines / omitted counts を含む
- compact text は raw diff の全文を含まない
- small `git diff` output は compact されない
- malformed `git diff` output は compact されない
- binary diff は binary body を含まず path summary だけ残す
- rename diff は old path / new path を残す
- mode-only diff は mode change summary を残す
- `git diff --stat` large output は list-style compact される
- `git diff --name-only` large output は list-style compact される
- `git diff --exit-code` exit status 1 でも parse 可能 diff だけなら git diff compact 対象になる
- real error / fatal git diff output は git-specific compact されない

#### show

- old large `git show` output は metadata + diff compact される
- commit hash / subject は残る
- author email は必須出力にしない
- patch body は diff compact ルールに従う

#### log / branch / ls-files

- old large `git log --oneline` output は commit-list compact される
- patch 付き `git log -p` は commit metadata + diff compact される
- old large `git branch -a` output は branch-list compact される
- old large `git ls-files` output は file-list compact される

#### dispatch / exclusions

- `git grep` は git state compact ではなく observation compact に送られる
- `git config --list` は git state compact ではなく sensitive safe placeholder に送られる
- `git remote -v` は git state compact ではなく external-state / config safe placeholder または skip になる
- git output は validation placeholder にされない
- git diff output は safe placeholder full omission されない

#### threshold / report

- threshold 未満では compact されず、Responses chain disabled にならない
- threshold 以上では compact され、Responses chain disabled になる
- dry-run では payload を変えず、eligible savings / report だけ出る
- dry-run で threshold 未満の git output を savings に過大計上しない
- report / status に git-specific classifier breakdown が出る

#### raw preservation

- raw `Agent.History` は保持される
- `Session.Messages` は保持される
- persisted JSONL / audit / change records は変更されない
- apply mode で実 compact が起きた場合だけ Responses chain disabled になる
- dry-run mode では payload を変更しない

### 実装候補ファイル

最新ソースから確認した候補。

providerhistory command / git output reduction owner:

- `internal/providerhistory/command_edit_dry_run.go`
- `internal/providerhistory/reduction.go`
- `internal/providerhistory/reduction_report.go`
- `internal/providerhistory/projection.go`
- `internal/providerhistory/projection_test.go`
- `internal/providerhistory/commandoutputs` を command output classifier / builder / compactor 用 subpackage の新設候補にする
- `internal/providerhistory/gitoutputs` を git-specific parser / compactor 用 subpackage の新設候補にする

agent request / Responses chain / status owner:

- `internal/agent/provider_history_projection.go`
- `internal/agent/provider_history_reduction.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/agent/provider_history_reduction_status_test.go`
- `internal/agent/provider_history_command_edit_dry_run_test.go`
- `internal/agent/provider_history_command_edit_test_helpers_test.go`
- `internal/agent/provider_history_reduction_phase5d_openai_test.go`
- `internal/agent/provider_history_request_apply_test.go`

command execution / result formatting owner。runtime behavior は変更しないが、command representation と success / error wording の確認に使う。

- `internal/tools/dev/bash_exec.go`
- `internal/tools/dev/bash_guard.go`
- `internal/tools/dev/bash_safe_command.go`
- `internal/tools/dev/bash_safety.go`
- `internal/tools/execute.go`
- `internal/tools/context.go`
- `internal/toolruntime/history.go`
- `internal/agent/tool_result_history.go`

command parser / shell classification reference:

- `internal/commandruntime/parse.go`
- `internal/commandruntime/parse_quote_policy.go`
- `internal/config/execution.go`
- `internal/config/execution_test.go`

## 4. failure command output compact

### 目的

古い失敗済み command output を provider-facing projection 上だけで安全に compact 化し、raw history を保持したまま provider request の入力トークンを削減する。

failure output は、単なるログではなく、失敗原因・修正対象・後続判断の evidence である。そのため、successful validation output のように placeholder だけへ潰すことは禁止する。

failure command output compact は、失敗原因の診断に必要な情報を残しつつ、巨大な stdout / stderr / stack trace / test log / build log を compact する。

### 非目的

以下は今回の対象外。

- raw `Agent.History` の変更
- `Session.Messages` の変更
- audit / change records / persisted JSONL の変更
- command runtime behavior の変更
- command の再実行
- latest tool suffix の置換
- no later assistant の直近 command result の置換
- successful command output replacement / compact
- git diff / git show / git status の successful compact
- failure output の full omission
- failure output の safe placeholder のみへの置換
- secret / env / credential / database dump などを含む可能性がある failure output の無条件 first/last compact

### 全体方針

failure command output compact は、command output classifier で以下に分類する。

1. validation failure
2. build / typecheck / lint failure
3. git failure
4. permission / security / denied failure
5. timeout / cancelled / killed failure
6. package / network / install / deploy failure
7. database failure
8. sensitive / config / auth failure
9. unknown failure

分類ごとに compact 形式を変える。

共通原則:

- command は必ず残す
- failure classifier は必ず残す
- exit / status / reason が分かる場合は必ず残す
- error-looking lines を優先して残す
- first / last lines は補助として使う
- output 全文は古い large failure では残さない
- full omission はしない
- safe placeholder のみにはしない
- secret / env / auth / database 系は raw excerpt を残しすぎない
- compact text が元 output より小さくない場合は置換しない
- saved tokens が command output replacement threshold 未満なら置換しない

### 共通安全条件

failure compact は以下をすべて満たす場合のみ行う。

- tool result が `bash` または `command` に対応している
- assistant tool call arguments から `command` を取得できる
- old command result である
- tool result の後に assistant message が存在する
- latest tool suffix に含まれない
- no later assistant の直近 command result ではない
- command result が failure と判定できる
- compact text が元 output より小さい
- saved tokens が command output replacement threshold 以上
- dry-run mode では payload を変更せず candidate / report のみ出す
- apply mode では provider-facing projection 上だけ変更する

failure 判定:

- tool result が `Error:` で始まる
- `exit status <n>` / `exit code <n>` / `Process exited with code <n>` の nonzero を含む
- `FAIL` / `FAILED` / `failed` を含む
- `panic:` / `fatal:` / `error:` を含む
- `permission denied` / `not allowed` / `unsafe path` を含む
- `timed out` / `timeout` / `killed` / `cancelled` / `interrupted` を含む
- classifier ごとの failure evidence を満たす

注意:

- `git diff --exit-code` の exit status 1 は、parse 可能な diff がある場合は section 3 の git-specific compact で扱う。failure compact に送らない。
- `grep` / `rg` など、match なしで exit code 1 になる command は必ずしも failure ではない。output が空または no-match evidence のみなら compact 対象外または observation no-result として扱う。
- command family によって nonzero exit の意味が違うため、exit code だけで failure compact にしない。

### command output replacement threshold

failure command output compact は section 2 の command output replacement threshold と同じ source of truth を使う。

方針:

```go
providerHistoryCommandReplacementMinSavedTokens = 128
```

または既存命名に合わせた helper / policy を使う。

large failure threshold:

- original line count が 80 lines 以上、または original bytes が 16 KiB 以上
- compact 後の saved tokens が command output replacement threshold 以上

小さい failure output は compact しない。失敗ログは短い場合、全文が最も有用な evidence になるため、そのまま provider-facing に残す。

### error-focused compact format

failure compact は first / last compact だけにしない。error-focused compact として以下を残す。

header:

```text
[compacted old failed command output; command="<command>"; classifier=<classifier>; exit=<code-or-unknown>; lines=<n>; bytes=<n>]
```

sections:

1. `summary:`
   - exit code / status / reason
   - detected failure category
   - matched error count
2. `key error lines:`
   - error-looking lines
   - file:line diagnostics
   - panic / fatal / FAIL / assertion lines
3. `first lines:`
   - first N lines
4. `last lines:`
   - last N lines
5. omitted marker

N の基本値:

- first lines: 20
- last lines: 40
- key error lines: max 120
- total retained raw-ish lines: max 180

同じ行は重複して残さない。key error lines と first / last lines が重複する場合は一度だけ出す。

### error line matcher

以下に一致する行は key error line として優先的に残す。

generic:

- `Error:`
- `error:`
- `ERROR`
- `fatal:`
- `panic:`
- `FAIL`
- `FAILED`
- `failed`
- `Exception`
- `Traceback`
- `AssertionError`
- `undefined`
- `cannot find`
- `not found`
- `permission denied`
- `access denied`
- `timed out`
- `timeout`
- `killed`
- `cancelled`
- `interrupted`
- `exit status`
- `exit code`

location-style:

- `file.go:123:`
- `file.go:123:45:`
- `path/to/file.ts:123:45`
- `path/to/file.rs:123:45`
- `path/to/file.py:123`
- `path/to/file:line:column`
- `at package/file:line`
- `FAIL: TestName`
- `--- FAIL: TestName`
- `FAILED tests/...`
- `E   <message>` in pytest-like output

The matcher should be conservative. If a line merely contains words like `failed` in a normal path or unrelated text, it may be retained as extra evidence, but it must not be the only reason to classify an otherwise successful command as failure.

### 1. validation failure

#### 対象

test command の失敗。

例:

```sh
go test ./...
go test ./internal/...
npm test
pnpm test
yarn test
cargo test
pytest
```

#### compact 方針

失敗 test 名、package / module、file:line、panic / error 周辺、summary を優先して残す。

残す情報:

- command
- classifier: `validation_failure`
- exit code
- failed package / module が分かる場合は残す
- failed test names
- file:line diagnostics
- panic / stack trace の top relevant lines
- assertion / error lines
- final summary

go test の例:

- `--- FAIL: TestName`
- `panic:`
- `file.go:123:`
- `FAIL <package>`
- `FAIL`
- `ok` lines は原則残さない。ただし final summary として必要なら compact summary に含める。

pytest の例:

- `FAILED <path>::<test>`
- `E   AssertionError`
- traceback の最初と最後
- short test summary info

cargo test の例:

- `failures:`
- `---- test_name stdout ----`
- `thread '...' panicked`
- `test result: FAILED`

placeholder ではなく compact にする。

### 2. build / typecheck / lint failure

#### 対象

build / compile / typecheck / lint / format check の失敗。

例:

```sh
go build ./...
go vet ./...
tsc --noEmit
npm run build
npm run lint
cargo build
cargo check
cargo clippy
ruff check
mypy
eslint
```

#### compact 方針

file:line diagnostics を優先して残す。

残す情報:

- command
- classifier: `build_failure` / `typecheck_failure` / `lint_failure`
- exit code
- diagnostic count が分かる場合は count
- file:line:col diagnostics
- error code が分かる場合は code
- final summary

TypeScript / ESLint / Ruff / MyPy / Go / Rust の diagnostic lines は、key error lines として優先的に残す。

success-looking lines、progress lines、dependency logs は原則省略する。

### 3. git failure

#### 対象

git command の実エラー。

例:

```sh
git status
git diff
git show
git log
git branch
git checkout
git merge
git rebase
```

#### compact 方針

section 3 の git-specific compact は successful repo state / diff output を扱う。git command の実エラーはこの failure compact で扱う。

対象:

- `fatal:`
- `error:`
- `not a git repository`
- `ambiguous argument`
- `unknown revision`
- `pathspec ... did not match`
- `permission denied`
- merge / rebase conflict command failure

残す情報:

- command
- classifier: `git_failure`
- exit code
- fatal / error lines
- relevant path / revision lines
- final summary

`git diff --exit-code` の exit status 1 は、parse 可能な diff output なら failure compact しない。section 3 の git diff compact に送る。

### 4. permission / security / denied failure

#### 対象

permission / safety policy / sandbox / path validation 系の失敗。

例:

- `permission denied`
- `access denied`
- `unsafe path`
- `invalid path`
- `outside repo`
- `not allowed`
- `blocked by policy`
- `approval required`
- `operation cancelled`

#### compact 方針

raw output を長く残す必要は少ない。reason と short excerpt を残す。

残す情報:

- command
- classifier: `permission_failure` / `security_failure`
- exit / status
- policy reason
- relevant path if present
- first matched error lines up to 20
- last lines up to 20

### 5. timeout / cancelled / killed failure

#### 対象

timeout / cancellation / killed process。

例:

- `timeout`
- `timed out`
- `killed`
- `signal: killed`
- `cancelled`
- `interrupted`
- `context deadline exceeded`

#### compact 方針

partial output がある場合、first / last を少し残す。ただし partial output を全文保持しない。

残す情報:

- command
- classifier: `timeout_failure` / `cancelled_failure`
- reason
- partial output lines count
- first 20 lines
- last 40 lines
- key error lines if present

### 6. package / network / install / deploy failure

#### 対象

package install、dependency download、network fetch、deploy、publish 系の失敗。

例:

```sh
npm install
pnpm install
cargo fetch
go mod download
curl ...
wget ...
gh release ...
npm publish
vercel deploy
firebase deploy
```

#### compact 方針

first / last lines を無条件に残さない。warning / URL / token / auth / publish result / local path が混ざる可能性があるため、error-focused lines を優先し、raw excerpt を強く制限する。

残す情報:

- command
- classifier: `package_failure` / `network_failure` / `deploy_failure`
- exit code
- package name / host / URL host が安全に取れる場合は host / package summary
- error lines
- final summary

raw excerpt caps:

- key error lines: max 80
- first lines: max 5
- last lines: max 20
- total retained raw-ish lines: max 100

URLs は可能なら host / path summary に正規化し、query string / token-like fragment は provider-facing compact output に残さない。

### 7. database failure

#### 対象

database query / dump / migration / schema command の失敗。

例:

```sh
psql ...
mysql ...
sqlite3 ...
mongosh ...
pg_dump ...
mysqldump ...
prisma db execute ...
prisma migrate ...
```

#### compact 方針

DB content / PII / connection string / query result が混ざる可能性があるため、first / last lines を無条件に残さない。

高信頼に database family と分類できる場合のみ、error-focused compact を行う。分類 confidence が低い場合は unknown failure として扱う。

残す情報:

- command family
- classifier: `database_failure`
- exit code
- error code / SQLSTATE が分かる場合は残す
- migration name が分かる場合は残す
- error lines

raw excerpt caps:

- key error lines: max 80
- first lines: max 0 by default
- last lines: max 20
- total retained raw-ish lines: max 80

connection strings、credentials、query result rows は provider-facing compact output に残さない。

### 8. sensitive / config / auth failure

#### 対象

env / config / auth / credential / system info 系 command の失敗。

例:

```sh
env
printenv
set
op read
gh auth status
npm config list
git config --list
docker info
kubectl config view
```

#### compact 方針

raw first / last lines を残さない。失敗 reason と high-level classifier だけを残す。

残す情報:

- command family
- classifier: `sensitive_failure` / `auth_failure` / `config_failure`
- exit code
- known failure reason if parse できる場合

compact 例:

```text
[compacted old failed sensitive command output; command_family=auth; classifier=auth_failure; exit=1; reason="authentication failed"]
```

raw output 行は原則残さない。ただし provider-facing に安全な fixed phrase のみ残してよい。

### 9. unknown failure

#### 対象

分類できない failed command output。

#### compact 方針

full omission はしない。safe placeholder のみにはしない。分類不能でも失敗原因を失わないよう、conservative error-focused compact を行う。

ただし sensitive / env / auth / database / network / package / deploy っぽい evidence がある場合は、unknown failure として雑に first / last compact せず、該当 family に送るか skip する。

unknown failure compact:

- command
- classifier: `unknown_failure`
- exit code if present
- key error lines max 80
- first 10 lines
- last 30 lines
- omitted line count

### shell composition

複合 command は慎重に扱う。

- `&&` / `||` / `;` / pipe / redirect / command substitution / newline を含む command は、全 parts を分類できる場合だけ family-specific compact を行う
- 1 つでも sensitive / database / network / package / unknown unsafe part が混ざる場合、raw excerpt caps は最も厳しい family に合わせる
- validation command + observation command のように mixed family の場合、unknown failure ではなく mixed_failure として error-focused compact してよい
- redirect output file path は provider-facing compact に必須ではない。安全な repo-relative path として取れる場合だけ残す
- command substitution を含む場合は、内部 command を安全に分類できない限り skip または strict unknown failure compact にする

### secret / credential redaction

failure compact は raw excerpts を残す場合があるため、provider-facing compact output に secret-like token を残さないようにする。

最低限、以下の redaction を行う。

- `token=...`
- `access_token=...`
- `refresh_token=...`
- `api_key=...`
- `apikey=...`
- `password=...`
- `passwd=...`
- `secret=...`
- `authorization: bearer ...`
- URL query string の `token` / `key` / `secret` / `password`
- common `.env` assignment style

redaction 例:

```text
access_token=[redacted]
Authorization: Bearer [redacted]
https://example.com/path?token=[redacted]
```

redaction が難しい line は残さない。secret redaction は raw history には適用しない。provider-facing compact output のみに適用する。

### report / status

report / status には failure compact の分類が見えるようにする。

最低限:

- failure command candidates
- failure command compacted count
- saved bytes
- approx saved tokens
- classifier breakdown

status 例:

```text
failure_command_replacements=3
failure_command_tools=validation_failure:1, typecheck_failure:1, timeout_failure:1
```

dry-run mode では compact eligibility を満たす failure output だけを savings に計上する。skip される failure output、threshold 未満の failure output、small failure output を estimated savings に過大計上しない。

### Responses `previous_response_id` chain

apply mode で failure command output compact が 1 件でも実際に起きた場合、既存契約どおり Responses `previous_response_id` chain を disabled にする。

dry-run で candidate があるだけの場合は chain disabled にしない。

no replacement の場合は chain を保持する。

### active context / rehydrate_context

failure command output compact は rehydrate_context evidence 対象外。

理由:

- command output は `read_file` / `search_code` / `gather_context` のような evidence pointer 管理対象ではない
- 古い failure output は再実行できるものと再実行できないものが混ざる
- failure evidence は rehydrate ではなく compact text として provider-facing に残す
- 必要なら agent は command を再実行できる

failure command output compact は active context transport availability に依存しない。

### テスト方針

最低限、以下をテストする。

#### validation failure

- old large `go test ./...` failure output は validation failure compact される
- compact text は failed test name / file:line / FAIL summary を含む
- compact text は full raw test log を含まない
- small test failure output は compact されない
- latest test failure output は compact されない
- no later assistant の test failure output は compact されない

#### build / typecheck / lint failure

- old large `go build ./...` failure output は build failure compact される
- old large `tsc --noEmit` failure output は typecheck failure compact される
- old large lint failure output は lint failure compact される
- compact text は file:line diagnostics を含む
- progress / success-looking lines は過剰に残さない

#### git failure

- `fatal: not a git repository` は git failure compact される
- ambiguous revision error は git failure compact される
- `git diff --exit-code` exit status 1 with parseable diff は failure compact されない
- real `git diff` fatal/error output は git failure compact される

#### permission / timeout

- permission denied output は permission failure compact される
- unsafe path output は security failure compact される
- timeout output は timeout failure compact される
- cancelled / killed output は cancelled / killed failure compact される

#### package / network / deploy

- `npm install` failure output は package failure compact される
- `curl` failure output は network failure compact される
- deploy failure output は deploy failure compact される
- compact text は query token / auth token を残さない
- raw first / last lines caps が strict である

#### database / sensitive

- database failure output は high confidence の場合だけ database failure compact される
- database compact は query result rows / credentials を残さない
- auth / config / env failure output は raw first / last lines を残さない
- sensitive compact は fixed reason / classifier だけを残す

#### unknown failure

- unknown large failure output は unknown failure compact される
- unknown compact は key error lines / first / last を残す
- unknown compact は full omission しない
- sensitive-like unknown output は strict caps または skip になる

#### redaction

- token-like values are redacted in provider-facing compact output
- Authorization bearer token is redacted
- URL query token is redacted
- redaction does not mutate raw `Agent.History`

#### threshold / report

- threshold 未満では compact されず、Responses chain disabled にならない
- threshold 以上では compact され、Responses chain disabled になる
- dry-run では payload を変えず、eligible savings / report だけ出る
- dry-run で threshold 未満の failure output を savings に過大計上しない
- report / status に failure classifier breakdown が出る

#### raw preservation

- raw `Agent.History` は保持される
- `Session.Messages` は保持される
- persisted JSONL / audit / change records は変更されない
- apply mode で実 compact が起きた場合だけ Responses chain disabled になる
- dry-run mode では payload を変更しない

### 実装候補ファイル

最新ソースから確認した候補。

providerhistory command / failure output reduction owner:

- `internal/providerhistory/command_edit_dry_run.go`
- `internal/providerhistory/reduction.go`
- `internal/providerhistory/reduction_report.go`
- `internal/providerhistory/projection.go`
- `internal/providerhistory/projection_test.go`
- `internal/providerhistory/commandoutputs` を command output classifier / builder / compactor / redaction 用 subpackage の新設候補にする
- `internal/providerhistory/failureoutputs` を failure-specific parser / compactor 用 subpackage の新設候補にする

agent request / Responses chain / status owner:

- `internal/agent/provider_history_projection.go`
- `internal/agent/provider_history_reduction.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/agent/provider_history_reduction_status_test.go`
- `internal/agent/provider_history_command_edit_dry_run_test.go`
- `internal/agent/provider_history_command_edit_test_helpers_test.go`
- `internal/agent/provider_history_reduction_phase5d_openai_test.go`
- `internal/agent/provider_history_request_apply_test.go`

command execution / result formatting owner。runtime behavior は変更しないが、command result format、exit / error wording、partial output wording の確認に使う。

- `internal/tools/dev/bash_exec.go`
- `internal/tools/dev/bash_stream.go`
- `internal/tools/dev/bash_guard.go`
- `internal/tools/dev/bash_safe_command.go`
- `internal/tools/dev/bash_safety.go`
- `internal/tools/dev/truncate.go`
- `internal/tools/execute.go`
- `internal/tools/context.go`
- `internal/toolruntime/history.go`
- `internal/agent/tool_result_history.go`

command parser / shell classification reference:

- `internal/commandruntime/parse.go`
- `internal/commandruntime/parse_quote_policy.go`
- `internal/config/execution.go`
- `internal/config/execution_test.go`

failure detection / plan failure wording reference。providerhistory の source of truth にはしないが、既存 failure phrase の確認に使う。

- `internal/agent/plan/failure.go`
- `internal/agent/plan/failure_test.go`
- `internal/agent/plan_failure_detection_test.go`

secret redaction helper owner:

- 既存の汎用 redaction owner は providerhistory 周辺には見当たらないため、provider-facing compact output 専用の redaction を `internal/providerhistory/commandoutputs` または failure-specific subpackage 側に置く候補にする
- raw history / audit / persisted JSONL へ redaction を適用しないことを API contract と tests で固定する

## 5. `delete_file` actual replacement の要否判断

### 結論

現行 `delete_file` は actual replacement しない。

`delete_file` の assistant tool call arguments は基本的に `path` のみであり、payload が小さい。また、削除された path は後続文脈として重要であるため、provider-facing projection 上でも保持する。

### 方針

- `delete_file` は candidate / dry-run report の検出対象として維持してよい
- apply mode でも actual replacement はしない
- `delete_file` が candidate として検出された場合、kept reason を明示する
  - `delete_file_path_kept_context`
  - `delete_file_replacement_not_beneficial`
  - 既存命名に合わせてよい
- `delete_file` だけを理由に Responses `previous_response_id` chain を disable しない
- `delete_file` path は provider-facing projection 上でも保持する
- raw history / session / audit / persisted JSONL は当然保持する

### 再検討条件

将来 `delete_file` が以下を持つようになった場合だけ actual replacement を再検討する。

- batch delete payload
- large metadata
- recursive delete report
- deleted file content snapshot
- long confirmation text

### テスト方針

- `delete_file` candidate は検出される
- apply mode でも arguments は置換されない
- `delete_file` だけでは Responses chain disabled にならない
- report に kept reason が残る
- raw `Agent.History` / `Session.Messages` は保持される

## 6. mode policy / default policy

### 結論

`provider_history_reduction.mode` は、公開・運用上は以下の 3 モードに整理する。

- `off`
- `dry_run`
- `apply`

`auto` を real policy として新規実装しない。

理由:

- `apply` は無条件削除ではなく、安全条件付き deterministic apply である
- 各 replacement / compact はすでに safety gate / parse gate / threshold gate を持つ
- `auto` を追加すると、`apply` との差分が曖昧になり、ユーザーにも実装者にも分かりにくい
- threshold / active context transport / parse failure / unknown classification などは `apply` mode 内の deterministic skip 条件として扱えばよい
- “自動判断” を別モードにすると、なぜ削られた / 削られなかったのか説明しづらくなる

### 現在確認できた mode / config representation

最新ソースで確認できた現状。

- providerhistory の内部 mode は `Disabled` / `DryRun` / `Apply` / `Auto`
- agent alias は `ProviderHistoryReductionDisabled` / `ProviderHistoryReductionDryRun` / `ProviderHistoryReductionApply` / `ProviderHistoryReductionAuto`
- `providerhistory.normalizePolicy` と agent 側 `normalizeProviderHistoryReductionPolicy` は `Auto` を `DryRun` に正規化している
- runtime resolution でも configured `Auto` の effective mode は `DryRun`
- project-local experimental config は `experimental.provider_history_reduction.mode` で `off` / `dry_run` / `apply` / `auto` を受け付けている
- env override は `XELYON_PROVIDER_HISTORY_REDUCTION`
- 現行 default は `off`
- `/config` / `docs/config.md` / generated config metadata の stable surface には provider history reduction を出していない

この section では、既存互換の `auto` 受け付けは維持しつつ、公開・運用上の policy を 3 モードへ整理する。

### mode semantics

#### `off`

provider-facing history reduction を無効化する。

- provider-facing projection を軽量化しない
- report / status の replacement savings は出さない、または disabled として表示する
- Responses `previous_response_id` chain はこの機能を理由に disable しない

#### `dry_run`

provider-facing payload は変更せず、削減見込みだけを report / status に出す。

- raw `Agent.History` は変更しない
- `Session.Messages` は変更しない
- provider-facing projection も変更しない
- replacement / compact candidate を検出する
- apply eligibility を満たす候補だけを estimated savings に計上する
- threshold 未満、parse failure、unknown、unsafe、latest suffix、no later assistant などで skip される候補は savings に過大計上しない
- dry-run candidate だけでは Responses `previous_response_id` chain を disable しない

#### `apply`

provider-facing projection 上だけで、安全条件を満たした replacement / compact を実際に適用する。

`apply` は無条件削除ではない。以下の gate を満たしたものだけが適用される。

- old result / old tool call argument である
- latest tool suffix に含まれない
- no later assistant の直近 result ではない
- tool / command family を高信頼に分類できる
- required parse が成功する
- unsafe / failed / denied / cancelled / unknown の扱いが仕様で定義されている
- placeholder / compact text が元 payload より小さい
- saved tokens が該当 threshold 以上
- evidence-backed replacement では active context / rehydrate transport 条件を満たす
- raw history / session / audit / persisted JSONL は変更しない

apply mode で実 replacement / compact が 1 件でも起きた場合、既存契約どおり Responses `previous_response_id` chain を disable する。

実 replacement / compact が 0 件の場合は Responses chain を保持する。

### `auto` の扱い

`auto` は real policy としては実装しない。

既存 config / env に `auto` が残っている場合は、互換性のため当面受け付けてもよい。ただし effective behavior は `dry_run` として扱う。

方針:

- `/config` には `auto` を表示しない
- public docs には `auto` を推奨しない
- generated config metadata に stable option として出さない
- `auto` が指定された場合、status には `configured=auto effective=dry_run` のように表示してよい
- 将来的には deprecated option として整理する

### public config surface

公開設定として出す場合は、ユーザー向けには 3 択にする。

UI 表示案:

- `Off`
- `Report only`
- `On`

内部値:

- `Off` -> `off`
- `Report only` -> `dry_run`
- `On` -> `apply`

`auto` は公開 UI に出さない。

### default policy

dogfood 前の default は `off` または既存値を維持する。

dogfood 中は project-local experimental YAML または env override で `apply` を使う。

default ON は、この section では決めない。default ON 判断は section 9 で扱う。

default ON の判断材料:

- dogfood で false replacement / false compact が出ない
- `/status` で削減量と chain disable 理由が説明できる
- rehydrate_context が provider 別に破綻しない
- Responses chain disable のコストが許容できる
- command classifier / git compact / failure compact の skip 条件が安全に働く
- unknown / parse failure / threshold 未満が savings に過大計上されない

### report / status

report / status には、configured mode と effective mode を表示する。

例:

```text
provider_history_reduction=apply
effective_mode=apply
```

`auto` 互換入力を受け付ける場合:

```text
provider_history_reduction=auto
effective_mode=dry_run
auto_policy=not_enabled
```

apply mode では、実際に replacement / compact された件数と saved tokens を表示する。

dry_run mode では、apply eligibility を満たす見込みの件数と estimated saved tokens を表示する。

skip reason / kept reason は、必要に応じて debug report または status の詳細に出す。

### テスト方針

最低限、以下をテストする。

#### mode behavior

- `off` では provider-facing projection が変更されない
- `dry_run` では provider-facing projection が変更されず、eligible savings だけ report される
- `apply` では safety gate を満たす replacement / compact だけが provider-facing projection 上で適用される
- `apply` でも threshold 未満 / parse failure / unknown / latest suffix / no later assistant は置換されない

#### Responses chain

- `dry_run` では candidate があっても Responses chain disabled にならない
- `apply` で実 replacement / compact が 1 件以上ある場合だけ Responses chain disabled になる
- `apply` で replacement / compact が 0 件の場合は Responses chain が保持される

#### auto compatibility

- `auto` が指定された場合、effective mode は `dry_run` になる
- `auto` は `/config` stable surface に出ない
- `auto` status では `configured=auto effective=dry_run` のように説明できる

#### raw preservation

- raw `Agent.History` は保持される
- `Session.Messages` は保持される
- audit / change records / persisted JSONL は変更されない

### 実装候補ファイル

最新ソースから確認した候補。

providerhistory mode / projection owner:

- `internal/providerhistory/reduction.go`
- `internal/providerhistory/projection.go`
- `internal/providerhistory/reduction_report.go`
- `internal/providerhistory/projection_test.go`

agent runtime / request / status owner:

- `internal/agent/provider_history_reduction.go`
- `internal/agent/provider_history_reduction_runtime.go`
- `internal/agent/provider_history_projection.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/agent/provider_history_reduction_status_test.go`
- `internal/agent/provider_history_reduction_runtime_test.go`
- `internal/agent/provider_history_request_apply_test.go`

project-local experimental config owner:

- `internal/config/project_experimental.go`
- `internal/config/project_experimental_test.go`
- `internal/config/project_template_test.go`
- `docs/dev/history-active-context.md`

stable `/config` / docs / generated config metadata owner。`auto` は stable option として出さない。

- `internal/config/registry_generated.go`
- `internal/config/show.go`
- `scripts/internal/configgen/sections.go`
- `scripts/gen-config-registry.go`
- `scripts/gen-config-docs.go`
- `scripts/gen-config-example.go`
- `docs/config.md`
- `config.yaml.example`

## 7. review command cost optimization

### 目的

review command の厳しさ・検出品質を維持したまま、provider-facing request に送る review 専用の中間成果物を compact / replacement し、レビュー実行時の入力トークンを削減する。

review command は通常チャットよりも、diff / probe plan / evidence / related context / web evidence / report draft / revision / scope coverage などの中間成果物が多い。そのため、通常の provider history reduction だけでなく、review command 専用の cost optimization を設計する。

### 非目的

以下は今回の対象外。

- review command の検出基準を甘くすること
- review pass 数を減らすこと
- probe / evidence collection を省略すること
- final findings を削ること
- finding evidence を削ること
- file path / line reference を削ること
- raw review audit / persisted JSON / debug artifacts の削除
- review report の品質低下
- “問題なし” 判断の根拠省略

### 最重要契約

review command optimization でも、以下を守る。

- raw review artifacts は保持する
- raw evidence は保持する
- final report は保持する
- final findings は保持する
- finding evidence は保持する
- provider-facing projection 上だけを軽量化する
- 中間成果物を compact しても、最終判断に必要な current state は残す
- compact できないものは安全側で保持する
- compact / replacement が起きた場合だけ Responses `previous_response_id` chain disable の既存契約に従う
- dry-run mode では provider-facing payload を変更しない
- apply mode では provider-facing projection 上だけ変更する

### 現在確認できた review request / artifact representation

最新ソースで確認できた現状。

- `internal/review` は `/review` の domain contract と orchestration owner
- `ReviewRunner.Run` は evidence build、probe plan、probe execution、report、saturation check、report revision を順に実行する
- review model request は通常会話 history ではなく、`reviewModelPromptHistory(req.Prompt)` の単一 user prompt として provider に渡る
- agent review model adapter は `suspendReviewModelResponseContinuation` で review model call 中の response continuation を一時停止している
- review artifact は `ReviewRunArtifactWriter` に保存され、`XELYON_REVIEW_RUN_ARTIFACTS=1` の場合は `.xelyon/review-runs/<runID>/...` に flush される
- artifact 名は `evidence.md`、`evidence_post_pass1.md`、`web_search_evidence.json`、`web_search_evidence_post_pass1.json`、`probe_requests.json`、`probe_results.json`、`probe_plan_prompt.md`、`probe_plan_raw.json`、`probe_plan_final.json`、`report_prompt.md`、`report_raw.json`、`report_final.json`、`saturation_prompt.md`、`saturation_raw.json`、`revision_prompt.md`、`revision_raw.json` など
- `modelinput.BuildProbeResultPromptContexts` は probe command output を command 単位 8 KiB、合計 64 KiB に cap している
- `reviewRunnerPromptRedactor` は prompt / artifact 向けに repo root、cwd、probe workdir などの path redaction を担当している
- external doc evidence は `externaldoc.Evidence` として URL、source credibility、content hash、snippet、fetched_at などを持つ
- report schema の `ReviewEvidenceRef` は `external_doc` の `doc_id` / `snippet_id` / `url` / `content_hash` を参照できる

この section の optimization は、既存の output cap / redaction / artifact preservation を置き換えない。review pass 間で再送される provider-facing prompt payload を短くするため、review runner / modelinput 側に review 専用の current state summary と obsolete intermediate compact を追加する方針にする。

### 基本方針

review command の provider-facing history は、以下の 3 層に分けて扱う。

#### 1. durable review state

レビュー品質に必要な現在状態。provider-facing に残す。

例:

- current diff summary
- current changed file list
- current impact surfaces
- current candidate risks
- current scope coverage
- current unresolved questions
- current final / latest report
- final findings
- finding evidence references

#### 2. raw evidence / audit artifacts

正確性・再検証のために保持するが、古くなったら provider-facing には raw 全量を再送しない。

例:

- raw probe outputs
- raw related context snippets
- raw web / external_doc snippets
- raw command outputs
- raw git diff snapshots
- raw intermediate JSON

これらは raw artifact / audit / persisted data として保持する。provider-facing には compact summary / evidence reference / content hash / citation-like pointer を送る。

#### 3. obsolete intermediate state

後続の current state に吸収済みの中間成果物。provider-facing では compact または placeholder 化する。

例:

- obsolete probe plan
- obsolete probe result raw body
- previous report draft
- previous scope coverage draft
- previous computed summary draft
- previous saturation check details
- previous revision input
- duplicated related context candidate list

### review state summary

review command は、pass が進むごとに provider-facing 用の `review state summary` を作る。

`review state summary` には最低限以下を含める。

- review target
- base / head / diff range が分かる場合はその情報
- changed files summary
- impact surfaces summary
- candidate risks summary
- confirmed findings summary
- dismissed / false-positive risks summary
- unresolved risks / questions
- scope coverage summary
- external evidence summary
- latest report status
- next probe focus

この summary は raw evidence の代替ではない。raw evidence は保持しつつ、provider-facing では現在のレビュー状態を短く再提示するためのもの。

### replacement / compact 対象

#### 1. obsolete probe plan

古い probe plan は、後続の probe result / scope coverage / current state summary に吸収された場合、provider-facing で compact できる。

placeholder 例:

```text
[omitted obsolete review probe plan; probes=12; absorbed_by=review_state_summary]
```

残す情報:

- probe count
- target surfaces
- unresolved count
- absorbed_by marker

#### 2. obsolete probe result

古い probe result の raw body は、finding / dismissed risk / scope coverage に反映済みなら compact できる。

compact 例:

```text
[compacted obsolete review probe result; probe="check provider history chain"; outcome=no_finding; evidence_refs=3]
summary:
- checked Responses chain disable behavior
- no finding after tests covered apply/dry-run split
```

finding に使われた probe result は、finding evidence reference を残す。finding evidence そのものは消さない。

#### 3. related context search results

古い related context candidate list は、selected evidence / dismissed candidate summary に畳む。

残す情報:

- query / surface
- selected files count
- selected refs
- dismissed count
- reason summary

raw list 全量は provider-facing に再送しない。

#### 4. web / external_doc evidence

web / external_doc evidence は、raw snippet を重複送信しない。

残す情報:

- source kind: official / external / unknown
- URL or source id if safe and already intended for report
- content hash
- short summary
- used_for finding id / risk id
- freshness if available

official / external の判定根拠は保持する。非公式サイトを official 扱いしない。

#### 5. report draft / revision input

final report より古い report draft は、latest report / revision summary に吸収された場合 provider-facing で compact できる。

placeholder 例:

```text
[omitted obsolete review report draft; superseded_by=report_revision_2; findings=3; dismissed=5]
```

ただし final report / latest report は compact しない。

#### 6. scope coverage intermediate

古い scope coverage intermediate は、current scope coverage に吸収済みなら compact できる。

残す情報:

- covered surfaces
- uncovered surfaces
- unresolved risks
- saturation status

#### 7. duplicated diff / git output

review command 内で同じ diff / status / show output が複数回 provider-facing に出る場合、古いものは section 3 の git-specific compact を使う。

current diff summary は残す。古い raw diff 全量を再送しない。

### compact 禁止対象

以下は provider-facing でも原則 compact しない。

- final findings
- finding evidence excerpts
- finding file path / line range
- latest final report
- current unresolved risks
- current scope coverage
- current impact surfaces
- current candidate risks
- current review decision summary
- user-provided review instructions
- safety / policy decisions that affect review behavior

### review-specific report / status

review command cost optimization の結果を report / status に出す。

最低限:

- review compact candidates
- review compact applied count
- saved bytes
- approx saved tokens
- compact family breakdown

例:

```text
review_history_replacements=6
review_history_tools=probe_plan:1, probe_result:2, report_draft:1, related_context:2
review_history_saved_tokens≈4200
```

通常の provider history reduction report に合算する場合でも、review-specific breakdown は dogfood 観測できるようにする。

### active context / rehydrate_context

review command の raw evidence は、通常チャットの `rehydrate_context` とは別に、review state summary / evidence reference として扱う。

方針:

- raw evidence は保持する
- provider-facing には compact summary / refs を送る
- 必要なら review runner が raw evidence を request-local に再注入できる
- history には再注入した raw evidence を保存しない

この機能は通常の `read_file` / `search_code` / `gather_context` rehydrate と混ぜない。review runner 専用の evidence lifecycle として扱う。

### Responses `previous_response_id` chain

apply mode で review-specific compact / replacement が 1 件でも実際に起きた場合、既存契約どおり Responses `previous_response_id` chain を disabled にする。

dry-run で candidate があるだけの場合は chain disabled にしない。

no replacement の場合は chain を保持する。

現行 review model adapter は review model call 中に response continuation を suspend している。実装時は `suspendReviewModelResponseContinuation` と二重の state owner を作らず、review-specific compact による chain disable の status / report 表示と runtime response-id handling の責務を整理する。

### テスト方針

最低限、以下をテストする。

#### raw preservation

- raw review evidence は保持される
- raw review artifacts は保持される
- final report は保持される
- final findings は保持される
- provider-facing projection だけ compact される

#### probe plan / result

- obsolete probe plan は compact される
- current probe plan は compact されない
- obsolete probe result は compact される
- finding evidence に使われた probe result は evidence ref を残す
- unresolved probe result は compact されない

#### related context

- old related context candidate list は compact される
- selected evidence refs は残る
- dismissed candidate summary は残る

#### web / external_doc evidence

- duplicated external_doc raw snippet は compact される
- source kind / content hash / summary は残る
- official / external の区別は保持される
- non-official source を official 扱いしない

#### report draft / revision

- obsolete report draft は compact される
- latest report は compact されない
- final report は compact されない
- revision summary は残る

#### scope coverage

- obsolete scope coverage intermediate は compact される
- current scope coverage は保持される
- unresolved risks は保持される

#### diff / git output

- duplicated old raw diff は section 3 git compact に送られる
- current diff summary は保持される

#### mode / chain

- dry-run では provider-facing payload を変更しない
- apply では provider-facing projection 上だけ compact される
- apply で実 compact が起きた場合だけ Responses chain disabled になる
- no replacement の場合は Responses chain が保持される

### 実装候補ファイル

最新ソースから確認した候補。

review orchestration / pass lifecycle owner:

- `internal/review/runner.go`
- `internal/review/runner_saturation.go`
- `internal/review/runner_artifact.go`
- `internal/review/review_model.go`
- `internal/review/runner_test.go`
- `internal/review/runner_saturation_test.go`
- `internal/review/runner_artifact_test.go`

review model input / compact prompt owner:

- `internal/review/modelinput/prompt.go`
- `internal/review/modelinput/saturation.go`
- `internal/review/modelinput/context.go`
- `internal/review/modelinput/context_test.go`
- `internal/review/modelinput/prompt_contract_test.go`
- `internal/review/modelinput/saturation_contract_test.go`

review evidence / current state summary owner:

- `internal/review/evidence/evidence_builder.go`
- `internal/review/evidence/evidence_model_input.go`
- `internal/review/evidence/evidence_render_markdown.go`
- `internal/review/evidence/evidence_types.go`
- `internal/review/evidence/evidence_diff.go`
- `internal/review/evidence/evidence_git_output.go`
- `internal/review/evidence/evidence_related_search.go`
- `internal/review/evidence/evidence_web_search.go`
- `internal/review/evidence/evidence_inventory.go`
- `internal/review/evidence/evidence_context.go`

review probe result owner:

- `internal/review/probe/probe_plan.go`
- `internal/review/probe/probe_runner.go`
- `internal/review/probe/probe_summary.go`
- `internal/review/probe/probe_runtime_types.go`
- `internal/review/probe/report_facade.go`

review report / scope coverage / revision owner:

- `internal/review/report/report_types.go`
- `internal/review/report/report_validation.go`
- `internal/review/report/report_computed_summary.go`
- `internal/review/report/coverage_audit.go`
- `internal/review/report/coverage_audit_scope.go`
- `internal/review/report/coverage_audit_external.go`
- `internal/review/report/coverage_audit_merge.go`
- `internal/review/modeloutput/report.go`
- `internal/review/modeloutput/saturation.go`

review web / external doc evidence owner:

- `internal/review/externaldoc/types.go`
- `internal/review/externaldoc/fetcher.go`
- `internal/review/externaldoc/content.go`
- `internal/review/externaldoc/credibility.go`
- `internal/review/externaldoc/support.go`
- `internal/review/externaldoc/query.go`
- `internal/review/evidence/externaldoc.go`

adapter / provider request / usage status owner:

- `internal/agent/review_runner.go`
- `internal/agent/review_model_adapter.go`
- `internal/reviewadapter/runner_factory.go`
- `internal/tuiagent/tui_adapter_review_test.go`
- `internal/tuiagent/tui_adapter_review_progress.go`
- `internal/tui/review_run_result.go`

providerhistory integration point。通常会話 history projection とは別 owner にするか、report/status だけ共有するかを実装時に判断する。

- `internal/providerhistory`
- `internal/providerhistory/reduction.go`
- `internal/providerhistory/reduction_report.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/agent/provider_history_reduction_status_test.go`

## 8. `/config` / docs / generated config metadata 公開方針

### 結論

provider history reduction は、XELYON の差別化要素として public config surface に出す。

これまでは experimental YAML / env override で dogfood できる内部機能だったが、以下の公開面に昇格する。

- `/config` TUI
- generated config metadata
- `docs/config.md`
- README の feature / differentiator section

ただし、default ON の判断はこの section では行わない。
default ON は section 9 の判断基準で扱う。

### 公開する理由

provider history reduction は、XELYON の重要な差別化要素である。

目的:

- provider-facing request の入力トークンを削減する
- raw history / session / audit / persisted JSONL を保持しながら、provider に送る履歴だけを軽量化する
- read/search/gather/edit/command/git/review の古い中間成果物を安全条件付きで compact / replacement する
- BYOK / API 利用時のコスト削減に寄与する
- 長時間作業や review command の実行コストを抑える
- `/status` で削減量と適用状況を観測できるようにする

この機能は単なる履歴削除ではなく、provider-facing projection の最適化である。
raw `Agent.History` / `Session.Messages` / audit / change records / persisted JSONL は保持する。

### 現在確認できた config surface

現行ソースでは、provider history reduction は stable config surface には出ていない。

確認済み:

- project-local experimental config は `ProjectConfig.Experimental.ProviderHistoryReduction` として `internal/config/project_experimental.go` にある
- YAML path は `experimental.provider_history_reduction.mode` / `experimental.provider_history_reduction.rehydrate_context`
- mode parser は `off` / `dry_run` / `apply` / `auto` を受け付ける
- env override は `XELYON_PROVIDER_HISTORY_REDUCTION` / `XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT`
- mode 解決は env > project experimental config > default off
- `rehydrate_context` の現行 default は false
- agent runtime では `auto` configured mode を effective `dry_run` として扱う
- `docs/dev/history-active-context.md` に dogfood 用 experimental config の説明がある
- `Config` / generated registry / `/config` TUI / `docs/config.md` / `config.yaml.example` には stable option としては出ていない

公開時は、この experimental-only surface を stable config surface へ昇格する。
既存 experimental config は互換入力として残す。

### stable config surface

stable config として以下を公開する。

推奨 config path は、既存 config 構造に合わせて実装時に確定する。
ただし、public docs では `experimental` ではない stable surface として扱う。

候補:

```yaml
provider_history_reduction:
  mode: dry_run
  rehydrate_context: true
```

または既存構造に合わせて:

```yaml
history:
  provider_reduction:
    mode: dry_run
    rehydrate_context: true
```

実装時は既存 config の命名規則を優先してよい。
ただし、公開 docs では “experimental-only feature” として扱わない。

stable config と experimental config の両方が指定された場合は、stable config を優先する。
env override は config より優先する既存方針に合わせる。

### mode

public mode は 3 つにする。

- `off`
- `dry_run`
- `apply`

ユーザー向け UI 表示:

- `Off`
- `Report only`
- `On`

内部値:

- `Off` -> `off`
- `Report only` -> `dry_run`
- `On` -> `apply`

`auto` は public config / `/config` / README / stable docs に出さない。

既存互換のために `auto` を読み取る場合は、effective mode を `dry_run` とする。

status 表示例:

```text
provider_history_reduction=auto
effective_mode=dry_run
auto_policy=not_enabled
```

### mode semantics

#### `off`

provider history reduction を無効化する。

- provider-facing projection を軽量化しない
- この機能を理由に Responses `previous_response_id` chain を disable しない

#### `dry_run` / `Report only`

provider-facing payload は変更せず、削減見込みだけを report / status に出す。

- replacement / compact candidate を検出する
- apply eligibility を満たす候補だけを estimated savings に計上する
- threshold 未満 / parse failure / unknown / unsafe / latest suffix / no later assistant は savings に過大計上しない
- Responses chain は disable しない

#### `apply` / `On`

provider-facing projection 上だけで、安全条件を満たした replacement / compact を実際に適用する。

`apply` は無条件削除ではない。
各 replacement / compact は以下の gate を通った場合だけ適用される。

- old result / old tool call argument
- latest tool suffix ではない
- no later assistant の直近 result ではない
- tool / command family を高信頼に分類できる
- required parse が成功する
- unknown / unsafe / parse failure は skip
- placeholder / compact text が元 payload より小さい
- saved tokens が threshold 以上
- evidence-backed replacement では active context / rehydrate transport 条件を満たす
- raw history / session / audit / persisted JSONL は変更しない

apply mode で実 replacement / compact が 1 件でも起きた場合、Responses `previous_response_id` chain を disable する。

実 replacement / compact が 0 件の場合は Responses chain を保持する。

### rehydrate_context

`rehydrate_context` は public config に出す。

理由:

- read_file / search_code / gather_context の old evidence replacement と対になる重要設定である
- provider-facing projection で古い evidence を軽量化した場合でも、必要な evidence を request-local active context として戻せる
- history には保存しない契約を維持する

UI 表示案:

- `Rehydrate context`
- `Restore omitted evidence when needed`
- `Request-local only`

public docs では次のように説明する。

- `rehydrate_context=true` は、placeholder 化された古い evidence を必要に応じて request-local active context として戻す
- raw history へ再保存しない
- provider / transport によって使える範囲が違う場合がある
- active context transport が使えない provider では、evidence-backed replacement は安全側で skip される

推奨 default:

```yaml
rehydrate_context: true
```

ただし `mode=off` の場合は実質無効。

現行 experimental config では default false なので、stable 公開時に default を true にする場合は migration / compatibility / status 表示で明確に扱う。

### backwards compatibility

既存の experimental config は、互換性のため当面読み取る。

例:

```yaml
experimental:
  provider_history_reduction:
    mode: apply
    rehydrate_context: true
```

stable config と experimental config の両方が指定された場合は、stable config を優先する。

env override は維持する。

例:

```sh
XELYON_PROVIDER_HISTORY_REDUCTION=apply
XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT=1
```

env override は config より優先する既存方針に合わせる。

`auto` は stable metadata には出さないが、互換入力として受けた場合は configured `auto` / effective `dry_run` を status で説明できるようにする。

### `/config` TUI

`/config` には provider history reduction を公開する。

表示案:

```text
Provider History Reduction
  Mode: Off / Report only / On
  Rehydrate context: On / Off
```

説明文:

```text
Reduce provider-facing history by replacing or compacting old tool results and command outputs.
Raw history, session messages, audit records, and persisted JSONL are preserved.
```

mode descriptions:

- `Off`: Do not reduce provider-facing history.
- `Report only`: Estimate savings without changing provider-facing payloads.
- `On`: Apply safe provider-facing replacements/compaction.

rehydrate description:

```text
Restore omitted read/search/gather evidence as request-local active context when needed.
```

`auto` は表示しない。

### `docs/config.md`

`docs/config.md` には詳細を書く。

含める内容:

- 機能概要
- raw history を消さないこと
- provider-facing projection だけを軽量化すること
- mode: `off` / `dry_run` / `apply`
- `rehydrate_context`
- env overrides
- safe gates
- Responses chain disable の説明
- `/status` で確認できる情報
- コスト削減と品質維持の tradeoff
- provider differences
- known limitations

docs/config.md の説明方針:

```text
Provider history reduction does not delete your local history.
It creates a lighter provider-facing projection for requests.
```

日本語 docs がある場合は、同じ意味で説明する。

### README

README には差別化要素として短く載せる。

載せる場所の候補:

- Features
- Cost control
- Provider support / BYOK
- Review command / long-running work section

README 例:

```md
### Provider-facing history reduction

XELYON can reduce provider-facing history without deleting local raw history.
Old read/search/gather results, safe command outputs, edit payloads, git output, and review intermediates can be replaced or compacted before sending requests to the provider.
This helps reduce input tokens during long-running coding sessions and review workflows while preserving local audit/session history.
```

README では細かい安全条件を全部書かず、`docs/config.md` へ誘導する。

### generated config metadata

generated config metadata に stable option として追加する。

必要項目:

- config key
- type
- allowed values
- default
- description
- env override
- stability: stable または advanced
- docs link

候補:

```yaml
provider_history_reduction.mode:
  type: enum
  values: [off, dry_run, apply]
  default: dry_run
  env: XELYON_PROVIDER_HISTORY_REDUCTION
```

```yaml
provider_history_reduction.rehydrate_context:
  type: bool
  default: true
  env: XELYON_PROVIDER_HISTORY_REHYDRATE_CONTEXT
```

実際の metadata format は既存 generator に合わせる。

現行の canonical source of truth は `scripts/internal/configgen/sections.go` で、`scripts/config_sections.go` は互換 shim である。
stable option 追加時は generator metadata を更新し、`internal/config/registry_generated.go` / `config.yaml.example` / `docs/config.md` は生成物として更新する。

### default

公開時点の default は `dry_run` / `Report only` にする。

理由:

- provider-facing payload は変更しない
- `/status` で削減見込みを見せられる
- 差別化要素として可視化できる
- `apply` は opt-in にできる

default ON は section 9 で判断する。
ただし、`/config` からユーザーが明示的に `On` を選べるようにする。

推奨公開ステップ:

1. stable config / `/config` / docs / README に出す
2. default は `dry_run` / `Report only`
3. dogfood と user opt-in で観測する
4. section 9 の条件を満たしたら default ON を検討する

### `/status`

`/status` には以下を表示する。

- configured mode
- effective mode
- `rehydrate_context` enabled/disabled
- replacement status
- content replacement count
- command output replacement count
- edit arg replacement count
- git output compact count
- failure compact count
- review history compact count
- total provider-facing savings
- Responses chain disabled
- active context / rehydrated evidence count
- skip / kept reason summary if available

例:

```text
provider_history_reduction=apply
effective_mode=apply
rehydrate_context=true
content_replacements=4
edit_arg_replacements=3
command_output_replacements=2
git_output_replacements=1
failure_command_replacements=1
review_history_replacements=0
provider_facing_saved_tokens≈12840
responses_chain_disabled=true
```

### warnings / limitations

docs には以下を明記する。

- provider-facing projection が軽くなる一方で、Responses chain が disable される場合がある
- chain disable により provider-side cached context の扱いが変わる可能性がある
- reduction が常にコスト削減になるとは限らない
- 小さい payload や threshold 未満の payload は置換しない
- parse 不能 / unknown / unsafe output は保持する
- raw local history は保持される
- provider ごとに active context transport の対応差がある

### tests

最低限、以下をテストする。

#### config parsing

- stable config の `mode=off` / `dry_run` / `apply` が読める
- `rehydrate_context` が読める
- env override が config より優先される
- experimental config からも後方互換で読める
- stable config と experimental config が両方ある場合は stable config が優先される
- invalid mode は validation error になる
- `auto` は stable metadata には出ない
- `auto` を互換入力として受けた場合は effective `dry_run` になる

#### `/config`

- `/config` に Provider History Reduction が表示される
- mode は Off / Report only / On として表示される
- auto は表示されない
- `rehydrate_context` を toggle できる
- 保存後の config が正しい stable key に書き込まれる

#### generated metadata

- mode が enum `[off, dry_run, apply]` として出る
- `rehydrate_context` が bool として出る
- env override が metadata に出る
- auto は stable option として出ない

#### docs / README

- `docs/config.md` に provider history reduction が記載される
- README に feature summary が記載される
- raw history は保持される説明がある
- provider-facing projection の説明がある
- Responses chain disable の注意がある

#### status

- configured/effective mode が表示される
- `rehydrate_context` が表示される
- replacement family breakdown が表示される
- total saved tokens が表示される
- Responses chain disabled が表示される

### 実装候補ファイル

stable config schema / defaults / validation / compatibility owner:

- `internal/config/config_types.go`
- `internal/config/defaults.go`
- `internal/config/defaults_apply.go`
- `internal/config/project_experimental.go`
- `internal/config/project_experimental_test.go`
- `internal/config/validator.go`
- `internal/config/config_validation_test.go`
- `internal/config/loader_compatibility.go`
- `internal/config/loader_compatibility_test.go`

generated config metadata / docs / example owner:

- `scripts/internal/configgen/sections.go`
- `scripts/internal/configgen/registry.go`
- `scripts/internal/configgen/docs_update.go`
- `scripts/internal/configgen/example.go`
- `scripts/gen-config-registry.go`
- `scripts/gen-config-docs.go`
- `scripts/gen-config-example.go`
- `internal/config/registry_generated.go`
- `internal/config/registry.go`
- `internal/config/registry_metadata_test.go`
- `docs/config.md`
- `config.yaml.example`
- `README.md`

`/config` TUI / command owner:

- `internal/agent/agent_commands_config.go`
- `internal/agent/agent_commands_config_test.go`
- `internal/agent/agent_commands_config_integration_test.go`
- `internal/tui/model_config.go`
- `internal/tui/model_config_render.go`
- `internal/tui/config_screen_state.go`
- `internal/tui/config_screen_render_lists.go`
- `internal/tui/config_screen_render_detail.go`
- `internal/tui/config_screen_edit_start.go`
- `internal/tui/config_screen_field_mutation.go`
- `internal/tui/model_config_screen_action_test.go`
- `internal/tui/model_config_scalar_editor_test.go`

runtime / status owner:

- `internal/agent/provider_history_reduction.go`
- `internal/agent/provider_history_reduction_runtime.go`
- `internal/agent/provider_history_reduction_runtime_test.go`
- `internal/agent/provider_history_reduction_startup_test.go`
- `internal/agent/project_config_sync.go`
- `internal/agent/project_config_sync_test.go`
- `internal/agent/provider_history_projection.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/agent/provider_history_reduction_status_test.go`
- `internal/agent/provider_history_rehydrate_context.go`
- `internal/agent/provider_history_rehydrate_context_test.go`

## 9. default ON 判断基準

### 結論

provider history reduction は public config / `/config` / docs / README に公開する。
ただし、公開直後から `apply` を default ON にはしない。

公開直後の default は `dry_run` / `Report only` を第一候補にする。

理由:

- provider-facing payload は変更しないため安全
- `/status` で削減見込みを確認できる
- ユーザーに差別化要素が見える
- false replacement / false compact のリスクを避けられる
- dogfood と opt-in `apply` の観測結果を集められる
- `apply` default ON の判断を後から安全にできる

`apply` default ON は、下記の判断基準を満たしてから検討する。

### 現在確認できた source

現行ソースでは、public release default はまだ実装されていない。

確認済み:

- `internal/config/project_experimental.go` の experimental mode 解決は env > project experimental config > default off
- `experimental.provider_history_reduction.rehydrate_context` の現行 default は false
- `internal/agent/provider_history_reduction_runtime.go` では configured `auto` を effective `dry_run` として扱う
- `internal/agent/provider_history_reduction_status.go` は mode / replacement status / family savings / Responses chain disabled を表示する owner

public default を `dry_run` / `rehydrate_context=true` にする場合は、stable config default と runtime sync の shared contract change として実装する。

### default policy stages

#### Stage 0: internal / experimental

対象:

- 開発中
- dogfood 前
- internal branch

default:

```yaml
provider_history_reduction:
  mode: off
```

または既存挙動維持。

使い方:

- project-local config / env override で `dry_run` または `apply` を明示する
- `/status` で report を確認する

#### Stage 1: public config, report-only default

対象:

- `/config` / docs / README に公開した直後
- user opt-in を受け付ける段階

default:

```yaml
provider_history_reduction:
  mode: dry_run
  rehydrate_context: true
```

UI 表示:

```text
Provider History Reduction: Report only
```

性質:

- provider-facing payload は変更しない
- Responses chain は disable しない
- `/status` で削減見込みを表示する
- ユーザーは `/config` から `On` / `apply` に変更できる
- README / docs では、`On` にすると provider-facing projection が軽量化されることを説明する

#### Stage 2: opt-in apply

対象:

- dogfood と初期ユーザーでの明示 opt-in
- `mode=apply` を選んだユーザー

default:

```yaml
provider_history_reduction:
  mode: dry_run
```

ユーザー設定:

```yaml
provider_history_reduction:
  mode: apply
  rehydrate_context: true
```

性質:

- safety gate を満たす replacement / compact だけ provider-facing projection に適用する
- raw history / session / audit / persisted JSONL は保持する
- 実 replacement / compact が起きた場合だけ Responses chain を disable する
- `/status` で family breakdown / saved tokens / chain disabled reason を表示する

#### Stage 3: apply default ON candidate

対象:

- dogfood と opt-in apply の観測が安定した後

default 候補:

```yaml
provider_history_reduction:
  mode: apply
  rehydrate_context: true
```

ただし、以下の判断基準を満たすまで Stage 3 に進めない。

### apply default ON の必須判断基準

#### 1. false replacement / false compact が出ない

dogfood / opt-in apply で、以下が発生していないこと。

- latest tool suffix が誤って置換される
- no later assistant の直近 result が誤って置換される
- parse 不能 output が誤って compact される
- unknown command output が誤って safe placeholder / first-last compact される
- sensitive / env / auth / database 系 output の raw lines が provider-facing compact に残る
- git diff が full omission される
- failure output が placeholder only に潰される
- final review findings / final report が compact される

#### 2. raw preservation 契約が破られていない

以下が常に保持されること。

- raw `Agent.History`
- `Session.Messages`
- audit records
- change records
- persisted JSONL
- review raw evidence
- final review report
- final findings

#### 3. report / status で説明できる

`/status` で最低限以下を確認できること。

- configured mode
- effective mode
- `rehydrate_context`
- content replacement count
- edit arg replacement count
- command output replacement count
- git output compact count
- failure compact count
- review history compact count
- family breakdown
- total provider-facing saved tokens
- Responses chain disabled
- chain disabled reason
- kept / skip reason summary

default ON 前に、ユーザーが「なぜ削られたか / なぜ削られなかったか」を確認できること。

#### 4. dry_run と apply の report が整合する

同じ history に対して、dry_run の eligible savings と apply の actual savings が大きく乖離しないこと。

特に:

- threshold 未満を savings に過大計上しない
- parse failure を savings に過大計上しない
- unknown / unsafe / latest suffix / no later assistant を savings に過大計上しない
- dry_run candidate count と apply replacement count の差分理由を report できる

#### 5. provider / transport 別に破綻しない

以下の provider family で挙動が破綻しないこと。

- OpenAI Responses
- OpenAI Chat Completions
- DeepSeek
- Gemini
- Claude
- Bedrock
- OpenRouter
- Kimi
- Groq
- Ollama

特に:

- active context transport が使える provider では `rehydrate_context` が機能する
- active context transport が使えない provider では evidence-backed replacement が安全側で skip される
- structured/list_dir/command/git/failure/review compact は active context transport availability に依存しない
- provider-specific message shape が壊れない
- Anthropic tool_use / OpenAI-style ToolCalls の同期契約が壊れない

#### 6. Responses chain disable のコストが許容できる

apply mode では replacement / compact が実際に起きた場合、Responses `previous_response_id` chain を disable する。

default ON 前に、以下を確認する。

- chain disable が起きる条件が正しい
- replacement / compact が 0 件なら chain を保持する
- chain disable による provider-side cached context 低下より、payload reduction の効果が上回るケースが多い
- `/status` に chain disabled reason が出る
- small savings のために chain disable しない threshold が機能している

#### 7. review command の品質が落ちない

review command で以下が維持されること。

- finding precision が落ちない
- final findings が欠落しない
- evidence refs / file:line refs が保持される
- unresolved risks が消えない
- final report / latest report が compact されない
- obsolete intermediate だけが compact される
- review state summary が十分に後続判断へ効く

#### 8. fallback が明確

default apply で問題が出た場合に、ユーザーがすぐ戻せること。

- `/config` で `Report only` / `Off` に戻せる
- env override で `off` / `dry_run` にできる
- docs に fallback 手順がある
- status で問題切り分けができる

### default ON にしない条件

以下が 1 つでも残る場合、default は `dry_run` のままにする。

- false replacement / false compact が 1 件でも確認された
- raw preservation 契約に不安がある
- provider-specific message shape の不具合がある
- Anthropic/OpenAI tool call argument sync に不安がある
- `rehydrate_context` が provider 別に不安定
- `/status` で理由説明が不十分
- review command の品質低下が疑われる
- Responses chain disable のコストが savings を上回る
- unknown / sensitive / database / failure compact の safety が十分でない

### default values

#### public release default

第一候補:

```yaml
provider_history_reduction:
  mode: dry_run
  rehydrate_context: true
```

理由:

- 機能を可視化できる
- provider-facing payload は変えない
- ユーザーが `/status` で削減見込みを見られる
- `On` への opt-in ができる

#### conservative fallback default

もし public default `dry_run` でも noise が多い場合:

```yaml
provider_history_reduction:
  mode: off
  rehydrate_context: true
```

この場合も `/config` で `Report only` / `On` を選べるようにする。

#### future apply default candidate

判断基準を満たした後に限り:

```yaml
provider_history_reduction:
  mode: apply
  rehydrate_context: true
```

### README / docs の説明

README では、default が `Report only` の場合でも差別化要素として説明する。

README 例:

```md
Provider-facing history reduction can estimate or apply safe history compaction without deleting local raw history.
By default, XELYON reports potential savings. You can turn it on from `/config` when you want provider-facing payload reduction during long-running coding or review sessions.
```

`docs/config.md` では以下を明記する。

- default は `Report only`
- `On` は opt-in
- raw local history は消さない
- provider-facing projection だけを軽量化する
- `/status` で savings と chain disable を確認できる
- 問題があれば `Report only` / `Off` に戻せる

### tests

最低限、以下をテストする。

#### default config

- public default が `dry_run` になる
- `rehydrate_context` default が true になる
- conservative fallback が必要な場合に `off` にできる
- explicit config `apply` は default を上書きする
- env override は default / config を上書きする

#### status

- default `dry_run` で estimated savings が表示される
- default `dry_run` では provider-facing payload が変わらない
- default `dry_run` では Responses chain disabled にならない
- explicit `apply` では safety gate を満たす場合だけ replacement / compact される

#### fallback

- `/config` で `Off` に戻せる
- `/config` で `Report only` に戻せる
- env override で `off` / `dry_run` に戻せる

#### apply default candidate safety

- false replacement guard tests が通る
- raw preservation tests が通る
- provider-specific projection tests が通る
- review command compact tests が通る

### 実装候補ファイル

config defaults / stable surface owner:

- `internal/config/config_types.go`
- `internal/config/defaults.go`
- `internal/config/defaults_apply.go`
- `internal/config/project_experimental.go`
- `internal/config/project_experimental_test.go`
- `internal/config/config_validation_test.go`
- `internal/config/default_config_test.go`
- `internal/config/loader_compatibility.go`
- `internal/config/loader_compatibility_test.go`

generated config metadata / docs owner:

- `scripts/internal/configgen/sections.go`
- `scripts/internal/configgen/registry.go`
- `scripts/internal/configgen/docs_update.go`
- `scripts/internal/configgen/example.go`
- `scripts/gen-config-registry.go`
- `scripts/gen-config-docs.go`
- `scripts/gen-config-example.go`
- `internal/config/registry_generated.go`
- `internal/config/registry_metadata_test.go`
- `docs/config.md`
- `config.yaml.example`
- `README.md`

`/config` TUI owner:

- `internal/agent/agent_commands_config.go`
- `internal/agent/agent_commands_config_test.go`
- `internal/agent/agent_commands_config_integration_test.go`
- `internal/tui/model_config.go`
- `internal/tui/model_config_screen_action_test.go`
- `internal/tui/model_config_scalar_editor_test.go`
- `internal/tui/config_screen_render_lists.go`
- `internal/tui/config_screen_render_detail.go`
- `internal/tui/config_screen_field_mutation.go`

runtime / status / provider projection owner:

- `internal/agent/provider_history_reduction.go`
- `internal/agent/provider_history_reduction_runtime.go`
- `internal/agent/provider_history_reduction_runtime_test.go`
- `internal/agent/provider_history_reduction_startup_test.go`
- `internal/agent/project_config_sync.go`
- `internal/agent/project_config_sync_test.go`
- `internal/agent/provider_history_projection.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/agent/provider_history_reduction_status_test.go`
- `internal/agent/provider_history_request_test.go`
- `internal/agent/provider_history_request_apply_test.go`
- `internal/agent/provider_history_reduction_phase5d_openai_test.go`
- `internal/agent/provider_history_reduction_phase5d_provider_state_test.go`

review command tests owner:

- `internal/review`
- `internal/review/reviewer.go`
- `internal/agent/review_runner.go`
- `internal/agent/review_model_adapter.go`
- `internal/agent/provider_history_reduction_status.go`
- `internal/tui/review_screen_render.go`

## 10. Goal handoff policy

### 目的

この master plan は、Codex Goal でまとめて実装するための仕様書である。

実装では、ここまでの安全契約と責務境界を守りつつ、最新ソースコードに合わせて自律的に設計・リファクタリング・テスト追加を行う。

細かい package 名、helper 名、report field の持ち方、status 表示の組み立て方は、既存コードの責務境界と命名規則に合わせて判断してよい。

section 1〜9 の設計を Codex Goal に渡す前に、全体の整合性も確認する。

実装差分が大きくなることは許容する。
ただし、検証単位・責務境界・報告単位は分ける。
大きな差分になっても、以下が追える状態にする。

- providerhistory foundation
- `list_dir` replacement
- command output classifier / compact
- git compact
- failure compact / redaction
- config / docs / status
- review command optimization

そのため、Goal 投入前に以下を master plan 上で固定する。

- 実装順
- 共通禁止事項
- cross-section invariant
- 検証・報告単位
- required verification matrix

### 実装中に仕様と最新ソース構造が衝突した場合

すぐに停止せず、以下の方針で臨機応変に問題解決する。

- 安全契約を弱めない
- raw history / session / audit / persisted JSONL は保持する
- provider-facing projection 上だけを軽量化する
- 既存責務境界を崩さない
- 必要なリファクタリングは行う
- 高信頼に分類できない output は安全側で skip / kept reason にする
- 実装できる範囲を進め、未対応・縮退・判断理由は最後に報告する

実装完了後は Codex Review で厳しく検証する前提とする。
そのため、実装段階では過度に停止せず、テストと自己確認を行いながら完走を優先する。

### pre/post implementation refactor policy

この master plan は大規模な provider history reduction 追加実装になるため、実装前後にリファクタリングフェーズを明示的に入れる。

目的は差分を小さくすることではない。
目的は、責務境界を保ち、テストしやすくし、Codex Review で指摘されやすい境界漏れ・重複・影響漏れを事前に潰すことである。

#### Phase 0: pre-implementation refactor / test foundation

本格実装に入る前に、必要な範囲で事前リファクタリングを行う。

このフェーズでは、機能追加そのものよりも、追加実装を安全に受け止めるための境界整理とテスト土台を優先する。

対象例:

- providerhistory の responsibility boundary を確認する
- evidence-backed / structured / command / git / failure / review compact の owner を整理する
- `toolresults` / `commandoutputs` / `gitoutputs` / `failureoutputs` などの package 境界を必要に応じて作る
- report / status / threshold / kept reason の共通 helper を整理する
- raw history preservation を検証しやすい test helper を整備する
- provider-facing projection と raw history の比較 helper を整備する
- command classifier / compact builder の test fixture を作りやすくする
- import cycle が起きそうな境界を先に解消する
- 既存テストの重複や読みづらさが実装を妨げる場合は、テストコードもリファクタリングする

このフェーズでは、既存挙動を変えない。
挙動変更が必要になった場合は、仕様に沿った機能実装フェーズで扱う。

#### Phase 1-N: implementation phases

Phase 0 の土台の上で、section 1〜9 の仕様に沿って実装する。

実装中に責務境界が悪化した場合は、その場で必要なリファクタリングを行ってよい。
小さく逃げるより、後続実装とレビューに耐える構造を優先する。

差分が大きくなることは許容する。
ただし、report/status/test の観測単位は分かるように保つ。

#### Phase Final-A: impact audit / review-hole sweep

機能実装後、Codex Review に出す前に、影響漏れとレビュー指摘されやすい穴を自律的に探索して潰す。

確認観点:

- raw `Agent.History` / `Session.Messages` / audit / persisted JSONL preservation
- provider-facing projection only の契約
- latest tool suffix / no later assistant protection
- Responses chain disable 条件
- dry_run estimated savings と apply actual savings の整合
- threshold 未満 / parse failure / unknown / unsafe の skip
- Anthropic tool_use と OpenAI-style ToolCalls の同期
- command classifier の false positive
- sensitive / env / auth / database raw line leakage
- git diff compact の情報欠落
- failure compact の診断情報欠落
- review final findings / final report preservation
- `/status` family breakdown
- `/config` / generated metadata / `docs/config.md` / README の整合
- provider-specific message shape
- generated files の更新漏れ

このフェーズでは、見つかった問題を可能な限り修正する。
単なる TODO 報告で終わらせず、修正できるものは修正する。

#### Phase Final-B: mandatory comprehensive refactor including tests

機能実装が完了した後、production code と test code の両方を対象に、必ず包括的なリファクタリングを行う。

このフェーズは optional ではない。
「必要なら行う」ではなく、実装完了後に必ず実施する品質仕上げフェーズである。

目的は、実装を動かすことではなく、Codex Review で指摘されそうな責務境界・重複・命名・テスト構造・影響漏れを、レビュー前に徹底的に潰すことである。

このフェーズでは、影響範囲が広がることを許容する。
むしろ、責務境界を正しくするために必要な package split / helper extraction / test helper consolidation / report/status aggregation redesign / naming cleanup / generated docs alignment がある場合は、積極的に実施する。

対象:

- production code の責務分離
- providerhistory / commandoutputs / gitoutputs / failureoutputs / toolresults などの package boundary
- report / status / threshold / kept reason / classifier reason の整理
- provider-specific projection adapter の責務整理
- Anthropic tool_use / OpenAI-style ToolCalls sync 周辺の重複整理
- command classifier / compact builder / redaction helper の責務分離
- git diff compact / failure compact の parser と formatter の分離
- review command optimization の owner 境界
- config / generated metadata / docs / README の整合
- test helper / fixture / assertion helper の再編成
- duplicated tests の統合
- brittle tests の修正
- test names / test setup の読みやすさ改善
- raw preservation / provider-facing projection only / Responses chain disable の shared assertions 化

このフェーズでは、テストコードも品質対象に含める。
テストコードが冗長・脆い・責務不明・fixture 重複・assertion 重複になっている場合は、production code と同じ基準でリファクタリングする。

歓迎する変更:

- レビュー指摘を減らすための責務境界変更
- 実装後に見えた package split
- helper / type / interface の再設計
- report/status の集計責務の移動
- test helper の大規模整理
- 既存テストの読み替え・再構成
- generated config/docs/README との整合を取るための追加修正
- 実装中に作った暫定 helper の撤去
- 重複 classifier / parser / formatter の統合

禁止する逃げ方:

- 「動いているのでリファクタリング不要」と判断してスキップする
- TODO コメントだけ残して終える
- テストコードの重複を放置する
- report/status の命名揺れを放置する
- providerhistory と agent/review/config の責務境界の違和感を放置する
- generated docs/config metadata の不整合を放置する
- Codex Review に任せる前提で明らかな設計違和感を放置する

このフェーズの完了条件:

- production code と test code の両方を見直した
- 実装中に増えた重複を整理した
- package / helper / type / report field / status label の命名が揃っている
- raw preservation / provider-facing projection only / chain disable の assertion が十分に共有化されている
- focused tests が読みやすく、失敗時に原因を追いやすい
- generated config metadata / `docs/config.md` / README / `config.yaml.example` の整合が取れている
- Codex Review に出す前に、自分で見つけられる境界漏れ・影響漏れ・テスト品質問題を潰した

このフェーズで見つけた問題は、可能な限りその場で修正する。
単なる報告や TODO 化で済ませず、修正できるものは修正してから最終報告に進む。

### 完了報告

完了報告では以下をまとめる。

- 実装した範囲
- 仕様から調整した点
- 安全側に skip / degrade した点
- 実行したテスト
- 残った follow-up

### 推奨実装順

1. content replacement の責務境界と threshold / report / status の土台を整える
2. structured tool result replacement として `list_dir` を追加する
3. successful command output classifier / compact を追加する
4. git-specific compact owner を追加する
5. failure command output compact / redaction を追加する
6. `delete_file` は candidate-only kept policy として固定する
7. mode policy / default policy / public config surface を実装する
8. review command cost optimization を review owner 側で追加する
9. default ON 判断に必要な status / docs / tests を揃える

review command optimization は通常 history projection とは owner が違うため、providerhistory 側の基盤が安定してから検証・報告単位を分ける候補にする。

### 共通禁止事項

- raw `Agent.History` を変更しない
- `Session.Messages` を変更しない
- audit / change records / persisted JSONL を変更しない
- provider-facing projection 以外を軽量化しない
- `list_dir` を evidence-backed replacement path に混ぜない
- command / git / failure / review compact を rehydrate_context evidence に混ぜない
- failed / denied / cancelled / unsafe の扱いが未定義な output を placeholder 化しない
- unknown success を full omission しない
- git diff を safe placeholder full omission しない
- failure output を placeholder only にしない
- sensitive / env / auth / database 系 output の raw first/last lines を provider-facing に残さない
- `ReductionCandidate` に raw assistant tool arguments を持たせない
- `auto` を real policy として新規実装しない
- `apply` default ON を section 9 の判断基準前に進めない

### 検証・報告単位

commit を作るかはユーザー指示に従う。
ただし、実装報告とテストは以下の単位で整理する。

1. providerhistory content replacement architecture / threshold / report/status foundation
2. `list_dir` structured replacement
3. successful command output classifier / compact
4. git-specific compact
5. failure command compact / redaction
6. `delete_file` candidate-only kept reason
7. mode/config/generated/docs/README public surface
8. review command cost optimization
9. default policy / status explanation / final docs alignment

各単位で focused tests を通し、最後に broad test / generated consistency / docs consistency を確認する。
差分が大きくても、どの検証単位で何を実装し、どのテストで固定したかを報告できる状態にする。

### Goal 投入前チェック

Goal に渡す前に、以下を確認する。

- section 1〜9 の terminology が一致している
- content replacement threshold と command replacement threshold の source of truth が分離されている
- report/status の family breakdown 名が重複していない
- dry_run estimated savings と apply actual savings の意味が統一されている
- Responses chain disable の条件が全 replacement / compact family で一致している
- active context / rehydrate_context の対象が evidence-backed replacement だけに限定されている
- public config default と docs/default policy が一致している
- generated metadata / `/config` / `docs/config.md` / README の公開内容が一致している
- provider-specific projection tests と review command tests が実装順に含まれている

## 実装時の想定コマンド

```sh
gofmt -w <changed files>
make gen-all
go test ./internal/providerhistory -run 'ListDir|Projection|ProviderHistory'
go test ./internal/agent -run 'ProviderHistory|ActiveContext|TokenBudget'
go test ./internal/config -run 'ProviderHistory|Experimental|Registry|Default'
go test ./scripts/internal/configgen
go test ./internal/review -run 'Review|Evidence|Report|Scope|History|Compact'
go test ./internal/review/modelinput -run 'Prompt|Context|Saturation|Compact'
go test ./internal/review/evidence -run 'Evidence|External|Diff|Context|Compact'
go test ./...
```

## 実装者に委ねる命名・配置判断

以下は設計上の未確定点ではなく、実装時に既存コードの命名規則・責務境界・リファクタ後の形に合わせて決めてよい。

- `internal/providerhistory/toolresults` の最終 API 名
- structured tool result replacement builder の関数名 / 型名
- status breakdown を `ProjectionReport` field として保持するか、status formatting 時に `Candidates` から集計するか
- content replacement threshold helper / const の正確な配置名

ただし、以下の設計契約は変更しない。

- evidence-backed tool result replacement と structured tool result replacement は分ける
- `list_dir` を evidence pointer 必須 flow に混ぜない
- `list_dir` を rehydrate_context / `AppliedEvidencePointers` / `BuildRehydratePlan` の対象にしない
- `ReductionCandidate` に raw assistant tool arguments を持たせない
- active context transport gating は per-candidate にする
- `list_dir` は active context transport availability に依存させない
- `list_dir` path は専用 helper で正規化し、result header の absolute path は placeholder に使わない
- content replacement threshold は `read_file` / `search_code` / `gather_context` / `list_dir` に共通適用する
- status で content replacement tool breakdown を確認できるようにする
