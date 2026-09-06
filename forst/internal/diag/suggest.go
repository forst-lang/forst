package diag

import (
	"strings"
	"unicode"
)

// ClosestName returns the best candidate for a typo when confidence is high.
// Empty string means no safe suggestion (do not autofix).
func ClosestName(got string, candidates []string) string {
	got = strings.TrimSpace(got)
	if got == "" || len(candidates) == 0 {
		return ""
	}
	gotLower := strings.ToLower(got)
	type scored struct {
		name  string
		dist  int
		pref  bool // got is a prefix of candidate (e.g. Printl → Println)
		score int  // lower is better after dist
	}
	var best *scored
	tied := false
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" || c == got {
			continue
		}
		cLower := strings.ToLower(c)
		d := levenshtein(gotLower, cLower)
		maxLen := max(len(got), len(c))
		pref := strings.HasPrefix(cLower, gotLower) && len(got) >= 2
		if pref {
			d = min(d, 1)
		}
		if d > 2 || (maxLen > 0 && d*5 > maxLen*2) {
			continue
		}
		// Prefer prefix matches, then shorter remaining suffix (Printl→Println over Printlx).
		s := scored{name: c, dist: d, pref: pref, score: len(c) - len(got)}
		if best == nil {
			best = &s
			tied = false
			continue
		}
		if s.dist < best.dist {
			best = &s
			tied = false
			continue
		}
		if s.dist > best.dist {
			continue
		}
		// Same distance: prefer prefix; then smaller length delta; else tie → no autofix.
		if s.pref && !best.pref {
			best = &s
			tied = false
			continue
		}
		if !s.pref && best.pref {
			continue
		}
		if s.score < best.score {
			best = &s
			tied = false
			continue
		}
		if s.score > best.score {
			continue
		}
		tied = true
	}
	if best == nil || tied {
		return ""
	}
	return best.name
}

// FormatKnownList caps and joins names for a note line.
func FormatKnownList(label string, names []string, capN int) string {
	if len(names) == 0 {
		return ""
	}
	if capN <= 0 {
		capN = 8
	}
	shown := names
	suffix := ""
	if len(shown) > capN {
		shown = names[:capN]
		suffix = ", …"
	}
	return label + strings.Join(shown, ", ") + suffix
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// IsExportedGoName reports whether s looks like an exported Go identifier.
func IsExportedGoName(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsUpper(r)
}
