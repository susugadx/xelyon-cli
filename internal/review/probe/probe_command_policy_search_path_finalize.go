package probe

func finalizeSearchCommandPathArgs(state searchCommandParseState) []string {
	pathArgs := append([]string(nil), state.pathArgs...)

	switch {
	case state.forceAllPositionalsAsPath:
		pathArgs = append(pathArgs, state.positionals...)
	case state.explicitPattern:
		pathArgs = append(pathArgs, state.positionals...)
	default:
		if len(state.positionals) >= 2 {
			pathArgs = append(pathArgs, state.positionals[1:]...)
		}
	}

	return pathArgs
}
