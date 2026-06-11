# XELYON openai_subscription Phase 0 Reality Check

この文書は `openai_subscription` provider 本実装の前に、XELYON が honest identity で ChatGPT/Codex subscription backend を使えるかだけを確認するための内部検証計画である。

公開 docs ではなく、Phase 0 probe の spike / Goal に渡す source of truth として使う。
full provider 実装はこの文書の対象外。

作成日: 2026-06-07

## Status

この文書は Phase 0 probe の evidence log として残す。
full provider の実装方針は以下を source of truth とする。

```text
docs/dev/openai-subscription-provider-master-plan.md
```

この文書から master plan v2 へ移植済みの結論:

- `originator=xelyon` browser OAuth / endpoint text smoke は OK。
- `prompt_cache_key` は enabled。
- `instructions` と `stream=true` は required。
- `store=true` は unsupported。
- `previous_response_id` は unsupported。
- `context_management` は `previous_response_id` 未対応により not applicable / disabled。
- `function_call` / `function_call_output` continuation は full payload fallback で enabled。

この文書に残すもの:

- retired probe で得た historical live result。
- 現在の doctor command shape。
- live result の詳細。
- request/response compatibility の分類根拠。
- redaction contract。
- Phase 0 実装の検証範囲。

実装者はこの文書を evidence として参照してよいが、runtime policy は master plan v2 を優先する。

## 0. 結論

この検証は full provider よりかなり軽い。
ただし `originator=xelyon` を証明するには browser OAuth の PKCE flow が必要で、単なる request smoke よりは重い。

### 0.1 Live result: 2026-06-07

当時の standalone probe で以下を確認した。
probe code は provider / auth CLI / doctor へ移植済みのため、現在の tree には残さない。
現在の再検証は `doctor openai-subscription` を使う。

```text
retired command:
  standalone Phase 0 probe --json
  removed after provider / doctor integration

result:
  status: ok
  gate_b: ok
  auth: logged_in_via_browser
  originator: xelyon
  endpoint: https://chatgpt.com/backend-api/codex/responses
  model: gpt-5.4-mini
  text: ok
  prompt_cache_key: enabled
```

確認できたこと:

- `originator=xelyon` を含む browser OAuth flow が通る。
- token exchange が成功する。
- account ID は token claims から抽出でき、report では mask できる。
- `originator: xelyon` header と OAuth bearer token で subscription endpoint text smoke が通る。
- `prompt_cache_key` は受け付けられる。
- response id は返る。

この smoke で分かった subscription endpoint の wire-shape 要求:

- `instructions` が必須。
- `stream=true` が必須。
- non-streaming JSON response ではなく SSE response として読む必要がある。

0.1 時点で未検証だった retention / tool / context capability は 0.2 で追加確認した。

### 0.2 Live result: technical gates, 2026-06-07

当時の追加 flags で以下を確認した。
現在の再検証は `doctor openai-subscription --retention-smoke --tool-smoke` と通常 smoke 系を使う。

```text
retired command:
  standalone Phase 0 probe --json
    --retention-smoke
    --tool-smoke
    --context-management-smoke
  removed after provider / doctor integration

result:
  status: warn
  gate_b: ok
  text: ok
  prompt_cache_key: enabled
  store: unsupported_required_false
  previous_response_id: unsupported
  invalid_previous_response_id_retry: possible
  context_management: not_applicable_previous_response_id_unsupported
  tool_call: enabled
  tool_continuation: enabled
```

確認できたこと:

- `store=true` は subscription endpoint で拒否される。
- backend は `Store must be set to false` を返す。
- `store=false` でも `previous_response_id` は `Unsupported parameter: previous_response_id` として拒否される。
- invalid `previous_response_id` は、拒否後に full payload retry できる。
- `context_management` smoke は `previous_response_id` 未対応が原因で成立しないため、この endpoint では適用不可として扱う。
- `function_call` item は streaming response で観測できる。
- `function_call_output` continuation は full payload fallback 形式で完走できる。

full provider 方針への影響:

```text
MUST:
  store=false
  previous_response_id disabled
  context_management disabled
  full provider-facing payload
  prompt_cache_key enabled
  streaming required
  tool call / function_call_output continuation enabled

SHOULD NOT:
  OpenAI API provider 同等の response-id chain optimization を claims しない。
  store / previous_response_id を doctor で enabled と表示しない。

CAN:
  subscription provider は stateless/full-payload fallback として成立し得る。
  XELYON tool loop は backend event shape 上は成立し得る。
```

目安:

