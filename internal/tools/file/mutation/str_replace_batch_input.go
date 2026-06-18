package mutation

import (
	"encoding/json"
	"fmt"
)

func parseBatchEditEntriesResult(editsJSON string) ([]EditEntry, fileMutationResult) {
	edits, err := parseBatchEditEntries(editsJSON)
	if err != nil {
		return nil, newErrorMutationResult(fmt.Sprintf("Error: invalid edits JSON: %v", err))
	}
	return edits, fileMutationResult{}
}

func parseBatchEditEntries(editsJSON string) ([]EditEntry, error) {
	var edits []EditEntry
	if err := json.Unmarshal([]byte(editsJSON), &edits); err != nil {
		return nil, err
	}
	return edits, nil
}

func validateBatchEditEntries(path string, edits []EditEntry) fileMutationResult {
	if len(edits) == 0 {
		return newErrorMutationResult("Error: edits array is empty")
	}

	for i, edit := range edits {
		if edit.OldStr == "" {
			return newErrorMutationResult(fmt.Sprintf("Error: edits[%d].old_str is empty in %s", i, path))
		}
		if edit.OldStr == edit.NewStr {
			return newErrorMutationResult(fmt.Sprintf("Error: edits[%d] old_str and new_str are identical (no change needed) in %s", i, path))
		}
	}

	return fileMutationResult{}
}
