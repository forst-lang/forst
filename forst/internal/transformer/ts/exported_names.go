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