```text
Phase 0 probe only:
  medium-light
  2-5 files
  provider catalog / config / model picker / runtime profile は触らない
  OAuth browser flow + one-off Responses request + redacted report

Full provider:
  heavy
  provider registration / config / model catalog / auth store / runtime profile /
  doctor / docs / tests / cost / UI integration まで必要
```

この Phase 0 の目的は、以下を判定すること。

```text
Can XELYON use the user's ChatGPT/Codex subscription entitlement
through honest originator=xelyon?
```

判定は二段階に分ける。

```text
Gate B: identity / OAuth / endpoint gate
  originator=xelyon で OAuth login できるか
  token が取れるか
  subscription endpoint に 1 request できるか

Gate C: Responses capability gate
  prompt_cache_key / streaming / tool call / store / previous_response_id /
  context_management がどこまで通るか
```

Gate B が通らない場合、provider 実装には進まない。
Gate C の一部だけが通らない場合、full payload fallback / WARN で provider を成立させる余地がある。

## 1. Non-goals

この Phase 0 ではやらない。

- `openai_subscription` provider の正式登録。
- `/provider` / `/model` / `/config` 統合。
- `config.yaml` への設定追加。
- persistent token store の本実装。
- OpenAI Responses runtime の profile 化。
- OpenAI provider の production path 変更。
- Codex app-server / SDK / CLI を外部 agent として呼ぶ実装。
- `~/.codex/auth.json` の読み取り。
- OpenCode auth cache / `~/.config/opencode` の読み取り。
- `originator=opencode` fallback。
- `originator=codex_cli_rs` fallback。
- OpenAI API key を使う fallback。
- user-facing docs への利用手順追加。

## 2. Why `originator=xelyon` Comes First

`originator` は request identity であり、最適化 capability ではない。

```text
OpenCode:
  originator=opencode

official Codex CLI:
  originator=codex_cli_rs

XELYON:
  originator=xelyon
```

XELYON が別 client の `originator` を名乗ると、技術的に通っても OSS provider としては安全に扱えない。
したがって Phase 0 では browser OAuth authorize URL に `originator=xelyon` を入れて検証する。

Device auth は headless には便利だが、OpenCode / official Codex の device flow では browser authorize URL のように `originator` が明示されていない。
そのため、device auth だけでは `originator=xelyon` gate の証明として弱い。
Phase 0 の primary check は browser OAuth とする。

## 3. What Counts As Success

### 3.1 Gate B success

すべて満たすこと。

- `originator=xelyon` を含む authorize URL で login flow が始まる。
- localhost callback に authorization code が返る。
- PKCE token exchange が成功する。
- `access_token` / `refresh_token` / optional `id_token` が取れる。
- token を stdout / log / debug output に出さない。
- account ID が取れた場合、masked 表示だけ行う。
- `Authorization: Bearer <access_token>` で subscription endpoint に 1 request できる。
- `originator: xelyon` header を付けても backend request が通る。

Gate B が失敗した場合:

```text
oauth rejected:
  stop
  OpenAI に XELYON client identity / OAuth client ID を相談する候補

token ok but endpoint rejected:
  stop
  endpoint / client identity / entitlement を調査

only works with originator=opencode or codex_cli_rs:
  stop
  fallback 実装は禁止
```

### 3.2 Gate C success / warn

Gate C は capability 分類であり、全部が fail でも identity gate とは扱いが違う。

```text
prompt_cache_key accepted:
  OK: enabled

prompt_cache_key rejected:
  WARN: disabled / full payload mode

store=true accepted and response.id returned:
  OK: store supported

previous_response_id accepted:
  OK: safe chain possible

previous_response_id rejected:
  WARN: full payload fallback possible

context_management rejected:
  WARN: subscription profile can disable context_management only

streaming / tool call event shape fundamentally incompatible:
  STOP or parser design rethink
```

## 4. Current Verification Command Shape

standalone probe は provider / auth CLI / doctor へ移植済みのため削除する。
現在の検証は production CLI の doctor を使う。

現在の候補:

```text
xelyon auth openai-subscription status
xelyon doctor openai-subscription --smoke
xelyon doctor openai-subscription --retention-smoke
xelyon doctor openai-subscription --tool-smoke
xelyon doctor openai-subscription --cache-smoke
xelyon doctor openai-subscription --thinking-smoke
xelyon doctor openai-subscription --compact-smoke
xelyon doctor openai-subscription --print-request --json
```

Phase 0 当時に standalone probe が軽かった理由:

