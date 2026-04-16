package typechecker

import (
	"io"
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func testTC(t *testing.T) *TypeChecker {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	return New(log, false)
}

func TestValidateReferencedTypesAfterCollect_unknownFieldType(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"x": ast.MakeTypeField("NoSuchType"),
	}))

	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), `unknown type "NoSuchType"`) {
		t.Fatalf("expected unknown type error, got %v", err)
	}
	if !strings.Contains(err.Error(), `type Row field "x"`) {
		t.Fatalf("expected ctx in error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_unknownParamType(t *testing.T) {
	tc := testTC(t)
	tc.Functions[ast.Identifier("f")] = FunctionSignature{
		Ident: ast.Ident{ID: "f"},
		Parameters: []ParameterSignature{
			{Ident: ast.Ident{ID: "a"}, Type: ast.TypeNode{Ident: "Typo"}},
		},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeVoid}},
	}

	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), `unknown type "Typo"`) {
		t.Fatalf("expected unknown type error, got %v", err)
	}
	if !strings.Contains(err.Error(), `function f parameter "a"`) {
		t.Fatalf("expected ctx in error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_unknownReturnType(t *testing.T) {
	tc := testTC(t)
	tc.Functions[ast.Identifier("g")] = FunctionSignature{
		Ident:       ast.Ident{ID: "g"},
		Parameters:  []ParameterSignature{},
		ReturnTypes: []ast.TypeNode{{Ident: "MissingRet"}},
	}

	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), `unknown type "MissingRet"`) {
		t.Fatalf("expected unknown type error, got %v", err)
	}
	if !strings.Contains(err.Error(), `function g return[0]`) {
		t.Fatalf("expected ctx in error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_okUserDefinedAndBuiltin(t *testing.T) {
	tc := testTC(t)
	tc.Defs["User"] = ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"u": ast.MakeTypeField("User"),
	}))

	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReferencedTypesAfterCollect_arrayElementUnknown(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"items": {Type: &ast.TypeNode{
			Ident: ast.TypeArray,
			TypeParams: []ast.TypeNode{
				{Ident: "NotThere"},
			},
		}},
	}))

	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), "NotThere") || !strings.Contains(err.Error(), "element") {
		t.Fatalf("expected nested array element error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_mapKeyValue(t *testing.T) {
	tc := testTC(t)
	tc.Defs["BadMap"] = ast.MakeTypeDef("BadMap", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"m": {Type: &ast.TypeNode{
			Ident: ast.TypeMap,
			TypeParams: []ast.TypeNode{
				{Ident: ast.TypeString},
				{Ident: "BadVal"},
			},
		}},
	}))

	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), "BadVal") || !strings.Contains(err.Error(), "map value") {
		t.Fatalf("expected map value error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_pointerStringForm(t *testing.T) {
	tc := testTC(t)
	tc.Defs["User"] = ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"u": {Type: &ast.TypeNode{Ident: "*User"}},
	}))

	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReferencedTypesAfterCollect_pointerStringForm_unknown(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"u": {Type: &ast.TypeNode{Ident: "*Ghost"}},
	}))

	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), `unknown type "Ghost"`) {
		t.Fatalf("expected unknown inner pointer type, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_typePointerForm(t *testing.T) {
	tc := testTC(t)
	tc.Defs["User"] = ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"u": {Type: &ast.TypeNode{
			Ident: ast.TypePointer,
			TypeParams: []ast.TypeNode{
				{Ident: "User"},
			},
		}},
	}))

	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReferencedTypesAfterCollect_assertionBaseType(t *testing.T) {
	tc := testTC(t)
	tc.Defs["User"] = ast.MakeTypeDef("User", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"name": ast.MakeTypeField(ast.TypeString),
	}))
	base := ast.TypeIdent("User")
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"a": {Type: &ast.TypeNode{
			Ident: ast.TypeAssertion,
			Assertion: &ast.AssertionNode{
				BaseType: &base,
			},
		}},
	}))

	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReferencedTypesAfterCollect_defIsNotTypeName(t *testing.T) {
	tc := testTC(t)
	// Register ident "User" as something that is not a TypeDefNode (simulates inconsistent state).
	tc.Defs["User"] = ast.FunctionNode{Ident: ast.Ident{ID: "User"}}
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"u": ast.MakeTypeField("User"),
	}))

	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), `not a type name`) || !strings.Contains(err.Error(), "User") {
		t.Fatalf("expected not a type name error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_implicitTypeSkipped(t *testing.T) {
	tc := testTC(t)
	tc.Functions[ast.Identifier("f")] = FunctionSignature{
		Ident: ast.Ident{ID: "f"},
		Parameters: []ParameterSignature{
			{Ident: ast.Ident{ID: "a"}, Type: ast.TypeNode{Ident: ast.TypeImplicit}},
		},
		ReturnTypes: []ast.TypeNode{},
	}

	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReferencedTypesAfterCollect_hashBasedMissingDef(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"x": {Type: &ast.TypeNode{
			Ident:    "H_xyz",
			TypeKind: ast.TypeKindHashBased,
		}},
	}))

	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), `unknown structural type`) {
		t.Fatalf("expected unknown structural type, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_T_prefixSkipped(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"x": {Type: &ast.TypeNode{Ident: "T_inferredLater"}},
	}))

	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReferencedTypesAfterCollect_resultFailureUnknown(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"r": {Type: &ast.TypeNode{
			Ident: ast.TypeResult,
			TypeParams: []ast.TypeNode{
				{Ident: ast.TypeInt},
				{Ident: "MissingErr"},
			},
		}},
	}))
	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), `unknown type "MissingErr"`) || !strings.Contains(err.Error(), "result failure") {
		t.Fatalf("expected Result failure branch error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_tupleMemberUnknown(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"t": {Type: &ast.TypeNode{
			Ident: ast.TypeTuple,
			TypeParams: []ast.TypeNode{
				{Ident: ast.TypeString},
				{Ident: "NoTuple"},
			},
		}},
	}))
	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), "NoTuple") || !strings.Contains(err.Error(), "tuple[1]") {
		t.Fatalf("expected tuple error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_unionMemberUnknown(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"u": {Type: &ast.TypeNode{
			Ident: ast.TypeUnion,
			TypeParams: []ast.TypeNode{
				{Ident: ast.TypeInt},
				{Ident: "NoUnion"},
			},
		}},
	}))
	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), "NoUnion") {
		t.Fatalf("expected union member error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_intersectionMemberUnknown(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"i": {Type: &ast.TypeNode{
			Ident: ast.TypeIntersection,
			TypeParams: []ast.TypeNode{
				{Ident: "A"},
				{Ident: "B"},
			},
		}},
	}))
	err := tc.validateReferencedTypesAfterCollect()
	if err == nil || !strings.Contains(err.Error(), `unknown type "A"`) {
		t.Fatalf("expected intersection error, got %v", err)
	}
}

func TestValidateReferencedTypesAfterCollect_syntheticParenTypeSkipped(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"x": {Type: &ast.TypeNode{Ident: "Pointer(String)"}},
	}))
	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReferencedTypesAfterCollect_pointerBuiltinScalarStringForm(t *testing.T) {
	tc := testTC(t)
	tc.Defs["Row"] = ast.MakeTypeDef("Row", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"p": {Type: &ast.TypeNode{Ident: "*String"}},
	}))
	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}
