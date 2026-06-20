package plan

const (
	planWrapperJSONKey              = "plan"
	planSummaryJSONKey              = "summary"
	planSchemaVersionJSONKey        = "schema_version"
	planGoalJSONKey                 = "goal"
	planAcceptanceCriteriaJSONKey   = "acceptance_criteria"
	planFindingsJSONKey             = "findings"
	planEvidenceJSONKey             = "evidence"
	planConstraintsJSONKey          = "constraints"
	planOpenQuestionsJSONKey        = "open_questions"
	planV2StepOutcomeJSONKey        = "outcome"
	planV2StepReasonJSONKey         = "reason"
	legacyPlanStepsKey              = "steps"
	legacyPlanAssumptionsKey        = "assumptions"
	legacyPlanExpectedOutputStepKey = "expected_output"
)

var planContentJSONKeys = [...]string{
	planSchemaVersionJSONKey,
	planGoalJSONKey,
	planSummaryJSONKey,
	planAcceptanceCriteriaJSONKey,
	planFindingsJSONKey,
	planEvidenceJSONKey,
	planConstraintsJSONKey,
	planOpenQuestionsJSONKey,
}

var legacyPlanTopLevelEvidenceJSONKeys = [...]string{
	planGoalJSONKey,
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

var planV2ShapeSignalJSONKeys = [...]string{
	planAcceptanceCriteriaJSONKey,
	planFindingsJSONKey,
	planConstraintsJSONKey,
	planOpenQuestionsJSONKey,
}

var planV2StepShapeSignalJSONKeys = [...]string{
	planV2StepOutcomeJSONKey,
	planV2StepReasonJSONKey,
}

var planStepShapeJSONKeys = [...]string{
	"id",
	"description",
	planV2StepOutcomeJSONKey,
	"purpose",
	planV2StepReasonJSONKey,
	"tools",
	"depends_on",
	"files",
	"verification",
}

var normalModeImplementationStepJSONKeys = [...]string{
	"description",
	"outcome",
	"purpose",
	"reason",
	"tools",
	"files",
	"verification",
}
