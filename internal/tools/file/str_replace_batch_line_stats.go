package file

import (
	"os"
	"strconv"
	"strings"
)

const defaultMyersDiagonalStepLimit = 2_000_000
const defaultMyersLineSpanLimit = 200_000
const defaultMyersMinDiagonalStepLimit = 20_000
const defaultMyersRewriteBudgetDivisor = 6
const defaultMyersRewriteBudgetFloor = 32

const (
	envMyersDiagonalStepLimit = "XELYON_STR_REPLACE_MYERS_DIAGONAL_STEP_LIMIT"
	envMyersLineSpanLimit     = "XELYON_STR_REPLACE_MYERS_LINE_SPAN_LIMIT"
	envMyersMinStepLimit      = "XELYON_STR_REPLACE_MYERS_MIN_STEP_LIMIT"
	envMyersRewriteDivisor    = "XELYON_STR_REPLACE_MYERS_REWRITE_DIVISOR"
	envMyersRewriteFloor      = "XELYON_STR_REPLACE_MYERS_REWRITE_FLOOR"
)

var myersDiagonalStepLimit = defaultMyersDiagonalStepLimit
var myersLineSpanLimit = defaultMyersLineSpanLimit
var myersMinDiagonalStepLimit = defaultMyersMinDiagonalStepLimit
var myersRewriteBudgetDivisor = defaultMyersRewriteBudgetDivisor
var myersRewriteBudgetFloor = defaultMyersRewriteBudgetFloor

type myersDiffTuning struct {
	diagonalStepLimit    int
	lineSpanLimit        int
	minDiagonalStep      int
	rewriteBudgetDivisor int
	rewriteBudgetFloor   int
}

func resolveBatchDiffLineStats(oldContent, newContent string) (added, removed int, exact bool) {
	tuning := resolveMyersDiffTuning()
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	trimmedOldLines, trimmedNewLines := trimSharedLineEdges(oldLines, newLines)
	if len(trimmedOldLines) == 0 && len(trimmedNewLines) == 0 {
		return 0, 0, true
	}
	if len(trimmedOldLines) == 0 {
		return len(trimmedNewLines), 0, true
	}
	if len(trimmedNewLines) == 0 {
		return 0, len(trimmedOldLines), true
	}
	return tryCountLineDiffWithMyers(trimmedOldLines, trimmedNewLines, resolveDynamicMyersStepLimit(len(trimmedOldLines), len(trimmedNewLines), tuning), tuning.lineSpanLimit)
}

func tryCountLineDiffWithMyers(oldLines, newLines []string, stepLimit, lineSpanLimit int) (added, removed int, ok bool) {
	n := len(oldLines)
	m := len(newLines)
	if lineSpanLimit > 0 && n+m > lineSpanLimit {
		return 0, 0, false
	}

	distance, ok := shortestEditDistanceMyersWithLimit(oldLines, newLines, stepLimit)
	if !ok {
		return 0, 0, false
	}

	removed = (distance - m + n) / 2
	added = (distance + m - n) / 2
	return added, removed, true
}

func resolveDynamicMyersStepLimit(oldLineCount, newLineCount int, tuning myersDiffTuning) int {
	if oldLineCount < 0 {
		oldLineCount = 0
	}
	if newLineCount < 0 {
		newLineCount = 0
	}

	distanceBudget := estimateMyersDistanceBudget(oldLineCount, newLineCount, tuning)
	stepLimit := (distanceBudget + 1) * (distanceBudget + 1)

	if stepLimit < tuning.minDiagonalStep {
		stepLimit = tuning.minDiagonalStep
	}
	if tuning.diagonalStepLimit > 0 && stepLimit > tuning.diagonalStepLimit {
		stepLimit = tuning.diagonalStepLimit
	}
	return stepLimit
}

func estimateMyersDistanceBudget(oldLineCount, newLineCount int, tuning myersDiffTuning) int {
	smaller := oldLineCount
	larger := newLineCount
	if newLineCount < oldLineCount {
		smaller = newLineCount
		larger = oldLineCount
	}

	imbalance := larger - smaller

	divisor := tuning.rewriteBudgetDivisor
	if divisor <= 0 {
		divisor = 1
	}
	rewriteBudget := smaller / divisor
	if rewriteBudget < tuning.rewriteBudgetFloor {
		rewriteBudget = tuning.rewriteBudgetFloor
	}
	return imbalance + rewriteBudget
}

