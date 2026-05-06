# Kimi built-in web_search 設計メモ

## 現状

Kimi native provider は Moonshot Chat Completions API の通常 function tools、streaming、thinking mode、`reasoning_content` の履歴再送、image input を扱う。Kimi built-in の `$web_search`、memory、code runner、video 入力はまだ runtime request に混ぜない。

XELYON の `web_search` ツールは現状 OpenAI / Gemini / Claude のネイティブ検索 provider surface に寄せており、メイン provider が Kimi の場合は `web_search.provider` で対応 provider を明示する。

## 将来方針

Kimi built-in `$web_search` を追加する場合は、通常 function tools とは別の Kimi provider-specific capability として扱う。`tools[]` の generic OpenAI-compatible function 定義へ混ぜるのではなく、Kimi request builder が Moonshot の仕様に沿った provider 固有 payload を生成する。

実装前に次を確認する。

- `$web_search` の request / response / streaming event 形状
- 通常 function tool calls と built-in search result の同一 turn 内での順序
- `reasoning_content`、`tool_calls`、role=`tool` result の履歴 replay と競合しないこと
- `web_search.provider` の明示 override と Kimi main provider の implicit built-in search の優先順位
- docs/providers.md、docs/config.md、docs/commands.md の capability 表示

このメモは将来設計の境界だけを固定する。今回の Kimi runtime request には `$web_search` を追加しない。
