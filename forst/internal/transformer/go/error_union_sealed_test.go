package transformergo

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

func TestFlattenUnionTypeParams_emptyAndNonUnion(t *testing.T) {
	t.Parallel()
	if flattenUnionTypeParams(ast.TypeNode{Ident: ast.TypeString}) != nil {
		t.Fatal("expected nil for non-union")
	}
	if flattenUnionTypeParams(ast.TypeNode{Ident: ast.TypeUnion}) != nil {
		t.Fatal("expected nil for empty union params")
	}
}

func TestFlattenUnionTypeParams_flatAndNested(t *testing.T) {
	t.Parallel()
	a := ast.TypeNode{Ident: ast.TypeIdent("A")}
	b := ast.TypeNode{Ident: ast.TypeIdent("B")}
	c := ast.TypeNode{Ident: ast.TypeIdent("C")}
	flat := ast.NewUnionType(a, b, c)
	got := flattenUnionTypeParams(flat)
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %#v", len(got), got)
	}
	nested := ast.NewUnionType(ast.NewUnionType(a, b), c)
	got2 := flattenUnionTypeParams(nested)
	if len(got2) != 3 {
		t.Fatalf("nested: len=%d want 3", len(got2))
	}
}

// TestFlattenUnionTypeParams_unionChildRecurses covers the recursive branch when a union member is
// itself TypeUnion with TypeParams. ast.NewUnionType normalizes nested unions, so this must be built manually.
func TestFlattenUnionTypeParams_unionChildRecurses(t *testing.T) {
	t.Parallel()
	a := ast.TypeNode{Ident: ast.TypeIdent("A")}
	b := ast.TypeNode{Ident: ast.TypeIdent("B")}
	c := ast.TypeNode{Ident: ast.TypeIdent("C")}
	inner := ast.TypeNode{
		Ident:      ast.TypeUnion,
		TypeParams: []ast.TypeNode{a, b},
		TypeKind:   ast.TypeKindBuiltin,
	}
	outer := ast.TypeNode{
		Ident:      ast.TypeUnion,
		TypeParams: []ast.TypeNode{inner, c},
		TypeKind:   ast.TypeKindBuiltin,
	}
	got := flattenUnionTypeParams(outer)
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %#v", len(got), got)
	}
	for i, want := range []ast.TypeIdent{"A", "B", "C"} {
		if got[i].Ident != want {
			t.Fatalf("got[%d].Ident=%q want %q", i, got[i].Ident, want)
		}
	}
}

func TestFlattenUnionTypeParams_balancedQuartetOrder(t *testing.T) {
	t.Parallel()
	a := ast.TypeNode{Ident: ast.TypeIdent("A")}
	b := ast.TypeNode{Ident: ast.TypeIdent("B")}
	c := ast.TypeNode{Ident: ast.TypeIdent("C")}
	d := ast.TypeNode{Ident: ast.TypeIdent("D")}
	// ((A|B)|(C|D)) — still a single chain of TypeUnion leaves
	u := ast.NewUnionType(ast.NewUnionType(a, b), ast.NewUnionType(c, d))
	got := flattenUnionTypeParams(u)
	if len(got) != 4 {
		t.Fatalf("len=%d want 4: %#v", len(got), got)
	}
	for i, want := range []ast.TypeIdent{"A", "B", "C", "D"} {
		if got[i].Ident != want {
			t.Fatalf("got[%d].Ident=%q want %q", i, got[i].Ident, want)
		}
	}
}

func TestSealedUnionMethodName(t *testing.T) {
	t.Parallel()
	if sealedUnionMethodName("ErrKind") != "isErrKind" {
		t.Fatalf("got %q", sealedUnionMethodName("ErrKind"))
	}
	if sealedUnionMethodName("E") != "isE" {
		t.Fatalf("got %q", sealedUnionMethodName("E"))
	}
	if sealedUnionMethodName("") != "isUnion" {
		t.Fatalf("got %q", sealedUnionMethodName(""))
	}
	if sealedUnionMethodName("MyUnion") != "isMyUnion" {
		t.Fatalf("got %q", sealedUnionMethodName("MyUnion"))
	}
}

