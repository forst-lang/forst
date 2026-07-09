package typechecker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testmod"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
	"go/types"
)

func loadNamedTypeFromModule(t *testing.T, moduleRoot, importPath, typeName string) types.Type {
	t.Helper()
	goload.ClearLoadCacheForTest()
	loaded, err := goload.LoadByPkgPath(moduleRoot, []string{importPath})
	if err != nil {
		t.Fatalf("load %s: %v", importPath, err)
	}
	pkg := loaded[importPath]
	if pkg == nil || pkg.Types == nil {
		t.Fatalf("%s types missing", importPath)
	}
	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		t.Fatalf("%s not found in %s", typeName, importPath)
	}
	return obj.Type()
}

func writeTaggedStructFixture(t *testing.T) (moduleRoot, importPath string, recv types.Type) {
	t.Helper()
	moduleRoot = t.TempDir()
	testmod.WriteGoMod(t, moduleRoot, "tagtest")
	dir := filepath.Join(moduleRoot, "tagged")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const src = `package tagged

type Label struct {
	Code string ` + "`json:\"wireName\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "tagged.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	importPath = "tagtest/tagged"
	recv = loadNamedTypeFromModule(t, moduleRoot, importPath, "Label")
	return moduleRoot, importPath, recv
}

func writeNestedStructFixture(t *testing.T) (recv types.Type) {
	t.Helper()
	root := t.TempDir()
	testmod.WriteGoMod(t, root, "nesttest")
	dir := filepath.Join(root, "nest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const src = `package nest

type Inner struct {
	Age int
}

type Box struct {
	Inner Inner
}
`
	if err := os.WriteFile(filepath.Join(dir, "nest.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return loadNamedTypeFromModule(t, root, "nesttest/nest", "Box")
}

func TestGoTypeAtFieldPath_matchesForstCamelCaseViaCapitalize(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	recv := loadNamedTypeFromModule(t, fx.ModuleRoot, fx.UsersImportPath, "User")
	got, err := goTypeAtFieldPath(recv, []string{"name"})
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "string" {
		t.Fatalf("want string, got %s", got)
	}
}

func TestGoTypeAtFieldPath_matchesForstNameViaJSONTag(t *testing.T) {
	t.Parallel()
	_, _, recv := writeTaggedStructFixture(t)
	got, err := goTypeAtFieldPath(recv, []string{"wireName"})
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "string" {
		t.Fatalf("want string, got %s", got)
	}
}

func TestGoTypeAtFieldPath_exactGoNameStillResolves(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	recv := loadNamedTypeFromModule(t, fx.ModuleRoot, fx.UsersImportPath, "User")
	got, err := goTypeAtFieldPath(recv, []string{"Name"})
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "string" {
		t.Fatalf("want string, got %s", got)
	}
}

func TestGoTypeAtFieldPath_nestedForstPath(t *testing.T) {
	t.Parallel()
	recv := writeNestedStructFixture(t)
	got, err := goTypeAtFieldPath(recv, []string{"inner", "age"})
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "int" {
		t.Fatalf("want int, got %s", got)
	}
}

func TestLookupFieldPathFromGoType_forstFieldNameMapsToForstType(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	recv := loadNamedTypeFromModule(t, fx.ModuleRoot, fx.UsersImportPath, "User")
	tc := New(logrus.New(), false)
	mapped, err := tc.lookupFieldPathFromGoType(recv, []string{"name"})
	if err != nil {
		t.Fatal(err)
	}
	if mapped.Ident != ast.TypeString {
		t.Fatalf("want String, got %v", mapped.Ident)
	}
}

func TestLookupFieldPath_qualifiedSiblingGoFallback_resolvesField(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	tc := siblingInteropTC(t, fx)
	got, err := tc.lookupFieldPath(ast.TypeNode{Ident: ast.TypeIdent("users.User")}, []string{"name"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != ast.TypeString {
		t.Fatalf("want String, got %v", got.Ident)
	}
}

func TestLookupFieldPath_qualifiedSiblingGoFallback_unknownFieldErrors(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	tc := siblingInteropTC(t, fx)
	_, err := tc.lookupFieldPath(ast.TypeNode{Ident: ast.TypeIdent("users.User")}, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestGoQualifiedCall_siblingFieldAccess_paramUserTypechecks(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	src := `package signup

import "` + fx.UsersImportPath + `"

func getName(user users.User): String {
  return user.name
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "signup/signup.ft", log).Lex()
	nodes, err := parser.New(toks, "signup/signup.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = fx.ModuleRoot
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestGoQualifiedCall_siblingFieldAccess_localLiteralTypechecks(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	src := `package signup

import "` + fx.UsersImportPath + `"

func localName(): String {
  user := users.User{ name: "ada" }
  return user.name
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "signup/signup.ft", log).Lex()
	nodes, err := parser.New(toks, "signup/signup.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = fx.ModuleRoot
	if err := tc.CheckTypes(nodes); err != nil {
		var diag *Diagnostic
		if errors.As(err, &diag) && diag != nil && diag.Code == "TYPE_MISMATCH" {
			t.Fatalf("TYPE_MISMATCH on sibling field access: %v", err)
		}
		t.Fatalf("CheckTypes: %v", err)
	}
}
