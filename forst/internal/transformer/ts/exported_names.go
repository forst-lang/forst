package transformerts

import "sort"

// sortDedupeStrings returns a sorted copy of s with duplicates and empty strings removed.
func sortDedupeStrings(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	var out []string
	for _, x := range s {
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

// CollectInvokeTypeNames gathers type identifiers referenced by invoke function signatures.
func CollectInvokeTypeNames(outputs []*TypeScriptOutput) []string {
	var names []string
	for _, out := range outputs {
		names = append(names, out.ExportedTypeNames...)
		for _, fn := range out.Functions {
			names = append(names, fn.ReturnType)
			for _, param := range fn.Parameters {
				names = append(names, param.Type)
			}
		}
	}
	return sortDedupeStrings(names)
}
