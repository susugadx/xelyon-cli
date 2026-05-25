package search

import (
	"reflect"
	"testing"
)

func assertGoSemanticShadowBundleEquivalent(t *testing.T, production *SymbolBundle, shadow *SymbolBundle, sectionKinds ...string) {
	t.Helper()
	if production == nil || shadow == nil {
		t.Fatalf("bundle comparison got production:%v shadow:%v", production, shadow)
	}
	if !reflect.DeepEqual(shadow.Identity, production.Identity) {
		t.Fatalf("shadow identity = %+v, want production %+v", shadow.Identity, production.Identity)
	}
	if !reflect.DeepEqual(shadow.Definition, production.Definition) {
		t.Fatalf("shadow definition = %+v, want production %+v", shadow.Definition, production.Definition)
	}
	if !reflect.DeepEqual(shadow.Diagnostics, production.Diagnostics) {
		t.Fatalf("shadow diagnostics = %+v, want production %+v", shadow.Diagnostics, production.Diagnostics)
	}
	if production.Impact == nil || shadow.Impact == nil {
		t.Fatalf("impact comparison got production:%v shadow:%v", production.Impact, shadow.Impact)
	}
	if shadow.Impact.RiskLevel != production.Impact.RiskLevel {
		t.Fatalf("shadow risk = %q, want production %q", shadow.Impact.RiskLevel, production.Impact.RiskLevel)
	}
	assertGoSemanticShadowRecommendedReadOrder(t, production, shadow)
	for _, kind := range sectionKinds {
		assertGoSemanticShadowSectionEquivalent(t, production, shadow, kind)
	}
}

func assertGoSemanticShadowEvidenceMatchesProductionImpact(t *testing.T, evidence *SemanticEvidence, impact *SymbolBundleImpact) {
	t.Helper()
	if evidence == nil {
		t.Fatal("SemanticShadowEvidence = nil")
	}
	if impact == nil {
		t.Fatal("production Impact = nil")
	}
	if evidence.RiskLevel != impact.RiskLevel {
		t.Fatalf("evidence RiskLevel = %q, want production %q", evidence.RiskLevel, impact.RiskLevel)
	}
	if got, want := goSemanticShadowReadKeys(evidence.RecommendedReads), goSemanticShadowReadKeys(impact.RecommendedReads); !reflect.DeepEqual(got, want) {
		t.Fatalf("evidence RecommendedReads = %v, want production %v", got, want)
	}
}

func assertGoSemanticShadowDiagnosticsEquivalent(t *testing.T, resolved symbolResolveResult) {
	t.Helper()
	if resolved.Bundle == nil || resolved.SemanticShadowBundle == nil {
		t.Fatalf("diagnostics comparison got production:%v shadow:%v", resolved.Bundle, resolved.SemanticShadowBundle)
	}
	if !reflect.DeepEqual(resolved.SemanticShadowBundle.Diagnostics, resolved.Bundle.Diagnostics) {
		t.Fatalf("shadow diagnostics = %+v, want production %+v", resolved.SemanticShadowBundle.Diagnostics, resolved.Bundle.Diagnostics)
	}
	if resolved.SemanticShadowEvidence == nil || resolved.SemanticShadowEvidence.Diagnostics == nil {
		t.Fatal("shadow evidence diagnostics = nil, want Go diagnostics")
	}
	if !reflect.DeepEqual(*resolved.SemanticShadowEvidence.Diagnostics, resolved.Bundle.Diagnostics) {
		t.Fatalf("shadow evidence diagnostics = %+v, want production %+v", *resolved.SemanticShadowEvidence.Diagnostics, resolved.Bundle.Diagnostics)
	}
}

func assertGoSemanticShadowSectionEquivalent(t *testing.T, production *SymbolBundle, shadow *SymbolBundle, kind string) {
	t.Helper()
	productionSection := symbolBundleSectionByKind(production, kind)
	shadowSection := symbolBundleSectionByKind(shadow, kind)
	if productionSection == nil || shadowSection == nil {
		t.Fatalf("section %q comparison got production:%v shadow:%v", kind, productionSection, shadowSection)
	}
	if productionSection.Total != shadowSection.Total || productionSection.More != shadowSection.More {
		t.Fatalf("section %q summary = total:%d more:%v, want total:%d more:%v", kind, shadowSection.Total, shadowSection.More, productionSection.Total, productionSection.More)
	}
	if !reflect.DeepEqual(shadowSection.Items, productionSection.Items) {
		t.Fatalf("section %q items = %+v, want production %+v", kind, shadowSection.Items, productionSection.Items)
	}
}

func assertGoSemanticShadowRecommendedReadOrder(t *testing.T, production *SymbolBundle, shadow *SymbolBundle) {
	t.Helper()
	if production == nil || production.Impact == nil || shadow == nil || shadow.Impact == nil {
		t.Fatalf("read comparison got production:%v shadow:%v", production, shadow)
	}
	got := goSemanticShadowReadKeys(shadow.Impact.RecommendedReads)
	want := goSemanticShadowReadKeys(production.Impact.RecommendedReads)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shadow RecommendedReads = %v, want production order %v", got, want)
	}
}

func assertGoSemanticShadowRecommendedReadKinds(t *testing.T, reads []SymbolBundleItem, kinds ...string) {
	t.Helper()
	if len(reads) != len(kinds) {
		t.Fatalf("RecommendedReads length = %d, want %d; reads = %+v", len(reads), len(kinds), reads)
	}
	for i, want := range kinds {
		if reads[i].Kind != want {
			t.Fatalf("RecommendedReads[%d].Kind = %q, want %q; reads = %+v", i, reads[i].Kind, want, reads)
		}
	}
}

type goSemanticShadowReadKey struct {
	kind string
	file string
	line int
}

func goSemanticShadowReadKeys(reads []SymbolBundleItem) []goSemanticShadowReadKey {
	keys := make([]goSemanticShadowReadKey, 0, len(reads))
	for _, read := range reads {
		keys = append(keys, goSemanticShadowReadKey{
			kind: read.Kind,
			file: read.File,
			line: read.Line,
		})
	}
	return keys
}
