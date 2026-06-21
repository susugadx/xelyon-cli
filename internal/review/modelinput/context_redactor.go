package modelinput

func normalizeRedactor(redactor Redactor) Redactor {
	if redactor == nil {
		return noopRedactor{}
	}
	return redactor
}

type noopRedactor struct{}

// RedactText は nil redactor 時に text をそのまま返す。
func (noopRedactor) RedactText(text string) string {
	return text
}

// RedactTexts は nil redactor 時に text 配列をそのまま返す。
func (noopRedactor) RedactTexts(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

// RedactPath は nil redactor 時に path をそのまま返す。
func (noopRedactor) RedactPath(path string) string {
	return path
}

// RedactPaths は nil redactor 時に path 配列をそのまま返す。
func (noopRedactor) RedactPaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{}
	}
	return append([]string(nil), paths...)
}
