package commandcatalog

import (
	"fmt"
	"regexp"
	"strings"
)

var commandTokenPattern = regexp.MustCompile(`^/[a-z0-9][a-z0-9-]*$`)

// ValidateCommands は catalog 定義の整合性を検証する。
func ValidateCommands(commands []CommandInfo) error {
	var problems []string
	seen := make(map[string]string)
	discoverableWeights := make(map[CommandSurface]map[int]string)

	for i, cmd := range commands {
		name := strings.TrimSpace(cmd.Name)
		if name == "" {
			problems = append(problems, fmt.Sprintf("commands[%d]: empty command name", i))
			continue
		}
		if !strings.HasPrefix(name, "/") {
			problems = append(problems, fmt.Sprintf("%s: command name must start with '/'", name))
		}
		if !isValidCommandToken(name) {
			problems = append(problems, fmt.Sprintf("%s: command name must match %s", name, commandTokenPattern.String()))
		}
		if len(cmd.Surfaces) == 0 {
			problems = append(problems, fmt.Sprintf("%s: surfaces must be explicitly declared", name))
		}

		checkCommandTokenDuplication(name, cmd.Name, seen, &problems)
		for _, alias := range cmd.Aliases {
			if strings.TrimSpace(alias) == "" {
				problems = append(problems, fmt.Sprintf("%s: empty alias is not allowed", name))
				continue
			}
			if !strings.HasPrefix(alias, "/") {
				problems = append(problems, fmt.Sprintf("%s: alias %q must start with '/'", name, alias))
			}
			if !isValidCommandToken(alias) {
				problems = append(problems, fmt.Sprintf("%s: alias %q must match %s", name, alias, commandTokenPattern.String()))
			}
			checkCommandTokenDuplication(alias, cmd.Name, seen, &problems)
		}

		if cmd.EffectiveTUILocalAction() != TUILocalActionNone && !cmd.SupportsSurface(CommandSurfaceTUI) {
			problems = append(problems, fmt.Sprintf("%s: TUI local action requires TUI surface", name))
		}
		if cmd.EffectiveOwner() == CommandOwnerTUIRouter && cmd.EffectiveTUILocalAction() == TUILocalActionNone {
			problems = append(problems, fmt.Sprintf("%s: TUI router owned command must declare TUI local action", name))
		}
		if !isValidTUILocalArgPolicy(cmd.TUILocalArgs) {
			problems = append(problems, fmt.Sprintf("%s: invalid TUI local arg policy %q", name, cmd.TUILocalArgs))
		}
		if !cmd.SupportsSurface(CommandSurfaceTUI) && cmd.Discoverable {
			problems = append(problems, fmt.Sprintf("%s: non-TUI command should not be discoverable", name))
		}
		if cmd.Discoverable {
			checkDiscoverableSortWeightUniqueness(cmd, discoverableWeights, &problems)
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("invalid command catalog:\n- %s", strings.Join(problems, "\n- "))
}

func checkCommandTokenDuplication(token, owner string, seen map[string]string, problems *[]string) {
	if existing, ok := seen[token]; ok {
		*problems = append(*problems, fmt.Sprintf("duplicate command token %q used by %s and %s", token, existing, owner))
		return
	}
	seen[token] = owner
}

func isValidTUILocalArgPolicy(policy TUILocalArgPolicy) bool {
	switch policy {
	case "", TUILocalArgBareOnly, TUILocalArgAllowAny:
		return true
	default:
		return false
	}
}

func isValidCommandToken(token string) bool {
	return commandTokenPattern.MatchString(token)
}

func checkDiscoverableSortWeightUniqueness(cmd CommandInfo, discoverableWeights map[CommandSurface]map[int]string, problems *[]string) {
	weight := cmd.EffectiveSortWeight()
	for _, surface := range cmd.effectiveSurfaces() {
		weightsBySurface, ok := discoverableWeights[surface]
		if !ok {
			weightsBySurface = make(map[int]string)
			discoverableWeights[surface] = weightsBySurface
		}
		if existing, duplicated := weightsBySurface[weight]; duplicated {
			*problems = append(*problems, fmt.Sprintf(
				"%s: duplicate discoverable sort weight %d on surface %s (already used by %s)",
				cmd.Name, weight, surface, existing,
			))
			continue
		}
		weightsBySurface[weight] = cmd.Name
	}
}
