package file

import (
	"bufio"
	"io"
)

// readFirstNLines は io.Reader から最初の n 行を読み、残りがあるかを返す
func readFirstNLines(r io.Reader, n int) (lines []string, totalRead int, hasMore bool, err error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		totalRead++
		if totalRead <= n {
			lines = append(lines, scanner.Text())
			continue
		}
		hasMore = true
		return lines, totalRead, hasMore, nil
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, 0, false, scanErr
	}
	return lines, totalRead, false, nil
}
