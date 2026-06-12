package skills

import (
	"fmt"
	"strings"
)

// FormatDoctorReport は /skills doctor 向けの診断文字列を返す。
func FormatDoctorReport(catalog SkillCatalog) string {
	if len(catalog.Diagnostics) == 0 {
		return "No diagnostics. Skills catalog is healthy.\n"
	}

	var b strings.Builder
	for _, diag := range catalog.Diagnostics {
		fmt.Fprintf(&b, "- [%s] %s", strings.ToUpper(string(diag.Severity)), diag.Code)
		if path := SanitizePromptLineValue(diag.Path); path != "" {
			fmt.Fprintf(&b, " (%s)", path)
		}
		message := SanitizePromptLineValue(diag.Message)
		if message == "" {
			message = "diagnostic unavailable"
		}
		fmt.Fprintf(&b, ": %s\n", message)
	}
	return b.String()
}
