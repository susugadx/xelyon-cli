package probe

import "strings"

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

type searchGenericOptionValueKind int

const (
	searchGenericOptionNoValue searchGenericOptionValueKind = iota
	searchGenericOptionConsumesNext
	searchGenericOptionConsumesNextPath
	searchGenericOptionAttachedPath
)

type parsedSearchGenericOption struct {
	matched                   bool
	valueKind                 searchGenericOptionValueKind
	attachedPathValue         string
	forceAllPositionalsAsPath bool
	isRecursiveFlag           bool
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

	if parsed, ok := parseClusteredSearchPatternOption(arg, spec); ok {
		return parsed
	}

	return parsedSearchPatternOption{}
}

func parseClusteredSearchPatternOption(arg string, spec searchCommandPatternSpec) (parsedSearchPatternOption, bool) {
	tokens, ok := parseProbeShortOptions(arg, spec.shortOptionsWithValue)
	if !ok {
		return parsedSearchPatternOption{}, false
	}

	if value, attached, consumesNext := probeShortOptionValue(tokens, optionByte(spec.regexpOptionShort)); attached || consumesNext {
		return parsedSearchPatternOption{
			kind:          searchPatternOptionRegexp,
			consumesNext:  consumesNext,
			attachedValue: value,
		}, true
	}
	if value, attached, consumesNext := probeShortOptionValue(tokens, optionByte(spec.fileOptionShort)); attached || consumesNext {
		return parsedSearchPatternOption{
			kind:          searchPatternOptionFile,
			consumesNext:  consumesNext,
			attachedValue: value,
		}, true
	}
	return parsedSearchPatternOption{}, false
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
			if _, ok := spec.longOptionsFileValue[optionName]; ok {
				return parsedSearchGenericOption{
					matched:           true,
					valueKind:         searchGenericOptionAttachedPath,
					attachedPathValue: strings.TrimPrefix(arg, optionName+"="),
				}
			}
			if want, ok := spec.recursiveLongOptionValues[optionName]; ok && strings.HasSuffix(arg, "="+want) {
				return parsedSearchGenericOption{
					matched:         true,
					isRecursiveFlag: true,
				}
			}
			return parsedSearchGenericOption{matched: true}
		}

		if _, ok := spec.longOptionsFileValue[optionName]; ok {
			return parsedSearchGenericOption{
				matched:   true,
				valueKind: searchGenericOptionConsumesNextPath,
			}
		}

		_, consumesNext := spec.longOptionsWithValue[optionName]
		if consumesNext {
			return parsedSearchGenericOption{
				matched:   true,
				valueKind: searchGenericOptionConsumesNext,
			}
		}
		return parsedSearchGenericOption{matched: true}
	}

	tokens, ok := parseProbeShortOptions(arg, spec.shortOptionsWithValue)
	if !ok {
		return parsedSearchGenericOption{}
	}

	result := parsedSearchGenericOption{matched: true}
	for _, token := range tokens {
		shortName := "-" + string(token.name)
		if _, ok := spec.recursiveFlags[shortName]; ok {
			result.isRecursiveFlag = true
		}

		if _, ok := spec.shortOptionsWithValue[token.name]; !ok {
			continue
		}

		if want, ok := spec.recursiveShortOptionValues[token.name]; ok && token.hasAttachedValue && token.attachedValue == want {
			result.isRecursiveFlag = true
		}

		if token.hasAttachedValue {
			return result
		}
		if token.consumesNext {
			result.valueKind = searchGenericOptionConsumesNext
			return result
		}
	}

	return result
}

func splitLongOption(arg string) (name string, hasValue bool) {
	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		return arg[:idx], true
	}
	return arg, false
}

func optionByte(option string) byte {
	if len(option) != 2 || option[0] != '-' {
		return 0
	}
	return option[1]
}
