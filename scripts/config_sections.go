//go:build ignore

// config_sections.go は `go run scripts/config_sections.go scripts/gen-*.go`
// 互換を維持するための薄い shim。
//
// 現在の canonical source of truth は scripts/internal/configmeta 側であり、
// この file には alias 以外のロジックを追加しない。
package main

import "github.com/susugadx/xelyon-cli/scripts/internal/configmeta"

type SectionInfo = configmeta.SectionInfo
type CategoryInfo = configmeta.CategoryInfo

var Sections = configmeta.Sections
var SectionOrder = configmeta.SectionOrder
var CategoryOrder = configmeta.CategoryOrder
var SectionToCategory = configmeta.SectionToCategory
var Categories = configmeta.Categories

var orderedSectionsForCategory = configmeta.OrderedSectionsForCategory
