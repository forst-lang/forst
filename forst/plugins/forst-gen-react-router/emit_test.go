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

func TestEmitReactRouter_layoutFixture(t *testing.T) {
	req := loadLayout(t)
	resp, err := emitReactRouter(req)
	if err != nil {
		t.Fatal(err)
	}
	routes := fileContent(resp, "routes.ts")
	if !strings.Contains(routes, `route("api/routes"`) || !strings.Contains(routes, "@react-router/dev/routes") {
		t.Fatalf("routes.ts:\n%s", routes)
	}
	if strings.Contains(routes, "app/") && strings.Contains(routes, "export default") {
		t.Fatal("must not emit user app modules")
	}
	h := fileContent(resp, "handlers/routes.ts")
	if !strings.Contains(h, "export async function loader") {
		t.Fatalf("GET should become loader:\n%s", h)
	}
	if !strings.Contains(h, "$api.GET") {
		t.Fatalf("package invoke:\n%s", h)
	}
	if strings.Contains(h, "export default") {
		t.Fatal("resource routes must not have a default component")
	}
	post := fileContent(resp, "handlers/handlers.ts")
	if !strings.Contains(post, "export async function action") {
		t.Fatalf("POST should become action:\n%s", post)
	}
	loaders := fileContent(resp, "loaders.ts")
	if !strings.Contains(loaders, "export async function load") {
		t.Fatalf("page load helpers:\n%s", loaders)
	}
}

func TestEmitReactRouter_pathParamLoader(t *testing.T) {
	req := &semantic.GenerateRequest{
		Packages: []semantic.SemanticPackage{{
			Name: "ordersid", Dir: "app/api/orders", TypeIDs: []string{"ordersid.OrdersId"},
			FunctionIDs: []string{"ordersid.GET"},
		}},
		Types: map[string]semantic.Type{
			"ordersid.OrdersId": {
				ID: "ordersid.OrdersId", Kind: "shape", Visibility: "exported",
				Constraints: []semantic.Constraint{{Name: "Router", Origin: "builtin"}},
				Fields:      []semantic.ShapeField{{Name: "GET", Method: true, Function: "ordersid.GET"}},
			},
		},
		Functions: map[string]semantic.Function{
			"ordersid.GET": {
				ID: "ordersid.GET", Name: "GET", Package: "ordersid",
				Visibility: "exported", Runnable: true,
				Params: []semantic.FuncParam{{Name: "id", Type: "string"}},
				Span:   &semantic.SourceSpan{File: "app/api/orders/$id.ft"},
			},
		},
	}
	resp, err := emitReactRouter(req)
	if err != nil {
		t.Fatal(err)
	}
	h := fileContent(resp, "handlers/orders.$id.ts")
	if !strings.Contains(h, `$ordersid.GET`) || !strings.Contains(h, `params["id"]`) {
		t.Fatalf("handler:\n%s", h)
	}
	routes := fileContent(resp, "routes.ts")
	if !strings.Contains(routes, `route("api/orders/:id", "./handlers/orders.$id.ts")`) {
		t.Fatalf("rr path:\n%s", routes)
	}
}

func TestEmitReactRouter_skipsUnrunnable(t *testing.T) {
	req := loadLayout(t)
	fn := req.Functions["api.GET"]
	fn.Runnable = false
	req.Functions["api.GET"] = fn
	resp, err := emitReactRouter(req)
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
	if strings.Contains(fileContent(resp, "handlers/routes.ts"), "function loader") {
		t.Fatal("should not emit loader for unrunnable GET")
	}
}

func loadLayout(t *testing.T) *semantic.GenerateRequest {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	golden := filepath.Join(filepath.Dir(file), "..", "..", "internal", "semantic", "testdata", "layout", "snapshot.golden.json")
	raw, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var req semantic.GenerateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
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
