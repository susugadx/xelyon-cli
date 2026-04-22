package file

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
