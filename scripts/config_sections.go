//go:build ignore

// config_sections.go keeps the historical go run entrypoint shape while
// delegating the canonical metadata to scripts/internal/configgen.
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
