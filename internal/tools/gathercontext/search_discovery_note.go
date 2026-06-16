package gathercontext

import "strings"

func appendSearchDiscoveryNote(discovery, note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return discovery
	}
	discovery = strings.TrimSpace(discovery)
	if discovery == "" {
		return note
	}
	return discovery + "\n" + note
}
