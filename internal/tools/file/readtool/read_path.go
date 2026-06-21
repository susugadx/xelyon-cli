package readtool

import "github.com/susugadx/xelyon-cli/internal/filequery"

// parsePath はパス文字列を解析し、ファイルパスと行範囲を返す
// "path" → path, 0, 0
// "path:10" → path, 10, 0
// "path:10-20" → path, 10, 20
func parsePath(entry string) (string, int, int) {
	return filequery.ParsePath(entry)
}
