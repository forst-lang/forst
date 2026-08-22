package importlocal

// TakenSet tracks import local names already in use.
type TakenSet map[string]struct{}

// Add records a local name as taken.
func (s TakenSet) Add(name string) {
	if name == "" {
		return
	}
	if s == nil {
		return
	}
	s[name] = struct{}{}
}

// Has reports whether name is already taken.
func (s TakenSet) Has(name string) bool {
	if s == nil || name == "" {
		return false
	}
	_, ok := s[name]
	return ok
}

// Clone returns a shallow copy of the set.
func (s TakenSet) Clone() TakenSet {
	if len(s) == 0 {
		return nil
	}
	out := make(TakenSet, len(s))
	for k := range s {
		out[k] = struct{}{}
	}
	return out
}

// Map returns the set as a map for APIs that still accept map[string]struct{}.
func (s TakenSet) Map() map[string]struct{} {
	if len(s) == 0 {
		return nil
	}
	return map[string]struct{}(s)
}
