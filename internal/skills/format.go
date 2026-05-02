package skills

import (
	"fmt"
	"strings"
)

// FormatCatalogList は /skills list 向けの表示文字列を返す。
func FormatCatalogList(catalog SkillCatalog) string {
	var b strings.Builder
	if len(catalog.Skills) == 0 {
		b.WriteString("No skills found.\n")
		return b.String()
	}
	for _, skill := range catalog.Skills {
		fmt.Fprintf(&b, "- %s: %s\n", skill.Name, skill.Description)
	}
	return b.String()
}

// FormatDoctorReport は /skills doctor 向けの診断文字列を返す。
func FormatDoctorReport(catalog SkillCatalog) string {
	if len(catalog.Diagnostics) == 0 {
		return "No diagnostics. Skills catalog is healthy.\n"
	}

	var b strings.Builder
	for _, diag := range catalog.Diagnostics {
		fmt.Fprintf(&b, "- [%s] %s", strings.ToUpper(string(diag.Severity)), diag.Code)
		if strings.TrimSpace(diag.Path) != "" {
			fmt.Fprintf(&b, " (%s)", diag.Path)
		}
		fmt.Fprintf(&b, ": %s\n", diag.Message)
	}
	return b.String()
}
