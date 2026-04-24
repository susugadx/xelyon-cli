package plan

// InferDependencies は依存関係を自動推論する。
// RAW (Read After Write): BがAの書き込みファイルを読む。
// WAW (Write After Write): 両方が同じファイルに書く。
// WAR (Write After Read): BがAの読み取りファイルに書く。
func (da *DependencyAnalyzer) InferDependencies(steps []PlanStep) []PlanStep {
	for i := range steps {
		currentID := steps[i].ID
		existingDeps := make(map[int]bool)
		for _, d := range steps[i].DependsOn {
			existingDeps[d] = true
		}

		newDeps := make(map[int]bool)

		// RAW: 書き込み後読み取り
		for _, readFile := range steps[i].ReadFiles {
			writers := da.fileWriters[readFile]
			for _, writerID := range writers {
				// 自分より前のステップで書き込んでいる場合
				if writerID < currentID && !existingDeps[writerID] {
					newDeps[writerID] = true
				}
			}
		}

		// WAW: 書き込み後書き込み
		for _, writeFile := range steps[i].WriteFiles {
			writers := da.fileWriters[writeFile]
			for _, writerID := range writers {
				// 自分より前のステップで書き込んでいる場合
				if writerID < currentID && !existingDeps[writerID] {
					newDeps[writerID] = true
				}
			}
		}

		// WAR: 読み取り後書き込み
		for _, writeFile := range steps[i].WriteFiles {
			readers := da.fileReaders[writeFile]
			for _, readerID := range readers {
				// 自分より前のステップで読み取っている場合
				if readerID < currentID && !existingDeps[readerID] {
					newDeps[readerID] = true
				}
			}
		}

		// 新しい依存関係を追加
		for depID := range newDeps {
			steps[i].DependsOn = append(steps[i].DependsOn, depID)
		}
	}

	return steps
}

// containsInt は int スライスに値が含まれているかを返す。
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
