package uifileview

import (
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/uistyle"
)

// FileOpHeader はファイル操作ツールのコンパクトヘッダーを出力する。
//
//	── tool  detail ──
func FileOpHeader(w io.Writer, toolLabel, detail string) {
	if w == nil {
		return
	}
	pal := uistyle.NewFileOpPalette(w)
	pal.Border(w, "── ")
	pal.Accent(w, toolLabel)
	if detail != "" {
		_, _ = fmt.Fprint(w, "  ")
		pal.Muted(w, detail)
	}
	pal.Border(w, " ──")
	_, _ = fmt.Fprintln(w)
}

// FileOpDivider はファイル操作用の薄い区切り線を出力する。
func FileOpDivider(w io.Writer, width int) {
	if w == nil {
		return
	}
	pal := uistyle.NewFileOpPalette(w)
	pal.Border(w, strings.Repeat("─", width))
	_, _ = fmt.Fprintln(w)
}

// FileOpStatsLine はファイル操作の変更統計を 1 行で出力する。
//
//	-5 / +8 (net +3)
func FileOpStatsLine(w io.Writer, removed, added int) {
	if w == nil {
		return
	}
	pal := uistyle.NewFileOpPalette(w)
	_, _ = fmt.Fprint(w, "  ")
	pal.DelLine(w, fmt.Sprintf("-%d", removed))
	_, _ = fmt.Fprint(w, " / ")
	pal.AddLine(w, fmt.Sprintf("+%d", added))
	net := added - removed
	switch {
	case net > 0:
		_, _ = fmt.Fprint(w, " (net ")
		pal.AddLine(w, fmt.Sprintf("+%d", net))
		_, _ = fmt.Fprintln(w, ")")
	case net < 0:
		_, _ = fmt.Fprint(w, " (net ")
		pal.DelLine(w, fmt.Sprintf("%d", net))
		_, _ = fmt.Fprintln(w, ")")
	default:
		_, _ = fmt.Fprintln(w, " (net 0)")
	}
}

// FileOpPathLine はファイル操作のパス行を出力する。
//
//	M path (+3, -2)
func FileOpPathLine(w io.Writer, action, path, counts string) {
	if w == nil {
		return
	}
	pal := uistyle.NewFileOpPalette(w)
	_, _ = fmt.Fprint(w, "  ")
	pal.Muted(w, action)
	_, _ = fmt.Fprint(w, " ")
	pal.Accent(w, path)
	if counts != "" {
		_, _ = fmt.Fprint(w, " ")
		pal.Muted(w, counts)
	}
	_, _ = fmt.Fprintln(w)
}
