package plan

const (
	planWrapperJSONKey              = "plan"
	planSummaryJSONKey              = "summary"
	planFindingsJSONKey             = "findings"
	planEvidenceJSONKey             = "evidence"
	planConstraintsJSONKey          = "constraints"
	legacyPlanStepsKey              = "steps"
	legacyPlanGoalKey               = "goal"
	legacyPlanAssumptionsKey        = "assumptions"
	legacyPlanExpectedOutputStepKey = "expected_output"
)

var planContentJSONKeys = [...]string{
	planSummaryJSONKey,
	planFindingsJSONKey,
	planEvidenceJSONKey,
	planConstraintsJSONKey,
}

var legacyPlanTopLevelEvidenceJSONKeys = [...]string{
	legacyPlanGoalKey,
	legacyPlanAssumptionsKey,
}

var planSpecificStepEvidenceJSONKeys = [...]string{
	"purpose",
	"tools",
	"depends_on",
	"files",
	"verification",
	legacyPlanExpectedOutputStepKey,
}

var planStepShapeJSONKeys = [...]string{
	"id",
	"description",
	"purpose",
	"tools",
	"depends_on",
	"files",
	"verification",
}

var normalModeImplementationStepJSONKeys = [...]string{
	"description",
	"purpose",
	"tools",
	"files",
	"verification",
}
