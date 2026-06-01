# Package Boundaries

## Phase 1: agent / TUI boundary

Phase 1 後の対話実行と TUI の依存方向は次の owner に分ける。

- `internal/agent`: conversation / turn / tool orchestration の core。`internal/tui`、Bubble Tea、Lip Gloss を直接知らない。`internal/ui` は interactive surface / runtime output contract として許可する。
- `internal/app`: CLI mode wiring と TUI startup の owner。TUI 起動時の package 接続をここで扱う。
- `internal/tuiagent`: `internal/agent` と `tui.AgentInterface` の adapter owner。agent の状態やイベントを TUI が扱う interface へ変換する。
- `internal/tui`: UI lifecycle、message、rendering の owner。Bubble Tea / Lip Gloss への直接依存はここに閉じる。

`internal/agent/package_boundaries_test.go` は、`internal/agent` 配下から `internal/tui`、Bubble Tea、Lip Gloss への import を禁止する。subpackage は禁止対象に含めるが、`internal/tuiagent` のように import path が似ている別 package は禁止しない。

## Phase 2-A: provider history / token boundary

Phase 2-A 後の provider-facing history projection と token helper は次の owner に分ける。

- `internal/providerhistory`: provider-facing history projection / reduction / report / rehydrate plan helper の pure logic owner。`internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss を直接知らない。raw history の永続化や request context の副作用は持たず、`api.Message` と `taskstate` の入力から provider payload 用 projection を返す。
- `internal/token`: model token limit と token count estimation の owner。agent だけでなく provider adapter や repomap から直接参照する shared helper として扱い、`internal/agent` には置かない。
- `internal/agent`: raw history の clone、runtime option 解決、TaskLedger snapshot からの evidence pointer 抽出、request context / response id chain / active context append の owner。provider history の pure policy 判断は `internal/providerhistory` に委譲する。

`internal/providerhistory/package_boundaries_test.go` は、`internal/providerhistory` 配下から `internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss への import を禁止する。Agent の runtime state や UI 表示に触る必要が出た場合は、providerhistory へ wrapper を増やさず `internal/agent` 側の caller で policy input に変換する。

## Phase 2-B: task state boundary

Phase 2-B 後の runtime task state は次の owner に分ける。

- `internal/taskstate`: `RuntimeTaskState`、snapshot / reset、recorder、tool/test observation、evidence pointer、rehydrate plan / execution、edit readiness、current task state snapshot rendering の provider-neutral な state owner。`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss を直接知らない。
- `internal/agent`: TaskLedger store の runtime 初期化と lifecycle、turn loop / mutation tracker / final check / edit readiness から state を更新・消費・reset するタイミング、provider-facing active context へ渡すかどうかの policy、`api.ActiveContextBlock` への wrapping、`/ledger` command surface の owner。
- `internal/providerhistory`: provider history projection / reduction から `taskstate.EvidencePointer` と `taskstate.RehydratePlan` を扱う pure helper の owner。TaskLedger store の所有や request context の副作用は持たない。

`internal/taskstate/package_boundaries_test.go` は、`internal/taskstate` 配下から `internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import を禁止する。Provider payload、TUI 表示、turn orchestration に触る必要が出た場合は、`internal/taskstate` へ wrapper を増やさず caller 側で `taskstate` の provider-neutral な型へ変換する。

## Phase 2-C: turn support / final check boundary

Phase 2-C 後の normal turn support と final check policy は次の owner に分ける。

- `internal/turnsupport`: normal turn の retry / stalled 判定、error fingerprint 正規化、turn-local mutation tracking、FileChange snapshot / progress fingerprint の pure state・policy owner。`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss を直接知らない。
- `internal/finalcheck`: final check の結果 DTO、failure feedback、failure fingerprint、no-progress retry gate、target file content fingerprint の owner。shell command 実行、taskstate 記録、colored output、taskTestResult 更新は持たない。
- `internal/agent`: final check command の読み取りと実行、timeout/env/process group、git diff / untracked context、taskstate observation 記録、history append、normal turn の continue / break / done 制御の owner。normal turn support と final check の deterministic policy は `internal/turnsupport` と `internal/finalcheck` に委譲する。

`internal/turnsupport/package_boundaries_test.go` と `internal/finalcheck/package_boundaries_test.go` は、それぞれ `internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import を禁止する。Agent runtime state、provider payload、TUI 表示に触る必要が出た場合は、新 package 側へ wrapper を増やさず `internal/agent` 側で policy input に変換する。

