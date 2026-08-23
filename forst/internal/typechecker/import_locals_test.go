package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/importlocal"
	"forst/internal/testutil"
)

func TestImportLocalContext_checkReserved_go(t *testing.T) {
	ctx := importLocalContext{}
	b := importlocal.Binding{Local: "type", ImportPath: "example.com/foo/type", ModuleID: "example.com/foo/type"}
	err := ctx.checkReserved(b, importlocal.KindGo)
	if err == nil {
		t.Fatal("expected reserved local error")
	}
	if !strings.Contains(err.Error(), "Go import local name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportLocalContext_checkReserved_js(t *testing.T) {
	ctx := importLocalContext{}
	b := importlocal.Binding{Local: "js", ImportPath: "./legacy/payment.ts", ModuleID: "legacy/payment.ts"}
	err := ctx.checkReserved(b, importlocal.KindBridge)
	if err == nil {
		t.Fatal("expected reserved local error for js")
	}
}

func TestImportLocalContext_checkCrossKindConflict(t *testing.T) {
	ctx := importLocalContext{
		goImports: []ast.ImportNode{{Path: "fmt", Alias: &ast.Ident{ID: "payment"}}},
		nodeImports: []ast.ImportNode{
			{Path: "./legacy/payment.ts", BridgeOptIn: true},
		},
	}
	err := ctx.checkCrossKindConflict("payment", importlocal.KindBridge)
	if err == nil {
		t.Fatal("expected cross-kind conflict")
	}
}

func TestImportLocalContext_checkDuplicate(t *testing.T) {
	ctx := importLocalContext{}
	seen := map[string]string{"fmt": "fmt"}
	err := ctx.checkDuplicate("fmt", "os", importlocal.KindGo, seen)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestGoBindingFromLoaded_prefersPkgName(t *testing.T) {
	imp := ast.ImportNode{Path: "example.com/foo/type"}
	b := goBindingFromLoaded(imp, nil)
	if b.Local != "type" {
		t.Fatalf("fallback local = %q, want type", b.Local)
	}
}

func TestImportLocals_goNodeLocalConflict_integration(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import payment "fmt"
import "./legacy/payment" js

func main() {}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
		ExpectError:      `JS import local name "payment" conflicts with Go import`,
	})
}

func TestGoImports_implicitLocalFromPathKeyword_requiresAlias(t *testing.T) {
	src := `package main
import "example.com/foo/type"

func main() {}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{
		ExpectError: `Go import local name "type" is a Forst keyword`,
	})
}

func TestGoImports_explicitAlias_ok(t *testing.T) {
	src := `package main
import typePkg "fmt"

func main() {
	typePkg.Println("ok")
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestGoImports_aliasNode_ok(t *testing.T) {
	src := `package main
import node "fmt"

func main() {
	node.Println("ok")
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{})
	if !tc.IsImportedLocalName("node") {
		t.Fatal("expected node as Go import local")
	}
}

func TestGoImports_duplicateExplicitAlias_rejected(t *testing.T) {
	src := `package main
import f "fmt"
import f "os"

func main() {}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{
		ExpectError: `duplicate Go import local name "f"`,
	})
}

func TestGoImports_stdlibUnaliased_ok(t *testing.T) {
	src := `package main
import "fmt"

func main() {
	fmt.Println("ok")
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}
