package tui

import tuiviewport "github.com/susugadx/xelyon-cli/internal/tui/viewport"

// lightViewport は bubbles/viewport の軽量代替。
// lipgloss を使わず、行スライスの直接操作のみで描画する。
type lightViewport struct {
	lines   []string
	yOffset int
	width   int
	height  int
}

// setContent はコンテンツを文字列で設定する。
func (v *lightViewport) setContent(s string) {
	v.lines = tuiviewport.LinesFromContent(s)
	if v.yOffset > v.maxYOffset() {
		v.gotoBottom()
	}
}

// setLines はコンテンツを行スライスで設定する（Split 不要、ゼロコピー）。
func (v *lightViewport) setLines(lines []string) {
	v.lines = lines
	if v.yOffset > v.maxYOffset() {
		v.gotoBottom()
	}
}

// visibleLines は現在表示中の行を返す。
func (v *lightViewport) visibleLines() []string {
	return tuiviewport.VisibleLines(v.lines, v.yOffset, v.height)
}

// view は表示領域の行を結合して返す。高さに足りない場合は空行で埋める。
func (v *lightViewport) view() string {
	return tuiviewport.Render(v.lines, v.yOffset, v.height, v.width)
}

func (v *lightViewport) maxYOffset() int {
	return tuiviewport.MaxYOffset(len(v.lines), v.height)
}

func (v *lightViewport) atBottom() bool {
	return v.yOffset >= v.maxYOffset()
}

func (v *lightViewport) gotoBottom() {
	v.yOffset = v.maxYOffset()
}

func (v *lightViewport) scrollUp(n int) {
	v.yOffset = tuiviewport.ScrollUp(v.yOffset, n, len(v.lines), v.height)
}

func (v *lightViewport) scrollDown(n int) {
	v.yOffset = tuiviewport.ScrollDown(v.yOffset, n, len(v.lines), v.height)
}

// gotoTop は先頭にスクロールする。
func (v *lightViewport) gotoTop() {
	v.yOffset = 0
}
