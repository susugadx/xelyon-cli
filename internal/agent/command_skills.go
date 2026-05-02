package agent

import (
	"fmt"
	"strings"

	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
)

func handleSkillsCommand(agent *Agent, args []string) bool {
	out := agent.output()
	catalog := agent.loadSkillCatalog()

	if len(args) == 0 || args[0] == "list" {
		printCommandHeaderToWriter(out, "Agent Skills")
		_, _ = fmt.Fprintf(out, "Discovered skills: %d\n", len(catalog.Skills))
		_, _ = fmt.Fprint(out, agentskills.FormatCatalogList(catalog))
		if len(catalog.Diagnostics) > 0 {
			yellow.Fprintf(out, "\n⚠️  Diagnostics found: %d (run /skills doctor)\n", len(catalog.Diagnostics))
		}
		return true
	}

	subcommand := strings.ToLower(strings.TrimSpace(args[0]))
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
		_, _ = fmt.Fprintln(out, activated.Content)
		return true

	case "doctor":
		printCommandHeaderToWriter(out, "Skills Doctor")
		_, _ = fmt.Fprint(out, agentskills.FormatDoctorReport(catalog))
		return true

	default:
		yellow.Fprintln(out, "Usage: /skills [list|show <name>|doctor]")
		return true
	}
}
