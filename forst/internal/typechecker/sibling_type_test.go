package typechecker

import (
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestResolveForstSiblingTypeDef_returnsAuthTypedef(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	goRoot := goload.FindModuleRoot(root)

	authMerged, _, err := forstpkg.ParseAndMergePackage(nil, []string{filepath.Join(root, "auth", "log.ft")})
	if err != nil {
		t.Fatal(err)
	}
	authTC := New(nil, false)
	authTC.GoWorkspaceDir = goRoot
	if err := authTC.CollectTypes(authMerged); err != nil {
		t.Fatalf("auth collect: %v", err)
	}

	view := &stubSiblingModuleView{
		importMap: map[string]string{"providers_cross_pkg_demo/auth": "auth"},
		pkgs:      map[string]*TypeChecker{"auth": authTC},
	}
	apiTC := New(nil, false)
	apiTC.GoWorkspaceDir = goRoot
	apiTC.SetForstPackage("api")
	apiTC.SetModuleResult(view)
	apiTC.imports = []ast.ImportNode{{Path: `"providers_cross_pkg_demo/auth"`}}
	apiTC.importPathByLocal = map[string]string{"auth": "providers_cross_pkg_demo/auth"}

	td, ok := apiTC.resolveForstSiblingTypeDef(ast.TypeIdent("auth.Logger"))
	if !ok {
		t.Fatal("expected auth.Logger typedef")
	}
	if td.Ident != "Logger" {
		t.Fatalf("want Logger typedef, got %q", td.Ident)
	}
	td2, ok2 := apiTC.resolveForstSiblingTypeDef(ast.TypeIdent("auth.Logger"))
	if !ok2 || td2.Ident != td.Ident {
		t.Fatalf("cache miss on second lookup: ok=%v ident=%q", ok2, td2.Ident)
	}
}

func TestIsTypeCompatible_siblingShapeMatchesLocalHashType(t *testing.T) {
	planShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"tier": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	billingTC := New(nil, false)
	billingTC.SetForstPackage("billing")
	if err := billingTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "Plan",
		Expr:  ast.TypeDefShapeExpr{Shape: planShape},
	}}); err != nil {
		t.Fatal(err)
	}

	shippingTC := New(nil, false)
	shippingTC.SetForstPackage("shipping")
	shippingTC.SetModuleResult(&stubSiblingModuleView{
		importMap: map[string]string{"demo/billing": "billing"},
		pkgs:      map[string]*TypeChecker{"billing": billingTC},
	})
	shippingTC.importPathByLocal = map[string]string{"billing": "demo/billing"}

	localShape := ast.TypeNode{Ident: "T_local123"}
	shippingTC.Defs[localShape.Ident] = ast.TypeDefNode{
		Ident: localShape.Ident,
		Expr:  ast.TypeDefShapeExpr{Shape: planShape},
	}
	expected := ast.TypeNode{Ident: ast.TypeIdent("billing.Plan")}
	if !shippingTC.IsTypeCompatible(localShape, expected) {
		t.Fatal("expected local hash shape compatible with billing.Plan")
	}
}

func TestIsTypeCompatible_siblingShapeMatchesExpectedHash(t *testing.T) {
	planShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"tier": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	billingTC := New(nil, false)
	billingTC.SetForstPackage("billing")
	if err := billingTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "Plan",
		Expr:  ast.TypeDefShapeExpr{Shape: planShape},
	}}); err != nil {
		t.Fatal(err)
	}

	shippingTC := New(nil, false)
	shippingTC.SetForstPackage("shipping")
	shippingTC.SetModuleResult(&stubSiblingModuleView{
		importMap: map[string]string{"demo/billing": "billing"},
		pkgs:      map[string]*TypeChecker{"billing": billingTC},
	})
	shippingTC.importPathByLocal = map[string]string{"billing": "demo/billing"}

	localShape := ast.TypeNode{Ident: "T_local123"}
	shippingTC.Defs[localShape.Ident] = ast.TypeDefNode{
		Ident: localShape.Ident,
		Expr:  ast.TypeDefShapeExpr{Shape: planShape},
	}
	actual := ast.TypeNode{Ident: ast.TypeIdent("billing.Plan")}
	if !shippingTC.IsTypeCompatible(actual, localShape) {
		t.Fatal("expected sibling shape compatible with local hash type")
	}
}

func TestResolveForstSiblingQualifiedVar_packageLevelString(t *testing.T) {
	billingTC := New(nil, false)
	billingTC.SetForstPackage("billing")
	if err := billingTC.collectPackageLevelVar(ast.AssignmentNode{
		IsPackageLevel: true,
		LValues:        []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "Version"}}},
		RValues:        []ast.ExpressionNode{ast.StringLiteralNode{Value: "1.0"}},
	}); err != nil {
		t.Fatal(err)
	}

	view := &stubSiblingModuleView{
		importMap: map[string]string{"demo/billing": "billing"},
		pkgs:      map[string]*TypeChecker{"billing": billingTC},
	}
	shippingTC := New(nil, false)
	shippingTC.SetForstPackage("shipping")
	shippingTC.SetModuleResult(view)
	shippingTC.imports = []ast.ImportNode{{Path: `"demo/billing"`}}
	shippingTC.importPathByLocal = map[string]string{"billing": "demo/billing"}

	types, resolved := shippingTC.resolveForstSiblingQualifiedVar("billing", "Version")
	if !resolved {
		t.Fatal("expected billing.Version to resolve")
	}
	if len(types) != 1 || types[0].Ident != ast.TypeString {
		t.Fatalf("got types %+v", types)
	}
}

