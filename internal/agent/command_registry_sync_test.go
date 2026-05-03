package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
)

func TestSpecialCommandRegistry_SyncedWithCatalog(t *testing.T) {
	surfaces := []commandcatalog.CommandSurface{
		commandcatalog.CommandSurfaceClassic,
		commandcatalog.CommandSurfaceTUI,
	}
	for _, surface := range surfaces {
		t.Run(string(surface), func(t *testing.T) {
			registry := specialCommandRegistry(&Agent{}, surface)
			registered := make(map[string]struct{}, len(registry))
			for name := range registry {
				registered[name] = struct{}{}
			}

			expected := make(map[string]struct{})
			for _, cmd := range commandcatalog.CommandsForSurface(surface) {
				if cmd.EffectiveOwner() != commandcatalog.CommandOwnerAgent {
					continue
				}
				expected[cmd.Name] = struct{}{}
				for _, alias := range cmd.Aliases {
					expected[alias] = struct{}{}
				}
			}

			for name := range expected {
				if _, ok := registered[name]; !ok {
					t.Fatalf("specialCommandRegistry missing command %q for surface=%s", name, surface)
				}
			}
		})
	}
}
