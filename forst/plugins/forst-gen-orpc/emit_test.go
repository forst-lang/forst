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

func TestEmitORPC_routerFixture(t *testing.T) {
	req := loadSnapshot(t, "router")
	resp, err := emitORPC(req)
	if err != nil {
		t.Fatalf("emitORPC: %v", err)
	}
	contract := fileContent(resp, "contract.ts")
	if !strings.Contains(contract, "PlaceOrder") || !strings.Contains(contract, "oc") {
		t.Fatalf("contract missing PlaceOrder/oc:\n%s", contract)
	}
	if !strings.Contains(contract, "invokePositional") {
		t.Fatalf("contract missing invoke wiring:\n%s", contract)
	}
	zod := fileContent(resp, "zod.ts")
	if !strings.Contains(zod, "UsernameSchema") || !strings.Contains(zod, ".min(3)") {
		t.Fatalf("zod missing Username constraints:\n%s", zod)
	}
	invoke := fileContent(resp, "invoke.ts")
	if !strings.Contains(invoke, "invokeFunction") {
		t.Fatalf("invoke.ts should call invokeFunction:\n%s", invoke)
	}
}

func TestEmitORPC_trpcAndSubscription(t *testing.T) {
	req := loadSnapshot(t, "router")
	req.Plugin = &semantic.PluginRef{Name: "orpc", Opt: json.RawMessage(`{"style":"trpc"}`)}
	watch := semantic.Function{
		ID: "catalog.WatchOrders", Name: "WatchOrders", Package: "catalog",
		Visibility: "exported", Runnable: true, Input: "void",
		Returns: []string{"t:ch"},
	}
	req.Types["t:ch"] = semantic.Type{ID: "t:ch", Kind: "channel", Element: "string"}
	req.Types["catalog.Catalog"] = withMethod(req.Types["catalog.Catalog"], semantic.ShapeField{
		Name: "WatchOrders", Method: true, Type: "t:ch.sig", Function: "catalog.WatchOrders",
	})
	req.Functions["catalog.WatchOrders"] = watch
	req.Types["catalog.GetOrderProc"] = semantic.Type{
		ID: "catalog.GetOrderProc", Kind: "shape",
		Constraints: []semantic.Constraint{{Name: "Query", Origin: "typeGuard"}},
	}

	resp, err := emitORPC(req)
	if err != nil {
		t.Fatal(err)
	}
	router := fileContent(resp, "router.ts")
	if !strings.Contains(router, "initTRPC") || !strings.Contains(router, ".mutation") {
		t.Fatalf("trpc router:\n%s", router)
	}
	if !strings.Contains(router, ".subscription") || !strings.Contains(router, "WatchOrders") {
		t.Fatalf("expected subscription:\n%s", router)
	}
}

func TestEmitORPC_optQueriesAndHTTP(t *testing.T) {
	req := loadSnapshot(t, "router")
	req.Plugin = &semantic.PluginRef{Name: "orpc", Opt: json.RawMessage(`{
		"queries": ["catalog.Catalog.PlaceOrder"],
		"routes": { "catalog.Catalog.PlaceOrder": { "method": "POST", "path": "/orders" } }
	}`)}
	resp, err := emitORPC(req)
	if err != nil {
		t.Fatal(err)
	}
	contract := fileContent(resp, "contract.ts")
	if !strings.Contains(contract, `path: "/orders"`) {
		t.Fatalf("missing http route:\n%s", contract)
	}
}

func withMethod(t semantic.Type, field semantic.ShapeField) semantic.Type {
	t.Fields = append(append([]semantic.ShapeField{}, t.Fields...), field)
	return t
}

func loadSnapshot(t *testing.T, name string) *semantic.GenerateRequest {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	golden := filepath.Join(filepath.Dir(file), "..", "..", "internal", "semantic", "testdata", name, "snapshot.golden.json")
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
