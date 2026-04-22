package agent

func normalizeParallelToolFamily(tool string) string {
	switch tool {
	case "read_file", "read_files", "list_dir":
		return "reads"
	case "search_code":
		return "searches"
	case "web_search":
		return "web"
	default:
		return tool
	}
}

func parallelGroupSummaryLabel(tool string) string {
	switch tool {
	case "bash":
		return "bash"
	default:
		return normalizeParallelToolFamily(tool)
	}
}

func parallelSpinnerBucket(tool string) string {
	switch tool {
	case "spawn_agent":
		return "spawn"
	case "wait_agent":
		return "wait"
	}

	family := normalizeParallelToolFamily(tool)
	switch family {
	case "reads", "searches", "web":
		return family
	default:
		return "tools"
	}
}
