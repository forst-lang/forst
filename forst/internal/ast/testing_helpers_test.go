package ast

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetupTestLogger_forceLevel(t *testing.T) {
	l := SetupTestLogger(&TestLoggerOptions{ForceLevel: logrus.ErrorLevel})
	if l == nil {
		t.Fatal("nil logger")
	}
	if l.GetLevel() != logrus.ErrorLevel {
		t.Fatalf("GetLevel = %v", l.GetLevel())
	}
}

func TestSetupTestLogger_nil_opts(t *testing.T) {
	l := SetupTestLogger(nil)
	if l == nil {
		t.Fatal("nil logger")
	}
}

func TestSetupTestLogger_verbose_branch(t *testing.T) {
	if !testing.Verbose() {
		t.Skip("run with -v to cover debug level branch in SetupTestLogger")
	}
	l := SetupTestLogger(nil)
	if l.GetLevel() != logrus.DebugLevel {
		t.Fatalf("verbose run: got level %v", l.GetLevel())
	}
}

func TestSetupTestLogger_injected_verbose(t *testing.T) {
	l := setupTestLogger(nil, func() bool { return true })
	if l.GetLevel() != logrus.DebugLevel {
		t.Fatalf("expected Debug when verbose returns true, got %v", l.GetLevel())
	}
	l2 := setupTestLogger(nil, func() bool { return false })
	if l2.GetLevel() != logrus.InfoLevel {
		t.Fatalf("expected default Info when verbose false and no opts, got %v", l2.GetLevel())
	}
	l3 := setupTestLogger(&TestLoggerOptions{ForceLevel: logrus.WarnLevel}, func() bool { return true })
	if l3.GetLevel() != logrus.WarnLevel {
		t.Fatalf("opts should override verbose level, got %v", l3.GetLevel())
	}
}

func TestMakeStringLiteral(t *testing.T) {
	l := MakeStringLiteral("hello")
	if l.Value != "hello" || l.Kind() != NodeKindStringLiteral {
		t.Fatal(l)
	}
}

func TestMakeTypeDef_and_shape_helpers(t *testing.T) {
	td := MakeTypeDef("T", MakeShape(map[string]ShapeFieldNode{"a": MakeTypeField(TypeInt)}))
	if td.GetIdent() != "T" || td.Kind() != NodeKindTypeDef {
		t.Fatalf("MakeTypeDef: %+v", td)
	}

	shape := MakeShape(map[string]ShapeFieldNode{"k": MakeShapeField(map[string]ShapeFieldNode{})})
	if !strings.Contains(shape.String(), "k") {
		t.Fatal(shape.String())
	}
	sp := MakeShapePtr(map[string]ShapeFieldNode{})
	if sp == nil || len(sp.Fields) != 0 {
		t.Fatal(sp)
	}

	af := MakeAssertionField(TypeIdent("U"))
	if af.Assertion == nil || af.Assertion.BaseType == nil || *af.Assertion.BaseType != "U" {
		t.Fatalf("MakeAssertionField: %+v", af)
	}
}

func TestMakeValue_and_type_helpers(t *testing.T) {
	if MakeValueNode(3).(IntLiteralNode).Value != 3 {
		t.Fatal()
	}
	if MakePointerType("X").Ident != "*X" {
		t.Fatal()
	}
	if MakeStringType().Ident != TypeString {
		t.Fatal()
	}
	if MakeTypeNode("Z").Ident != "Z" {
		t.Fatal()
	}
	st := MakeShapeType(map[string]ShapeFieldNode{})
	if st.Ident != TypeShape {
		t.Fatal(st)
	}
}

func TestMakeAddressOf_reference_struct_and_assignment(t *testing.T) {
	u := MakeAddressOf(IntLiteralNode{Value: 1})
	if u.Operator != "&" {
		t.Fatal(u)
	}

	v := MakeReferenceNode("n")
	if v.GetIdent() != "n" {
		t.Fatal(v)
	}

	slit := MakeStructLiteral("Base", map[string]ShapeFieldNode{"f": MakeStructField(IntLiteralNode{Value: 1})})
	if slit.BaseType == nil || *slit.BaseType != "Base" {
		t.Fatal(slit)
	}
	if MakeStructFieldWithType(NewBuiltinType(TypeInt)).Type == nil {
		t.Fatal()
	}
	ns := MakeNestedStructField(&ShapeNode{Fields: map[string]ShapeFieldNode{}})
	if ns.Shape == nil {
		t.Fatal()
	}

	asg := MakeAssignment("x", NewBuiltinType(TypeInt), IntLiteralNode{Value: 0})
	if asg == nil || len(asg.LValues) != 1 {
		t.Fatal(asg)
	}
}

func TestMakeFunction_package_and_constraint(t *testing.T) {
	fn := MakeFunction("f", []ParamNode{MakeSimpleParam("a", NewBuiltinType(TypeInt))}, nil)
	if fn.GetIdent() != "f" {
		t.Fatal(fn)
	}

	call := MakeFunctionCall("g", []ExpressionNode{IntLiteralNode{Value: 2}})
	if !strings.Contains(call.String(), "g") {
		t.Fatal(call.String())
	}

	pkg := MakePackage("main", nil)
	if pkg.GetIdent() != "main" || !pkg.IsMainPackage() {
		t.Fatal(pkg)
	}
	if MakePackage("other", nil).IsMainPackage() {
		t.Fatal("expected not main")
	}

	cn := MakeConstraint("C", MakeShapePtr(map[string]ShapeFieldNode{}))
	if cn.Name != "C" || len(cn.Args) != 1 {
		t.Fatal(cn)
	}
}