`internal/token/package_boundaries_test.go` は、shared token helper が `internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss に依存しないことを固定する。

## Phase 3-A: review report / artifact boundary

Phase 3-A 後の `/review` report schema と artifact 保存境界は次の owner に分ける。

- `internal/review`: `/review` の外部入口、facade、runner / probe / evidence orchestration の owner。既存の `ReviewReport` / `ReviewRunArtifactWriter` などの public-ish 名は alias / wrapper として維持し、agent / TUI / cmd 側の import path を変えない。
- `internal/review/domain`: report / probe / request が共有する最小 enum owner。`TargetKind`、`ReviewProbeMode`、`ReviewProbeStatus` だけを持ち、runner state、provider payload、UI 表示を持たない。
- `internal/review/report`: review report schema DTO、evidence ref DTO、strict decode、report validation、Pass1 scope cross validation、computed summary、saturation check DTO / decode / validation の owner。`ReviewProbePlan` 全体には依存せず、facade が `PlanScope` へ変換する。
- `internal/review/artifact`: review run artifact writer、artifact directory / name / repo-local path validation、buffered writer の owner。runner の保存制御、warning 出力、redaction 適用タイミングは `internal/review` に残す。

`internal/review/report/package_boundaries_test.go` と `internal/review/domain/package_boundaries_test.go` は、`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import を禁止する。`internal/review/artifact/package_boundaries_test.go` は、`internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss への import を禁止する。evidence path / probe path policy は Phase 3-A では動かさず、repo root 内判定の owner を決める後続工程で扱う。

## Phase 3-B: review probe / path policy boundary

Phase 3-B 後の probe plan / runtime と repo-root path policy は次の owner に分ける。

- `internal/review`: `/review` の外部入口、runner orchestration、model prompt、evidence cross validation、progress / artifact / redaction、report / saturation flow の owner。既存の probe plan / runtime public-ish 名は alias / wrapper として維持し、agent / cmd 側の import path を変えない。
- `internal/review/probe`: probe plan DTO / strict decode / basic validation / request conversion、probe runtime request/result DTO、`ProbeRunner`、host_readonly / scratch_only / repo_sandbox executor、command allowlist、sandbox policy、generated file / worktree snapshot / Go toolchain helper、review Git args/env policy、probe result mutation outcome helper の owner。`internal/review/domain` と `internal/review/report` には依存してよいが、親 `internal/review` には依存しない。
- `internal/review/pathpolicy`: repo-root containment の lexical / symlink helper owner。evidence path schema や sandbox-specific error contract は持たず、caller がそれぞれの error message / `errors.Is` contract に変換する。
- `internal/review/report`: probe result から report schema への assembly は持たない。runner facade が `ReviewProbeResult` を `ReviewProbeSummary` に変換し、report package は report schema validation に集中する。

`internal/review/probe/package_boundaries_test.go` は、`internal/agent`、`internal/tui`、Bubble Tea、Lip Gloss への import を禁止する。`internal/review/pathpolicy/package_boundaries_test.go` は、`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import を禁止する。probe 実行順、mutation 後 skip、timeout / failure / blocked / mutation outcome、allowed / denied command、report / evidence JSON schema は Phase 3-B では変更しない。

## Phase 3-C: review analysis / external doc boundary

Phase 3-C 後の Pass1 evidence analysis と external_doc evidence helper は次の owner に分ける。

