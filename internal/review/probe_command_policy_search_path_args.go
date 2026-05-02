package review

import "strings"

type searchCommandPatternSpec struct {
	regexpOptionShort string
	regexpOptionLong  string
	fileOptionShort   string
	fileOptionLong    string

	shortOptionsWithValue      map[byte]struct{}
	longOptionsWithValue       map[string]struct{}
	longOptionsPathOnlyMode    map[string]struct{}
	recursiveFlags             map[string]struct{}
	recursiveLongOptionValues  map[string]string
	recursiveShortOptionValues map[byte]string
}

type searchPatternOptionKind int

const (
	searchPatternOptionNone searchPatternOptionKind = iota
	searchPatternOptionRegexp
	searchPatternOptionFile
)

type parsedSearchPatternOption struct {
	kind          searchPatternOptionKind
	consumesNext  bool
	attachedValue string
}

type parsedSearchGenericOption struct {
	matched                   bool
	consumesNext              bool
	forceAllPositionalsAsPath bool
	isRecursiveFlag           bool
}

type searchCommandParseState struct {
	explicitPattern           bool
	forceAllPositionalsAsPath bool
	hasRecursiveFlag          bool
	positionals               []string
	pathArgs                  []string
}

var searchCommandPatternSpecs = map[string]searchCommandPatternSpec{
	"rg": {
		regexpOptionShort: "-e",
		regexpOptionLong:  "--regexp",
		fileOptionShort:   "-f",
		fileOptionLong:    "--file",
		shortOptionsWithValue: byteSet(
			'A', 'B', 'C', 'd', 'E', 'f', 'g', 'j', 'M', 'm', 'r', 't', 'T',
		),
		longOptionsWithValue: stringSet(
			"--after-context",
			"--before-context",
			"--colors",
			"--context",
			"--context-separator",
			"--dfa-size-limit",
			"--encoding",
			"--engine",
			"--field-context-separator",
			"--file",
			"--glob",
			"--iglob",
			"--ignore-file",
			"--max-columns",
			"--max-count",
			"--max-depth",
			"--max-filesize",
			"--path-separator",
			"--pre-glob",
			"--regex-size-limit",
			"--regexp",
			"--replace",
			"--sort",
			"--sortr",
			"--threads",
			"--type",
			"--type-add",
			"--type-clear",
			"--type-not",
		),
		longOptionsPathOnlyMode: stringSet(
			"--files",
		),
	},
	"grep": {
		regexpOptionShort: "-e",
		regexpOptionLong:  "--regexp",
		fileOptionShort:   "-f",
		fileOptionLong:    "--file",
		shortOptionsWithValue: byteSet(
			'A', 'B', 'C', 'D', 'd', 'e', 'f', 'm',
		),
		longOptionsWithValue: stringSet(
			"--after-context",
			"--before-context",
			"--binary-files",
			"--context",
			"--devices",
			"--directories",
			"--exclude",
			"--exclude-dir",
			"--exclude-from",
			"--file",
			"--include",
			"--label",
			"--max-count",
			"--regexp",
		),
		recursiveFlags: stringSet(
			"-r",
			"-R",
			"--recursive",
			"--dereference-recursive",
		),
		recursiveLongOptionValues: map[string]string{
			"--directories": "recurse",
		},
		recursiveShortOptionValues: map[byte]string{
			'd': "recurse",
		},
	},
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

	if parsed.consumesNext {
		if *i+1 < len(args) {
			*i++
		}
		return
	}
}

func finalizeSearchCommandPathArgs(state searchCommandParseState) []string {
	pathArgs := append([]string(nil), state.pathArgs...)

	switch {
	case state.forceAllPositionalsAsPath:
		pathArgs = append(pathArgs, state.positionals...)
	case state.explicitPattern:
		pathArgs = append(pathArgs, state.positionals...)
	case state.hasRecursiveFlag && len(state.positionals) >= 2:
		// GNU grep 互換の実用上、再帰モードでは末尾を pattern とみなし、それ以前を探索対象 path として扱う。
		pathArgs = append(pathArgs, state.positionals[:len(state.positionals)-1]...)
	default:
		if len(state.positionals) >= 2 {
			pathArgs = append(pathArgs, state.positionals[1:]...)
		}
	}

	return pathArgs
}

