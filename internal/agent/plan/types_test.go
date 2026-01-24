package plan

import (
	"testing"
)

func TestGetStep(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Description: "Step 1"},
			{ID: 2, Description: "Step 2"},
			{ID: 3, Description: "Step 3"},
		},
	}

	tests := []struct {
		id       int
		expected string
		isNil    bool
	}{
		{1, "Step 1", false},
		{2, "Step 2", false},
		{3, "Step 3", false},
		{4, "", true},  // Non-existent
		{0, "", true},  // Non-existent
		{-1, "", true}, // Invalid
	}

	for _, tt := range tests {
		step := plan.GetStep(tt.id)
		if tt.isNil {
			if step != nil {
				t.Errorf("GetStep(%d) should return nil, got %+v", tt.id, step)
			}
		} else {
			if step == nil {
				t.Errorf("GetStep(%d) returned nil, expected step", tt.id)
			} else if step.Description != tt.expected {
				t.Errorf("GetStep(%d).Description = %q, want %q", tt.id, step.Description, tt.expected)
			}
		}
	}
}

func TestCanExecute(t *testing.T) {
	tests := []struct {
		name     string
		plan     *Plan
		stepID   int
		expected bool
	}{
		{
			name: "pending with no deps",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "pending"},
				},
			},
			stepID:   1,
			expected: true,
		},
		{
			name: "pending with completed deps",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "pending", DependsOn: []int{1}},
				},
			},
			stepID:   2,
			expected: true,
		},
		{
			name: "pending with incomplete deps",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "pending"},
					{ID: 2, Status: "pending", DependsOn: []int{1}},
				},
			},
			stepID:   2,
			expected: false,
		},
		{
			name: "already completed",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
				},
			},
			stepID:   1,
			expected: false,
		},
		{
			name: "running",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "running"},
				},
			},
			stepID:   1,
			expected: false,
		},
		{
			name: "non-existent step",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "pending"},
				},
			},
			stepID:   99,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.CanExecute(tt.stepID)
			if result != tt.expected {
				t.Errorf("CanExecute(%d) = %v, want %v", tt.stepID, result, tt.expected)
			}
		})
	}
}

func TestGetParallelSteps(t *testing.T) {
	tests := []struct {
		name     string
		plan     *Plan
		expected []int
	}{
		{
			name: "multiple parallel steps with same deps",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "pending", DependsOn: []int{1}},
					{ID: 3, Status: "pending", DependsOn: []int{1}},
					{ID: 4, Status: "pending", DependsOn: []int{1}},
				},
			},
			expected: []int{2, 3, 4},
		},
		{
			name: "no parallel steps - different deps",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "completed"},
					{ID: 3, Status: "pending", DependsOn: []int{1}},
					{ID: 4, Status: "pending", DependsOn: []int{2}},
				},
			},
			expected: nil, // Only one executable per dep group
		},
		{
			name: "single executable step",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "pending"},
				},
			},
			expected: nil,
		},
		{
			name: "no deps - parallel at start",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "pending"},
					{ID: 2, Status: "pending"},
				},
			},
			expected: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.GetParallelSteps()

			if tt.expected == nil {
				if result != nil {
					t.Errorf("GetParallelSteps() = %v, want nil", result)
				}
			} else {
				if len(result) != len(tt.expected) {
					t.Errorf("GetParallelSteps() length = %d, want %d", len(result), len(tt.expected))
				}
				resultSet := make(map[int]bool)
				for _, id := range result {
					resultSet[id] = true
				}
				for _, expID := range tt.expected {
					if !resultSet[expID] {
						t.Errorf("Expected step ID %d not in result %v", expID, result)
					}
				}
			}
		})
	}
}

func TestGetNextStep(t *testing.T) {
	tests := []struct {
		name     string
		plan     *Plan
		expected int
	}{
		{
			name: "first pending",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "pending"},
					{ID: 2, Status: "pending"},
				},
			},
			expected: 1,
		},
		{
			name: "skip completed",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "pending"},
				},
			},
			expected: 2,
		},
		{
			name: "respect dependencies",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "pending"},
					{ID: 2, Status: "pending", DependsOn: []int{1}},
				},
			},
			expected: 1,
		},
		{
			name: "all completed",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "completed"},
				},
			},
			expected: -1,
		},
		{
			name: "blocked by dependency",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "running"},
					{ID: 2, Status: "pending", DependsOn: []int{1}},
				},
			},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.GetNextStep()
			if result != tt.expected {
				t.Errorf("GetNextStep() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "pending"},
		},
	}

	plan.UpdateStatus(1, "completed", "Done")

	step := plan.GetStep(1)
	if step.Status != "completed" {
		t.Errorf("Status = %q, want 'completed'", step.Status)
	}
	if step.Result != "Done" {
		t.Errorf("Result = %q, want 'Done'", step.Result)
	}

	// Update non-existent step (should not panic)
	plan.UpdateStatus(99, "completed", "")
}

func TestIncrementToolsExecuted(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, ToolsExecuted: 0},
		},
	}

	plan.IncrementToolsExecuted(1)
	plan.IncrementToolsExecuted(1)
	plan.IncrementToolsExecuted(1)

	if plan.GetToolsExecuted(1) != 3 {
		t.Errorf("ToolsExecuted = %d, want 3", plan.GetToolsExecuted(1))
	}

	// Non-existent step
	if plan.GetToolsExecuted(99) != 0 {
		t.Error("GetToolsExecuted for non-existent step should return 0")
	}

	// Increment non-existent (should not panic)
	plan.IncrementToolsExecuted(99)
}

func TestIsCompleted(t *testing.T) {
	tests := []struct {
		name     string
		plan     *Plan
		expected bool
	}{
		{
			name: "all completed",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "completed"},
				},
			},
			expected: true,
		},
		{
			name: "some pending",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "pending"},
				},
			},
			expected: false,
		},
		{
			name: "empty plan",
			plan: &Plan{
				Steps: []PlanStep{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.IsCompleted()
			if result != tt.expected {
				t.Errorf("IsCompleted() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasFailed(t *testing.T) {
	tests := []struct {
		name     string
		plan     *Plan
		expected bool
	}{
		{
			name: "has failed step",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "failed"},
				},
			},
			expected: true,
		},
		{
			name: "no failed steps",
			plan: &Plan{
				Steps: []PlanStep{
					{ID: 1, Status: "completed"},
					{ID: 2, Status: "pending"},
				},
			},
			expected: false,
		},
		{
			name: "empty plan",
			plan: &Plan{
				Steps: []PlanStep{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.HasFailed()
			if result != tt.expected {
				t.Errorf("HasFailed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSameDependencies(t *testing.T) {
	tests := []struct {
		name     string
		a        []int
		b        []int
		expected bool
	}{
		{
			name:     "same order",
			a:        []int{1, 2, 3},
			b:        []int{1, 2, 3},
			expected: true,
		},
		{
			name:     "different order",
			a:        []int{1, 2, 3},
			b:        []int{3, 2, 1},
			expected: true,
		},
		{
			name:     "different length",
			a:        []int{1, 2},
			b:        []int{1, 2, 3},
			expected: false,
		},
		{
			name:     "different values",
			a:        []int{1, 2},
			b:        []int{1, 3},
			expected: false,
		},
		{
			name:     "both empty",
			a:        []int{},
			b:        []int{},
			expected: true,
		},
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sameDependencies(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("sameDependencies(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
