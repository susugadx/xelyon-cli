package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
	tuiviewport "github.com/susugadx/xelyon-cli/internal/tui/viewport"
)

type reviewBodyViewport struct {
	yOffset int
}

type reviewBodyScrollBounds struct {
	lineCount int
	height    int
}

func newReviewBodyScrollBounds(lineCount int, height int) reviewBodyScrollBounds {
	return reviewBodyScrollBounds{
		lineCount: lineCount,
		height:    max(0, height),
	}
}

func reviewBodyScrollBoundsForLines(lines []string, height int) reviewBodyScrollBounds {
	return newReviewBodyScrollBounds(len(lines), height)
}

func (b reviewBodyScrollBounds) hasOverflow() bool {
	return b.maxYOffset() > 0
}

func (b reviewBodyScrollBounds) maxYOffset() int {
	return tuiviewport.MaxYOffset(b.lineCount, b.height)
}

func (b reviewBodyScrollBounds) pageSize() int {
	return max(1, b.height)
}

func (v *reviewBodyViewport) reset() {
	v.yOffset = 0
}

func (v *reviewBodyViewport) clamp(bounds reviewBodyScrollBounds) {
	v.yOffset = tuiviewport.ClampYOffset(v.yOffset, bounds.lineCount, bounds.height)
}

func (v *reviewBodyViewport) scrollUp(n int, bounds reviewBodyScrollBounds) {
	v.yOffset = tuiviewport.ScrollUp(v.yOffset, n, bounds.lineCount, bounds.height)
}

func (v *reviewBodyViewport) scrollDown(n int, bounds reviewBodyScrollBounds) {
	v.yOffset = tuiviewport.ScrollDown(v.yOffset, n, bounds.lineCount, bounds.height)
}

func (v *reviewBodyViewport) gotoTop() {
	v.yOffset = 0
}

func (v *reviewBodyViewport) gotoBottom(bounds reviewBodyScrollBounds) {
	v.yOffset = bounds.maxYOffset()
}

func (v reviewBodyViewport) render(lines []string, height int, width int) string {
	bounds := reviewBodyScrollBoundsForLines(lines, height)
	yOffset := tuiviewport.ClampYOffset(v.yOffset, bounds.lineCount, bounds.height)
	visible := tuiviewport.VisibleLines(lines, yOffset, bounds.height)

	var body strings.Builder
	body.Grow(bounds.height * (max(0, width) + 1))
	for row := 0; row < bounds.height; row++ {
		if row > 0 {
			body.WriteByte('\n')
		}
		if row < len(visible) {
			body.WriteString(termtext.FillANSITextWidth(visible[row], width, theme.Config.BgNormal))
			continue
		}
		body.WriteString(termtext.FillANSITextWidth("", width, theme.Config.BgNormal))
	}
	return body.String()
}
