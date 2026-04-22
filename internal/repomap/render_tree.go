package repomap

import "strings"

func (pm *ProjectMap) render(options []renderOption, omittedFiles int) string {
	var b strings.Builder
	b.WriteString("## Project Map\n\n")

	renderState := buildRenderTreeState(pm.Files, options)
	for dirIndex, dir := range renderState.dirs {
		if dirIndex > 0 {
			b.WriteString("\n")
		}
		writeRenderedDirectory(&b, dir, renderState.grouped[dir], renderState.pathIndex, options)
	}

	writeRenderedOmittedFiles(&b, len(renderState.dirs), omittedFiles)
	writeRenderedGitStatus(&b, len(renderState.dirs), omittedFiles, pm.GitStatus)

	return strings.TrimRight(b.String(), "\n")
}