- provider registration を触らない。
- config schema を触らない。
- generated registry を触らない。
- `/provider` / `/model` UI を触らない。
- OpenAI provider production path を触らない。

Phase 0 が通って provider 化したため、reusable 部分は `internal/api/providers/openai` と doctor 実装へ移植済み。
probe-only helper は production tree に残さない。

## 5. OAuth Probe Details

constants:

```text
issuer: https://auth.openai.com
client_id: app_EMoamEEZ73f0CkXaXp7hrann
redirect_uri: http://localhost:1455/auth/callback
originator: xelyon
endpoint: https://chatgpt.com/backend-api/codex/responses
```

env override:

```text
XELYON_OPENAI_SUBSCRIPTION_ISSUER
XELYON_OPENAI_SUBSCRIPTION_ENDPOINT
XELYON_OPENAI_SUBSCRIPTION_CLIENT_ID
XELYON_OPENAI_SUBSCRIPTION_ORIGINATOR
XELYON_OPENAI_SUBSCRIPTION_PROBE_PORT
```

browser OAuth request:

```text
response_type=code
client_id=<client_id>
redirect_uri=http://localhost:1455/auth/callback
scope=openid profile email offline_access
code_challenge=<pkce S256 challenge>
code_challenge_method=S256
id_token_add_organizations=true
codex_cli_simplified_flow=true
state=<random>
originator=xelyon
```

Probe requirements:

- state を検証する。
- callback timeout は 5 分。
- token を stdout に出さない。
- success / failure HTML は最小。
- browser を開けない場合は URL を表示する。
- token は default では保存しない。
- optional `--save-token` を作る場合でも `~/.xelyon/auth/openai_subscription_probe.json` のような probe 専用 path にし、0600 で保存する。

## 6. Request Probe Details

headers:

```text
Content-Type: application/json
Authorization: Bearer <access_token>
ChatGPT-Account-Id: <account_id> // optional
originator: xelyon
User-Agent: xelyon/<version> (<os> <arch>)
```

絶対に使わない:

```text
OPENAI_API_KEY
originator=opencode
originator=codex_cli_rs
~/.codex/auth.json
OpenCode auth cache
```

minimal text smoke:

```json
{
  "instructions": "Reply briefly.",
  "model": "gpt-5.4-mini",
  "input": [
    {
      "role": "user",
      "content": "Reply exactly: xelyon subscription probe ok"
    }
  ],
  "stream": true,
  "store": false,
  "prompt_cache_key": "xelyon-openai-subscription-probe"
}
```

If `prompt_cache_key` is rejected, retry once without it and classify:

```text
prompt_cache_key: unsupported
mode: full payload fallback
```

## 7. Retention Probe Details

Retention smoke is separate from identity.

initial request:

```json
{
  "instructions": "Remember the marker and reply briefly.",
  "model": "gpt-5.4-mini",
  "input": [
    {
      "role": "user",
      "content": "Remember the marker xelyon-retention-probe-47. Reply ok."
    }
  ],
  "stream": true,
  "store": true,
  "prompt_cache_key": "xelyon-openai-subscription-retention-probe"
}
```

Required observations:

- HTTP 200.
- response `id` exists.
- content is non-empty.

follow-up request:

```json
{
  "instructions": "Answer with the remembered marker only.",
  "model": "gpt-5.4-mini",
  "input": [
    {
      "role": "user",
      "content": "What marker did I ask you to remember?"
    }
  ],
  "stream": true,
  "store": true,
  "previous_response_id": "<response.id from initial>"
}
```

Required observations:

- HTTP 200.
- response `id` exists.
- answer preserves the marker or otherwise proves chain context.

invalid chain probe:

```json
{
  "instructions": "Reply briefly.",
  "model": "gpt-5.4-mini",
  "input": [
    {
      "role": "user",
      "content": "Reply ok."
    }
  ],
  "stream": true,
  "store": true,
  "previous_response_id": "resp_xelyon_invalid_probe"
}
```

Expected classification:

```text
invalid previous_response_id rejected:
  OK, if error is recognizable and full payload retry succeeds

invalid previous_response_id accepted:
  suspicious, inspect response semantics

invalid previous_response_id error shape unrecognizable:
  WARN, retry detection needs provider-specific parser
```

If `store=true` or `previous_response_id` is rejected:

```text
provider can still be useful
but cannot claim OpenAI provider equivalent response-id chain optimization
doctor must report WARN
runtime should use full payload fallback if provider implementation proceeds
```

## 8. Streaming / Tool Probe

