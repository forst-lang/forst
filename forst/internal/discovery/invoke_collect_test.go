package discovery

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"
	"forst/internal/typechecker"
)

func TestCollectInvokeFunctionsFromNodes_publicRunnableOnly(t *testing.T) {
	src := `package bcrypt

func Hash() {
	return {h: "x"}
}

func helper() {
	return {ok: true}
}

func main() {}
`
	tc, nodes := typechecker.MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	out := CollectInvokeFunctionsFromNodes(nodes, tc)
	if len(out) != 1 {
		t.Fatalf("len = %d want 1: %#v", len(out), out)
	}
	if out[0].Package != "bcrypt" || out[0].Name != "Hash" {
		t.Fatalf("got %#v", out[0])
	}
}

func TestCollectInvokeFunctionsFromNodes_defaultPackageMain(t *testing.T) {
	nodes := []ast.Node{
		ast.FunctionNode{
			Ident: ast.Ident{ID: "Ping"},
			Body:  []ast.Node{},
		},
	}
	tc := typechecker.New(nil, false)
	out := CollectInvokeFunctionsFromNodes(nodes, tc)
	if len(out) != 1 {
		t.Fatalf("len = %d want 1: %#v", len(out), out)
	}
	if out[0].Package != "main" || out[0].Name != "Ping" {
		t.Fatalf("got %#v", out[0])
	}
}

func TestCollectInvokeFunctionsFromNodes_nilTcExcludesNonRunnable(t *testing.T) {
	nodes := []ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "helper"}},
		ast.FunctionNode{
			Ident: ast.Ident{ID: "Ping"},
			Body:  []ast.Node{},
		},
	}
	out := CollectInvokeFunctionsFromNodes(nodes, nil)
	if len(out) != 0 {
		t.Fatalf("nil tc leaves Runnable false, got %#v", out)
	}
}
