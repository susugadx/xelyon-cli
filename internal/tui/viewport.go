package tui

import "strings"

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
	v.lines = strings.Split(s, "\n")
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
	if len(v.lines) == 0 || v.height <= 0 {
		return nil
	}

	top := v.yOffset
	if top < 0 {
		top = 0
	}
	end := top + v.height
	if end > len(v.lines) {
		end = len(v.lines)
	}
	return v.lines[top:end]
}

// view は表示領域の行を結合して返す。高さに足りない場合は空行で埋める。
func (v *lightViewport) view() string {
	visible := v.visibleLines()
	var sb strings.Builder
	sb.Grow(v.height * (v.width + 1))
	for i := 0; i < v.height; i++ {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i < len(visible) {
			sb.WriteString(visible[i])
		}
	}
	return sb.String()
}

func (v *lightViewport) maxYOffset() int {
	m := len(v.lines) - v.height
	if m < 0 {
		return 0
	}
	return m
}

func (v *lightViewport) atBottom() bool {
	return v.yOffset >= v.maxYOffset()
}

func (v *lightViewport) gotoBottom() {
	v.yOffset = v.maxYOffset()
}

func (v *lightViewport) scrollUp(n int) {
	v.yOffset -= n
	if v.yOffset < 0 {
		v.yOffset = 0
	}
}

func (v *lightViewport) scrollDown(n int) {
	v.yOffset += n
	if v.yOffset > v.maxYOffset() {
		v.yOffset = v.maxYOffset()
	}
}

// gotoTop は先頭にスクロールする。
func (v *lightViewport) gotoTop() {
	v.yOffset = 0
}

// halfPageUp は半ページ上にスクロールする。
func (v *lightViewport) halfPageUp() {
	v.scrollUp(max(1, v.height/2))
}

// halfPageDown は半ページ下にスクロールする。
func (v *lightViewport) halfPageDown() {
	v.scrollDown(max(1, v.height/2))
}
