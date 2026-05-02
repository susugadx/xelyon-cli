package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type discoverRoot struct {
	Path   string
	Source Source
	Order  int
}

// Discover は .agents/skills と ~/.agents/skills を走査して SKILL.md 候補を検出する。
func Discover(opts DiscoverOptions) DiscoverResult {
	cwd := strings.TrimSpace(opts.InvocationCWD)
	if cwd == "" {
		if current, err := os.Getwd(); err == nil {
			cwd = current
		}
	}
	if cwd == "" {
		cwd = "."
	}

	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}

	roots := resolveDiscoverRoots(cwd, home)
	result := DiscoverResult{Roots: make([]string, 0, len(roots))}
	for _, root := range roots {
		result.Roots = append(result.Roots, root.Path)
		skills, diags := discoverFromRoot(root)
		result.Skills = append(result.Skills, skills...)
		result.Diagnostics = append(result.Diagnostics, diags...)
	}

	sort.SliceStable(result.Skills, func(i, j int) bool {
		left := result.Skills[i]
		right := result.Skills[j]
		if left.RootOrder != right.RootOrder {
			return left.RootOrder < right.RootOrder
		}
		return left.PathOrder < right.PathOrder
	})

	return result
}

func resolveDiscoverRoots(cwd, home string) []discoverRoot {
	projectRoot := resolveProjectSkillsRoot(cwd, home)
	homeRoot := ""
	if strings.TrimSpace(home) != "" {
		homeRoot = cleanAbsPathOrFallback(filepath.Join(home, ".agents", "skills"))
	}
	projectSource := SourceProject
	if homeRoot != "" && sameCleanPath(projectRoot, homeRoot) {
		projectSource = SourceHome
	}
	roots := []discoverRoot{{
		Path:   projectRoot,
		Source: projectSource,
		Order:  0,
	}}

	if homeRoot != "" && !sameCleanPath(projectRoot, homeRoot) {
		roots = append(roots, discoverRoot{Path: homeRoot, Source: SourceHome, Order: 1})
	}

	for i := range roots {
		roots[i].Path = cleanAbsPathOrFallback(roots[i].Path)
	}
	return roots
}

func resolveProjectSkillsRoot(cwd, home string) string {
	cwd = cleanAbsPathOrFallback(cwd)
	homeRoot := ""
	if strings.TrimSpace(home) != "" {
		homeRoot = cleanAbsPathOrFallback(filepath.Join(home, ".agents", "skills"))
	}

	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, ".agents", "skills")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if homeRoot == "" || !sameCleanPath(candidate, homeRoot) {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return filepath.Join(cwd, ".agents", "skills")
}

func discoverFromRoot(root discoverRoot) ([]DiscoveredSkill, []Diagnostic) {
	entries, err := os.ReadDir(root.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, []Diagnostic{newDiagnostic(SeverityError, "discover_failed", root.Path, fmt.Sprintf("failed to scan skills root: %v", err))}
	}

	items := make([]DiscoveredSkill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root.Path, entry.Name())
		skillPath := filepath.Join(dir, "SKILL.md")
		info, statErr := os.Stat(skillPath)
		if statErr != nil || info.IsDir() {
			continue
		}

		items = append(items, DiscoveredSkill{
			Directory:   cleanAbsPathOrFallback(dir),
			SkillPath:   cleanAbsPathOrFallback(skillPath),
			Source:      root.Source,
			RootPath:    root.Path,
			RootOrder:   root.Order,
			PathOrder:   filepath.ToSlash(cleanAbsPathOrFallback(dir)),
			DisplayPath: cleanAbsPathOrFallback(dir),
		})
	}
	return items, nil
}

func cleanAbsPathOrFallback(path string) string {
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func sameCleanPath(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return cleanAbsPathOrFallback(left) == cleanAbsPathOrFallback(right)
}
