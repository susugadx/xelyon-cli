package uiconfig

import (
	"fmt"
	"io"
	"strings"
)

type stringSliceSnapshot struct {
	items []string
}

func buildStringSliceSnapshot(items []string) stringSliceSnapshot {
	copied := make([]string, len(items))
	copy(copied, items)
	return stringSliceSnapshot{items: copied}
}

// Run は []string 編集UIを表示し、編集結果を返す。
func (e *StringSliceEditor) Run() ([]string, bool, error) {
	ctx := newConfigPromptContext(e.Runtime)
	promptIO := ctx.promptIO
	out := ctx.out

	result := make([]string, len(e.Current))
	copy(result, e.Current)

	for {
		snapshot := buildStringSliceSnapshot(result)
		e.renderStringSliceMenu(out, snapshot)

		input := readConfigEditorChoice(&promptIO)
		done, saved := e.handleStringSliceChoice(&result, &promptIO, out, snapshot, input)
		if done {
			if !saved {
				return nil, false, nil
			}
			return result, true, nil
		}
	}
}

func (e *StringSliceEditor) renderStringSliceMenu(out io.Writer, snapshot stringSliceSnapshot) {
	_, _ = fmt.Fprintf(out, "\n%s── %s ───────────────────────────────────%s\n\n", colorCyan, e.Path, colorReset)
	_, _ = fmt.Fprintln(out, "  Current items:")

	if len(snapshot.items) == 0 {
		_, _ = fmt.Fprintf(out, "    %s(empty)%s\n", colorDim, colorReset)
	} else {
		for i, item := range snapshot.items {
			_, _ = fmt.Fprintf(out, "    %d. %s\n", i+1, truncateString(item, 40))
		}
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "  [a] Add item")
	_, _ = fmt.Fprintln(out, "  [d] Delete item (enter number)")
	_, _ = fmt.Fprintln(out, "  [s] Save and back")
	_, _ = fmt.Fprintln(out, "  [c] Cancel (discard changes)")
	_, _ = fmt.Fprintf(out, "\n%sChoice:%s ", colorCyan, colorReset)
}

func (e *StringSliceEditor) handleStringSliceChoice(result *[]string, promptIO *PromptIO, out io.Writer, snapshot stringSliceSnapshot, input string) (done bool, saved bool) {
	switch input {
	case "a", "add":
		e.addStringSliceItem(result, promptIO, out)
	case "d", "delete":
		e.deleteStringSliceItem(result, promptIO, out, snapshot)
	case "s", "save":
		return true, true
	case "c", "cancel":
		return true, false
	default:
		_, _ = fmt.Fprintf(out, "%sUnknown command. Use a/d/s/c%s\n", colorDim, colorReset)
	}
	return false, false
}

func (e *StringSliceEditor) addStringSliceItem(result *[]string, promptIO *PromptIO, out io.Writer) {
	_, _ = fmt.Fprint(out, "Enter new item: ")
	newItem := strings.TrimSpace(readLineWithIO(promptIO))
	if newItem == "" {
		return
	}

	*result = append(*result, newItem)
	_, _ = fmt.Fprintf(out, "%s✓ Added: %s%s\n", colorGreen, newItem, colorReset)
}

func (e *StringSliceEditor) deleteStringSliceItem(result *[]string, promptIO *PromptIO, out io.Writer, snapshot stringSliceSnapshot) {
	if len(snapshot.items) == 0 {
		_, _ = fmt.Fprintf(out, "%sNo items to delete%s\n", colorDim, colorReset)
		return
	}

	_, _ = fmt.Fprintf(out, "Enter number to delete (1-%d): ", len(snapshot.items))
	numStr := readLineWithIO(promptIO)
	idx, ok := parseConfigEditorIndex(numStr, len(snapshot.items))
	if !ok {
		_, _ = fmt.Fprintf(out, "%sInvalid number%s\n", colorDim, colorReset)
		return
	}

	deleted := snapshot.items[idx]
	updated := append([]string{}, snapshot.items[:idx]...)
	updated = append(updated, snapshot.items[idx+1:]...)
	*result = updated
	_, _ = fmt.Fprintf(out, "%s✓ Deleted: %s%s\n", colorGreen, deleted, colorReset)
}