func TestInferTypes_qualifiedSiblingParamType(t *testing.T) {
	sessionShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"workDir": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	workspaceTC := New(nil, false)
	workspaceTC.SetForstPackage("workspace")
	if err := workspaceTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "Session",
		Expr:  ast.TypeDefShapeExpr{Shape: sessionShape},
	}}); err != nil {
		t.Fatal(err)
	}

	reportsMerged := parseTestPackage(t, `package reports

func buildSummary(ctx workspace.Session): String {
	return ctx.workDir
}
`)
	view := &stubSiblingModuleView{
		importMap: map[string]string{"demo/workspace": "workspace"},
		pkgs:      map[string]*TypeChecker{"workspace": workspaceTC},
	}
	reportsTC := New(nil, false)
	reportsTC.SetForstPackage("reports")
	reportsTC.SetModuleResult(view)
	reportsTC.imports = []ast.ImportNode{{Path: `"demo/workspace"`}}
	reportsTC.importPathByLocal = map[string]string{"workspace": "demo/workspace"}
	if err := reportsTC.CollectTypes(reportsMerged); err != nil {
		t.Fatal(err)
	}
	if err := reportsTC.InferTypes(reportsMerged); err != nil {
		t.Fatalf("infer: %v", err)
	}
	got := reportsTC.Functions["buildSummary"].Parameters[0].Type.Ident
	if got != ast.TypeIdent("workspace.Session") {
		t.Fatalf("want workspace.Session param, got %q", got)
	}
}

func TestResolveForstSiblingTypeInImports_uniqueImportedType(t *testing.T) {
	modelsTC := New(nil, false)
	modelsTC.SetForstPackage("models")
	if err := modelsTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "Options",
		Expr: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"Flags": {Type: &ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}},
			},
		}},
	}}); err != nil {
		t.Fatal(err)
	}

	view := &stubSiblingModuleView{
		importMap: map[string]string{"demo/models": "models"},
		pkgs:      map[string]*TypeChecker{"models": modelsTC},
	}
	mainTC := New(nil, false)
	mainTC.SetForstPackage("main")
	mainTC.SetModuleResult(view)
	mainTC.imports = []ast.ImportNode{{Path: `"demo/models"`}}
	mainTC.importPathByLocal = map[string]string{"models": "demo/models"}

	td, ok := mainTC.typeDefForIdent("Options")
	if !ok {
		t.Fatal("expected Options from imported models package")
	}
	if td.Ident != "Options" {
		t.Fatalf("want Options typedef, got %q", td.Ident)
	}
	td2, ok2 := mainTC.typeDefForIdent("Options")
	if !ok2 || td2.Ident != td.Ident {
		t.Fatalf("cache miss on second lookup: ok=%v ident=%q", ok2, td2.Ident)
	}
}

func TestResolveForstSiblingTypeInImports_ambiguousImportedType(t *testing.T) {
	planShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"tier": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	makePkg := func(name string) *TypeChecker {
		tc := New(nil, false)
		tc.SetForstPackage(name)
		if err := tc.CollectTypes([]ast.Node{ast.TypeDefNode{
			Ident: "Plan",
			Expr:  ast.TypeDefShapeExpr{Shape: planShape},
		}}); err != nil {
			t.Fatal(err)
		}
		return tc
	}
	billingTC := makePkg("billing")
	shippingTC := makePkg("shipping")

	view := &stubSiblingModuleView{
		importMap: map[string]string{
			"demo/billing":  "billing",
			"demo/shipping": "shipping",
		},
		pkgs: map[string]*TypeChecker{
			"billing":  billingTC,
			"shipping": shippingTC,
		},
	}
	consumerTC := New(nil, false)
	consumerTC.SetForstPackage("consumer")
	consumerTC.SetModuleResult(view)
	consumerTC.imports = []ast.ImportNode{
		{Path: `"demo/billing"`},
		{Path: `"demo/shipping"`},
	}
	consumerTC.importPathByLocal = map[string]string{
		"billing":  "demo/billing",
		"shipping": "demo/shipping",
	}

	if _, ok := consumerTC.typeDefForIdent("Plan"); ok {
		t.Fatal("expected ambiguous Plan to not resolve")
	}
}

func parseTestPackage(t *testing.T, src string) []ast.Node {
	t.Helper()
	log := logrus.New()
	log.SetOutput(nil)
	toks := lexer.New([]byte(src), "test.ft", log).Lex()
	nodes, err := parser.New(toks, "test.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	return nodes
}

type stubSiblingModuleView struct {
	importMap map[string]string
	pkgs      map[string]*TypeChecker
}

func (s *stubSiblingModuleView) ImportPathToForstPkg() map[string]string { return s.importMap }
func (s *stubSiblingModuleView) ForstPackageTypeChecker(pkg string) *TypeChecker {
	return s.pkgs[pkg]
}