func TestNominalErrorUnionMembers(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	log.SetOutput(io.Discard)
	tc := typechecker.New(log, false)

	payload := ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}
	tc.Defs["NomA"] = ast.TypeDefNode{Ident: "NomA", Expr: ast.TypeDefErrorExpr{Payload: payload}}
	tc.Defs["NomB"] = ast.TypeDefNode{Ident: "NomB", Expr: ast.TypeDefErrorExpr{Payload: payload}}
	str := ast.TypeString
	tc.Defs["AliasStr"] = ast.TypeDefNode{
		Ident: "AliasStr",
		Expr: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &str}},
	}
	tc.Defs["Rec"] = ast.TypeDefNode{
		Ident: "Rec",
		Expr:  ast.TypeDefShapeExpr{Shape: payload},
	}
	tc.Defs["NomC"] = ast.TypeDefNode{Ident: "NomC", Expr: ast.TypeDefErrorExpr{Payload: payload}}

	t.Run("two_nominals_ok", func(t *testing.T) {
		members := []ast.TypeNode{{Ident: "NomA"}, {Ident: "NomB"}}
		ids, ok := nominalErrorUnionMembers(tc, members)
		if !ok || len(ids) != 2 || ids[0] != "NomA" || ids[1] != "NomB" {
			t.Fatalf("got ok=%v ids=%v", ok, ids)
		}
	})
	t.Run("builtin_error_rejected", func(t *testing.T) {
		_, ok := nominalErrorUnionMembers(tc, []ast.TypeNode{{Ident: ast.TypeError}, {Ident: "NomA"}})
		if ok {
			t.Fatal("expected false when mixing built-in Error")
		}
	})
	t.Run("non_nominal_rejected", func(t *testing.T) {
		_, ok := nominalErrorUnionMembers(tc, []ast.TypeNode{{Ident: "NomA"}, {Ident: "AliasStr"}})
		if ok {
			t.Fatal("expected false for non-nominal member")
		}
	})
	t.Run("missing_def_rejected", func(t *testing.T) {
		_, ok := nominalErrorUnionMembers(tc, []ast.TypeNode{{Ident: "NomA"}, {Ident: "MissingType"}})
		if ok {
			t.Fatal("expected false for unknown type")
		}
	})
	t.Run("single_member_false", func(t *testing.T) {
		_, ok := nominalErrorUnionMembers(tc, []ast.TypeNode{{Ident: "NomA"}})
		if ok {
			t.Fatal("expected false for single member (need >= 2)")
		}
	})
	t.Run("empty_ident_rejected", func(t *testing.T) {
		_, ok := nominalErrorUnionMembers(tc, []ast.TypeNode{{Ident: ""}, {Ident: "NomA"}})
		if ok {
			t.Fatal("expected false")
		}
	})
	t.Run("shape_typedef_rejected", func(t *testing.T) {
		_, ok := nominalErrorUnionMembers(tc, []ast.TypeNode{{Ident: "NomA"}, {Ident: "Rec"}})
		if ok {
			t.Fatal("expected false when a member is a non-error shape typedef")
		}
	})
	t.Run("three_nominals_ok", func(t *testing.T) {
		members := []ast.TypeNode{{Ident: "NomA"}, {Ident: "NomB"}, {Ident: "NomC"}}
		ids, ok := nominalErrorUnionMembers(tc, members)
		if !ok || len(ids) != 3 {
			t.Fatalf("got ok=%v ids=%v", ok, ids)
		}
	})
}

func TestEmitSealedUnionMemberMethodIfNew_dedupes(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	log.SetOutput(io.Discard)
	tc := typechecker.New(log, false)
	tr := New(tc, log)
	tr.emitSealedUnionMemberMethodIfNew("ParseError", "isErrKind")
	tr.emitSealedUnionMemberMethodIfNew("ParseError", "isErrKind")
	tr.emitSealedUnionMemberMethodIfNew("IoError", "isErrKind")
	if len(tr.Output.functions) != 2 {
		t.Fatalf("want 2 unique funcs, got %d", len(tr.Output.functions))
	}
}

