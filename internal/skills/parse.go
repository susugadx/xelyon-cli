package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ParseSKILL は SKILL.md を解析し、frontmatter と本文・resource 一覧を返す。
func ParseSKILL(skillPath string) (ParsedSkill, error) {
	parsed, _, err := ParseSKILLWithDiagnostics(skillPath)
	return parsed, err
}

// ParseSKILLWithDiagnostics は SKILL.md と optional XELYON routing sidecar を解析する。
func ParseSKILLWithDiagnostics(skillPath string) (ParsedSkill, []Diagnostic, error) {
	skillPath = cleanAbsPathOrFallback(skillPath)
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return ParsedSkill{}, nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}
	return parseSKILLContentWithDiagnostics(skillPath, data)
}

func parseSKILLContentWithDiagnostics(skillPath string, data []byte) (ParsedSkill, []Diagnostic, error) {
	frontmatterRaw, body, err := splitSkillFrontmatter(string(data))
	if err != nil {
		return ParsedSkill{}, nil, err
	}

	meta := skillFrontmatter{}
	if err := yaml.Unmarshal([]byte(frontmatterRaw), &meta); err != nil {
		return ParsedSkill{}, nil, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	meta.Name = strings.TrimSpace(meta.Name)
	meta.Description = strings.TrimSpace(meta.Description)
	if meta.Name == "" {
		return ParsedSkill{}, nil, fmt.Errorf("missing required frontmatter field: name")
	}
	if meta.Description == "" {
		return ParsedSkill{}, nil, fmt.Errorf("missing required frontmatter field: description")
	}

	dir := filepath.Dir(skillPath)
	parsed := ParsedSkill{
		Name:        meta.Name,
		Description: meta.Description,
		Body:        body,
		Directory:   cleanAbsPathOrFallback(dir),
		SkillPath:   skillPath,
	}
	for _, group := range skillResourceGroupOrder {
		items, err := listDirectFiles(filepath.Join(dir, group.String()), group.String())
		if err != nil {
			return ParsedSkill{}, nil, err
		}
		setSkillResourceItems(&parsed, group, items)
	}
	routing, diagnostics := loadXelyonRoutingMetadata(dir)
	parsed.Routing = routing

	return parsed, diagnostics, nil
}

func splitSkillFrontmatter(content string) (frontmatter string, body string, err error) {
	content = strings.TrimPrefix(content, "\ufeff")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("missing YAML frontmatter start delimiter")
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return "", "", fmt.Errorf("missing YAML frontmatter end delimiter")
	}

	frontmatter = strings.Join(lines[1:end], "\n")
	body = strings.Join(lines[end+1:], "\n")
	body = strings.TrimLeft(body, "\r\n")
	return frontmatter, body, nil
}

func listDirectFiles(dirPath string, group string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list %s files: %w", group, err)
	}

	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		items = append(items, filepath.ToSlash(filepath.Join(group, entry.Name())))
	}
	sort.Strings(items)
	return items, nil
}
