package analysis

import (
	"path/filepath"
	"strings"
)

func reviewPressureSignalProductionWithoutTestsEvidence(input EvidenceInput) []string {
	inventory := input.ChangeInventory
	if len(inventory.Production) == 0 || len(inventory.Tests) > 0 {
		return nil
	}
	return append(reviewPressureSignalPathEvidence("production", inventory.Production), "tests: []")
}

func reviewPressureSignalConfigOrSchemaEvidence(input EvidenceInput) []string {
	inventory := input.ChangeInventory
	evidence := make([]string, 0)
	evidence = append(evidence, reviewPressureSignalPathEvidence("config", inventory.Config)...)
	evidence = append(evidence, reviewPressureSignalTokenPathEvidence("schema_or_contract_path", reviewPressureSignalAllInventoryPaths(inventory), reviewPressureSignalMatchesSchemaOrContractPath)...)
	return evidence
}

func reviewPressureSignalPromptContractEvidence(input EvidenceInput) []string {
	return reviewPressureSignalTokenPathEvidence("prompt_or_instruction_path", reviewPressureSignalAllInventoryPaths(input.ChangeInventory), func(path string) bool {
		return reviewPressureSignalMatchesPromptContractPath(path, input.KnownRuleFilePaths)
	})
}

func reviewPressureSignalDeletedOrRenamedEvidence(input EvidenceInput) []string {
	inventory := input.ChangeInventory
	if len(inventory.DeletedFiles) == 0 && len(inventory.RenamedFiles) == 0 {
		return nil
	}
	evidence := make([]string, 0, len(inventory.DeletedFiles)+len(inventory.RenamedFiles))
	evidence = append(evidence, reviewPressureSignalPathEvidence("deleted_files", inventory.DeletedFiles)...)
	evidence = append(evidence, reviewPressureSignalPathEvidence("renamed_files", inventory.RenamedFiles)...)
	return evidence
}

func reviewPressureSignalUntrackedEvidence(input EvidenceInput) []string {
	if len(input.ChangeInventory.Untracked) == 0 {
		return nil
	}
	return reviewPressureSignalPathEvidence("untracked", input.ChangeInventory.Untracked)
}

func reviewPressureSignalGeneratedEvidence(input EvidenceInput) []string {
	if len(input.ChangeInventory.Generated) == 0 {
		return nil
	}
	return reviewPressureSignalPathEvidence("generated", input.ChangeInventory.Generated)
}

func reviewPressureSignalAllInventoryPaths(inventory ChangeInventory) []string {
	paths := make([]string, 0,
		len(inventory.Generated)+
			len(inventory.Tests)+
			len(inventory.Docs)+
			len(inventory.Config)+
			len(inventory.Production)+
			len(inventory.NewFiles)+
			len(inventory.DeletedFiles)+
			len(inventory.RenamedFiles)+
			len(inventory.Untracked),
	)
	paths = append(paths, inventory.Generated...)
	paths = append(paths, inventory.Tests...)
	paths = append(paths, inventory.Docs...)
	paths = append(paths, inventory.Config...)
	paths = append(paths, inventory.Production...)
	paths = append(paths, inventory.NewFiles...)
	paths = append(paths, inventory.DeletedFiles...)
	paths = append(paths, inventory.RenamedFiles...)
	paths = append(paths, inventory.Untracked...)
	return paths
}

func reviewPressureSignalMatchesSchemaOrContractPath(path string) bool {
	normalized := strings.ToLower(path)
	return strings.Contains(normalized, "schema") ||
		strings.Contains(normalized, "contract")
}

func reviewPressureSignalMatchesPromptContractPath(path string, ruleFilePaths []string) bool {
	normalized := strings.ToLower(filepath.ToSlash(path))
	return reviewPressureSignalMatchesKnownRuleFilePath(normalized, ruleFilePaths) ||
		strings.Contains(normalized, "prompt") ||
		strings.Contains(normalized, "instruction") ||
		strings.Contains(normalized, "agents")
}

func reviewPressureSignalMatchesKnownRuleFilePath(normalizedPath string, ruleFilePaths []string) bool {
	for _, rulePath := range ruleFilePaths {
		normalizedRulePath := strings.ToLower(filepath.ToSlash(rulePath))
		if normalizedPath == normalizedRulePath || strings.HasSuffix(normalizedPath, "/"+normalizedRulePath) {
			return true
		}
	}
	return false
}
