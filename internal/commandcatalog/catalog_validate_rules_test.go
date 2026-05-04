package commandcatalog

import (
	"strings"
	"testing"
)

func TestValidateCommands_RejectsInvalidCommandTokenStyle(t *testing.T) {
	commands := []CommandInfo{
		{Name: "/Good", Surfaces: []CommandSurface{CommandSurfaceTUI}},
	}

	err := ValidateCommands(commands)
	if err == nil {
		t.Fatal("ValidateCommands() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "command name must match") {
		t.Fatalf("ValidateCommands() error = %v, want command token style error", err)
	}
}

func TestValidateCommands_RejectsInvalidAliasTokenStyle(t *testing.T) {
	commands := []CommandInfo{
		{
			Name:     "/status",
			Aliases:  []string{"/Stats"},
			Surfaces: []CommandSurface{CommandSurfaceTUI},
		},
	}

	err := ValidateCommands(commands)
	if err == nil {
		t.Fatal("ValidateCommands() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "alias \"/Stats\" must match") {
		t.Fatalf("ValidateCommands() error = %v, want alias token style error", err)
	}
}

func TestValidateCommands_RejectsDuplicateDiscoverableSortWeightPerSurface(t *testing.T) {
	commands := []CommandInfo{
		{
			Name:         "/a",
			Surfaces:     []CommandSurface{CommandSurfaceTUI},
			Discoverable: true,
			SortWeight:   10,
		},
		{
			Name:         "/b",
			Surfaces:     []CommandSurface{CommandSurfaceTUI},
			Discoverable: true,
			SortWeight:   10,
		},
	}

	err := ValidateCommands(commands)
	if err == nil {
		t.Fatal("ValidateCommands() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "duplicate discoverable sort weight 10") {
		t.Fatalf("ValidateCommands() error = %v, want duplicate sort weight error", err)
	}
}

func TestValidateCommands_AllowsUniqueDiscoverableSortWeights(t *testing.T) {
	commands := []CommandInfo{
		{
			Name:         "/a",
			Surfaces:     []CommandSurface{CommandSurfaceTUI},
			Discoverable: true,
			SortWeight:   10,
		},
		{
			Name:         "/b",
			Surfaces:     []CommandSurface{CommandSurfaceTUI},
			Discoverable: true,
			SortWeight:   20,
		},
	}

	if err := ValidateCommands(commands); err != nil {
		t.Fatalf("ValidateCommands() error = %v, want nil", err)
	}
}
