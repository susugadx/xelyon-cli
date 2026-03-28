package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func formatSymbolBundle(bundle *SymbolBundle, reg *locator.Registry, matchedPatterns []string) string {
	if bundle == nil {
		return ""
	}

	var sb strings.Builder
	header := fmt.Sprintf("── %s %s (L%d", bundle.Identity.Kind, bundle.Identity.DisplayName, bundle.Identity.Line)
	if bundle.Identity.EndLine > bundle.Identity.Line {
		header += fmt.Sprintf("-L%d", bundle.Identity.EndLine)
	}
	header += fmt.Sprintf(") in %s", bundle.Identity.File)
	if reg != nil {
		id := reg.Register(locator.Location{
			FilePath: bundle.Identity.File,
			Line:     bundle.Identity.Line,
			EndLine:  bundle.Identity.EndLine,
			Name:     fmt.Sprintf("%s %s", bundle.Identity.Kind, bundle.Identity.DisplayName),
		})
		header += " " + id
	}
	sb.WriteString(header + " ──\n")

	if len(matchedPatterns) > 1 {
		fmt.Fprintf(&sb, "Matched patterns: %s\n", strings.Join(matchedPatterns, ", "))
	}

	sb.WriteString("Definition:\n")
	body := bundle.Definition.Body
	if len(body) == 0 && bundle.Definition.Signature != "" {
		body = []string{fmt.Sprintf("%d: %s", bundle.Definition.Line, bundle.Definition.Signature)}
	}
	for _, line := range body {
		sb.WriteString("  " + line + "\n")
	}

	appendSymbolBundleDiagnostics(&sb, bundle)

	for _, section := range bundle.Sections {
		if len(section.Items) == 0 {
			continue
		}
		if section.More {
			fmt.Fprintf(&sb, "\n%s: %d shown (of %d)\n", section.Title, len(section.Items), section.Total)
		} else {
			fmt.Fprintf(&sb, "\n%s (%d):\n", section.Title, len(section.Items))
		}
		for _, item := range section.Items {
			line := formatSymbolBundleItem(section.Kind, item)
			if reg != nil {
				id := reg.Register(locator.Location{
					FilePath: item.File,
					Line:     item.Line,
					EndLine:  item.EndLine,
					Name:     item.Name,
				})
				line += " " + id
			}
			fmt.Fprintf(&sb, "  - %s\n", line)
		}
		if section.More {
			sb.WriteString("  (+ more available via broader search_code)\n")
		}
	}

	if symbolBundleReferenceCount(bundle) == 0 {
		sb.WriteString("\nNo references found.\n")
	}

	return sb.String()
}

func appendSymbolBundleDiagnostics(sb *strings.Builder, bundle *SymbolBundle) {
	if bundle == nil {
		return
	}

	if bundle.Diagnostics.UpstreamIncomplete {
		sb.WriteString("\nWarning: upstream search may be incomplete.\n")
	} else if bundle.Diagnostics.UpstreamTruncated {
		sb.WriteString("\nNote: upstream results were truncated.\n")
	}

	if bundle.Diagnostics.ResolvedViaLSP {
		sb.WriteString("\nNote: resolved via gopls.\n")
	}
}

func formatSymbolBundleItem(kind string, item SymbolBundleItem) string {
	switch kind {
	case "tests":
		if item.Name != "" {
			return fmt.Sprintf("%s:%d | func %s", item.File, item.Line, item.Name)
		}
	case "callers":
		scope := ""
		if item.Scope != "" && item.Scope != "package-level" {
			scope = " in " + item.Scope
		}
		if item.Snippet != "" {
			return fmt.Sprintf("%s:%d%s | %s", item.File, item.Line, scope, item.Snippet)
		}
		return fmt.Sprintf("%s:%d%s", item.File, item.Line, scope)
	case "implementations":
		if item.Snippet != "" {
			return fmt.Sprintf("%s:%d | %s", item.File, item.Line, item.Snippet)
		}
	}
	if item.Snippet != "" {
		return fmt.Sprintf("%s:%d | %s", item.File, item.Line, item.Snippet)
	}
	return fmt.Sprintf("%s:%d", item.File, item.Line)
}

func symbolBundleReferenceCount(bundle *SymbolBundle) int {
	total := 0
	for _, section := range bundle.Sections {
		total += len(section.Items)
	}
	return total
}
