package reviewscreen

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
	tuiviewport "github.com/susugadx/xelyon-cli/internal/tui/viewport"
)

type reviewBodyViewport struct {
	yOffset int
}

func (v *reviewBodyViewport) reset() {
	v.yOffset = 0
}

func (v reviewBodyViewport) render(lines []string, height int, width int) string {
	height = max(0, height)
	yOffset := tuiviewport.ClampYOffset(v.yOffset, len(lines), height)
	visible := tuiviewport.VisibleLines(lines, yOffset, height)

	var body strings.Builder
	body.Grow(height * (max(0, width) + 1))
	for row := 0; row < height; row++ {
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
