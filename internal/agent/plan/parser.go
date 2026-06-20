package plan

// ParsePlan はJSON文字列からPlanを解析
//
// 互換対応:
// - xelyon.plan.v2 形式: {"schema_version":"xelyon.plan.v2", ...}
// - V2形式: {"plan": {...}}
// - 旧形式: {"summary": "...", "steps": [...]}
func ParsePlan(jsonStr string) (*Plan, error) {
	if v2Plan, handled, err := parsePlanV2IfPresent(jsonStr); handled || err != nil {
		return v2Plan, err
	}
	if wrappedPlan, handled, err := parseWrappedPlanIfPresent(jsonStr); handled || err != nil {
		return wrappedPlan, err
	}
	return parseLegacyPlan(jsonStr)
}
