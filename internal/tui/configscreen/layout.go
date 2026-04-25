package configscreen

// Layout は /config 画面のレイアウト計算条件を保持する。
type Layout struct {
	width  int
	height int
}

// NewLayout は幅と高さから Layout を構築する。
func NewLayout(width, height int) Layout {
	return Layout{width: width, height: height}
}

// PaneWidths は3ペインの幅を計算する。
func PaneWidths(totalWidth int) (left, mid, right int) {
	if totalWidth < 40 {
		return totalWidth, 0, 0
	}
	if totalWidth < 80 {
		left = totalWidth * 30 / 100
		mid = totalWidth - left
		right = 0
		return
	}
	left = maxInt(18, totalWidth*20/100)
	mid = maxInt(25, totalWidth*30/100)
	right = totalWidth - left - mid
	return
}

// PaneWidths は Layout の幅から3ペインの幅を計算する。
func (l Layout) PaneWidths() (left, mid, right int) {
	return PaneWidths(l.width)
}

// BodyHeight は header/status を除いた body 高さを返す。
func (l Layout) BodyHeight() int {
	bodyHeight := l.height - 2
	if bodyHeight < 1 {
		return 1
	}
	return bodyHeight
}

// DetailVisible は detail pane が表示可能かを返す。
func (l Layout) DetailVisible() bool {
	_, _, right := l.PaneWidths()
	return right > 0
}

// EditTargetPane は編集 UI を表示する pane を返す。
func (l Layout) EditTargetPane() Pane {
	if l.DetailVisible() {
		return PaneDetail
	}
	return PaneField
}

// FieldPaneVisibleRows は field pane の表示可能行数を返す。
func (l Layout) FieldPaneVisibleRows(filterVisible bool) int {
	bodyHeight := l.BodyHeight()
	if filterVisible {
		bodyHeight--
	}
	if bodyHeight < 1 {
		return 1
	}
	return bodyHeight
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
