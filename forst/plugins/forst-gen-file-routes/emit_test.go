package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/semantic"
)

func TestEmitFileRoutes_layoutFixture(t *testing.T) {
	req := loadLayoutSnapshot(t)
	req.Plugin = &semantic.PluginRef{
		Name: "file-routes",
		Opt:  json.RawMessage(`{"routesRoot":"app/api","paramStyle":"$id"}`),
	}
	resp, err := emitFileRoutes(req)
	if err != nil {
		t.Fatalf("emitFileRoutes: %v", err)
	}
	registry := fileContent(resp, "registry.ts")
	if !strings.Contains(registry, "/api/routes") || !strings.Contains(registry, "GET") {
		t.Fatalf("registry missing route:\n%s", registry)
	}
	if strings.Contains(registry, "/app/api/") {
		t.Fatalf("URL must strip app/: \n%s", registry)
	}
	if !strings.Contains(registry, "invokeFunction") || !strings.Contains(registry, "matchRoute") {
		t.Fatalf("dispatch incomplete:\n%s", registry)
	}
	if fileContent(resp, "handlers/routes.ts") == "" {
		t.Fatal("missing handlers/routes.ts")
	}
	if fileContent(resp, "runtime.ts") == "" {
		t.Fatal("missing runtime.ts")
	}
}

func TestEmitFileRoutes_pathParams(t *testing.T) {
	req := paramSnapshot()
	resp, err := emitFileRoutes(req)
	if err != nil {
		t.Fatal(err)
	}
	registry := fileContent(resp, "registry.ts")
	if !strings.Contains(registry, "/api/orders/:id") {
		t.Fatalf("param path:\n%s", registry)
	}
	h := fileContent(resp, "handlers/orders.$id.ts")
	if !strings.Contains(h, `params["id"]`) {
		t.Fatalf("handler should bind path param:\n%s", h)
	}
}

func TestEmitFileRoutes_skipsUnrunnable(t *testing.T) {
	req := paramSnapshot()
	fn := req.Functions["ordersid.GET"]
	fn.Runnable = false
	req.Functions["ordersid.GET"] = fn
	resp, err := emitFileRoutes(req)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Message, "not runnable") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected runnable diagnostic: %#v", resp.Diagnostics)
	}
	registry := fileContent(resp, "registry.ts")
	if strings.Contains(registry, `"GET": {`) {
		t.Fatalf("registry should not emit GET for unrunnable handler:\n%s", registry)
	}
	h := fileContent(resp, "handlers/orders.$id.ts")
	if strings.Contains(h, "export async function GET") {
		t.Fatal("should not emit GET handler for unrunnable function")
	}
}

func TestEmitFileRoutes_paramMismatchDiagnostic(t *testing.T) {
	req := paramSnapshot()
	fn := req.Functions["ordersid.GET"]
	fn.Params = nil
	req.Functions["ordersid.GET"] = fn
	del := req.Functions["ordersid.DELETE"]
	del.Params = nil
	req.Functions["ordersid.DELETE"] = del
	resp, err := emitFileRoutes(req)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Message, "path param") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected param mismatch diag: %#v", resp.Diagnostics)
	}
}

func paramSnapshot() *semantic.GenerateRequest {
	return &semantic.GenerateRequest{
		Packages: []semantic.SemanticPackage{{
			Name: "ordersid", Dir: "app/api/orders", Files: []string{"$id.ft"},
			TypeIDs: []string{"ordersid.OrdersId"}, FunctionIDs: []string{"ordersid.GET"},
		}},
		Types: map[string]semantic.Type{
			"ordersid.OrdersId": {
				ID: "ordersid.OrdersId", Kind: "shape", Visibility: "exported",
				Constraints: []semantic.Constraint{{Name: "Router", Origin: "builtin"}},
				Fields: []semantic.ShapeField{
					{Name: "GET", Method: true, Type: "sig", Function: "ordersid.GET"},
					{Name: "DELETE", Method: true, Type: "sig", Function: "ordersid.DELETE"},
				},
			},
		},
		Functions: map[string]semantic.Function{
			"ordersid.GET": {
				ID: "ordersid.GET", Name: "GET", Package: "ordersid",
				Visibility: "exported", Runnable: true,
				Params: []semantic.FuncParam{{Name: "id", Type: "string"}},
				Input:  "string", Returns: []string{"string"},
				Span: &semantic.SourceSpan{File: "app/api/orders/$id.ft"},
			},
			"ordersid.DELETE": {
				ID: "ordersid.DELETE", Name: "DELETE", Package: "ordersid",
				Visibility: "exported", Runnable: true,
				Params: []semantic.FuncParam{{Name: "id", Type: "string"}},
				Input:  "string", Returns: []string{"void"},
				Span: &semantic.SourceSpan{File: "app/api/orders/$id.ft"},
			},
		},
	}
}

func loadLayoutSnapshot(t *testing.T) *semantic.GenerateRequest {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	golden := filepath.Join(filepath.Dir(file), "..", "..", "internal", "semantic", "testdata", "layout", "snapshot.golden.json")
	raw, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var req semantic.GenerateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &req
}

func fileContent(resp semantic.GenerateResponse, name string) string {
	for _, f := range resp.Files {
		if f.Path == name {
			return f.Content
		}
	}
	return ""
}
