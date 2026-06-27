package discovery

import (
	"encoding/json"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func TestBuildUsablesDiscoveryV1_schema(t *testing.T) {
	functions := map[string]map[string]FunctionInfo{
		"example.com/users": {
			"handleCreateUser": {
				Usables:  []string{"Logger", "UserRepo"},
				Runnable: false,
			},
			"healthCheck": {
				Runnable: true,
			},
		},
	}
	doc := BuildUsablesDiscoveryV1(functions)
	if doc.Version != 1 {
		t.Fatalf("version = %d", doc.Version)
	}
	pkg := doc.Packages["example.com/users"]
	if len(pkg.Functions) != 2 {
		t.Fatalf("functions: %d", len(pkg.Functions))
	}
	if got := pkg.Functions["handleCreateUser"].Usables; len(got) != 2 || got[0] != "Logger" {
		t.Fatalf("handleCreateUser usables: %v", got)
	}
	if pkg.Functions["healthCheck"].Runnable != true {
		t.Fatal("healthCheck should be runnable")
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid JSON: %s", string(raw))
	}
}

func TestDiscoverer_extractUsablesFromTypechecker(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test", logger, nil)
	tc := typechecker.New(nil, false)
	tc.FunctionUsables = map[ast.Identifier][]typechecker.UsableSlot{
		"ExpireToken": {{RootIdent: "Logger", Key: "Logger"}},
	}

	functions := make(map[string]FunctionInfo)
	fn := &ast.FunctionNode{Ident: ast.Ident{ID: "ExpireToken"}}
	discoverer.extractFunctionsFromNode(fn, "main", "/svc.ft", functions, tc)

	info, ok := functions["ExpireToken"]
	if !ok {
		t.Fatal("ExpireToken not discovered")
	}
	if len(info.Usables) != 1 || info.Usables[0] != "Logger" {
		t.Fatalf("usables = %v", info.Usables)
	}
	if info.Runnable {
		t.Fatal("ExpireToken should not be runnable")
	}
}
