package typechecker

import (
	"fmt"
	"strings"
)

// AccessStepKind distinguishes field, coarse index, and pointer dereference steps.
type AccessStepKind uint8

const (
	// AccessField is a named field selection (x.f).
	AccessField AccessStepKind = iota
	// AccessIndexAny is a coarse element path (x[*]).
	AccessIndexAny
	// AccessDeref is a pointer dereference (*x / x's pointee slot path).
	AccessDeref
)

// AccessStep is one hop on an AccessPath.
type AccessStep struct {
	Kind  AccessStepKind
	Field string // set when Kind == AccessField
}

// AccessPath is a root SymbolID plus zero or more access steps.
type AccessPath struct {
	Root  SymbolID
	Steps []AccessStep
}

// PathKey is a stable interning key for AccessPath.
func (p AccessPath) PathKey() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d", p.Root)
	for _, s := range p.Steps {
		switch s.Kind {
		case AccessField:
			b.WriteString(".f:")
			b.WriteString(s.Field)
		case AccessIndexAny:
			b.WriteString("[*]")
		case AccessDeref:
			b.WriteString(".*")
		default:
			fmt.Fprintf(&b, ".?%d", s.Kind)
		}
	}
	return b.String()
}

// Equal reports structural equality (same root and steps).
func (p AccessPath) Equal(o AccessPath) bool {
	if p.Root != o.Root || len(p.Steps) != len(o.Steps) {
		return false
	}
	for i := range p.Steps {
		if p.Steps[i].Kind != o.Steps[i].Kind || p.Steps[i].Field != o.Steps[i].Field {
			return false
		}
	}
	return true
}

// CloneSteps returns a copy of Steps (nil-safe).
func (p AccessPath) CloneSteps() []AccessStep {
	if len(p.Steps) == 0 {
		return nil
	}
	out := make([]AccessStep, len(p.Steps))
	copy(out, p.Steps)
	return out
}

// PathInterner canonicalizes AccessPath values by PathKey.
type PathInterner struct {
	byKey map[string]*AccessPath
}

// NewPathInterner creates an empty interner.
func NewPathInterner() *PathInterner {
	return &PathInterner{byKey: make(map[string]*AccessPath)}
}

// Intern returns the canonical AccessPath for p (shared pointer).
func (pi *PathInterner) Intern(p AccessPath) *AccessPath {
	if pi == nil {
		cp := p
		cp.Steps = p.CloneSteps()
		return &cp
	}
	if pi.byKey == nil {
		pi.byKey = make(map[string]*AccessPath)
	}
	key := p.PathKey()
	if existing, ok := pi.byKey[key]; ok {
		return existing
	}
	cp := &AccessPath{Root: p.Root, Steps: p.CloneSteps()}
	pi.byKey[key] = cp
	return cp
}

// Len returns how many distinct paths are interned.
func (pi *PathInterner) Len() int {
	if pi == nil {
		return 0
	}
	return len(pi.byKey)
}
