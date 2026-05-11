package search

import "testing"

func TestStructuredTypeScriptImpactTargetPathPolicy(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		wantTarget     bool
		wantSuffix     string
		wantStructured bool
		wantImpl       bool
		wantDecl       bool
		wantSource     bool
		wantSourceImpl bool
		wantSourceDecl bool
	}{
		{
			name:           "ts implementation",
			path:           "src/build.ts",
			wantTarget:     true,
			wantSuffix:     ".ts",
			wantStructured: true,
			wantImpl:       true,
			wantSource:     true,
			wantSourceImpl: true,
		},
		{
			name:           "declaration",
			path:           "src/build.d.ts",
			wantTarget:     true,
			wantSuffix:     ".d.ts",
			wantStructured: true,
			wantDecl:       true,
			wantSource:     true,
			wantSourceDecl: true,
		},
		{
			name:           "tsx implementation",
			path:           "src/view.tsx",
			wantTarget:     true,
			wantSuffix:     ".tsx",
			wantStructured: true,
			wantImpl:       true,
			wantSource:     true,
			wantSourceImpl: true,
		},
		{
			name: "cts is outside TypeScript structured impact targets",
			path: "src/build.cts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, ok := structuredTypeScriptImpactTargetForPath(tt.path)
			if ok != tt.wantTarget {
				t.Fatalf("structuredTypeScriptImpactTargetForPath(%q) ok = %v, want %v", tt.path, ok, tt.wantTarget)
			}
			if ok {
				if target.suffix != tt.wantSuffix {
					t.Fatalf("target suffix = %q, want %q", target.suffix, tt.wantSuffix)
				}
				if target.structuredImpact != tt.wantStructured {
					t.Fatalf("target structuredImpact = %v, want %v", target.structuredImpact, tt.wantStructured)
				}
				if target.implementation != tt.wantImpl {
					t.Fatalf("target implementation = %v, want %v", target.implementation, tt.wantImpl)
				}
				if target.declaration != tt.wantDecl {
					t.Fatalf("target declaration = %v, want %v", target.declaration, tt.wantDecl)
				}
			}

			if got := isTypeScriptSourceFilePath(tt.path); got != tt.wantSource {
				t.Fatalf("isTypeScriptSourceFilePath(%q) = %v, want %v", tt.path, got, tt.wantSource)
			}
			if got := isTypeScriptImplementationFilePath(tt.path); got != tt.wantSourceImpl {
				t.Fatalf("isTypeScriptImplementationFilePath(%q) = %v, want %v", tt.path, got, tt.wantSourceImpl)
			}
			if got := isTypeScriptDeclarationFilePath(tt.path); got != tt.wantSourceDecl {
				t.Fatalf("isTypeScriptDeclarationFilePath(%q) = %v, want %v", tt.path, got, tt.wantSourceDecl)
			}
		})
	}
}

func TestNormalizeStructuredTypeScriptImpactOptionsUsesTargetPolicy(t *testing.T) {
	tests := []struct {
		name            string
		opts            SearchOptions
		wantOK          bool
		wantFileType    string
		wantFilePattern string
	}{
		{
			name:         "ts file filter is structured",
			opts:         SearchOptions{FileType: "ts"},
			wantOK:       true,
			wantFileType: "ts",
		},
		{
			name:         "tsx file filter is structured",
			opts:         SearchOptions{FileType: "tsx"},
			wantOK:       true,
			wantFileType: "tsx",
		},
		{
			name: "typescript file filter stays fallback",
			opts: SearchOptions{FileType: "typescript"},
		},
		{
			name:            "ts file pattern is structured",
			opts:            SearchOptions{FilePattern: "*.ts"},
			wantOK:          true,
			wantFilePattern: "*.ts",
		},
		{
			name:            "declaration file pattern is structured",
			opts:            SearchOptions{FilePattern: "**/*.d.ts"},
			wantOK:          true,
			wantFilePattern: "**/*.d.ts",
		},
		{
			name:            "tsx file pattern is structured",
			opts:            SearchOptions{FilePattern: "*.tsx"},
			wantOK:          true,
			wantFilePattern: "*.tsx",
		},
		{
			name:   "ts path is structured",
			opts:   SearchOptions{Path: "src/build.ts"},
			wantOK: true,
		},
		{
			name:   "tsx path is structured",
			opts:   SearchOptions{Path: "src/view.tsx"},
			wantOK: true,
		},
		{
			name:   "declaration path is structured",
			opts:   SearchOptions{Path: "src/build.d.ts"},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeStructuredTypeScriptImpactOptions(tt.opts)
			if ok != tt.wantOK {
				t.Fatalf("normalizeStructuredTypeScriptImpactOptions() ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.FileType != tt.wantFileType {
				t.Fatalf("FileType = %q, want %q", got.FileType, tt.wantFileType)
			}
			if got.FilePattern != tt.wantFilePattern {
				t.Fatalf("FilePattern = %q, want %q", got.FilePattern, tt.wantFilePattern)
			}
		})
	}
}

