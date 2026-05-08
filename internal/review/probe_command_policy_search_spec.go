package review

type searchCommandPatternSpec struct {
	regexpOptionShort string
	regexpOptionLong  string
	fileOptionShort   string
	fileOptionLong    string

	shortOptionsWithValue      map[byte]struct{}
	longOptionsWithValue       map[string]struct{}
	longOptionsFileValue       map[string]struct{}
	longOptionsPathOnlyMode    map[string]struct{}
	recursiveFlags             map[string]struct{}
	recursiveLongOptionValues  map[string]string
	recursiveShortOptionValues map[byte]string
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
		longOptionsFileValue: stringSet(
			"--ignore-file",
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
		longOptionsFileValue: stringSet(
			"--exclude-from",
		),
	},
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
