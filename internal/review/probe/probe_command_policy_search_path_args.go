package probe

type searchCommandParseState struct {
	explicitPattern           bool
	forceAllPositionalsAsPath bool
	hasRecursiveFlag          bool
	positionals               []string
	pathArgs                  []string
}

func collectSearchCommandPathCandidates(command string, args []string) []string {
	spec, ok := searchCommandPatternSpecs[command]
	if !ok {
		return nil
	}
	return collectSearchCommandPathArgs(args, spec)
}

func collectSearchCommandPathArgs(args []string, spec searchCommandPatternSpec) []string {
	if len(args) == 0 {
		return nil
	}

	state := searchCommandParseState{
		positionals: make([]string, 0, len(args)),
		pathArgs:    make([]string, 0, len(args)),
	}

	optionsTerminated := false
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if !optionsTerminated && arg == "--" {
			optionsTerminated = true
			continue
		}

		if optionsTerminated {
			state.positionals = append(state.positionals, arg)
			continue
		}

		if consumeSearchCommandOption(&state, arg, args, &i, spec) {
			continue
		}

		state.positionals = append(state.positionals, arg)
	}

	return finalizeSearchCommandPathArgs(state)
}

func consumeSearchCommandOption(state *searchCommandParseState, arg string, args []string, i *int, spec searchCommandPatternSpec) bool {
	if parsed := parseSearchPatternOption(arg, spec); parsed.kind != searchPatternOptionNone {
		applySearchPatternOption(state, parsed, args, i)
		return true
	}

	if parsed := parseSearchGenericOption(arg, spec); parsed.matched {
		applySearchGenericOption(state, parsed, args, i)
		return true
	}

	return false
}

func applySearchPatternOption(state *searchCommandParseState, parsed parsedSearchPatternOption, args []string, i *int) {
	state.explicitPattern = true

	switch parsed.kind {
	case searchPatternOptionRegexp:
		if parsed.consumesNext && *i+1 < len(args) {
			*i++
		}
	case searchPatternOptionFile:
		if parsed.consumesNext {
			if *i+1 < len(args) {
				state.pathArgs = append(state.pathArgs, args[*i+1])
				*i++
			}
			return
		}
		state.pathArgs = append(state.pathArgs, parsed.attachedValue)
	}
}

func applySearchGenericOption(state *searchCommandParseState, parsed parsedSearchGenericOption, args []string, i *int) {
	if parsed.forceAllPositionalsAsPath {
		state.forceAllPositionalsAsPath = true
	}
	if parsed.isRecursiveFlag {
		state.hasRecursiveFlag = true
	}

	switch parsed.valueKind {
	case searchGenericOptionAttachedPath:
		state.pathArgs = append(state.pathArgs, parsed.attachedPathValue)
	case searchGenericOptionConsumesNext:
		if *i+1 < len(args) {
			*i++
		}
	case searchGenericOptionConsumesNextPath:
		if *i+1 < len(args) {
			state.pathArgs = append(state.pathArgs, args[*i+1])
			*i++
		}
	}
}