- `internal/review`: `/review` の外部入口、runner orchestration、evidence collection、web search collector、model prompt / repair、progress / artifact / redaction、report / saturation flow の owner。既存の `ReviewExternalDoc*`、`ValidateReviewProbePlanAgainstEvidence`、report / saturation validation entrypoint は alias / wrapper として維持し、agent / cmd 側の import path を変えない。
- `internal/review/analysis`: Pass1 probe plan と evidence input からの deterministic analysis owner。material path / inventory category / untracked / generic impact / truncation / no-probe related evidence coverage、review pressure signals、Pass1 plan から report `PlanScope` への変換、external_doc evidence ref と fetched snippet の照合を扱う。親 `internal/review` は import せず、親 facade が `ReviewEvidenceModelInput` / `ReviewEvidenceBundle` から analysis 用 DTO へ変換する。
- `internal/review/externaldoc`: external documentation source DTO、source credibility classification、focus term sanitation、snippet construction、bounded HTTPS fetcher、external-document search subject helper の owner。Web search provider 呼び出しや bundle 依存の query collection orchestration は持たず、`internal/review` の collector が caller として扱う。
- `internal/review/report`: report / saturation schema validation の owner。Pass1 probe plan 全体には依存せず、`internal/review/analysis` が作る `PlanScope` を受け取る。

`internal/review/analysis/package_boundaries_test.go` と `internal/review/externaldoc/package_boundaries_test.go` は、`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import を禁止する。report schema、evidence schema、probe behavior、web search query quality、external_doc official / third_party / unknown 判定基準は Phase 3-C では変更しない。

## Phase 3-D: review externaldoc / web search query boundary

Phase 3-D 後の external_doc と Web 検索 query 計画は次の owner に分ける。

- `internal/review`: `/review` の外部入口、collector / runner orchestration、Web 検索 provider 実行境界、external doc fetcher 呼び出し、max query / max result 制御、error / truncated / inconclusive 集約、artifact / prompt / progress の owner。`ReviewEvidenceBundle` から `internal/review/externaldoc.SearchQueryPlanningInput` への変換もここに残す。Phase 3-C の「web search collector は親 package」方針は、provider 実行と収集制御が親 package の責務である、という意味で維持する。
- `internal/review/externaldoc`: external_doc / external source / source credibility、Web search evidence DTO、`SearchQueryPlanningInput` / `SearchQueryCandidate`、`BuildSearchQueryCandidates`、`BuildFetchRequest` の owner。focus token selection、generic token concrete 判定、query dedupe、`official documentation` query 組み立て、candidate cap はここに閉じる。`ReviewEvidenceBundle`、provider 実行、artifact / prompt / progress には依存しない。
- `internal/review/analysis`: Pass1 evidence analysis、pressure signal、report / probe の external_doc ref cross validation の owner。Web search evidence と query DTO は `internal/review/externaldoc` の型を analysis 用 DTO として共有し、親 facade が runtime evidence から analysis input へコピーする。

Phase 3-D では report schema、evidence JSON、probe behavior、external_doc official / third_party / unknown 判定基準、Web 検索 query 生成結果は変更しない。`internal/review/externaldoc/package_boundaries_test.go` は、`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import 禁止を維持する。

## Phase 3-E: review evidence boundary

Phase 3-E 後の review evidence 収集と render は次の owner に分ける。

- `internal/review`: `/review` の外部入口、facade、runner orchestration、model prompt / repair、progress / artifact / redaction、report / saturation flow の owner。既存の `ReviewEvidence*`、`ReviewEvidenceBuilder`、`ReviewEvidenceCommandRunner`、`ReviewWebSearchEvidenceCollector*` などの public-ish 名は alias / wrapper として維持し、agent / cmd 側の import path を変えない。
- `internal/review/evidence`: current changes evidence bundle、git / file / rule / untracked / related context collection、generic impact expansion、evidence JSON / Markdown render、model input DTO、repo-root path display / redaction helper、analysis DTO への変換、review pressure signals、外部 Web 検索 evidence collector の owner。`internal/review/domain`、`internal/review/probe`、`internal/review/analysis`、`internal/review/externaldoc` には依存してよいが、親 `internal/review` には依存しない。
- `internal/review/probe`: probe 実行後の mutation outcome canonicalization、mutation 後の skipped probe result 生成、probe result から report 用 `ReviewProbeSummary` への変換の owner。
- `internal/review/report`: trusted probe summary の copy / canonicalization、blocked outcome に基づく report verification status normalization、report schema validation の owner。runner は redaction 適用、external_doc ref cross validation 呼び出し、error prefix 付与に集中する。
- `internal/review/externaldoc`: Phase 3-D の Web 検索 query 計画と external_doc DTO / fetch request owner を維持する。Phase 3-D で親 package に残していた Web 検索 provider 実行と収集制御は、Phase 3-E から `internal/review/evidence` の collector owner へ移る。

