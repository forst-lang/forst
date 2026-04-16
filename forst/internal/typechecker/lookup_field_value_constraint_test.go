package typechecker

import (
	"io"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestInferValueConstraintType_basicErrors(t *testing.T) {
	tc := New(logrus.New(), false)

	_, err := tc.inferValueConstraintType(ast.ConstraintNode{Name: ast.ValueConstraint}, "f", nil)
	if err == nil {
		t.Fatal("expected error for missing Value args")
	}

	c := ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: nil}},
	}
	_, err = tc.inferValueConstraintType(c, "f", nil)
	if err == nil {
		t.Fatal("expected error for nil Value arg")
	}
}

func TestInferValueConstraintType_literalAndFallbackExpectedType(t *testing.T) {
	tc := New(logrus.New(), false)

	vs := ast.ValueNode(ast.StringLiteralNode{Value: "abc"})
	typ, err := tc.inferValueConstraintType(ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: &vs}},
	}, "name", nil)
	if err != nil {
		t.Fatalf("string literal infer: %v", err)
	}
	if typ.Ident != ast.TypeString {
		t.Fatalf("expected TYPE_STRING, got %q", typ.Ident)
	}

	vVar := ast.ValueNode(ast.VariableNode{Ident: ast.Ident{ID: "missing"}})
	expected := ast.TypeNode{Ident: ast.TypeBool}
	typ, err = tc.inferValueConstraintType(ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: &vVar}},
	}, "flag", &expected)
	if err != nil {
		t.Fatalf("fallback expected type infer: %v", err)
	}
	if typ.Ident != ast.TypeBool {
		t.Fatalf("expected fallback TYPE_BOOL, got %q", typ.Ident)
	}
}

func TestInferValueConstraintType_literalsReferenceAndNil(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("n"), []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)

	valC := func(v ast.ValueNode) ast.ConstraintNode {
		x := v
		return ast.ConstraintNode{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &x}},
		}
	}

	t.Run("int_literal_value_and_pointer", func(t *testing.T) {
		for _, v := range []ast.ValueNode{
			ast.IntLiteralNode{Value: 3},
			&ast.IntLiteralNode{Value: 4},
		} {
			typ, err := tc.inferValueConstraintType(valC(v), "f", nil)
			if err != nil {
				t.Fatalf("%T: %v", v, err)
			}
			if typ.Ident != ast.TypeInt {
				t.Fatalf("%T: got %q", v, typ.Ident)
			}
		}
	})
	t.Run("float_and_bool", func(t *testing.T) {
		typ, err := tc.inferValueConstraintType(valC(ast.FloatLiteralNode{Value: 1.5}), "f", nil)
		if err != nil || typ.Ident != ast.TypeFloat {
			t.Fatalf("float: err=%v ident=%q", err, typ.Ident)
		}
		typ, err = tc.inferValueConstraintType(valC(ast.BoolLiteralNode{Value: true}), "f", nil)
		if err != nil || typ.Ident != ast.TypeBool {
			t.Fatalf("bool: err=%v ident=%q", err, typ.Ident)
		}
	})
	t.Run("nil_pointer_with_expected_string_wraps_pointer", func(t *testing.T) {
		strT := ast.TypeNode{Ident: ast.TypeString}
		var nilPtr ast.ValueNode = &ast.NilLiteralNode{}
		typ, err := tc.inferValueConstraintType(valC(nilPtr), "p", &strT)
		if err != nil {
			t.Fatal(err)
		}
		if typ.Ident != ast.TypePointer || len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeString {
			t.Fatalf("got %+v", typ)
		}
	})
	t.Run("nil_value_returns_expected_type_directly", func(t *testing.T) {
		strT := ast.TypeNode{Ident: ast.TypeString}
		typ, err := tc.inferValueConstraintType(valC(ast.NilLiteralNode{}), "p", &strT)
		if err != nil {
			t.Fatal(err)
		}
		if typ.Ident != ast.TypeString {
			t.Fatalf("got %+v", typ)
		}
	})
	t.Run("reference_to_registered_var", func(t *testing.T) {
		vn := ast.ValueNode(&ast.ReferenceNode{Value: &ast.VariableNode{Ident: ast.Ident{ID: "n"}}})
		typ, err := tc.inferValueConstraintType(valC(vn), "f", nil)
		if err != nil {
			t.Fatal(err)
		}
		if typ.Ident != ast.TypePointer || len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeInt {
			t.Fatalf("want *Int, got %+v", typ)
		}
	})
	t.Run("variable_resolves_from_scope", func(t *testing.T) {
		vn := ast.ValueNode(&ast.VariableNode{Ident: ast.Ident{ID: "n"}})
		typ, err := tc.inferValueConstraintType(valC(vn), "f", nil)
		if err != nil {
			t.Fatal(err)
		}
		if typ.Ident != ast.TypeInt {
			t.Fatalf("got %q", typ.Ident)
		}
	})
}

func TestInferValueConstraintType_dotNotationFieldPath(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	tc.Defs[ast.TypeIdent("UserQuery")] = ast.MakeTypeDef("UserQuery", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"id": ast.MakeTypeField(ast.TypeString),
	}))
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("query"), []ast.TypeNode{{Ident: ast.TypeIdent("UserQuery")}}, SymbolVariable)

	valC := func(v ast.ValueNode) ast.ConstraintNode {
		x := v
		return ast.ConstraintNode{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &x}},
		}
	}

	for _, v := range []ast.ValueNode{
		&ast.VariableNode{Ident: ast.Ident{ID: "query.id"}},
		ast.VariableNode{Ident: ast.Ident{ID: "query.id"}},
	} {
		typ, err := tc.inferValueConstraintType(valC(v), "f", nil)
		if err != nil {
			t.Fatalf("%T: %v", v, err)
		}
		if typ.Ident != ast.TypeString {
			t.Fatalf("%T: want String, got %q", v, typ.Ident)
		}
	}
}

