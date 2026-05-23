package impactplan

import "testing"

func TestPlanForRisk(t *testing.T) {
	tests := []struct {
		name string
		risk string
		want Plan
	}{
		{
			name: "high",
			risk: RiskHigh,
			want: Plan{RiskLevel: RiskHigh, Budget: HighBudget, ImplementationLimit: 8},
		},
		{
			name: "medium",
			risk: RiskMedium,
			want: Plan{RiskLevel: RiskMedium, Budget: MediumBudget, ImplementationLimit: 4},
		},
		{
			name: "low",
			risk: RiskLow,
			want: Plan{RiskLevel: RiskLow, Budget: LowBudget, ImplementationLimit: 2},
		},
		{
			name: "unknown defaults to low",
			risk: "unknown",
			want: Plan{RiskLevel: RiskLow, Budget: LowBudget, ImplementationLimit: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlanForRisk(tt.risk); got != tt.want {
				t.Fatalf("PlanForRisk(%q) = %#v, want %#v", tt.risk, got, tt.want)
			}
		})
	}
}

func TestPlanEqual(t *testing.T) {
	base := Plan{RiskLevel: RiskMedium, Budget: MediumBudget, ImplementationLimit: 4}
	tests := []struct {
		name  string
		left  Plan
		right Plan
		want  bool
	}{
		{name: "equal", left: base, right: base, want: true},
		{name: "risk differs", left: base, right: Plan{RiskLevel: RiskHigh, Budget: MediumBudget, ImplementationLimit: 4}, want: false},
		{name: "budget differs", left: base, right: Plan{RiskLevel: RiskMedium, Budget: HighBudget, ImplementationLimit: 4}, want: false},
		{name: "implementation limit differs", left: base, right: Plan{RiskLevel: RiskMedium, Budget: MediumBudget, ImplementationLimit: 8}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlanEqual(tt.left, tt.right); got != tt.want {
				t.Fatalf("PlanEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanRank(t *testing.T) {
	tests := []struct {
		risk string
		want int
	}{
		{risk: RiskHigh, want: 3},
		{risk: RiskMedium, want: 2},
		{risk: RiskLow, want: 1},
		{risk: "unknown", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.risk, func(t *testing.T) {
			if got := PlanRank(Plan{RiskLevel: tt.risk}); got != tt.want {
				t.Fatalf("PlanRank(%q) = %d, want %d", tt.risk, got, tt.want)
			}
		})
	}
}
