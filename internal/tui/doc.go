// Package tui は Bubble Tea ベースの TUI 実装を提供する。
//
// Model は top-level orchestration を担当し、下位の file は責務ごとに分割する。
//
// 主な責務境界:
//   - model_types.go, model_lifecycle.go, model_update.go, model_interrupt.go:
//     Model の状態定義と root lifecycle / update orchestration
//   - model_input*.go, model_input_submission_*.go, composer*.go:
//     chat 入力、composer 状態、submit 判定、command/chat 振り分け
//   - model_config.go, model_config_render.go, config_screen_*.go:
//     /config 画面の orchestration、state、input、mutation、save、render
//   - model_review.go, review_screen_*.go:
//     /review 画面の orchestration、preset/custom 入力、ReviewRequest 生成、render
//   - model_navigation*.go:
//     NAV mode、cursor、vim motion、visual selection、copy、pending state
//   - model_output.go, model_stream*.go:
//     chat 出力、stream 更新、Model への反映
//   - model_render.go, render_*.go, viewport.go:
//     viewport/chrome 描画と TUI 固有の表示合成
//   - model_toolblock*.go:
//     tool block の content/focus/render と viewport jump
//   - model_mouse_selection*.go:
//     mouse selection の state、drag/autoscroll、copy、overlay render
//
// pure logic は internal/tui/termtext に置き、Bubble Tea の更新ループから切り離す。
// TUI lifecycle の terminal 復旧や exit hook は internal/tui/lifecycle が owner する。
package tui
