package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

type agentProjectPromptState struct {
	projectMapFileCount            int
	projectMapSymbolCount          int
	projectMap                     *repomap.ProjectMap
	projectMapRootPath             string
	projectMapIgnoreKey            string
	projectMapStateKey             string
	projectMapWatchDirs            []string
	projectMapBaseSection          string
	projectMapFocusSection         string
	projectMapSection              string
	projectMapBaseKey              string
	projectMapFocusKey             string
	projectMapDirty                bool
	projectInstructionBundle       *config.ProjectInstructionBundle
	projectInstructionBundleLoaded bool
	projectInstructionBundleKey    string
}

func (s *agentProjectPromptState) resetProjectMapState() {
	if s == nil {
		return
	}
	s.clearProjectMapState(true)
}

func (s *agentProjectPromptState) clearProjectMapState(dirty bool) {
	if s == nil {
		return
	}
	s.projectMap = nil
	s.projectMapRootPath = ""
	s.projectMapIgnoreKey = ""
	s.projectMapStateKey = ""
	s.projectMapWatchDirs = nil
	s.projectMapBaseSection = ""
	s.projectMapFocusSection = ""
	s.projectMapSection = ""
	s.projectMapBaseKey = ""
	s.projectMapFocusKey = ""
	s.projectMapFileCount = 0
	s.projectMapSymbolCount = 0
	s.projectMapDirty = dirty
}

func (s *agentProjectPromptState) hasProjectMapState() bool {
	if s == nil {
		return false
	}
	return s.projectMap != nil ||
		s.projectMapRootPath != "" ||
		s.projectMapIgnoreKey != "" ||
		s.projectMapStateKey != "" ||
		len(s.projectMapWatchDirs) != 0 ||
		s.projectMapBaseSection != "" ||
		s.projectMapFocusSection != "" ||
		s.projectMapSection != "" ||
		s.projectMapBaseKey != "" ||
		s.projectMapFocusKey != "" ||
		s.projectMapFileCount != 0 ||
		s.projectMapSymbolCount != 0
}
