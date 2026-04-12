# internal/tui

Bubble Tea ベースの TUI 実装です。`Model` 自体は top-level orchestration に寄せ、下位の file は owner ごとに分割しています。

## 責務マップ

- `model_types.go`, `model_lifecycle.go`, `model_update.go`, `model_interrupt.go`
  - `Model` の状態定義と root lifecycle / update orchestration
- `model_input*.go`, `model_input_submission_*.go`, `composer*.go`
  - chat 入力、composer 状態、submit 判定、command/chat 振り分け
- `model_config.go`, `model_config_render.go`, `config_screen_*.go`
  - `/config` 画面の orchestration、state、input、mutation、save、render
- `model_navigation*.go`
  - NAV mode、cursor、vim motion、visual selection、copy、pending state
- `model_output.go`, `model_stream*.go`, `stream_*.go`, `stream_cells_*.go`
  - chat 出力、stream merge、ANSI 付き chunk 更新、cell rebuild
- `model_render.go`, `render_*.go`, `layout_*.go`
  - viewport/chrome 描画、ANSI helper、layout build と cursor map
- `model_toolblock*.go`
  - tool block の content/focus/render と viewport jump
- `model_mouse_selection*.go`
  - mouse selection の state、drag/autoscroll、copy、overlay render

## 境界の原則

- `model_*.go` の上位 file には top-level dispatch と orchestration だけを置く
- pure logic は `layout_*`, `render_ansi_*`, `stream_cells_*` のように Bubble Tea 依存から切る
- `/config` と NAV は `state / input / mutation / render` を跨がせない
- clipboard や save のような I/O は state 更新 helper と分離する

## 代表フロー

- chat submit
  - `model_input.go` -> `model_input_chat.go` -> `model_input_submission_*.go` -> `composer*.go`
- config save
  - `model_config.go` -> `config_screen_input.go` -> `config_screen_*mutation.go` -> `config_screen_save_state.go`
- nav copy
  - `model_navigation_input.go` -> `model_navigation_vim_*.go` -> `model_navigation_visual_state.go` / `model_navigation_copy.go`
- stream render
  - `model_stream.go` / `model_output.go` -> `stream_merge.go` / `stream_cells_*.go` -> `layout_*.go` -> `render_viewport*.go`

## package 分割をまだしていない理由

現状は file owner の分離で、`Model` の共有状態と Bubble Tea 更新ループの境界はかなり明確になっています。ここで sub-package 化まで進めると、`Model` の内部状態や `tea.Msg` の共有をまたぐ exported API が増えやすく、逆に境界が薄くなりやすいです。

次に package 分割へ進めるなら、候補は次の単位です。

- `config`
  - `config_screen_*` と `model_config*.go`
- `nav`
  - `model_navigation*`, `model_mouse_selection*`
- `render`
  - `render_*`, `layout_*`, `stream_cells_*`

この段階では、まず file owner と integration test を安定させてから package 境界へ進める方が安全です。
