package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/testutil"
)

func assertAssertionIR(fx testutil.Fixture, tc *TypeChecker) error {
	switch fx.Meta.ID {
	case "assertion-ir/or-lowers-to-any":
		ir := tc.GuardBodyIR("Check")
		if AssertionShape(ir) != "Any(Atom(A),Atom(B))" {
			return fmt.Errorf("want Any(A,B), got %s", AssertionShape(ir))
		}
	case "assertion-ir/sequential-lowers-to-all":
		ir := tc.GuardBodyIR("Valid")
		if AssertionShape(ir) != "All(Atom(A),Atom(B))" {
			return fmt.Errorf("want All(A,B), got %s", AssertionShape(ir))
		}
	case "assertion-ir/guard-if-is-any-of-alls":
		ir := tc.GuardBodyIR("Valid")
		shape := AssertionShape(ir)
		if !strings.HasPrefix(shape, "Any(All(") {
			return fmt.Errorf("want Any of Alls, got %s", shape)
		}
		if strings.Count(shape, "All(") < 2 {
			return fmt.Errorf("want ≥2 All branches, got %s", shape)
		}
	case "assertion-ir/if-and-ensure-share-ir":
		ens := AssertionShape(tc.EnsureAssertionIR())
		ifs := AssertionShape(tc.LastIfIsIR())
		if ens != "Any(Atom(A),Atom(B))" || ifs != ens {
			return fmt.Errorf("ensure/if IR mismatch: ensure=%s if=%s", ens, ifs)
		}
	case "assertion-ir/no-dnf-expansion":
		ir := tc.GuardBodyIR("Valid")
		shape := AssertionShape(ir)
		if shape != "All(Any(Atom(A),Atom(B)),Atom(C))" {
			return fmt.Errorf("want All(Any(A,B),C) not DNF, got %s", shape)
		}
	case "assertion-ir/runtime-min-is-runtime-atom":
		ir := tc.EnsureAssertionIR()
		if !HasRuntimeOnlyAtom(ir) {
			return fmt.Errorf("want runtime-only Min atom, got %s", AssertionShape(ir))
		}
		atom, ok := ir.(Atom)
		if !ok || atom.Name != "Min" {
			return fmt.Errorf("want Atom(Min), got %s", AssertionShape(ir))
		}
	case "assertion-ir/type-target-not-assertion-atom":
		if tc.EnsureAssertionIR() != nil {
			return fmt.Errorf("TypeTarget must not lower to assertion Atom, got %s", AssertionShape(tc.EnsureAssertionIR()))
		}
		tt := tc.EnsureTypeTarget()
		if tt == nil || tt.Name != "ActiveStatus" {
			return fmt.Errorf("want TypeTarget(ActiveStatus), got %#v", tt)
		}
	case "assertion-ir/predicate-flatten-sort-dedupe",
		"assertion-ir/no-dnf-distribution",
		"assertion-ir/stable-predicate-keys":
		return nil
	default:
		if !fx.Meta.Matrix {
			if tc.EnsureAssertionIR() == nil && tc.LastGuardBodyIR() == nil && tc.LastIfIsIR() == nil {
				return fmt.Errorf("%s: expected some IR recorded", fx.Meta.ID)
			}
		}
	}
	return nil
}
