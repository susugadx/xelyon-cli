package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type stringMapSnapshot struct {
	entries map[string]string
	keys    []string
}

func buildStringMapSnapshot(entries map[string]string) stringMapSnapshot {
	copied := make(map[string]string, len(entries))
	keys := make([]string, 0, len(entries))
	for key, value := range entries {
		copied[key] = value
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return stringMapSnapshot{
		entries: copied,
		keys:    keys,
	}
}

// Run は map[string]string 編集UIを表示し、編集結果を返す。
func (e *StringMapEditor) Run() (map[string]string, bool, error) {
	ctx := newConfigPromptContext(e.Runtime)
	promptIO := ctx.promptIO
	out := ctx.out

	result := make(map[string]string)
	for k, v := range e.Current {
		result[k] = v
	}

	for {
		snapshot := buildStringMapSnapshot(result)
		e.renderStringMapMenu(out, snapshot)

		input := readConfigEditorChoice(&promptIO)
		done, saved := e.handleStringMapChoice(result, &promptIO, out, snapshot, input)
		if done {
			if !saved {
				return nil, false, nil
			}
			return result, true, nil
		}
	}
}

func (e *StringMapEditor) renderStringMapMenu(out io.Writer, snapshot stringMapSnapshot) {
	_, _ = fmt.Fprintf(out, "\n%s── %s ───────────────────────────────────%s\n\n", colorCyan, e.Path, colorReset)
	_, _ = fmt.Fprintln(out, "  Current entries:")

	if len(snapshot.keys) == 0 {
		_, _ = fmt.Fprintf(out, "    %s(empty)%s\n", colorDim, colorReset)
	} else {
		for i, key := range snapshot.keys {
			value := snapshot.entries[key]
			_, _ = fmt.Fprintf(out, "    %d. %s → %s\n", i+1, key, truncateString(value, 25))
		}
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  [a] Add entry")
	_, _ = fmt.Fprintln(out, "  [e] Edit entry (enter number)")
	_, _ = fmt.Fprintln(out, "  [d] Delete entry (enter number)")
	_, _ = fmt.Fprintln(out, "  [s] Save and back")
	_, _ = fmt.Fprintln(out, "  [c] Cancel (discard changes)")
	_, _ = fmt.Fprintf(out, "\n%sChoice:%s ", colorCyan, colorReset)
}

func (e *StringMapEditor) handleStringMapChoice(result map[string]string, promptIO *PromptIO, out io.Writer, snapshot stringMapSnapshot, input string) (done bool, saved bool) {
	switch input {
	case "a", "add":
		e.addStringMapEntry(result, promptIO, out)
	case "e", "edit":
		e.editStringMapEntry(result, promptIO, out, snapshot)
	case "d", "delete":
		e.deleteStringMapEntry(result, promptIO, out, snapshot)
	case "s", "save":
		return true, true
	case "c", "cancel":
		return true, false
	default:
		_, _ = fmt.Fprintf(out, "%sUnknown command. Use a/e/d/s/c%s\n", colorDim, colorReset)
	}
	return false, false
}

func (e *StringMapEditor) addStringMapEntry(result map[string]string, promptIO *PromptIO, out io.Writer) {
	_, _ = fmt.Fprint(out, "Enter key: ")
	key := strings.TrimSpace(readLineWithIO(promptIO))
	if key == "" {
		_, _ = fmt.Fprintf(out, "%sKey cannot be empty%s\n", colorDim, colorReset)
		return
	}

	_, _ = fmt.Fprint(out, "Enter value: ")
	value := strings.TrimSpace(readLineWithIO(promptIO))
	result[key] = value
	_, _ = fmt.Fprintf(out, "%s✓ Added: %s → %s%s\n", colorGreen, key, value, colorReset)
}

func (e *StringMapEditor) editStringMapEntry(result map[string]string, promptIO *PromptIO, out io.Writer, snapshot stringMapSnapshot) {
	if len(snapshot.keys) == 0 {
		_, _ = fmt.Fprintf(out, "%sNo entries to edit%s\n", colorDim, colorReset)
		return
	}

	_, _ = fmt.Fprintf(out, "Enter number to edit (1-%d): ", len(snapshot.keys))
	numStr := readLineWithIO(promptIO)
	idx, ok := parseConfigEditorIndex(numStr, len(snapshot.keys))
	if !ok {
		_, _ = fmt.Fprintf(out, "%sInvalid number%s\n", colorDim, colorReset)
		return
	}

	key := snapshot.keys[idx]
	_, _ = fmt.Fprintf(out, "Enter new value for '%s' (current: %s): ", key, snapshot.entries[key])
	newValue := strings.TrimSpace(readLineWithIO(promptIO))
	if newValue == "" {
		return
	}

	result[key] = newValue
	_, _ = fmt.Fprintf(out, "%s✓ Updated: %s → %s%s\n", colorGreen, key, newValue, colorReset)
}

func (e *StringMapEditor) deleteStringMapEntry(result map[string]string, promptIO *PromptIO, out io.Writer, snapshot stringMapSnapshot) {
	if len(snapshot.keys) == 0 {
		_, _ = fmt.Fprintf(out, "%sNo entries to delete%s\n", colorDim, colorReset)
		return
	}

	_, _ = fmt.Fprintf(out, "Enter number to delete (1-%d): ", len(snapshot.keys))
	numStr := readLineWithIO(promptIO)
	idx, ok := parseConfigEditorIndex(numStr, len(snapshot.keys))
	if !ok {
		_, _ = fmt.Fprintf(out, "%sInvalid number%s\n", colorDim, colorReset)
		return
	}

	key := snapshot.keys[idx]
	delete(result, key)
	_, _ = fmt.Fprintf(out, "%s✓ Deleted: %s%s\n", colorGreen, key, colorReset)
}
