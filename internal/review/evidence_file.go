package review

import (
	"bytes"
	"io"
	"os"
)

type reviewEvidenceRegularFileReadStatus int

const (
	reviewEvidenceRegularFileReadOK reviewEvidenceRegularFileReadStatus = iota
	reviewEvidenceRegularFileReadMissing
	reviewEvidenceRegularFileReadStatFailed
	reviewEvidenceRegularFileReadSymlink
	reviewEvidenceRegularFileReadDir
	reviewEvidenceRegularFileReadNonRegular
	reviewEvidenceRegularFileReadInvalidPath
	reviewEvidenceRegularFileReadFileTooLarge
	reviewEvidenceRegularFileReadBudgetExceeded
	reviewEvidenceRegularFileReadFailed
)

type reviewEvidenceRegularFileReadStage int

const (
	reviewEvidenceRegularFileReadStageNone reviewEvidenceRegularFileReadStage = iota
	reviewEvidenceRegularFileReadStageLstat
	reviewEvidenceRegularFileReadStageValidate
	reviewEvidenceRegularFileReadStageStat
	reviewEvidenceRegularFileReadStageRead
)

type reviewEvidenceRegularFileReadInput struct {
	repoRoot       string
	absPath        string
	relPath        string
	maxBytes       int64
	allowSymlink   bool
	maxFileBytes   int64
	enforceFileMax bool
	maxBudgetBytes int64
	enforceBudget  bool
}

type reviewEvidenceRegularFileReadResult struct {
	status    reviewEvidenceRegularFileReadStatus
	stage     reviewEvidenceRegularFileReadStage
	err       error
	data      []byte
	binary    bool
	truncated bool
	sizeBytes int64
	readBytes int64
	regular   bool
}

func readReviewEvidenceRegularFile(input reviewEvidenceRegularFileReadInput) reviewEvidenceRegularFileReadResult {
	lstatInfo, err := os.Lstat(input.absPath)
	if os.IsNotExist(err) {
		return reviewEvidenceRegularFileReadResult{
			status: reviewEvidenceRegularFileReadMissing,
			stage:  reviewEvidenceRegularFileReadStageLstat,
			err:    err,
		}
	}
	if err != nil {
		return reviewEvidenceRegularFileReadResult{
			status: reviewEvidenceRegularFileReadStatFailed,
			stage:  reviewEvidenceRegularFileReadStageLstat,
			err:    err,
		}
	}

	result := reviewEvidenceRegularFileReadResult{
		sizeBytes: lstatInfo.Size(),
	}
	lstatMode := lstatInfo.Mode()
	if lstatMode&os.ModeSymlink != 0 && !input.allowSymlink {
		result.status = reviewEvidenceRegularFileReadSymlink
		return result
	}
	if lstatInfo.IsDir() {
		result.status = reviewEvidenceRegularFileReadDir
		return result
	}
	if lstatMode&os.ModeSymlink == 0 && !lstatMode.IsRegular() {
		result.status = reviewEvidenceRegularFileReadNonRegular
		return result
	}

	if err := validateReviewEvidenceExistingPath(input.repoRoot, input.absPath, input.relPath); err != nil {
		result.status = reviewEvidenceRegularFileReadInvalidPath
		result.stage = reviewEvidenceRegularFileReadStageValidate
		result.err = err
		return result
	}

	statInfo, err := os.Stat(input.absPath)
	if err != nil {
		result.status = reviewEvidenceRegularFileReadStatFailed
		result.stage = reviewEvidenceRegularFileReadStageStat
		result.err = err
		return result
	}
	result.sizeBytes = statInfo.Size()
	if statInfo.IsDir() {
		result.status = reviewEvidenceRegularFileReadDir
		return result
	}
	if !statInfo.Mode().IsRegular() {
		result.status = reviewEvidenceRegularFileReadNonRegular
		return result
	}

	result.regular = true
	if input.enforceFileMax && statInfo.Size() > input.maxFileBytes {
		result.status = reviewEvidenceRegularFileReadFileTooLarge
		return result
	}
	if input.enforceBudget && (input.maxBudgetBytes <= 0 || statInfo.Size() > input.maxBudgetBytes) {
		result.status = reviewEvidenceRegularFileReadBudgetExceeded
		return result
	}

	data, truncated, err := readReviewEvidenceFilePrefix(input.absPath, input.maxBytes)
	if err != nil {
		result.status = reviewEvidenceRegularFileReadFailed
		result.stage = reviewEvidenceRegularFileReadStageRead
		result.err = err
		return result
	}

	result.status = reviewEvidenceRegularFileReadOK
	result.data = data
	result.binary = bytes.IndexByte(data, 0) >= 0
	result.readBytes = int64(len(data))
	result.truncated = truncated || statInfo.Size() > input.maxBytes
	return result
}

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
