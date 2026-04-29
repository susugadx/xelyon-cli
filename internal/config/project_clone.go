package config

// CloneProjectConfig は ProjectConfig の編集用コピーを返す。
func CloneProjectConfig(pc *ProjectConfig) *ProjectConfig {
	if pc == nil {
		return nil
	}
	clone := *pc
	clone.Rules = append([]string(nil), pc.Rules...)
	clone.Conditional = cloneProjectConditionalBlocks(pc.Conditional)
	clone.Ignore.Patterns = append([]string(nil), pc.Ignore.Patterns...)
	if pc.FinalChecks != nil {
		fc := *pc.FinalChecks
		fc.Commands = append([]string(nil), pc.FinalChecks.Commands...)
		clone.FinalChecks = &fc
	}
	return &clone
}

func cloneProjectConditionalBlocks(blocks []ProjectConditionalBlock) []ProjectConditionalBlock {
	if len(blocks) == 0 {
		return nil
	}
	clone := make([]ProjectConditionalBlock, len(blocks))
	for i, block := range blocks {
		clone[i] = block
		clone[i].Paths = append([]string(nil), block.Paths...)
		clone[i].Rules = append([]string(nil), block.Rules...)
	}
	return clone
}
