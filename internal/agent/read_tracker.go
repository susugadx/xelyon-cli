package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// readTrackerThreshold は同一ファイルの read_file 呼び出し回数がこの値以上で
// soft guidance を付与する閾値。
const readTrackerThreshold = 3

// readTracker は同一ファイルの read_file 呼び出しを追跡し、
// 過剰な micro-read ループを抑制する soft guidance を生成する。
// hard block ではなく、ツール結果に guidance メッセージを追記するのみ。
// 同一ファイルへの guidance は 1 回だけ出し、繰り返さない。
type readTracker struct {
	// fileCounts はファイルパスごとの read_file 呼び出し回数
	fileCounts map[string]int
	// guidanceShown は guidance を実際に表示済みのファイルパスの集合
	guidanceShown map[string]bool
}

// newReadTracker は新しい readTracker を生成する。
func newReadTracker() *readTracker {
	return &readTracker{
		fileCounts:    make(map[string]int),
		guidanceShown: make(map[string]bool),
	}
}

// record は read_file のツール呼び出しを記録し、soft guidance メッセージを返す。
// guidance が不要な場合は空文字を返す。
// 同一ファイルへの guidance は 1 回だけ出す（guidanceShown で管理）。
// guidanceShown の更新はここでは行わない。呼び出し元が実際に guidance を
// 結果に追記できた場合に confirmShown を呼んで確定する。
// str_replace 等の書き込みツールが呼ばれた場合は対象ファイルのカウントをリセットする。
func (rt *readTracker) record(toolCall *tools.ToolCall) string {
	if rt == nil {
		return ""
	}

	toolName := toolCall.Tool

	// write 系ツールが呼ばれたら対象ファイルのカウントと guidance 状態をリセット
	// （編集後の再読は正当な再読であるため）
	if tools.IsWriteTool(toolName) {
		path := toolCall.Args["path"]
		if path != "" {
			delete(rt.fileCounts, path)
			delete(rt.guidanceShown, path)
		}
		return ""
	}

	if toolName != "read_file" {
		return ""
	}

	path := toolCall.Args["path"]
	if path == "" {
		return ""
	}

	rt.fileCounts[path]++
	count := rt.fileCounts[path]

	if count < readTrackerThreshold {
		return ""
	}

	// 同一ファイルへの guidance は 1 回だけ
	if rt.guidanceShown[path] {
		return ""
	}

	// soft guidance: read_file symbol mode や broader range を促す
	return fmt.Sprintf("\n\n[GUIDANCE] You have read this file %d times in this session. "+
		"Consider reading a larger range or using symbol mode instead of multiple micro-reads. "+
		"If you already have enough context, proceed to edit.", count)
}

// confirmShown は guidance が実際に結果に追記されたことを記録する。
// record() が guidance を返しても、Error 結果等で append されなかった場合は
// この関数を呼ばないことで、次回に再度 guidance を出せるようにする。
func (rt *readTracker) confirmShown(path string) {
	if rt == nil {
		return
	}
	rt.guidanceShown[path] = true
}

// reset はファイルのカウントをリセットする（タスク完了時など）。
func (rt *readTracker) reset() {
	if rt == nil {
		return
	}
	rt.fileCounts = make(map[string]int)
	rt.guidanceShown = make(map[string]bool)
}

// appendGuidanceWithConfirm は結果文字列に guidance を追記し、
// 実際に追記できた場合に readTracker の shown 状態を確定する。
// guidance が空の場合や Error 結果の場合は result をそのまま返す。
func appendGuidanceWithConfirm(rt *readTracker, path, result, guidance string) string {
	if guidance == "" {
		return result
	}
	// Error 結果には guidance を付けない（shown 状態も更新しない）
	if strings.HasPrefix(strings.TrimSpace(result), "Error:") {
		return result
	}
	rt.confirmShown(path)
	return result + guidance
}
