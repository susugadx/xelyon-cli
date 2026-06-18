package applypatch

import "fmt"

// ApplyResult は apply_patch の適用結果を表す。
type ApplyResult struct {
	Added    []string
	Modified []string
	Deleted  []string
	details  []applyResultDetail
}

type applyResultDetail struct {
	Path         string
	Action       string
	LinesAdded   int
	LinesRemoved int
	OldContent   string
	NewContent   string
}

// ApplyPatch はパッチテキストを解析してファイルへ適用する。
func ApplyPatch(patchText string) (*ApplyResult, error) {
	parsed, err := ParsePatch(patchText)
	if err != nil {
		return nil, err
	}
	return applyParsedPatch(parsed)
}

func applyParsedPatch(parsed *ParsedPatch) (*ApplyResult, error) {
	if parsed == nil || len(parsed.Hunks) == 0 {
		return nil, fmt.Errorf("no files were modified")
	}

	ops, snapshots, result, err := buildPlannedFileOps(parsed)
	if err != nil {
		return nil, err
	}

	if err := applyFileOps(ops, snapshots); err != nil {
		return nil, err
	}

	return result, nil
}
