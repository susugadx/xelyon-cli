package review

import (
	"io"
	"os"
)

func readReviewEvidenceFilePrefix(path string, maxBytes int64) ([]byte, bool, error) {
	if maxBytes < 0 {
		maxBytes = 0
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) <= maxBytes {
		return data, false, nil
	}
	return data[:maxBytes], true, nil
}
