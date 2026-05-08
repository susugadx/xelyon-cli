# Kimi built-in web_search 設計メモ

## 現状

Kimi native provider は Moonshot Chat Completions API の通常 function tools、streaming、thinking mode、`reasoning_content` の履歴再送、image input を扱う。built-in `$web_search` は通常 `ChatWithTools` の tools には混ぜず、XELYON の `web_search` tool が呼ぶ native search backend として実装する。

`web_search.provider = kimi` / `moonshot`、またはメイン provider が Kimi で `web_search.provider` 未設定の場合、`internal/api/providers/kimi` の専用 route が使われる。`moonshot` は provider_models の alias owner として扱うため、`provider_models.moonshot.default_model` があれば検索 model もそれを使う。

## Request Contract

Kimi `$web_search` request は Chat Completions streaming payload のまま送る。

- `tools[].type = "builtin_function"`
- `tools[].function.name = "$web_search"`
- `thinking: {"type":"disabled"}`
- `stream: true`
- `stream_options.include_usage = true`
- `prompt_cache_key` を通常 Kimi request と同じ owner で送る
- `max_completion_tokens` を送る
- `max_tokens` と `tool_choice` は送らない

通常 function tools はこの route に入れない。画像 / video / file upload / memory / code runner とも混ぜない。

## Tool Loop

1. system/user + `$web_search` tool declaration を送る。
2. `finish_reason = "tool_calls"` の場合、assistant message に返却された `tool_calls` を保持する。
3. 各 `$web_search` call の `tool_call.function.arguments` を変換せず、その文字列を role=`tool` の `content` に入れる。
4. tool message には `tool_call_id` と Kimi 仕様の `name` を含める。
5. 次 request にも `$web_search` tool declaration を含める。
6. `finish_reason = "stop"` の final content を XELYON の `web_search` result text として返す。

ループは最大 3 request まで。さらに tool loop が続く場合は、検索が完了しなかった明示エラーにする。

## Usage / Cost

Kimi stream usage が返った場合だけ token usage として `api.UsageCallback` に流す。`cached_tokens` は `api.Usage.CachedInputTokens` に乗せる。

Moonshot は `$web_search` call fee と Chat Completions token 使用量を別々に課金する。`finish_reason = "tool_calls"` で `tool_call.function.name = "$web_search"` が返った場合、XELYON は call count を数え、[Kimi API Platform WebSearch Pricing](https://platform.moonshot.ai/docs/pricing/tools.en-US) の `$0.005 / invocation` を `api.Usage.StorageCost` に載せる。`finish_reason = "stop"` で `$web_search` tool call がない場合は call fee を載せない。call fee callback は token usage とは別に発火し、`api.Usage.WebSearchCalls` と `api.Usage.WebSearchResultTokens` に観測値を載せる。

`tool_call.function.arguments` は replay 用にそのまま保持する。別途 best-effort で JSON parse し、`usage.total_tokens` または top-level `total_tokens` があれば検索結果 token 観測値として記録する。parse 失敗は tool loop の成否に影響させない。検索結果 tokens は次 request の `prompt_tokens` に含まれるため、`InputTokens` へは二重加算しない。

`api.Usage.HasTokenObservation` は endpoint が返した token / cache usage だけを見る。`api.Usage.HasTokenOrWebSearchObservation` は token usage に加えて Kimi `$web_search` call fee 観測も含める。doctor smoke の `usage` check は前者を使い、web search の fee / call count 観測だけで endpoint token usage を観測済みにはしない。

## Diagnostics

`doctor kimi --web-search-smoke` と `make kimi-web-search-smoke` は live API で以下を確認する。

- `$web_search` payload が `builtin_function` になっていること
- thinking disabled が送られること
- `prompt_cache_key` があること
- 最終回答が空でないこと
- usage が返った場合は token/cached usage が観測されること
- `$web_search` call count と call fee estimate が観測されること
- `tool_call.function.arguments` に検索結果 token 使用量が含まれる場合、表示用に観測されること

`--web-search-smoke` は `$web_search` call count が 1 以上の場合だけ成功扱いにする。通常の `stop` response で request が返っても、tool call がなければ call fee / search result token 観測の診断にならないため fail する。endpoint の token usage が返らない場合は `usage` check を warn に留め、`web_search_usage_observed` は `$web_search` call fee / call count 側の観測として text / JSON に出す。`cached_input_tokens` は API が返した場合だけ増えるため 0 でも成功条件にはしない。`MOONSHOT_API_KEY` がない場合、`make kimi-web-search-smoke` は実行前に失敗する。
