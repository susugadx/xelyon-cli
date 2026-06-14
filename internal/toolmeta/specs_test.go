package toolmeta

import "testing"

func TestBuiltinSpecsReturnsCopy(t *testing.T) {
	specs := BuiltinSpecs()
	if len(specs) == 0 {
		t.Fatal("BuiltinSpecs() returned no specs")
	}

	original := specs[0]
	specs[0] = Spec{
		Name:        "mutated_tool",
		Description: "mutated description",
		Safety:      SafetyLow,
		HelpSummary: "mutated summary",
		HelpOrder:   -1,
	}

	if _, ok := Lookup("mutated_tool"); ok {
		t.Fatal("Lookup(\"mutated_tool\") ok = true, want false after mutating returned slice")
	}
	got, ok := Lookup(original.Name)
	if !ok {
		t.Fatalf("Lookup(%q) ok = false, want true", original.Name)
	}
	if got != original {
		t.Fatalf("Lookup(%q) = %#v, want original spec %#v", original.Name, got, original)
	}

	fresh := BuiltinSpecs()
	if fresh[0] != original {
		t.Fatalf("fresh BuiltinSpecs()[0] = %#v, want original spec %#v", fresh[0], original)
	}
}

func TestDescriptionMapReturnsCopy(t *testing.T) {
	descriptions := DescriptionMap()
	original, ok := descriptions["gather_context"]
	if !ok {
		t.Fatal("DescriptionMap()[\"gather_context\"] missing")
	}
	if original == "" {
		t.Fatal("DescriptionMap()[\"gather_context\"] is empty")
	}

	descriptions["gather_context"] = "mutated"
	descriptions["new_tool"] = "new description"

	fresh := DescriptionMap()
	if fresh["gather_context"] != original {
		t.Fatalf("fresh gather_context description = %q, want original %q", fresh["gather_context"], original)
	}
	if _, ok := fresh["new_tool"]; ok {
		t.Fatal("fresh DescriptionMap()[\"new_tool\"] exists after mutating returned map")
	}
}

func TestLookupAndSafetyByName(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		wantLookup   bool
		wantSafety   SafetyLevel
		wantSafetyOK bool
	}{
		{
			name:         "gather context is high safety",
			toolName:     "gather_context",
			wantLookup:   true,
			wantSafety:   SafetyHigh,
			wantSafetyOK: true,
		},
		{
			name:         "apply patch is low safety",
			toolName:     "apply_patch",
			wantLookup:   true,
			wantSafety:   SafetyLow,
			wantSafetyOK: true,
		},
		{
			name:         "bash is low safety",
			toolName:     "bash",
			wantLookup:   true,
			wantSafety:   SafetyLow,
			wantSafetyOK: true,
		},
		{
			name:         "unknown tool falls back to medium safety",
			toolName:     "unknown_tool",
			wantLookup:   false,
			wantSafety:   SafetyMedium,
			wantSafetyOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, lookupOK := Lookup(tt.toolName)
			if lookupOK != tt.wantLookup {
				t.Fatalf("Lookup(%q) ok = %v, want %v", tt.toolName, lookupOK, tt.wantLookup)
			}

			gotSafety, safetyOK := SafetyByName(tt.toolName)
			if safetyOK != tt.wantSafetyOK {
				t.Fatalf("SafetyByName(%q) ok = %v, want %v", tt.toolName, safetyOK, tt.wantSafetyOK)
			}
			if gotSafety != tt.wantSafety {
				t.Fatalf("SafetyByName(%q) safety = %v, want %v", tt.toolName, gotSafety, tt.wantSafety)
			}
		})
	}
}

func TestHelpDisplayOrderSortsByOrderThenName(t *testing.T) {
	original := builtinSpecs
	builtinSpecs = append(BuiltinSpecs(),
		Spec{Name: "zz_same_order", HelpOrder: 1000},
		Spec{Name: "aa_same_order", HelpOrder: 1000},
	)
	t.Cleanup(func() {
		builtinSpecs = original
	})

	names := HelpDisplayOrder()
	for i := 1; i < len(names); i++ {
		prev, prevOK := specInBuiltinSpecs(names[i-1])
		curr, currOK := specInBuiltinSpecs(names[i])
		if !prevOK || !currOK {
			t.Fatalf("HelpDisplayOrder() returned unknown adjacent names %q/%q", names[i-1], names[i])
		}
		if prev.HelpOrder > curr.HelpOrder {
			t.Fatalf("HelpDisplayOrder() order %q(%d) before %q(%d), want ascending HelpOrder", prev.Name, prev.HelpOrder, curr.Name, curr.HelpOrder)
		}
		if prev.HelpOrder == curr.HelpOrder && prev.Name > curr.Name {
			t.Fatalf("HelpDisplayOrder() tie order %q before %q, want ascending name", prev.Name, curr.Name)
		}
	}

	aaIndex := indexOfName(names, "aa_same_order")
	zzIndex := indexOfName(names, "zz_same_order")
	if aaIndex == -1 || zzIndex == -1 {
		t.Fatalf("HelpDisplayOrder() = %v, want synthetic tie specs included", names)
	}
	if aaIndex > zzIndex {
		t.Fatalf("HelpDisplayOrder() tie placed aa_same_order at %d after zz_same_order at %d", aaIndex, zzIndex)
	}
}

func TestHelpSummary(t *testing.T) {
	summary, ok := HelpSummary("apply_patch")
	if !ok {
		t.Fatal("HelpSummary(\"apply_patch\") ok = false, want true")
	}
	if summary == "" {
		t.Fatal("HelpSummary(\"apply_patch\") returned empty summary")
	}

	if summary, ok := HelpSummary("gather_context"); ok || summary != "" {
		t.Fatalf("HelpSummary(\"gather_context\") = %q, %v; want empty, false for spec without summary", summary, ok)
	}
	if summary, ok := HelpSummary("unknown_tool"); ok || summary != "" {
		t.Fatalf("HelpSummary(\"unknown_tool\") = %q, %v; want empty, false for unknown tool", summary, ok)
	}
}

func specInBuiltinSpecs(name string) (Spec, bool) {
	for _, spec := range builtinSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return Spec{}, false
}

func indexOfName(names []string, name string) int {
	for i, got := range names {
		if got == name {
			return i
		}
	}
	return -1
}
