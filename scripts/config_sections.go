//go:build ignore

// config_sections.go は `go run scripts/config_sections.go scripts/gen-*.go`
// 互換を維持するための薄い shim。
//
// 現在の canonical source of truth は scripts/internal/configgen 側であり、
// この file には alias 以外のロジックを追加しない。
package main

import configgen "github.com/susugadx/xelyon-cli/scripts/internal/configgen"

type SectionInfo = configgen.SectionInfo
type CategoryInfo = configgen.CategoryInfo

var Sections = configgen.Sections
var SectionOrder = configgen.SectionOrder
var CategoryOrder = configgen.CategoryOrder
var SectionToCategory = configgen.SectionToCategory
var Categories = configgen.Categories

var orderedSectionsForCategory = configgen.OrderedSectionsForCategory
