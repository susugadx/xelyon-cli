package skills

import (
	"fmt"
	"sort"
	"strings"
)

type discoveredSkillParser func(found DiscoveredSkill) (ParsedSkill, error)

// LoadCatalogForInvocationCWD は invocation cwd を指定して skill catalog を読み込む。
func LoadCatalogForInvocationCWD(invocationCWD string) SkillCatalog {
	return LoadCatalog(DiscoverOptions{InvocationCWD: invocationCWD})
}

// LoadCatalog は現在環境の discover + parse + catalog を一括で実行する。
func LoadCatalog(opts DiscoverOptions) SkillCatalog {
	return defaultCatalogStore.Load(opts)
}

// Catalog は discover 結果を解析し、重複名を deterministic に解決した catalog を返す。
func Catalog(discover DiscoverResult) SkillCatalog {
	return catalogWithParser(discover, func(found DiscoveredSkill) (ParsedSkill, error) {
		return ParseSKILL(found.SkillPath)
	})
}

// CatalogWithContentCache は SKILL.md の内容キャッシュを優先利用して catalog を構築する。
func CatalogWithContentCache(discover DiscoverResult, skillContents map[string][]byte) SkillCatalog {
	return catalogWithParser(discover, func(found DiscoveredSkill) (ParsedSkill, error) {
		skillPath := cleanAbsPathOrFallback(found.SkillPath)
		if len(skillContents) > 0 {
			if data, ok := skillContents[skillPath]; ok {
				return parseSKILLContent(skillPath, data)
			}
		}
		return ParseSKILL(skillPath)
	})
}

func catalogWithParser(discover DiscoverResult, parser discoveredSkillParser) SkillCatalog {
	catalog := SkillCatalog{
		Skills:      make([]ParsedSkill, 0, len(discover.Skills)),
		Diagnostics: append([]Diagnostic(nil), discover.Diagnostics...),
	}

	nameIndex := make(map[string]int)
	for _, found := range discover.Skills {
		parsed, err := parser(found)
		if err != nil {
			catalog.Diagnostics = append(catalog.Diagnostics, newDiagnostic(SeverityError, "parse_skill_failed", found.SkillPath, err.Error()))
			continue
		}
		parsed.Source = found.Source
		parsed.Directory = cleanAbsPathOrFallback(found.Directory)
		parsed.SkillPath = cleanAbsPathOrFallback(found.SkillPath)

		if idx, exists := nameIndex[parsed.Name]; exists {
			chosen := catalog.Skills[idx]
			catalog.Diagnostics = append(catalog.Diagnostics, newDiagnostic(
				SeverityWarning,
				"duplicate_skill_name",
				parsed.SkillPath,
				fmt.Sprintf("duplicate skill name %q: keep %s, skip %s", parsed.Name, chosen.SkillPath, parsed.SkillPath),
			))
			continue
		}

		nameIndex[parsed.Name] = len(catalog.Skills)
		catalog.Skills = append(catalog.Skills, parsed)
	}

	sort.SliceStable(catalog.Skills, func(i, j int) bool {
		left := catalog.Skills[i]
		right := catalog.Skills[j]
		leftKey := strings.ToLower(left.Name)
		rightKey := strings.ToLower(right.Name)
		if leftKey != rightKey {
			return leftKey < rightKey
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.SkillPath < right.SkillPath
	})

	return catalog
}

// SkillNames は catalog の skill 名一覧を返す。
func SkillNames(catalog SkillCatalog) []string {
	names := make([]string, 0, len(catalog.Skills))
	for _, skill := range catalog.Skills {
		names = append(names, skill.Name)
	}
	sort.Strings(names)
	return names
}
