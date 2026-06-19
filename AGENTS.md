# XELYON CLI

## Rules
- コミット前は make ci-check 必須（go fmt → build → lint → test）
- 新機能・バグ修正時はテストも追加
- エラーハンドリング必須（I/O 操作、HTTP には Timeout 設定）
- コミットは日本語で機能単位で小さく、具体的に記述
- 新機能追加時は README.md と docs/ を更新（バグ修正は不要）
- 設定追加時は make gen-all を実行
- 未使用コードは作らない（呼び出し元を確認してから実装）
- 公開関数・型には日本語コメント必須（godoc 形式）

## Verification policy

- 実装中は `make ci-check` を毎回走らせない。
- まず `make verify-fast`、touched package の focused `go test`、必要な `rg`、`git diff --check` を使う。
- shared contract / provider-facing / runtime state / config を触った場合は、caller package まで test を広げる。
- commit 前は必ず `make ci-check` を 1 回実行する。
- `make ci-check` は race + coverage 付きの最終 gate として扱う。

## Worktree / PR Workflow

- 重要変更は作業用 worktree + branch で実装し、ローカル review / `make ci-check` 後に branch を push して PR を作る。
- main への取り込みは GitHub PR merge を標準とし、ローカル main に先行取り込みしてからレビューする運用は例外とする。
- ローカル Codex review は read-only の current changes review として扱い、利用可能なら `strict-diff-review` を使う。
- commit / push / PR 作成は、ユーザーの明示指示がある場合だけ行う。
- PR merge 後はローカル main を pull してから、作業 worktree とローカル作業 branch を削除する。
- worktree 操作は利用可能なら `xelyon-worktree-ops` の `wt-start` / `wt-ci` / `wt-publish` / `wt-close` を使う。
- Squash merge 後に `wt-close` が Git 履歴だけで取り込み済み判定できない場合は、GitHub 上で merge 済みを確認したうえで `wt-close --allow-squash` を使う。

## Package / folder boundary policy

- Go の file 分割だけで責務境界が固定されたとは扱わない。同一 package 内では private symbol も共有されるため、contract owner を守れない場合は package boundary を検討する。
- 巨大 package、複数 owner を持つ package、`*_builder.go` / `*_types.go` / `*_resolver.go` / `*_runtime.go` などが同居する package に新規責務を追加する前に、`package-boundary-map` で owner / public surface / dependency direction を確認する。
- `package-boundary-map` の結果、package split / move が必要と判断した場合は、実装タスクに混ぜず、挙動変更なしの `package-boundary-refactor` として別工程に切る。
- package split では、export を増やすことを目的にしない。外部から壊れた状態を作れない API、constructor / builder / parser、focused tests を優先する。
- `utils` / `helpers` / `common` への domain policy 退避は禁止する。
- import cycle 回避だけを目的にした薄い interface / wrapper は、根本の依存方向が誤っていないか確認する。
- package boundary の問題が見えた場合、実装を続ける前に「この変更で触るべき owner はどの package か」「今の package に置くと何が共有されすぎるか」を短く報告する。

## Go contract / state / security / concurrency change checks

- domain struct、enum-like kind、constructor、parser、builder、validation、DTO/domain 境界、error contract を触る場合は `go-contract-design` を使う。
- registry、cache、ledger、session/runtime state、history、active context、provider chain state、global state、test shared state を触る場合は `go-state-lifecycle-change` を使う。
- external input が file / network / process / provider / MCP / env / path / cwd / repo root 境界に到達する場合は `security-boundary-change` を使う。
- goroutine、channel、context cancellation、timer/ticker、lock、streaming、background task、concurrent provider/tool execution を触る場合は `go-concurrency-lifecycle` を使う。

## Skill routing discipline

- Skill に該当する変更を、局所 patch / helper 追加 / fallback 追加 / interface 追加だけで進めない。
- 複数 Skill に該当する場合は、主目的の Skill を 1 つ選び、必要な補助 Skill を明示する。
- 作業中に別 Skill の stop condition に当たった場合は、同じ diff で押し切らず、別 task として報告する。
- 「一旦動かすため」の exported API、global state、fallback、sleep、wrapper、test hook を追加しない。
- どうしても最小 patch にする場合は、owner / source of truth / 残る risk を file / function / package 単位で説明する。

## 通常実装後 Final-A / Final-B gate

- Goal 機能や明示的な Goal 指示がなくても、非 trivial な実装、shared contract、provider-facing projection/prompt、history、tool result replay、web search replay、config/runtime/state、classifier/reason/status、fallback/helper、test fixture/fake を触った場合は、完了報告前に Final-A / Final-B 相当の実装後ゲートを通常運用として実施する。
- Final-A は correctness / impact recovery の工程として扱う。provider-facing data loss、classification precedence、片側 path 漏れ、caller/test/docs/generated/config drift、review prompt と apply mode の不一致を疑う場合は `post-implementation-impact-recovery` に切り替え、counterexample test と caller 経由検証で閉じる。
- Final-B は behavior-preserving な comprehensive refactor として扱う。production diff と test diff の両方、owner graph、direct callers、sibling helpers、fixtures/fakes/table tests、docs/generated/config chain まで広めに見て、外部挙動を変えずにできる構造整理は実施する。
- Final-B では scope を小さくすること自体を品質と扱わない。owner / source of truth / helper boundary を改善できる場合は、影響範囲が広がっても必要な caller と tests を追って整理する。
- Final-B 中に correctness risk を見つけた場合は、整理を続けず Final-A / `post-implementation-impact-recovery` に戻す。壊れた behavior をきれいに整理して完了扱いしない。
- test diff、fixture、fake、assertion helper、table test の整理が必要な場合は `test-boundary-refactor` を使う。テストも owner / source of truth / helper boundary の対象として扱う。
- XELYON の provider history、review command、command output reduction、MCP/tool result、provider-native built-in replay、web search projection、token/compression/runtime 周りの変更は、provider-facing data loss と prompt/review 品質劣化を最優先リスクとして扱う。

## XELYON-specific config change checks

config schema、YAML key、defaults、validation、migration、registry、`/config` UI、docs、generated files を触る場合は、利用可能なら `xelyon-config-contract-change` skill を使う。

## XELYON-specific shared change checks

`shared-contract-change` に該当する変更では、汎用 preflight / self-review に加えて以下も確認する。

- compression / provider / model / pricing / token / request path / Responses API 周りを触る場合は、利用可能なら `xelyon-provider-runtime-change` skill を使う。
- 既存 config に保存済みの旧 default がある場合は migration 要否を確認する。
