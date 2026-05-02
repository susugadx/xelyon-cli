package configgen

type compiledExampleFilterSpec struct {
	allowedSections map[string]bool
	sectionFilters  map[string]sectionFieldFilterSpec
}

type sectionFieldFilterSpec struct {
	mode           ExampleFilterMode
	topLevelFields map[string]bool
	fieldTree      *fieldPathNode
	omittedFields  map[string]bool
}

func buildExampleFilterSpec(sections map[string]SectionInfo) compiledExampleFilterSpec {
	spec := compiledExampleFilterSpec{
		allowedSections: make(map[string]bool, len(sections)),
		sectionFilters:  make(map[string]sectionFieldFilterSpec, len(sections)),
	}
	for sectionName, info := range sections {
		spec.allowedSections[sectionName] = true
		if len(info.Fields) == 0 {
			continue
		}
		mode := resolveExampleFilterMode(info)
		omitted := cloneBoolMap(info.Example.OmittedFields)
		filterSpec := sectionFieldFilterSpec{
			mode:          mode,
			omittedFields: omitted,
		}
		if mode == ExampleFilterModeFields {
			topLevelFields := make(map[string]bool, len(info.Fields))
			for fieldName := range info.Fields {
				topLevelFields[fieldName] = true
			}
			filterSpec.topLevelFields = topLevelFields
			filterSpec.fieldTree = buildFieldPathTree(info.Fields)
		}
		spec.sectionFilters[sectionName] = filterSpec
	}
	return spec
}

func resolveExampleFilterMode(info SectionInfo) ExampleFilterMode {
	if info.Example.FilterMode != "" {
		return info.Example.FilterMode
	}
	if sectionHasOnlyMapLikeFieldTypes(info.FieldTypes) {
		return ExampleFilterModeKeepAll
	}
	return ExampleFilterModeFields
}

func sectionHasOnlyMapLikeFieldTypes(fieldTypes map[string]string) bool {
	if len(fieldTypes) == 0 {
		return false
	}
	for _, fieldType := range fieldTypes {
		if fieldType != "structmap" && fieldType != "map" {
			return false
		}
	}
	return true
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]bool, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
