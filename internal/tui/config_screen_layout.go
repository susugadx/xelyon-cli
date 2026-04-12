package tui

// configLayout は /config 画面のレイアウト計算をまとめる。
type configLayout struct {
	width  int
	height int
}

func newConfigLayout(width, height int) configLayout {
	return configLayout{width: width, height: height}
}

// configPaneWidths は3ペインの幅を計算する。
func configPaneWidths(totalWidth int) (left, mid, right int) {
	// 最小幅確保
	if totalWidth < 40 {
		return totalWidth, 0, 0
	}
	if totalWidth < 80 {
		left = totalWidth * 30 / 100
		mid = totalWidth - left
		right = 0
		return
	}
	left = max(18, totalWidth*20/100)
	mid = max(25, totalWidth*30/100)
	right = totalWidth - left - mid
	return
}

func (l configLayout) paneWidths() (left, mid, right int) {
	return configPaneWidths(l.width)
}

func (l configLayout) bodyHeight() int {
	bodyHeight := l.height - 2 // ヘッダー1行 + ステータス1行
	if bodyHeight < 1 {
		return 1
	}
	return bodyHeight
}

func (l configLayout) detailVisible() bool {
	_, _, right := l.paneWidths()
	return right > 0
}

func (l configLayout) editTargetPane() configPane {
	if l.detailVisible() {
		return paneDetail
	}
	return paneField
}

func (l configLayout) fieldPaneVisibleRows(cs *configScreen) int {
	bodyHeight := l.bodyHeight()
	if cs != nil && (cs.filterMode || cs.filterText != "") {
		bodyHeight--
	}
	if bodyHeight < 1 {
		return 1
	}
	return bodyHeight
}
