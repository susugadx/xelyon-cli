package commanddocs

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

var commandHeadingTokenRE = regexp.MustCompile("`?(/[a-z][a-z0-9_-]*)`?")

// findExistingCommands は docs/commands.md の見出しから documented な slash command token を抽出する。
func findExistingCommands(content string) map[string]bool {
	existing := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		for _, match := range commandHeadingTokenRE.FindAllStringSubmatch(line, -1) {
			if len(match) >= 2 {
				existing[match[1]] = true
			}
		}
	}

	return existing
}

// missingCommands は command catalog のうち docs に見出しがない command を返す。
func missingCommands(commands []commandcatalog.CommandInfo, documented map[string]bool) []commandcatalog.CommandInfo {
	var missing []commandcatalog.CommandInfo
	for _, cmd := range commands {
		if cmd.HiddenFromHelp {
			continue
		}
		if commandDocumented(cmd, documented) {
			continue
		}
		missing = append(missing, cmd)
	}
	return missing
}

// renderMissingCommandSkeleton は未記載 command の追記用 Markdown 骨格を生成する。
func renderMissingCommandSkeleton(commands []commandcatalog.CommandInfo) string {
	if len(commands) == 0 {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("\n## 未ドキュメント化コマンド（自動追加）\n\n")
	buf.WriteString("<!-- TODO: 以下のコマンドに詳細な説明を追加してください -->\n\n")

	for _, cmd := range commands {
		header := cmd.Name
		if len(cmd.Aliases) > 0 {
			header += ", " + strings.Join(cmd.Aliases, ", ")
		}
		fmt.Fprintf(&buf, "### `%s`\n\n", header)
		fmt.Fprintf(&buf, "%s\n\n", cmd.Description)

		usage := cmd.Name
		if cmd.Args != "" {
			usage += " " + cmd.Args
		}
		buf.WriteString("```\n")
		fmt.Fprintf(&buf, "> %s\n", usage)
		buf.WriteString("```\n\n")

		if len(cmd.SubCommands) > 0 {
			buf.WriteString("**サブコマンド:**\n\n")
			for _, sub := range cmd.SubCommands {
				fmt.Fprintf(&buf, "- `%s` - %s\n", sub.Name, sub.Description)
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// AppendMissingCommandSkeleton は docs content の末尾へ不足 command の skeleton を追記する。
func AppendMissingCommandSkeleton(content string, commands []commandcatalog.CommandInfo) (string, []commandcatalog.CommandInfo) {
	existingCommands := findExistingCommands(content)
	missingCommands := missingCommands(commands, existingCommands)
	if len(missingCommands) == 0 {
		return content, nil
	}
	return content + renderMissingCommandSkeleton(missingCommands), missingCommands
}

func commandDocumented(cmd commandcatalog.CommandInfo, documented map[string]bool) bool {
	if documented[cmd.Name] {
		return true
	}
	for _, alias := range cmd.Aliases {
		if documented[alias] {
			return true
		}
	}
	return false
}
