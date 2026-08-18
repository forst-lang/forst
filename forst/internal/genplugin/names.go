package genplugin

import (
	"fmt"
	"strings"
)

// UniqueTypeName returns a stable identifier for type id.
// First collision-free use: TypeShortName(id).
// Second collision: PackageOfTypeID(id) + "_" + TypeShortName(id).
// Further collisions: sanitized full type id (dots → underscores).
func UniqueTypeName(id string, used map[string]int) string {
	base := TypeShortName(id)
	if base == "" {
		base = "Type"
	}
	n := used[base]
	used[base] = n + 1
	switch n {
	case 0:
		return base
	case 1:
		if pkg := PackageOfTypeID(id); pkg != "" {
			return fmt.Sprintf("%s_%s", pkg, base)
		}
		fallthrough
	default:
		return TSIdentifier(strings.ReplaceAll(id, ".", "_"))
	}
}
