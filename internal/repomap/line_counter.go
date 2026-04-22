package repomap

import (
	"bufio"
	"io"
	"os"
)

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	reader := bufio.NewReaderSize(f, 32*1024)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return count, nil
			}
			return 0, err
		}
		if b == '\n' {
			count++
		}
	}
}