func TestEmitSealedUnionMemberMethodIfNew_sameReceiverDistinctMethods(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	log.SetOutput(io.Discard)
	tc := typechecker.New(log, false)
	tr := New(tc, log)
	tr.emitSealedUnionMemberMethodIfNew("ParseError", "isErrKind")
	tr.emitSealedUnionMemberMethodIfNew("ParseError", "isAltSeal")
	if len(tr.Output.functions) != 2 {
		t.Fatalf("want 2 funcs (same receiver, different seal method names), got %d", len(tr.Output.functions))
	}
}

func TestPipeline_nominalErrorUnion_sealedGoInterface(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "..", "..", "examples", "in", "union_error_types.ft")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	out := compileForstPipeline(t, string(src))
	for _, needle := range []string{
		"type ErrKind interface",
		"isErrKind()",
		"func (ParseError) isErrKind()",
		"func (IoError) isErrKind()",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("generated Go missing %q\n%s", needle, out)
		}
	}
	if strings.Contains(out, "type ErrKind error") {
		t.Fatalf("expected sealed interface, not error alias; got:\n%s", out)
	}
}

func TestPipeline_errorUnion_table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		src          string
		mustContain  []string
		mustNotContain []string
	}{
		{
			name: "sealed_three_nominal_errors",
			src: `package main
error A { code: Int }
error B { path: String }
error C { n: Int }
type U = A | B | C
func main() {}
`,
			mustContain: []string{
				"type U interface",
				"isU()",
				"func (A) isU()",
				"func (B) isU()",
				"func (C) isU()",
			},
			mustNotContain: []string{"type U error"},
		},
		{
			name: "builtin_error_mixed_with_nominal_not_sealed",
			src: `package main
error E { code: Int }
type U = Error | E
func main() {}
`,
			mustContain:    []string{"type U error"},
			mustNotContain: []string{"type U interface", "isU()"},
		},
		{
			name: "non_error_union_string_int",
			src: `package main
type U = String | Int
func main() {}
`,
			mustContain:    []string{"type U any"},
			mustNotContain: []string{"type U interface", "type U error"},
		},
		{
			name: "two_sealed_typedefs_distinct_methods",
			src: `package main
error P { code: Int }
error Q { n: Int }
type First = P | Q
type Second = P | Q
func main() {}
`,
			mustContain: []string{
				"type First interface",
				"isFirst()",
				"type Second interface",
				"isSecond()",
				"func (P) isFirst()",
				"func (Q) isFirst()",
				"func (P) isSecond()",
				"func (Q) isSecond()",
			},
		},
		{
			name: "leading_pipe_same_as_infix",
			src: `package main
error A { code: Int }
error B { n: Int }
type U =
| A
| B
func main() {}
`,
			mustContain: []string{
				"type U interface",
				"func (A) isU()",
				"func (B) isU()",
			},
		},
		{
			name: "sealed_four_nominal_errors",
			src: `package main
error A { code: Int }
error B { path: String }
error C { n: Int }
error D { k: Int }
type U = A | B | C | D
func main() {}
`,
			mustContain: []string{
				"type U interface",
				"func (A) isU()",
				"func (B) isU()",
				"func (C) isU()",
				"func (D) isU()",
				"// U is a closed union of nominal errors",
			},
			mustNotContain: []string{"type U error"},
		},
		{
			name: "sealed_emits_doc_comment",
			src: `package main
error X { code: Int }
error Y { n: Int }
type Kind = X | Y
func main() {}
`,
			mustContain: []string{
				"// Kind is a closed union of nominal errors",
				"type Kind interface",
				"func (X) isKind()",
				"func (Y) isKind()",
			},
		},
		{
			name: "builtin_error_or_string_not_sealed",
			src: `package main
type U = Error | String
func main() {}
`,
			mustContain: []string{"type U any"},
			mustNotContain: []string{
				"type U interface",
				"isU()",
			},
		},
		{
			name: "nominal_error_or_shape_record_not_sealed",
			src: `package main
error E { code: Int }
type R = { n: Int }
type U = E | R
func main() {}
`,
			mustContain: []string{"type U any"},
			mustNotContain: []string{
				"type U interface",
				"func (E) isU()",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := compileForstPipeline(t, tt.src)
			for _, s := range tt.mustContain {
				if !strings.Contains(out, s) {
					t.Fatalf("missing %q\n%s", s, out)
				}
			}
			for _, s := range tt.mustNotContain {
				if strings.Contains(out, s) {
					t.Fatalf("must not contain %q\n%s", s, out)
				}
			}
		})
	}
}