Phase 3-E では report schema、evidence markdown / JSON、probe behavior、external_doc behavior、runner 実行順は変更しない。`internal/review/evidence/package_boundaries_test.go` は、`internal/agent`、`internal/tui`、`internal/api`、Bubble Tea、Lip Gloss への import を禁止する。

## Phase 3-F: review model input assembly boundary

Phase 3-F 後の review model input assembly は次の owner に分ける。

- `internal/review`: `/review` の外部入口、facade、runner orchestration、evidence 収集、probe 実行順、model phase 選択、`ReviewModelRequest` 送信、strict decode / validation / finalize、external_doc ref cross validation、artifact / progress、path replacement 発見と final report/probe summary redaction の owner。既存の `ReviewRequest`、`ReviewRunner`、`ReviewModel` はここに残し、agent / cmd 側の import path を変えない。
- `internal/review/modelinput`: probe plan / report / saturation check / report revision の prompt text、JSON contract text、repair prompt、probe result prompt DTO、output budget limiter、prompt section / fence / JSON formatter の deterministic assembly owner。`internal/review/domain`、`internal/review/probe`、`internal/review/report` の型・定数を直接受け取り、親 `internal/review`、evidence collection、artifact、provider runtime、UI には依存しない。
- `internal/review/evidence`: evidence bundle と markdown render の owner を維持する。`modelinput` には rendered markdown string として渡し、evidence package の型を prompt assembly に漏らさない。
- `internal/review/report` / `internal/review/probe`: schema DTO、probe result DTO、trusted probe summary DTO の owner を維持する。`modelinput` はこれらを入力として prompt DTO へ写像するだけで、decode / validation / runtime 実行は持たない。

Phase 3-F では prompt 文面、section order、JSON schema / validation、evidence markdown、probe behavior、external_doc behavior、runner の model phase / retry 制御は変更しない。`internal/review/modelinput/package_boundaries_test.go` は、親 `internal/review`、`internal/review/evidence`、artifact、agent / TUI / provider runtime、Bubble Tea、Lip Gloss への import を禁止する。

## Phase 3-G: review model output boundary

Phase 3-G 後の review model output decode / finalization は次の owner に分ける。

- `internal/review`: `/review` の外部入口、facade、runner orchestration、evidence 収集、probe 実行順、model phase 選択、`ReviewModelRequest` 送信、repair / revision / saturation の順序制御、artifact / progress、path replacement 発見の owner。既存の `ReviewRequest`、`ReviewRunner`、`ReviewModel` はここに残し、agent / cmd 側の import path を変えない。
- `internal/review/modeloutput`: LLM raw output の strict decode、trusted probe summary の redacted copy 注入、report / saturation check の validation、external_doc ref cross validation、computed summary 注入の deterministic finalization owner。runner からは fetched external docs だけを受け取り、`ReviewEvidenceBundle`、artifact、modelinput、provider call には依存しない。
- `internal/review/modelinput`: Phase 3-F の prompt / repair prompt assembly owner を維持する。decode / validation / finalization は持たない。
- `internal/review/report` / `internal/review/analysis` / `internal/review/externaldoc` / `internal/review/probe`: schema DTO、PlanScope 変換、external_doc ref 照合、probe plan DTO の owner を維持する。`modeloutput` はこれらを直接使い、親 `internal/review` facade には依存しない。

Phase 3-G では prompt 文面、report / saturation schema、validation rule、trusted probe outcome normalization、saturation 判定、repair / retry 回数と順序、provider call は変更しない。`internal/review/modeloutput/package_boundaries_test.go` は、親 `internal/review`、`internal/review/evidence`、artifact、modelinput、agent / TUI / provider runtime、Bubble Tea、Lip Gloss への import を禁止する。