func parseSearchPatternOption(arg string, spec searchCommandPatternSpec) parsedSearchPatternOption {
	if arg == spec.regexpOptionShort || arg == spec.regexpOptionLong {
		return parsedSearchPatternOption{
			kind:         searchPatternOptionRegexp,
			consumesNext: true,
		}
	}
	if strings.HasPrefix(arg, spec.regexpOptionLong+"=") {
		return parsedSearchPatternOption{
			kind: searchPatternOptionRegexp,
		}
	}
	if strings.HasPrefix(arg, spec.regexpOptionShort) && arg != spec.regexpOptionShort {
		return parsedSearchPatternOption{
			kind: searchPatternOptionRegexp,
		}
	}

	if arg == spec.fileOptionShort || arg == spec.fileOptionLong {
		return parsedSearchPatternOption{
			kind:         searchPatternOptionFile,
			consumesNext: true,
		}
	}
	if strings.HasPrefix(arg, spec.fileOptionLong+"=") {
		return parsedSearchPatternOption{
			kind:          searchPatternOptionFile,
			attachedValue: strings.TrimPrefix(arg, spec.fileOptionLong+"="),
		}
	}
	if strings.HasPrefix(arg, spec.fileOptionShort) && arg != spec.fileOptionShort {
		return parsedSearchPatternOption{
			kind:          searchPatternOptionFile,
			attachedValue: strings.TrimPrefix(arg, spec.fileOptionShort),
		}
	}

	return parsedSearchPatternOption{}
}

func parseSearchGenericOption(arg string, spec searchCommandPatternSpec) parsedSearchGenericOption {
	if arg == "-" || !strings.HasPrefix(arg, "-") {
		return parsedSearchGenericOption{}
	}

	if strings.HasPrefix(arg, "--") {
		if arg == "--" {
			return parsedSearchGenericOption{}
		}

		optionName, hasValue := splitLongOption(arg)
		if _, ok := spec.longOptionsPathOnlyMode[optionName]; ok {
			return parsedSearchGenericOption{
				matched:                   true,
				forceAllPositionalsAsPath: true,
			}
		}
		if _, ok := spec.recursiveFlags[optionName]; ok {
			return parsedSearchGenericOption{
				matched:         true,
				isRecursiveFlag: true,
			}
		}
		if hasValue {
			if want, ok := spec.recursiveLongOptionValues[optionName]; ok && strings.HasSuffix(arg, "="+want) {
				return parsedSearchGenericOption{
					matched:         true,
					isRecursiveFlag: true,
				}
			}
			return parsedSearchGenericOption{
				matched: true,
			}
		}

		_, consumesNext := spec.longOptionsWithValue[optionName]
		return parsedSearchGenericOption{
			matched:      true,
			consumesNext: consumesNext,
		}
	}

	result := parsedSearchGenericOption{
		matched: true,
	}

	for idx := 1; idx < len(arg); idx++ {
		ch := arg[idx]
		shortName := "-" + string(ch)
		if _, ok := spec.recursiveFlags[shortName]; ok {
			result.isRecursiveFlag = true
		}
		if _, ok := spec.shortOptionsWithValue[ch]; !ok {
			continue
		}
		if want, ok := spec.recursiveShortOptionValues[ch]; ok && idx+1 < len(arg) && arg[idx+1:] == want {
			result.isRecursiveFlag = true
		}

		if idx+1 < len(arg) {
			return result
		}
		result.consumesNext = true
		return result
	}

	return result
}

func splitLongOption(arg string) (name string, hasValue bool) {
	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		return arg[:idx], true
	}
	return arg, false
}

func byteSet(values ...byte) map[byte]struct{} {
	m := make(map[byte]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}

func stringSet(values ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}