func TestTransformTypeDef_twice_no_duplicate_seal_methods(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	log.SetOutput(io.Discard)
	src := `package main
error A { code: Int }
error B { n: Int }
type U = A | B
func main() {}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	var udef ast.TypeDefNode
	for _, n := range nodes {
		if d, ok := n.(ast.TypeDefNode); ok && d.Ident == "U" {
			udef = d
			break
		}
	}
	tr := New(tc, log)
	_, err = tr.TransformForstFileToGo(nodes)
	if err != nil {
		t.Fatal(err)
	}
	firstCount := countSealMethods(tr, "isU")
	if firstCount < 2 {
		t.Fatalf("expected at least 2 isU methods, got %d", firstCount)
	}
	// Second full transform simulates re-entrancy (e.g. emit pass): new transformer, same AST
	tr2 := New(tc, log)
	if _, err := tr2.TransformForstFileToGo(nodes); err != nil {
		t.Fatal(err)
	}
	secondCount := countSealMethods(tr2, "isU")
	if secondCount != firstCount {
		t.Fatalf("method count mismatch: first=%d second=%d", firstCount, secondCount)
	}
	if udef.Ident != "U" {
		t.Fatal("expected U typedef")
	}
}

func TestTryEmitNominalErrorUnionSealedInterface_emptyTypeNameReturnsNil(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	log.SetOutput(io.Discard)
	tc := typechecker.New(log, false)
	payload := ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}
	tc.Defs["A"] = ast.TypeDefNode{Ident: "A", Expr: ast.TypeDefErrorExpr{Payload: payload}}
	tc.Defs["B"] = ast.TypeDefNode{Ident: "B", Expr: ast.TypeDefErrorExpr{Payload: payload}}
	aName := ast.TypeIdent("A")
	bName := ast.TypeIdent("B")
	bin := ast.TypeDefBinaryExpr{
		Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &aName}},
		Op:    ast.TokenBitwiseOr,
		Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &bName}},
	}
	node := ast.TypeDefNode{Ident: "", Expr: bin}
	tr := New(tc, log)
	got, err := tr.tryEmitNominalErrorUnionSealedInterface(node, bin)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil *ast.GenDecl when typedef ident is empty, got %#v", got)
	}
}

// If TypeDefExprToTypeNode fails on the binary body, tryEmit must bail without error (same as "not a sealed nominal union").
func TestTryEmitNominalErrorUnionSealedInterface_typeDefExprErrorReturnsNil(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	log.SetOutput(io.Discard)
	tc := typechecker.New(log, false)
	payload := ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}
	tc.Defs["A"] = ast.TypeDefNode{Ident: "A", Expr: ast.TypeDefErrorExpr{Payload: payload}}
	aName := ast.TypeIdent("A")
	bin := ast.TypeDefBinaryExpr{
		Left:  ast.TypeDefErrorExpr{Payload: payload},
		Op:    ast.TokenBitwiseOr,
		Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &aName}},
	}
	node := ast.TypeDefNode{Ident: "U", Expr: bin}
	tr := New(tc, log)
	got, err := tr.tryEmitNominalErrorUnionSealedInterface(node, bin)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil when binary cannot lower to TypeNode, got %#v", got)
	}
}

func countSealMethods(tr *Transformer, method string) int {
	n := 0
	for _, f := range tr.Output.functions {
		if f.Name != nil && f.Name.Name == method {
			n++
		}
	}
	return n
}
