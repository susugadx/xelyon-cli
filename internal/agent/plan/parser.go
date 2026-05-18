package plan

// ParsePlan はJSON文字列からPlanを解析
//
// 互換対応:
// - V2形式: {"plan": {...}}
// - 旧形式: {"summary": "...", "steps": [...]}
func ParsePlan(jsonStr string) (*Plan, error) {
	if wrappedPlan, handled, err := parseWrappedPlanIfPresent(jsonStr); handled || err != nil {
		return wrappedPlan, err
	}
	return parseLegacyPlan(jsonStr)
}
