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

func TestResolveForstSiblingTypeDef_returnsAlphaTypedef(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	goRoot := goload.FindModuleRoot(root)

	alphaMerged, _, err := forstpkg.ParseAndMergePackage(nil, []string{filepath.Join(root, "alpha", "log.ft")})
	if err != nil {
		t.Fatal(err)
	}
	alphaTC := New(nil, false)
	alphaTC.GoWorkspaceDir = goRoot
	if err := alphaTC.CollectTypes(alphaMerged); err != nil {
		t.Fatalf("alpha collect: %v", err)
	}

	view := &stubSiblingModuleView{
		importMap: map[string]string{"providers_cross_pkg_demo/alpha": "alpha"},
		pkgs:      map[string]*TypeChecker{"alpha": alphaTC},
	}
	betaTC := New(nil, false)
	betaTC.GoWorkspaceDir = goRoot
	betaTC.SetForstPackage("beta")
	betaTC.SetModuleResult(view)
	betaTC.imports = []ast.ImportNode{{Path: `"providers_cross_pkg_demo/alpha"`}}
	betaTC.importPathByLocal = map[string]string{"alpha": "providers_cross_pkg_demo/alpha"}

	td, ok := betaTC.resolveForstSiblingTypeDef(ast.TypeIdent("alpha.Logger"))
	if !ok {
		t.Fatal("expected alpha.Logger typedef")
	}
	if td.Ident != "Logger" {
		t.Fatalf("want Logger typedef, got %q", td.Ident)
	}
	td2, ok2 := betaTC.resolveForstSiblingTypeDef(ast.TypeIdent("alpha.Logger"))
	if !ok2 || td2.Ident != td.Ident {
		t.Fatalf("cache miss on second lookup: ok=%v ident=%q", ok2, td2.Ident)
	}
}

func TestIsTypeCompatible_siblingShapeMatchesLocalHashType(t *testing.T) {
	configShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"Version": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	alphaTC := New(nil, false)
	alphaTC.SetForstPackage("alpha")
	if err := alphaTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "Config",
		Expr:  ast.TypeDefShapeExpr{Shape: configShape},
	}}); err != nil {
		t.Fatal(err)
	}

	betaTC := New(nil, false)
	betaTC.SetForstPackage("beta")
	betaTC.SetModuleResult(&stubSiblingModuleView{
		importMap: map[string]string{"demo/alpha": "alpha"},
		pkgs:      map[string]*TypeChecker{"alpha": alphaTC},
	})
	betaTC.importPathByLocal = map[string]string{"alpha": "demo/alpha"}

	localShape := ast.TypeNode{Ident: "T_local123"}
	betaTC.Defs[localShape.Ident] = ast.TypeDefNode{
		Ident: localShape.Ident,
		Expr:  ast.TypeDefShapeExpr{Shape: configShape},
	}
	expected := ast.TypeNode{Ident: ast.TypeIdent("alpha.Config")}
	if !betaTC.IsTypeCompatible(localShape, expected) {
		t.Fatal("expected local hash shape compatible with alpha.Config")
	}
}

func TestIsTypeCompatible_siblingShapeMatchesExpectedHash(t *testing.T) {
	configShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"Version": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	alphaTC := New(nil, false)
	alphaTC.SetForstPackage("alpha")
	if err := alphaTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "Config",
		Expr:  ast.TypeDefShapeExpr{Shape: configShape},
	}}); err != nil {
		t.Fatal(err)
	}

	betaTC := New(nil, false)
	betaTC.SetForstPackage("beta")
	betaTC.SetModuleResult(&stubSiblingModuleView{
		importMap: map[string]string{"demo/alpha": "alpha"},
		pkgs:      map[string]*TypeChecker{"alpha": alphaTC},
	})
	betaTC.importPathByLocal = map[string]string{"alpha": "demo/alpha"}

	localShape := ast.TypeNode{Ident: "T_local123"}
	betaTC.Defs[localShape.Ident] = ast.TypeDefNode{
		Ident: localShape.Ident,
		Expr:  ast.TypeDefShapeExpr{Shape: configShape},
	}
	actual := ast.TypeNode{Ident: ast.TypeIdent("alpha.Config")}
	if !betaTC.IsTypeCompatible(actual, localShape) {
		t.Fatal("expected sibling shape compatible with local hash type")
	}
}

