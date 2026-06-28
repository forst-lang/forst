package discovery

import (
	"encoding/json"
	"fmt"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func TestBuildProvidersDiscoveryV1_schema(t *testing.T) {
	functions := map[string]map[string]FunctionInfo{
		"example.com/users": {
			"handleCreateUser": {
				Providers:  []string{"Logger", "UserRepo"},
				Runnable: false,
			},
			"healthCheck": {
				Runnable: true,
			},
		},
	}
	doc := BuildProvidersDiscoveryV1(functions)
	if doc.Version != 1 {
		t.Fatalf("version = %d", doc.Version)
	}
	pkg := doc.Packages["example.com/users"]
	if len(pkg.Functions) != 2 {
		t.Fatalf("functions: %d", len(pkg.Functions))
	}
	if got := pkg.Functions["handleCreateUser"].Providers; len(got) != 2 || got[0] != "Logger" {
		t.Fatalf("handleCreateUser providers: %v", got)
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

func TestDiscoverer_extractProvidersFromTypechecker(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test", logger, nil)
	tc := typechecker.New(nil, false)
	tc.FunctionProviders = map[ast.Identifier][]typechecker.ProviderSlot{
		"ExpireToken": {{RootIdent: "Logger", Key: "Logger"}},
	}

	functions := make(map[string]FunctionInfo)
	fn := &ast.FunctionNode{Ident: ast.Ident{ID: "ExpireToken"}}
	discoverer.extractFunctionsFromNode(fn, "main", "/svc.ft", functions, tc)

	info, ok := functions["ExpireToken"]
	if !ok {
		t.Fatal("ExpireToken not discovered")
	}
	if len(info.Providers) != 1 || info.Providers[0] != "Logger" {
		t.Fatalf("providers = %v", info.Providers)
	}
	if info.Runnable {
		t.Fatal("ExpireToken should not be runnable")
	}
}

func TestDiscoverer_DiscoverProvidersJSONV1_configError(t *testing.T) {
	discoverer := NewDiscoverer("/test", &MockLogger{}, &MockConfig{err: fmt.Errorf("config error")})
	_, err := discoverer.DiscoverProvidersJSONV1()
	if err == nil {
		t.Fatal("expected error when discovery fails")
	}
}
