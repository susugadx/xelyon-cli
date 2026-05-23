package jsast

import "testing"

func TestRequireBindingsWithParsedCollectsCommonJSBindings(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []ImportBinding
	}{
		{
			name: "destructured named require",
			src:  "const { buildUser: makeUser, buildTeam } = require('./build')\n",
			want: []ImportBinding{
				{Kind: ImportBindingNamed, Imported: "buildUser", Local: "makeUser", Source: "./build", Line: 1},
				{Kind: ImportBindingNamed, Imported: "buildTeam", Local: "buildTeam", Source: "./build", Line: 1},
			},
		},
		{
			name: "default require",
			src:  "const makeDefault = require('./default-build')\n",
			want: []ImportBinding{
				{Kind: ImportBindingDefault, Imported: "default", Local: "makeDefault", Source: "./default-build", Line: 1},
			},
		},
		{
			name: "named default require property",
			src:  "const { default: makeDefaultAlias } = require('./default-alias')\n",
			want: []ImportBinding{
				{Kind: ImportBindingNamed, Imported: "default", Local: "makeDefaultAlias", Source: "./default-alias", Line: 1},
			},
		},
		{
			name: "member named require",
			src:  "const makeUserMember = require('./member').buildUser\n",
			want: []ImportBinding{
				{Kind: ImportBindingNamed, Imported: "buildUser", Local: "makeUserMember", Source: "./member", Line: 1},
			},
		},
		{
			name: "nested destructuring is not top level",
			src:  "const { api: { buildUser: nestedBuildUser } } = require('./nested')\n",
		},
		{
			name: "non require call ignored",
			src:  "const ignored = requireAlias('./other')\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseBytes("src/app.js", []byte(tt.src))
			if err != nil {
				t.Fatalf("ParseBytes() error = %v", err)
			}
			defer parsed.Close()

			bindings := RequireBindingsWithParsed(parsed)

			if len(bindings) != len(tt.want) {
				t.Fatalf("bindings = %+v, want %+v", bindings, tt.want)
			}
			for i := range tt.want {
				if bindings[i].Kind != tt.want[i].Kind ||
					bindings[i].Imported != tt.want[i].Imported ||
					bindings[i].Local != tt.want[i].Local ||
					bindings[i].Source != tt.want[i].Source ||
					bindings[i].Line != tt.want[i].Line {
					t.Fatalf("binding %d = %+v, want %+v", i, bindings[i], tt.want[i])
				}
				if !ImportBindingCoversLine(bindings[i], tt.want[i].Line) {
					t.Fatalf("ImportBindingCoversLine(%+v, %d) = false", bindings[i], tt.want[i].Line)
				}
			}
			for _, binding := range bindings {
				if binding.Source == "./nested" {
					t.Fatalf("binding = %+v, did not want nested require destructuring binding", binding)
				}
			}
		})
	}
}
