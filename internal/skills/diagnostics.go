package skills

// Diagnostics は catalog 生成時の診断一覧を返す。
func Diagnostics(catalog SkillCatalog) []Diagnostic {
	out := make([]Diagnostic, len(catalog.Diagnostics))
	copy(out, catalog.Diagnostics)
	return out
}

func newDiagnostic(severity DiagnosticSeverity, code, path, message string) Diagnostic {
	return Diagnostic{
		Severity: severity,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}

// HasWarnings は warning 診断の有無を返す。
func HasWarnings(catalog SkillCatalog) bool {
	for _, diag := range catalog.Diagnostics {
		if diag.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// HasErrors は error 診断の有無を返す。
func HasErrors(catalog SkillCatalog) bool {
	for _, diag := range catalog.Diagnostics {
		if diag.Severity == SeverityError {
			return true
		}
	}
	return false
}
