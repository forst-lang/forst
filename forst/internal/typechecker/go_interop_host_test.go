package typechecker

import (
	"go/types"
	"testing"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestGoTypeForForstType_functionType_assignableToHTTPHandlerFunc(t *testing.T) {
	t.Parallel()
	dir := testutil.ModuleRoot(t)
	loaded, err := goload.LoadByPkgPath(dir, []string{"net/http"})
	if err != nil || len(loaded) == 0 {
		t.Skip("go/packages could not load net/http")
	}
	pkg := loaded["net/http"]
	if pkg == nil || pkg.Types == nil {
		t.Skip("net/http types unavailable")
	}
	handlerFunc := pkg.Types.Scope().Lookup("HandlerFunc")
	if handlerFunc == nil {
		t.Fatal("http.HandlerFunc not found")
	}
	goHandlerType := handlerFunc.Type()

	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	tc.importPathByLocal = map[string]string{"http": "net/http"}
	tc.goPkgsByLocal = map[string]*types.Package{"http": pkg.Types}

	forstFn := ast.NewFunctionType(
		[]ast.ParamNode{
			ast.SimpleParamNode{Ident: ast.Ident{ID: "w"}, Type: ast.TypeNode{Ident: ast.TypeIdent("http.ResponseWriter")}},
			ast.SimpleParamNode{Ident: ast.Ident{ID: "r"}, Type: ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeIdent("http.Request")}}}},
		},
		nil,
	)
	got := tc.goTypeForForstType(forstFn)
	if got == nil {
		t.Fatal("expected Go signature from Forst function type")
	}
	if !tc.forstAssignableToGoType(forstFn, goHandlerType) {
		t.Fatalf("Forst handler func not assignable to %s", goHandlerType.String())
	}
}