Streaming smoke:

- send `stream=true`.
- parse enough SSE to confirm text deltas and final completion.
- do not require full production parser changes in Phase 0.
- classify event names and payload shapes.

Tool smoke:

- define one dummy function tool.
- ask model to call it.
- verify a `function_call` item or equivalent event appears.
- send a `function_call_output` continuation only if the event shape matches existing OpenAI Responses parser.

If event shape is fundamentally different from existing OpenAI Responses parser, stop full provider work until parser owner is decided.

## 9. Redaction Contract

Never print:

- access token
- refresh token
- id token
- raw Authorization header
- raw `Bearer ...`
- unmasked `ChatGPT-Account-Id`
- JWT-looking long strings

Report shape:

```text
OpenAI Subscription Phase 0 probe
Status: OK/WARN/FAIL
Auth: logged in via browser / failed
Originator: xelyon
Account: acct_****abcd / unavailable
Endpoint: https://chatgpt.com/backend-api/codex/responses
Billing: ChatGPT subscription
API cost: N/A

Capabilities:
  text: ok/fail
  prompt_cache_key: enabled/unsupported
  streaming: ok/fail/not_run
  tool_call: ok/fail/not_run
  store: enabled/unsupported
  previous_response_id: enabled/unsupported
  invalid_previous_response_id_retry: possible/unknown
  context_management: enabled/unsupported/not_run
```

## 10. Implementation Weight

### 10.1 Light path

Scope:

- browser OAuth only.
- non-streaming text smoke.
- retention smoke.
- redacted JSON/text report.
- no persistent token store by default.
- no full provider registration.

Estimated work:

```text
2-5 files
medium-light
main complexity: PKCE browser callback + redaction + request classification
```

Why it is not too heavy:

- existing OpenAI diagnostic smoke already has retention smoke concepts.
- existing OpenAI provider already has Responses request structs and parsers.
- probe can bypass config/model catalog/provider picker.

Why it is not trivial:

- OAuth callback and token exchange must be correct.
- token handling must be secure even in a probe.
- `originator=xelyon` must be tested through browser flow, not only device auth.
- error body redaction must be applied before logging.

### 10.2 Heavy path to avoid in Phase 0

Do not include these in the first spike:

- auth CLI with persistent refresh.
- token refresh mutex.
- `/providers` auth status.
- model catalog / aliases.
- config defaults / registry generation.
- OpenAI Responses runtime profile refactor.
- tool loop production integration.
- cost display integration.
- docs for end users.

Those are full provider work after Gate B/C.

## 11. Stop Conditions

Stop and report if any of these happen:

- `originator=xelyon` browser OAuth authorize is rejected.
- token exchange succeeds only when using another client identity.
- backend request succeeds only with `originator=opencode` or `originator=codex_cli_rs`.
- endpoint rejects XELYON identity even with valid ChatGPT token.
- endpoint requires reading Codex/OpenCode auth cache.
- token or account ID would need to be logged for debugging.
- streaming/tool event shape is too different to classify safely.

Do not add fallback impersonation to keep moving.

## 12. Result Matrix

```text
Gate B OK, Gate C mostly OK:
  proceed to full provider plan

Gate B OK, previous_response_id unsupported:
  provider can proceed with full payload fallback if user accepts degraded mode

Gate B OK, prompt_cache_key unsupported:
  provider can proceed, but cache optimization disabled/WARN

Gate B OK, streaming/tool incompatible:
  pause provider implementation; parser design needed

Gate B FAIL:
  do not implement provider
  consider OpenAI contact for XELYON client identity/OAuth client ID
```

## 13. Historical Phase 0 Goal Handoff Prompt

この prompt は Phase 0 probe 実装時の historical handoff である。
full provider 実装 Goal には `docs/dev/openai-subscription-provider-master-plan.md` の handoff prompt を使う。

Phase 0 probe Goal:

```text
Use docs/dev/openai-subscription-phase0-reality-check.md as the source of truth.
Do not implement the full openai_subscription provider. Build only the minimum
Phase 0 probe needed to test honest originator=xelyon browser OAuth and the
subscription Responses endpoint. Do not read Codex/OpenCode auth caches, do not
use OPENAI_API_KEY, and do not add originator=opencode or originator=codex_cli_rs
fallbacks. Redact all tokens and account IDs. Report Gate B identity result and
Gate C capability classification for prompt_cache_key, streaming, tool call,
store, previous_response_id, invalid previous_response_id retry, and
context_management if tested. Do not commit or push unless explicitly requested.
```