func TestResolveForstSiblingQualifiedVar_packageLevelString(t *testing.T) {
	alphaTC := New(nil, false)
	alphaTC.SetForstPackage("alpha")
	if err := alphaTC.collectPackageLevelVar(ast.AssignmentNode{
		IsPackageLevel: true,
		LValues:        []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "Version"}}},
		RValues:        []ast.ExpressionNode{ast.StringLiteralNode{Value: "1.0"}},
	}); err != nil {
		t.Fatal(err)
	}

	view := &stubSiblingModuleView{
		importMap: map[string]string{"demo/alpha": "alpha"},
		pkgs:      map[string]*TypeChecker{"alpha": alphaTC},
	}
	betaTC := New(nil, false)
	betaTC.SetForstPackage("beta")
	betaTC.SetModuleResult(view)
	betaTC.imports = []ast.ImportNode{{Path: `"demo/alpha"`}}
	betaTC.importPathByLocal = map[string]string{"alpha": "demo/alpha"}

	types, resolved := betaTC.resolveForstSiblingQualifiedVar("alpha", "Version")
	if !resolved {
		t.Fatal("expected alpha.Version to resolve")
	}
	if len(types) != 1 || types[0].Ident != ast.TypeString {
		t.Fatalf("got types %+v", types)
	}
}

func TestInferTypes_qualifiedSiblingParamType(t *testing.T) {
	runctxShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"WorkDir": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	runctxTC := New(nil, false)
	runctxTC.SetForstPackage("runctx")
	if err := runctxTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "RunContext",
		Expr:  ast.TypeDefShapeExpr{Shape: runctxShape},
	}}); err != nil {
		t.Fatal(err)
	}

	exportMerged := parseTestPackage(t, `package export

func buildManifest(ctx runctx.RunContext): String {
	return ctx.WorkDir
}
`)
	view := &stubSiblingModuleView{
		importMap: map[string]string{"demo/runctx": "runctx"},
		pkgs:      map[string]*TypeChecker{"runctx": runctxTC},
	}
	exportTC := New(nil, false)
	exportTC.SetForstPackage("export")
	exportTC.SetModuleResult(view)
	exportTC.imports = []ast.ImportNode{{Path: `"demo/runctx"`}}
	exportTC.importPathByLocal = map[string]string{"runctx": "demo/runctx"}
	if err := exportTC.CollectTypes(exportMerged); err != nil {
		t.Fatal(err)
	}
	if err := exportTC.InferTypes(exportMerged); err != nil {
		t.Fatalf("infer: %v", err)
	}
	got := exportTC.Functions["buildManifest"].Parameters[0].Type.Ident
	if got != ast.TypeIdent("runctx.RunContext") {
		t.Fatalf("want runctx.RunContext param, got %q", got)
	}
}

func TestResolveForstSiblingTypeInImports_uniqueImportedType(t *testing.T) {
	orchTC := New(nil, false)
	orchTC.SetForstPackage("orchestrator")
	if err := orchTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "ParsedArgs",
		Expr: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"Flags": {Type: &ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}},
			},
		}},
	}}); err != nil {
		t.Fatal(err)
	}

	view := &stubSiblingModuleView{
		importMap: map[string]string{"demo/orchestrator": "orchestrator"},
		pkgs:      map[string]*TypeChecker{"orchestrator": orchTC},
	}
	mainTC := New(nil, false)
	mainTC.SetForstPackage("main")
	mainTC.SetModuleResult(view)
	mainTC.imports = []ast.ImportNode{{Path: `"demo/orchestrator"`}}
	mainTC.importPathByLocal = map[string]string{"orchestrator": "demo/orchestrator"}

	td, ok := mainTC.typeDefForIdent("ParsedArgs")
	if !ok {
		t.Fatal("expected ParsedArgs from imported orchestrator package")
	}
	if td.Ident != "ParsedArgs" {
		t.Fatalf("want ParsedArgs typedef, got %q", td.Ident)
	}
	td2, ok2 := mainTC.typeDefForIdent("ParsedArgs")
	if !ok2 || td2.Ident != td.Ident {
		t.Fatalf("cache miss on second lookup: ok=%v ident=%q", ok2, td2.Ident)
	}
}

func TestResolveForstSiblingTypeInImports_ambiguousImportedType(t *testing.T) {
	configShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"Version": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	makePkg := func(name string) *TypeChecker {
		tc := New(nil, false)
		tc.SetForstPackage(name)
		if err := tc.CollectTypes([]ast.Node{ast.TypeDefNode{
			Ident: "Config",
			Expr:  ast.TypeDefShapeExpr{Shape: configShape},
		}}); err != nil {
			t.Fatal(err)
		}
		return tc
	}
	alphaTC := makePkg("alpha")
	betaTC := makePkg("beta")

	view := &stubSiblingModuleView{
		importMap: map[string]string{
			"demo/alpha": "alpha",
			"demo/beta":  "beta",
		},
		pkgs: map[string]*TypeChecker{
			"alpha": alphaTC,
			"beta":  betaTC,
		},
	}
	consumerTC := New(nil, false)
	consumerTC.SetForstPackage("consumer")
	consumerTC.SetModuleResult(view)
	consumerTC.imports = []ast.ImportNode{
		{Path: `"demo/alpha"`},
		{Path: `"demo/beta"`},
	}
	consumerTC.importPathByLocal = map[string]string{
		"alpha": "demo/alpha",
		"beta":  "demo/beta",
	}

	if _, ok := consumerTC.typeDefForIdent("Config"); ok {
		t.Fatal("expected ambiguous Config to not resolve")
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
