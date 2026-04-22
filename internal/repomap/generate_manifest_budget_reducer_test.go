package repomap

import (
	"reflect"
	"testing"
)

func TestNewProjectMapManifestBudgetReducer_InitialLimits(t *testing.T) {
	pm := &ProjectMap{
		GitStatus: make([]GitChange, 12),
	}
	topDirs := make([]string, 12)
	topFiles := make([]string, 11)
	priorityFiles := make([]string, 15)

	reducer := newProjectMapManifestBudgetReducer(pm, topDirs, topFiles, priorityFiles)

	want := manifestSectionLimits{
		dirLimit:      maxManifestTopLevelDirs,
		fileLimit:     maxManifestTopLevelFiles,
		priorityLimit: maxManifestPriorityFiles,
		changeLimit:   12,
	}
	if !reflect.DeepEqual(reducer.limits, want) {
		t.Fatalf("limits = %+v, want %+v", reducer.limits, want)
	}
}

func TestProjectMapManifestBudgetReducer_ShrinkOrder(t *testing.T) {
	reducer := &projectMapManifestBudgetReducer{
		limits: manifestSectionLimits{
			dirLimit:      1,
			fileLimit:     1,
			priorityLimit: 2,
			changeLimit:   2,
		},
	}

	var got []manifestSectionLimits
	for reducer.shrink() {
		got = append(got, reducer.limits)
	}

	want := []manifestSectionLimits{
		{dirLimit: 1, fileLimit: 1, priorityLimit: 1, changeLimit: 2},
		{dirLimit: 1, fileLimit: 1, priorityLimit: 0, changeLimit: 2},
		{dirLimit: 1, fileLimit: 1, priorityLimit: 0, changeLimit: 1},
		{dirLimit: 1, fileLimit: 1, priorityLimit: 0, changeLimit: 0},
		{dirLimit: 1, fileLimit: 0, priorityLimit: 0, changeLimit: 0},
		{dirLimit: 0, fileLimit: 0, priorityLimit: 0, changeLimit: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shrink order = %+v, want %+v", got, want)
	}
}
