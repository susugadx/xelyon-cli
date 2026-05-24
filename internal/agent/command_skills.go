package agent

import (
	"fmt"
	"strings"

	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

func handleSkillsCommand(agent *Agent, args []string) bool {
	out := agent.output()
	catalog := agent.loadSkillCatalog()

	subcommand := ""
	if len(args) > 0 {
		subcommand = strings.ToLower(strings.TrimSpace(args[0]))
	}

	if subcommand == "" || subcommand == "overview" || subcommand == "list" {
		printSkillsOverview(out, catalog)
		return true
	}

	switch subcommand {
	case "show":
		if len(args) < 2 {
			yellow.Fprintln(out, "Usage: /skills show <name>")
			return true
		}
		name := strings.TrimSpace(strings.Join(args[1:], " "))
		activated, err := agentskills.Activate(catalog, name)
		if err != nil {
			red.Fprintf(out, "❌ %v\n", err)
			return true
		}
		printSkillDetail(out, activated.Skill)
		return true

	case "doctor":
		printCommandHeaderToWriter(out, "Skills Doctor")
		_, _ = fmt.Fprint(out, agentskills.FormatDoctorReport(catalog))
		return true

	default:
		yellow.Fprintln(out, "Usage: /skills [overview|show <name>|doctor]")
		dim.Fprintln(out, "       /skills list is accepted as an alias for overview")
		return true
	}
}
