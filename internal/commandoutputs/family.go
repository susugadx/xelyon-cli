package commandoutputs

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
)

type commandFamily string

const (
	commandFamilyUnknown     commandFamily = ""
	commandFamilyValidation  commandFamily = "validation"
	commandFamilyObservation commandFamily = "observation"
	commandFamilyFileDump    commandFamily = "file_dump"
	commandFamilyGitStatus   commandFamily = "git_status"
	commandFamilyGitDiff     commandFamily = "git_diff"
	commandFamilyGitShow     commandFamily = "git_show"
	commandFamilyGitLog      commandFamily = "git_log"
	commandFamilyGitBranch   commandFamily = "git_branch"
	commandFamilyGitFileList commandFamily = "git_file_list"
	commandFamilySensitive   commandFamily = "sensitive"
	commandFamilyPackage     commandFamily = "package"
	commandFamilyNetwork     commandFamily = "network"
	commandFamilyDeploy      commandFamily = "deploy"
	commandFamilyDatabase    commandFamily = "database"
)

func classifyCommandFamily(command string) commandFamily {
	if commandHasShellComposition(command) {
		return commandFamilyUnknown
	}
	words := commandWords(command)
	if len(words) == 0 {
		return commandFamilyUnknown
	}
	head := wordBase(words[0])
	second := wordAt(words, 1)
	third := wordAt(words, 2)

	if head == "git" {
		switch second {
		case "status":
			return commandFamilyGitStatus
		case "diff":
			return commandFamilyGitDiff
		case "show":
			return commandFamilyGitShow
		case "log":
			return commandFamilyGitLog
		case "branch":
			return commandFamilyGitBranch
		case "ls-files":
			return commandFamilyGitFileList
		case "grep":
			return commandFamilyObservation
		case "config":
			return commandFamilySensitive
		case "remote":
			if containsWord(words, "-v") || containsWord(words, "--verbose") {
				return commandFamilySensitive
			}
		}
	}

	switch {
	case isValidationCommand(head, second, third):
		return commandFamilyValidation
	case isObservationCommand(head):
		return commandFamilyObservation
	case isFileDumpCommand(head):
		return commandFamilyFileDump
	case isSensitiveCommand(head, second, third):
		return commandFamilySensitive
	case isPackageCommand(head, second, third):
		return commandFamilyPackage
	case isDeployCommand(head, second, third):
		return commandFamilyDeploy
	case isNetworkCommand(head, second):
		return commandFamilyNetwork
	case isDatabaseCommand(head, second, third):
		return commandFamilyDatabase
	default:
		return commandFamilyUnknown
	}
}

func commandWords(command string) []string {
	parts, status := commandruntime.SplitStrict(command)
	if !status.IsOK() {
		return nil
	}
	words := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		words = append(words, part)
	}
	for len(words) > 0 && looksLikeEnvAssignment(words[0]) {
		words = words[1:]
	}
	return words
}

func isValidationCommand(head, second, third string) bool {
	switch {
	case head == "go" && (second == "test" || second == "build" || second == "vet" || second == "fmt"):
		return true
	case head == "cargo" && (second == "test" || second == "build" || second == "clippy" || second == "check"):
		return true
	case head == "pytest" || head == "mypy" || head == "tsc" || head == "golangci-lint":
		return true
	case head == "ruff" && second == "check":
		return true
	case head == "eslint":
		return true
	case head == "npx" && (second == "eslint" || second == "ruff" || second == "tsc"):
		return true
	case head == "npm" || head == "pnpm" || head == "yarn":
		return second == "test" || second == "t" || second == "lint" || second == "build" || second == "typecheck" ||
			second == "run" && (third == "test" || third == "lint" || third == "build" || third == "typecheck")
	case head == "make":
		return second == "test" || second == "build" || second == "lint" || second == "typecheck" || second == "ci-check"
	default:
		return false
	}
}

func isObservationCommand(head string) bool {
	switch head {
	case "rg", "grep", "find", "ls", "wc", "tree", "fd":
		return true
	default:
		return false
	}
}

func isFileDumpCommand(head string) bool {
	switch head {
	case "cat", "sed", "head", "tail", "bat":
		return true
	default:
		return false
	}
}

func isSensitiveCommand(head, second, third string) bool {
	switch head {
	case "env", "printenv", "set", "op":
		return true
	case "gh":
		return second == "auth"
	case "npm", "pnpm", "yarn":
		return second == "config"
	case "docker":
		return second == "info" || second == "context" || second == "config"
	case "kubectl":
		return second == "config" || second == "cluster-info"
	case "aws", "gcloud", "az":
		return second == "config" || second == "auth" || third == "config"
	default:
		return false
	}
}

func isPackageCommand(head, second, third string) bool {
	switch head {
	case "npm", "pnpm", "yarn":
		return second == "install" || second == "i" || second == "add" || second == "ci"
	case "cargo":
		return second == "fetch"
	case "go":
		return second == "mod" && third == "download"
	case "pip", "pip3":
		return second == "install"
	default:
		return false
	}
}

func isNetworkCommand(head, second string) bool {
	switch head {
	case "curl", "wget":
		return true
	case "gh":
		return second == "release"
	default:
		return false
	}
}

func isDeployCommand(head, second, third string) bool {
	switch head {
	case "npm":
		return second == "publish"
	case "vercel", "firebase":
		return second == "deploy" || second == ""
	case "gh":
		return second == "release" && third == "upload"
	default:
		return false
	}
}

func isDatabaseCommand(head, second, third string) bool {
	switch head {
	case "psql", "mysql", "sqlite3", "mongosh", "pg_dump", "mysqldump":
		return true
	case "prisma":
		return second == "db" || second == "migrate"
	case "npx":
		return second == "prisma" && (third == "db" || third == "migrate")
	default:
		return false
	}
}

func wordAt(words []string, index int) string {
	if index < 0 || index >= len(words) {
		return ""
	}
	return wordBase(words[index])
}

func wordBase(word string) string {
	word = strings.TrimSpace(word)
	if idx := strings.LastIndexAny(word, `/\`); idx >= 0 && idx+1 < len(word) {
		return word[idx+1:]
	}
	return word
}

func containsWord(words []string, want string) bool {
	for _, word := range words {
		if word == want {
			return true
		}
	}
	return false
}

func looksLikeEnvAssignment(word string) bool {
	eq := strings.IndexByte(word, '=')
	if eq <= 0 {
		return false
	}
	for _, r := range word[:eq] {
		if r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func commandHasShellComposition(command string) bool {
	quoteChar := rune(0)
	runes := []rune(command)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quoteChar == '\'' {
			if r == '\'' {
				quoteChar = 0
			}
			continue
		}
		if quoteChar == '"' {
			if r == '\\' {
				i++
				continue
			}
			switch r {
			case '"':
				quoteChar = 0
			case '`':
				return true
			case '$':
				if i+1 < len(runes) && runes[i+1] == '(' {
					return true
				}
			}
			continue
		}
		switch r {
		case '\'', '"':
			quoteChar = r
		case '\n', '\r', ';', '|', '&', '<', '>', '`':
			return true
		case '$':
			if i+1 < len(runes) && runes[i+1] == '(' {
				return true
			}
		}
	}
	return false
}