func TestStructuredTypeScriptImpactTargetPairedDeclarationPolicy(t *testing.T) {
	implementationPath, ok := structuredTypeScriptDeclarationImplementationPath("src/build.d.ts")
	if !ok {
		t.Fatal("structuredTypeScriptDeclarationImplementationPath() ok = false, want true")
	}
	if implementationPath != "src/build.ts" {
		t.Fatalf("implementation path = %q, want src/build.ts", implementationPath)
	}
	implementationPaths := structuredTypeScriptDeclarationImplementationPaths("src/build.d.ts")
	if len(implementationPaths) != 2 || implementationPaths[0] != "src/build.ts" || implementationPaths[1] != "src/build.tsx" {
		t.Fatalf("implementation paths = %v, want src/build.ts and src/build.tsx", implementationPaths)
	}

	preferred := preferStructuredTypeScriptImplementationDefs([]genericSymbolDef{
		{Name: "buildUser", File: "src/build.ts", Line: 1},
		{Name: "buildUser", File: "src/build.d.ts", Line: 1},
	})
	if len(preferred.defs) != 1 || preferred.defs[0].File != "src/build.ts" {
		t.Fatalf("preferred defs = %+v, want only src/build.ts", preferred.defs)
	}
	if len(preferred.suppressedDeclarationDefs) != 1 || preferred.suppressedDeclarationDefs[0].File != "src/build.d.ts" {
		t.Fatalf("suppressed declarations = %+v, want src/build.d.ts", preferred.suppressedDeclarationDefs)
	}

	tsxPreferred := preferStructuredTypeScriptImplementationDefs([]genericSymbolDef{
		{Name: "Button", File: "src/Button.tsx", Line: 1},
		{Name: "Button", File: "src/Button.d.ts", Line: 1},
	})
	if len(tsxPreferred.defs) != 1 || tsxPreferred.defs[0].File != "src/Button.tsx" {
		t.Fatalf("tsx preferred defs = %+v, want only src/Button.tsx", tsxPreferred.defs)
	}
	if len(tsxPreferred.suppressedDeclarationDefs) != 1 || tsxPreferred.suppressedDeclarationDefs[0].File != "src/Button.d.ts" {
		t.Fatalf("tsx suppressed declarations = %+v, want src/Button.d.ts", tsxPreferred.suppressedDeclarationDefs)
	}
}

func TestStructuredTypeScriptImpactTargetNearbyTestPolicy(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts":        "export function buildUser(id: string) { return id }\n",
		"src/build.test.ts":   "buildUser('test')\n",
		"src/build.spec.ts":   "describe('build', () => {})\n",
		"src/types.d.ts":      "export interface BuildOptions { id: string }\n",
		"src/types.d.test.ts": "describe('types', () => {})\n",
		"src/view.tsx":        "export function View() { return <div /> }\n",
		"src/view.test.tsx":   "View()\n",
	})
	opts := SearchOptions{Path: dir, FileType: "ts", InvocationCWD: dir}

	implementationTests := findNearbyTypeScriptTests(genericSymbolDef{File: "src/build.ts"}, opts, nil)
	if len(implementationTests) != 2 {
		t.Fatalf("implementation nearby tests = %+v, want build.test.ts and build.spec.ts", implementationTests)
	}
	for _, want := range []string{"src/build.test.ts", "src/build.spec.ts"} {
		if !genericRefsContainFile(implementationTests, want) {
			t.Fatalf("implementation nearby tests = %+v, want %s", implementationTests, want)
		}
	}

	declarationTests := findNearbyTypeScriptTests(genericSymbolDef{File: "src/types.d.ts"}, opts, nil)
	if len(declarationTests) != 0 {
		t.Fatalf("declaration nearby tests = %+v, want none", declarationTests)
	}

	tsxOpts := opts
	tsxOpts.FileType = "tsx"
	tsxTests := findNearbyTypeScriptTests(genericSymbolDef{File: "src/view.tsx"}, tsxOpts, nil)
	if len(tsxTests) != 1 || tsxTests[0].File != "src/view.test.tsx" {
		t.Fatalf("tsx nearby tests = %+v, want src/view.test.tsx", tsxTests)
	}
}

func genericRefsContainFile(refs []genericSymbolRef, file string) bool {
	for _, ref := range refs {
		if ref.File == file {
			return true
		}
	}
	return false
}