func TestInferValueConstraintType_referenceNonVarFallsBackToExpected(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)

	valC := func(v ast.ValueNode) ast.ConstraintNode {
		x := v
		return ast.ConstraintNode{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &x}},
		}
	}
	ref := &ast.ReferenceNode{Value: ast.IntLiteralNode{Value: 9}}
	exp := ast.TypeNode{Ident: ast.TypeString}
	typ, err := tc.inferValueConstraintType(valC(ref), "f", &exp)
	if err != nil {
		t.Fatal(err)
	}
	if typ.Ident != ast.TypeString {
		t.Fatalf("got %q", typ.Ident)
	}
}

func TestInferValueConstraintType_referenceNilValueUsesExpected(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)

	valC := func(v ast.ValueNode) ast.ConstraintNode {
		x := v
		return ast.ConstraintNode{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &x}},
		}
	}
	exp := ast.TypeNode{Ident: ast.TypeFloat}
	var ref ast.ValueNode = &ast.ReferenceNode{Value: nil}
	typ, err := tc.inferValueConstraintType(valC(ref), "f", &exp)
	if err != nil {
		t.Fatal(err)
	}
	if typ.Ident != ast.TypeFloat {
		t.Fatalf("got %q", typ.Ident)
	}
}

func TestInferValueConstraintType_unsupportedValueNodeErrors(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)

	valC := func(v ast.ValueNode) ast.ConstraintNode {
		x := v
		return ast.ConstraintNode{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &x}},
		}
	}
	arr := ast.ArrayLiteralNode{
		Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}},
	}
	_, err := tc.inferValueConstraintType(valC(arr), "f", nil)
	if err == nil {
		t.Fatal("expected error for array literal in Value constraint")
	}
}

func TestInferValueConstraintType_nilPointerExpectedIsPointerType(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	ptrToInt := ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}}}
	var nilPtr ast.ValueNode = &ast.NilLiteralNode{}
	x := nilPtr
	c := ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: &x}},
	}
	typ, err := tc.inferValueConstraintType(c, "f", &ptrToInt)
	if err != nil {
		t.Fatal(err)
	}
	if typ.Ident != ast.TypePointer || len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeInt {
		t.Fatalf("got %+v", typ)
	}
}

func TestInferValueConstraintType_nilNoExpectedDefaultsToStringPtr(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	var nilPtr ast.ValueNode = &ast.NilLiteralNode{}
	x := nilPtr
	c := ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: &x}},
	}
	typ, err := tc.inferValueConstraintType(c, "f", nil)
	if err != nil {
		t.Fatal(err)
	}
	if typ.Ident != ast.TypePointer || len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeString {
		t.Fatalf("got %+v", typ)
	}
	vNil := ast.ValueNode(ast.NilLiteralNode{})
	typ, err = tc.inferValueConstraintType(ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: &vNil}},
	}, "f", nil)
	if err != nil {
		t.Fatal(err)
	}
	if typ.Ident != ast.TypePointer || len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeString {
		t.Fatalf("value nil: got %+v", typ)
	}
}

func TestInferValueConstraintType_referenceValueFormVarLookup(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("n"), []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)

	valC := func(v ast.ValueNode) ast.ConstraintNode {
		x := v
		return ast.ConstraintNode{
			Name: ast.ValueConstraint,
			Args: []ast.ConstraintArgumentNode{{Value: &x}},
		}
	}

	refPtr := ast.ValueNode(&ast.ReferenceNode{Value: &ast.VariableNode{Ident: ast.Ident{ID: "n"}}})
	typ, err := tc.inferValueConstraintType(valC(refPtr), "f", nil)
	if err != nil {
		t.Fatalf("*Ref: %v", err)
	}
	if typ.Ident != ast.TypePointer || len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeInt {
		t.Fatalf("*Ref: got %+v", typ)
	}

	refVal := ast.ValueNode(ast.ReferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "n"}}})
	typ, err = tc.inferValueConstraintType(valC(refVal), "f", nil)
	if err != nil {
		t.Fatalf("Ref value + value Var: %v", err)
	}
	if typ.Ident != ast.TypePointer || len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeInt {
		t.Fatalf("Ref value + value Var: got %+v", typ)
	}

	refValPtrInner := ast.ValueNode(ast.ReferenceNode{Value: &ast.VariableNode{Ident: ast.Ident{ID: "n"}}})
	typ, err = tc.inferValueConstraintType(valC(refValPtrInner), "f", nil)
	if err != nil {
		t.Fatalf("Ref value + *Var: %v", err)
	}
	if typ.Ident != ast.TypePointer || len(typ.TypeParams) != 1 || typ.TypeParams[0].Ident != ast.TypeInt {
		t.Fatalf("Ref value + *Var: got %+v", typ)
	}
}

func TestInferValueConstraintType_mapLiteralErrors(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	v := ast.ValueNode(ast.MapLiteralNode{})
	x := v
	_, err := tc.inferValueConstraintType(ast.ConstraintNode{
		Name: ast.ValueConstraint,
		Args: []ast.ConstraintArgumentNode{{Value: &x}},
	}, "f", nil)
	if err == nil {
		t.Fatal("expected error for map literal in Value constraint")
	}
}
