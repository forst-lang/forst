package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func collectTypesFromSource(t *testing.T, src string) *TypeChecker {
	t.Helper()
	log := ast.SetupTestLogger(nil)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CollectTypes(nodes); err != nil {
		t.Fatalf("CollectTypes: %v", err)
	}
	return tc
}

func TestIsRuntimeEnsureTypeTarget(t *testing.T) {
	t.Parallel()
	src := `package main

type Sku = String.Min(1).Max(64)
type Password = String
type User = { name: String }
type Positive = Int.Min(1)
type Port = Positive.Max(65535)
type Loop = Loop.Min(1)

func main() {}
`
	tc := collectTypesFromSource(t, src)

	cases := []struct {
		name            string
		ident           ast.TypeIdent
		runtimeTarget   bool
		wantConstraints int
		wantCarrier     ast.TypeIdent
	}{
		{
			name:            "constrainedScalarAlias",
			ident:           "Sku",
			runtimeTarget:   true,
			wantConstraints: 2,
			wantCarrier:     ast.TypeString,
		},
		{
			name:          "bareNominalScalar",
			ident:         "Password",
			runtimeTarget: true,
		},
		{
			name:          "structuralShapeRejected",
			ident:         "User",
			runtimeTarget: false,
		},
		{
			name:            "inheritedConstrainedAlias",
			ident:           "Port",
			runtimeTarget:   true,
			wantConstraints: 2,
			wantCarrier:     ast.TypeInt,
		},
		{
			name:          "recursiveAliasTerminates",
			ident:         "Loop",
			runtimeTarget: false,
		},
	}

	for _, tcCase := range cases {
		t.Run(tcCase.name, func(t *testing.T) {
			gotRuntime := tc.isRuntimeEnsureTypeTarget(tcCase.ident)
			if gotRuntime != tcCase.runtimeTarget {
				t.Fatalf("isRuntimeEnsureTypeTarget(%s)=%v want %v", tcCase.ident, gotRuntime, tcCase.runtimeTarget)
			}
			assertion, ok := tc.ConstrainedScalarAliasAssertion(tcCase.ident)
			if tcCase.wantConstraints == 0 {
				if ok {
					t.Fatalf("ConstrainedScalarAliasAssertion(%s) unexpectedly ok: %+v", tcCase.ident, assertion)
				}
				if tcCase.wantCarrier == "" {
					return
				}
			}
			if tcCase.wantConstraints > 0 {
				if !ok || assertion == nil || len(assertion.Constraints) != tcCase.wantConstraints {
					t.Fatalf("ConstrainedScalarAliasAssertion(%s): ok=%v constraints=%v want %d",
						tcCase.ident, ok, assertion, tcCase.wantConstraints)
				}
			}
			if tcCase.wantCarrier != "" {
				carrier, cok := tc.carrierTypeForNamedType(ast.TypeNode{Ident: tcCase.ident, TypeKind: ast.TypeKindUserDefined})
				if !cok || carrier.Ident != tcCase.wantCarrier {
					t.Fatalf("carrier: ok=%v ident=%s want %s", cok, carrier.Ident, tcCase.wantCarrier)
				}
			}
		})
	}
}