func resolveMyersDiffTuning() myersDiffTuning {
	tuning := myersDiffTuning{
		diagonalStepLimit:    myersDiagonalStepLimit,
		lineSpanLimit:        myersLineSpanLimit,
		minDiagonalStep:      myersMinDiagonalStepLimit,
		rewriteBudgetDivisor: myersRewriteBudgetDivisor,
		rewriteBudgetFloor:   myersRewriteBudgetFloor,
	}

	tuning.diagonalStepLimit = resolveEnvIntOrDefault(envMyersDiagonalStepLimit, tuning.diagonalStepLimit)
	tuning.lineSpanLimit = resolveEnvIntOrDefault(envMyersLineSpanLimit, tuning.lineSpanLimit)
	tuning.minDiagonalStep = resolveEnvIntOrDefault(envMyersMinStepLimit, tuning.minDiagonalStep)
	tuning.rewriteBudgetDivisor = resolveEnvIntOrDefault(envMyersRewriteDivisor, tuning.rewriteBudgetDivisor)
	tuning.rewriteBudgetFloor = resolveEnvIntOrDefault(envMyersRewriteFloor, tuning.rewriteBudgetFloor)

	if tuning.diagonalStepLimit < 1 {
		tuning.diagonalStepLimit = myersDiagonalStepLimit
	}
	if tuning.lineSpanLimit < 1 {
		tuning.lineSpanLimit = myersLineSpanLimit
	}
	if tuning.minDiagonalStep < 1 {
		tuning.minDiagonalStep = 1
	}
	if tuning.diagonalStepLimit < 1 {
		tuning.diagonalStepLimit = 1
	}
	if tuning.lineSpanLimit < 1 {
		tuning.lineSpanLimit = 1
	}
	if tuning.diagonalStepLimit > 0 && tuning.diagonalStepLimit < tuning.minDiagonalStep {
		tuning.diagonalStepLimit = tuning.minDiagonalStep
	}
	if tuning.rewriteBudgetDivisor <= 0 {
		tuning.rewriteBudgetDivisor = 1
	}
	if tuning.rewriteBudgetFloor < 0 {
		tuning.rewriteBudgetFloor = 0
	}

	return tuning
}

func resolveEnvIntOrDefault(name string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func shortestEditDistanceMyersWithLimit(oldLines, newLines []string, stepLimit int) (int, bool) {
	n := len(oldLines)
	m := len(newLines)
	maxDistance := n + m
	offset := maxDistance

	diagonalEnds := make([]int, 2*maxDistance+1)
	for i := range diagonalEnds {
		diagonalEnds[i] = -1
	}
	diagonalEnds[offset+1] = 0

	steps := 0
	for distance := 0; distance <= maxDistance; distance++ {
		for diagonal := -distance; diagonal <= distance; diagonal += 2 {
			steps++
			if stepLimit > 0 && steps > stepLimit {
				return 0, false
			}

			index := offset + diagonal
			var x int
			if diagonal == -distance || (diagonal != distance && diagonalEnds[index-1] < diagonalEnds[index+1]) {
				x = diagonalEnds[index+1]
			} else {
				x = diagonalEnds[index-1] + 1
			}

			y := x - diagonal
			for x < n && y < m && oldLines[x] == newLines[y] {
				x++
				y++
			}
			diagonalEnds[index] = x

			if x >= n && y >= m {
				return distance, true
			}
		}
	}
	return maxDistance, true
}

func trimSharedLineEdges(oldLines, newLines []string) ([]string, []string) {
	prefix := countCommonLinePrefix(oldLines, newLines)
	remainingOld := oldLines[prefix:]
	remainingNew := newLines[prefix:]

	suffix := countCommonLineSuffix(remainingOld, remainingNew)
	return remainingOld[:len(remainingOld)-suffix], remainingNew[:len(remainingNew)-suffix]
}

func countCommonLinePrefix(oldLines, newLines []string) int {
	limit := len(oldLines)
	if len(newLines) < limit {
		limit = len(newLines)
	}

	count := 0
	for count < limit && oldLines[count] == newLines[count] {
		count++
	}
	return count
}

func countCommonLineSuffix(oldLines, newLines []string) int {
	limit := len(oldLines)
	if len(newLines) < limit {
		limit = len(newLines)
	}

	count := 0
	for count < limit {
		oldIdx := len(oldLines) - 1 - count
		newIdx := len(newLines) - 1 - count
		if oldLines[oldIdx] != newLines[newIdx] {
			break
		}
		count++
	}
	return count
}
