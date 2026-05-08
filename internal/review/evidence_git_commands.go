package review

func reviewStatusShortGitArgs() []string {
	return []string{"status", "--short", "--untracked-files=all", "--renames"}
}

func reviewUntrackedListGitArgs() []string {
	return []string{"ls-files", "-z", "--others", "--exclude-standard"}
}

func reviewDiffBaseGitArgs(staged bool) []string {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--no-ext-diff", "--no-textconv")
	return args
}

func reviewDiffMetadataGitArgs(staged bool, suffix ...string) []string {
	args := reviewDiffBaseGitArgs(staged)
	args = append(args, suffix...)
	return args
}

func reviewDiffBodyGitArgs(staged bool) []string {
	args := reviewDiffBaseGitArgs(staged)
	return append(args, "--unified=3")
}
