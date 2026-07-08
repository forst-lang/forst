package typechecker

import (
	"errors"
	"testing"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
	"go/types"
)

func siblingInteropTC(t *testing.T, fx testutil.SiblingGoInteropFixture) *TypeChecker {
	t.Helper()
	goload.ClearLoadCacheForTest()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(fx.CallerSource), "signup/signup.ft", log).Lex()
	nodes, err := parser.New(toks, "signup/signup.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = fx.ModuleRoot
	if err := tc.CollectTypes(nodes); err != nil {
		t.Fatal(err)
	}
	tc.initGoImportPackages()
	return tc
}

func loadedAccountsGoParamType(t *testing.T, fx testutil.SiblingGoInteropFixture) types.Type {
	t.Helper()
	loaded, err := goload.LoadByPkgPath(fx.ModuleRoot, []string{fx.AccountsImportPath})
	if err != nil {
		t.Fatalf("load accounts: %v", err)
	}
	pkg := loaded[fx.AccountsImportPath]
	if pkg == nil || pkg.Types == nil {
		t.Fatal("accounts package types missing")
	}
	obj := pkg.Types.Scope().Lookup("CreateAccount")
	if obj == nil {
		t.Fatal("CreateAccount not found")
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		t.Fatal("CreateAccount not a func")
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() == 0 {
		t.Fatal("invalid CreateAccount signature")
	}
	return sig.Params().At(0).Type()
}

func TestForstTypeForGoType_inModuleNamedMapsToQualifiedForstType(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	tc := siblingInteropTC(t, fx)
	goParam := loadedAccountsGoParamType(t, fx)

	got, ok := tc.forstTypeForGoType(goParam)
	if !ok {
		t.Fatal("expected mapping for in-module User")
	}
	if got.Ident != ast.TypeIdent("users.User") {
		t.Fatalf("forstTypeForGoType = %q, want users.User", got.Ident)
	}
}

func TestForstTypeForGoType_stdlibNamedReturnsFalse(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	loaded, err := goload.LoadByPkgPath(dir, []string{"strings"})
	if err != nil {
		t.Skip("go/packages strings:", err)
	}
	pkg := loaded["strings"]
	if pkg == nil || pkg.Types == nil {
		t.Skip("strings types unavailable")
	}
	obj := pkg.Types.Scope().Lookup("Reader")
	if obj == nil {
		t.Fatal("strings.Reader not found")
	}
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	tc.importPathByLocal = map[string]string{"strings": "strings"}
	if _, ok := tc.forstTypeForGoType(obj.Type()); ok {
		t.Fatal("expected stdlib named type not to map to Forst sibling type")
	}
}

func TestForstTypeForGoType_withoutModuleResult_whenGoPkgLoaded(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	tc := siblingInteropTC(t, fx)
	if tc.moduleResult != nil {
		t.Fatal("expected no moduleResult in LSP-style setup")
	}
	goParam := loadedAccountsGoParamType(t, fx)
	if _, ok := tc.forstTypeForGoType(goParam); !ok {
		t.Fatal("expected mapping without moduleResult when Go package is loaded")
	}
}

func TestForstAssignableToGoType_qualifiedSiblingToGoNamedStruct(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	tc := siblingInteropTC(t, fx)
	goParam := loadedAccountsGoParamType(t, fx)
	forstArg := ast.TypeNode{Ident: ast.TypeIdent("users.User")}
	if !tc.forstAssignableToGoType(forstArg, goParam) {
		t.Fatal("expected users.User assignable to Go users.User param")
	}
}

func TestGoQualifiedCall_siblingUserArgToAccountsCreateAccount(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(fx.CallerSource), "signup/signup.ft", log).Lex()
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

func TestGoQualifiedCall_siblingUserArg_wrongTypeRejected(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	badSrc := `package signup

import "` + fx.AccountsImportPath + `"

func registerUser(): Error {
  return accounts.CreateAccount("not-a-user")
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(badSrc), "signup/signup.ft", log).Lex()
	nodes, err := parser.New(toks, "signup/signup.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = fx.ModuleRoot
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected go-call type error")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag == nil || diag.Code != "go-call" {
		t.Fatalf("expected go-call diagnostic, got %T: %v", err, err)
	}
}

func TestGoQualifiedCall_siblingUserArg_withModuleResult(t *testing.T) {
	t.Parallel()
	fx := testutil.WriteSiblingGoInteropModuleFixture(t)
	userShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	usersTC := New(nil, false)
	usersTC.SetForstPackage("users")
	if err := usersTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	}}); err != nil {
		t.Fatal(err)
	}
	view := &stubSiblingModuleView{
		importMap: map[string]string{fx.UsersImportPath: "users"},
		pkgs:      map[string]*TypeChecker{"users": usersTC},
	}
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(fx.CallerSource), "signup/signup.ft", log).Lex()
	nodes, err := parser.New(toks, "signup/signup.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = fx.ModuleRoot
	tc.SetForstPackage("signup")
	tc.SetModuleResult(view)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes with moduleResult: %v", err)
	}
}
