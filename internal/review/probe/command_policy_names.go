package probe

// HostReadOnlyCommandNames は host_readonly で許可する command 名を返す。
func HostReadOnlyCommandNames() []string {
	return commandSpecNames(hostReadOnlyCommandSpecs)
}

// ScratchOnlyCommandNames は scratch_only で許可する command 名を返す。
func ScratchOnlyCommandNames() []string {
	return commandSpecNames(scratchOnlyCommandSpecs)
}

// RepoSandboxCommandNames は repo_sandbox で許可する command 名を返す。
func RepoSandboxCommandNames() []string {
	return commandSpecNames(repoSandboxCommandSpecs)
}

func commandSpecNames[T any](specs map[string]T) []string {
	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	return names
}
