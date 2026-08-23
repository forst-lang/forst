package importlocal

import "fmt"

// SuggestAlias returns a valid, unused import alias for moduleID/importPath.
func SuggestAlias(moduleID, importPath string, taken map[string]struct{}) string {
	return SuggestAliasForKind(moduleID, importPath, TakenSet(taken), KindGo)
}

// SuggestNodeAlias returns a valid, unused JS import alias.
func SuggestNodeAlias(moduleID, importPath string, taken map[string]struct{}) string {
	return SuggestAliasForKind(moduleID, importPath, TakenSet(taken), KindBridge)
}

// SuggestAliasForKind returns a valid, unused import alias for the given kind.
func SuggestAliasForKind(moduleID, importPath string, taken TakenSet, kind Kind) string {
	seen := make(map[string]struct{})
	var candidates []string
	add := func(c string) {
		if c == "" {
			return
		}
		if _, ok := seen[c]; ok {
			return
		}
		seen[c] = struct{}{}
		candidates = append(candidates, c)
	}

	base := DefaultLocalFromModuleID(moduleID)
	sanitized := sanitizeSegment(base)
	add(base)
	add(sanitized)
	add(base + "Pkg")
	add(sanitized + "Pkg")

	for _, c := range candidates {
		if Validate(c, kind) != nil {
			continue
		}
		if taken.Has(c) {
			continue
		}
		return c
	}

	root := sanitized + "Pkg"
	if root == "Pkg" {
		root = "jsPkg"
	}
	if Validate(root, kind) != nil {
		root = "jsPkg"
	}
	for i := 2; i < 100; i++ {
		c := fmt.Sprintf("%s%d", root, i)
		if Validate(c, kind) != nil {
			continue
		}
		if taken.Has(c) {
			continue
		}
		return c
	}
	return root + "Mod"
}

// SuggestAliasWithValidator is deprecated; use SuggestAliasForKind.
func SuggestAliasWithValidator(moduleID, importPath string, taken map[string]struct{}, validate func(string) error) string {
	kind := KindGo
	if validate != nil {
		if validate("node") != nil {
			kind = KindBridge
		}
	}
	return SuggestAliasForKind(moduleID, importPath, TakenSet(taken), kind)
}
