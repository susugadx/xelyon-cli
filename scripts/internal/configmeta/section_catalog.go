package configmeta

type sectionCatalogEntry struct {
	Name     string
	Category string
}

var sectionCatalog = []sectionCatalogEntry{
	{Name: "default_provider", Category: "provider"},
	{Name: "default_model", Category: "provider"},
	{Name: "provider_models", Category: "provider"},
	{Name: "gemini", Category: "provider"},
	{Name: "review", Category: "review"},
	{Name: "general", Category: "general"},
	{Name: "execution", Category: "execution"},
	{Name: "compression", Category: "compression"},
	{Name: "provider_history_reduction", Category: "provider_history_reduction"},
	{Name: "paste", Category: "paste"},
	{Name: "project_map", Category: "project_map"},
	{Name: "agent_instructions", Category: "agent_instructions"},
	{Name: "skills", Category: "skills"},
	{Name: "lsp", Category: "lsp"},
	{Name: "output", Category: "output"},
	{Name: "web_search", Category: "web_search"},
	{Name: "sub_agent", Category: "sub_agent"},
	{Name: "mcp", Category: "mcp"},
	{Name: "final_checks", Category: "final_checks"},
}

// SectionOrder は user-facing section の表示順。
var SectionOrder = buildSectionOrder(sectionCatalog)

// SectionToCategory は section と UI category の対応表。
var SectionToCategory = buildSectionCategoryMap(sectionCatalog)

type categoryCatalogEntry struct {
	Name string
	Info CategoryInfo
}

var categoryCatalog = []categoryCatalogEntry{
	{Name: "provider", Info: CategoryInfo{DisplayName: "Provider & Model", Icon: "🤖"}},
	{Name: "review", Info: CategoryInfo{DisplayName: "Review", Icon: "🔎"}},
	{Name: "general", Info: CategoryInfo{DisplayName: "General", Icon: "⚙️"}},
	{Name: "execution", Info: CategoryInfo{DisplayName: "Execution Mode", Icon: "🛡️"}},
	{Name: "compression", Info: CategoryInfo{DisplayName: "Compression", Icon: "📦"}},
	{Name: "provider_history_reduction", Info: CategoryInfo{DisplayName: "Provider History Reduction", Icon: "📉"}},
	{Name: "paste", Info: CategoryInfo{DisplayName: "Paste Mode", Icon: "📋"}},
	{Name: "project_map", Info: CategoryInfo{DisplayName: "Project Map", Icon: "🗺️"}},
	{Name: "agent_instructions", Info: CategoryInfo{DisplayName: "Agent Instructions", Icon: "📚"}},
	{Name: "skills", Info: CategoryInfo{DisplayName: "Agent Skills", Icon: "🧭"}},
	{Name: "lsp", Info: CategoryInfo{DisplayName: "LSP Servers", Icon: "🔧"}},
	{Name: "output", Info: CategoryInfo{DisplayName: "Output", Icon: "📤"}},
	{Name: "web_search", Info: CategoryInfo{DisplayName: "Web Search", Icon: "🔍"}},
	{Name: "sub_agent", Info: CategoryInfo{DisplayName: "Sub-agent", Icon: "🚀"}},
	{Name: "mcp", Info: CategoryInfo{DisplayName: "MCP Servers", Icon: "🔌"}},
	{Name: "final_checks", Info: CategoryInfo{DisplayName: "Final Checks", Icon: "🧪"}},
}

// CategoryOrder は UI category の表示順。
var CategoryOrder = buildCategoryOrder(categoryCatalog)

// Categories は category の表示メタデータ。
var Categories = buildCategoryMap(categoryCatalog)

func buildSectionOrder(catalog []sectionCatalogEntry) []string {
	order := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		order = append(order, entry.Name)
	}
	return order
}

func buildSectionCategoryMap(catalog []sectionCatalogEntry) map[string]string {
	mapping := make(map[string]string, len(catalog))
	for _, entry := range catalog {
		mapping[entry.Name] = entry.Category
	}
	return mapping
}

func buildCategoryOrder(catalog []categoryCatalogEntry) []string {
	order := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		order = append(order, entry.Name)
	}
	return order
}

func buildCategoryMap(catalog []categoryCatalogEntry) map[string]CategoryInfo {
	categories := make(map[string]CategoryInfo, len(catalog))
	for _, entry := range catalog {
		categories[entry.Name] = entry.Info
	}
	return categories
}

// OrderedSectionsForCategory は category に属する section を表示順で返す。
func OrderedSectionsForCategory(category string) []string {
	var sections []string
	for _, sectionName := range SectionOrder {
		if SectionToCategory[sectionName] == category {
			sections = append(sections, sectionName)
		}
	}
	return sections
}
