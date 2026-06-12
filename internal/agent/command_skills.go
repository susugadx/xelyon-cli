package agent

import (
	"fmt"
	"strings"

	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/skills/doctor"
	"github.com/susugadx/xelyon-cli/internal/skills/router"
	"github.com/susugadx/xelyon-cli/internal/skills/usageledger"
)

const (
	skillsUsageCommandUsage                = "Usage: /skills usage [clear [--all]]"
	skillUsageLedgerRootUnavailableMessage = "No project root found; current repo skill routing usage ledger is unavailable."
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
			red.Fprintf(out, "❌ %s\n", agentskills.SanitizePromptLineValue(err.Error()))
			return true
		}
		printSkillDetail(out, activated.Skill)
		return true

	case "suggest":
		if len(args) < 2 {
			yellow.Fprintln(out, "Usage: /skills suggest <text>")
			return true
		}
		taskText := strings.TrimSpace(strings.Join(args[1:], " "))
		input := agent.skillRouterInput(agent.currentRequestContext(), taskText, "")
		rec := router.Recommend(catalog, input)
		_, _ = fmt.Fprint(out, router.FormatSuggestReport(rec))
		return true

	case "doctor":
		routingMode := hasSkillCommandFlag(args[1:], "--routing")
		var usageDiagnostics []agentskills.Diagnostic
		if routingMode {
			if store, ok := agent.skillUsageLedgerStoreForProject(true); ok {
				if summary, err := store.Summary(); err == nil {
					usageDiagnostics = usageledger.Diagnostics(summary)
				} else {
					usageDiagnostics = append(usageDiagnostics, agentskills.Diagnostic{
						Severity: agentskills.SeverityWarning,
						Code:     "usage_ledger_read_failed",
						Message:  err.Error(),
					})
				}
			} else {
				usageDiagnostics = append(usageDiagnostics, agentskills.Diagnostic{
					Severity: agentskills.SeverityInfo,
					Code:     "usage_ledger_project_root_unavailable",
					Message:  skillUsageLedgerRootUnavailableMessage,
				})
			}
		}
		report := doctor.BuildReport(catalog, doctor.Options{
			Routing:               routingMode,
			PromptCatalogMaxItems: promptSkillCatalogMaxEntries,
			AdditionalDiagnostics: usageDiagnostics,
		})
		_, _ = fmt.Fprint(out, doctor.FormatReport(report))
		return true

	case "usage":
		handleSkillsUsageCommand(agent, args[1:])
		return true

	default:
		yellow.Fprintln(out, "Usage: /skills [overview|show <name>|doctor [--routing]|suggest <text>|usage [clear [--all]]]")
		dim.Fprintln(out, "       /skills list is accepted as an alias for overview")
		return true
	}
}

func handleSkillsUsageCommand(agent *Agent, args []string) {
	out := agent.output()
	if len(args) > 0 && strings.ToLower(strings.TrimSpace(args[0])) == "clear" {
		clearAll, ok := parseSkillsUsageClearArgs(args[1:])
		if !ok {
			yellow.Fprintln(out, skillsUsageCommandUsage)
			return
		}
		if clearAll {
			store := agent.skillUsageLedgerStoreForAll(true)
			if err := store.ClearAll(); err != nil {
				red.Fprintf(out, "❌ %v\n", err)
				return
			}
			green.Fprintln(out, "✓ Cleared all skill routing usage ledgers")
			return
		}
		store, ok := agent.skillUsageLedgerStoreForProject(true)
		if !ok {
			yellow.Fprintln(out, skillUsageLedgerRootUnavailableMessage)
			return
		}
		if err := store.Clear(); err != nil {
			red.Fprintf(out, "❌ %v\n", err)
			return
		}
		green.Fprintln(out, "✓ Cleared current repo skill routing usage ledger")
		return
	}
	if len(args) > 0 {
		yellow.Fprintln(out, skillsUsageCommandUsage)
		return
	}
	store, ok := agent.skillUsageLedgerStoreForProject(true)
	if !ok {
		yellow.Fprintln(out, skillUsageLedgerRootUnavailableMessage)
		return
	}
	summary, err := store.Summary()
	if err != nil {
		red.Fprintf(out, "❌ %v\n", err)
		return
	}
	_, _ = fmt.Fprint(out, usageledger.FormatSummary(summary))
}

func parseSkillsUsageClearArgs(args []string) (clearAll bool, ok bool) {
	if len(args) == 0 {
		return false, true
	}
	if len(args) == 1 && strings.EqualFold(strings.TrimSpace(args[0]), "--all") {
		return true, true
	}
	return false, false
}

func hasSkillCommandFlag(args []string, flag string) bool {
	for _, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), flag) {
			return true
		}
	}
	return false
}
