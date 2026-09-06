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
	if !strings.Contains(zod, "UsernameSchema") || !strings.Contains(zod, "[...v].length >= 3") {
		t.Fatalf("zod missing Username code-point Min:\n%s", zod)
	}
	invoke := fileContent(resp, "invoke.ts")
	if !strings.Contains(invoke, "invokeFunction") {
		t.Fatalf("invoke.ts should call invokeFunction:\n%s", invoke)
	}
}

func TestEmitORPC_zodUniqueSchemaNames(t *testing.T) {
	req := &semantic.GenerateRequest{
		Packages: []semantic.SemanticPackage{{Name: "catalog", TypeIDs: []string{"catalog.Id", "orders.Id"}}},
		Types: map[string]semantic.Type{
			"catalog.Id": {ID: "catalog.Id", Kind: "string", Visibility: "exported"},
			"orders.Id":  {ID: "orders.Id", Kind: "string", Visibility: "exported"},
		},
	}
	z := newZodEnc(req.Types)
	z.need("catalog.Id")
	z.need("orders.Id")
	out := z.emit()
	if !strings.Contains(out, "export const IdSchema") {
		t.Fatalf("expected first id schema:\n%s", out)
	}
	if !strings.Contains(out, "export const orders_IdSchema") {
		t.Fatalf("expected disambiguated orders id schema:\n%s", out)
	}
	if strings.Count(out, "export const IdSchema") > 1 {
		t.Fatalf("duplicate IdSchema exports:\n%s", out)
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

func TestZodEnc_valueConstraint(t *testing.T) {
	types := map[string]semantic.Type{
		"p.Status": {
			ID: "p.Status", Kind: "string",
			Constraints: []semantic.Constraint{{Name: "Value", Args: []any{"active"}, Origin: "builtin"}},
		},
	}
	z := newZodEnc(types)
	z.need("p.Status")
	out := z.emit()
	if !strings.Contains(out, `z.literal("active")`) {
		t.Fatalf("expected Value literal:\n%s", out)
	}
}

func TestZodEnc_stringMinMax_codePointsNotUtf16(t *testing.T) {
	types := map[string]semantic.Type{
		"p.Name": {
			ID: "p.Name", Kind: "string",
			Constraints: []semantic.Constraint{
				{Name: "Min", Args: []any{float64(3)}, Origin: "builtin", Applies: "length"},
				{Name: "Max", Args: []any{float64(10)}, Origin: "builtin", Applies: "length"},
				{Name: "MaxBytes", Args: []any{float64(32)}, Origin: "builtin", Applies: "bytes"},
			},
		},
		"p.Qty": {
			ID: "p.Qty", Kind: "int",
			Constraints: []semantic.Constraint{
				{Name: "Min", Args: []any{float64(1)}, Origin: "builtin", Applies: "value"},
			},
		},
	}
	z := newZodEnc(types)
	z.need("p.Name")
	z.need("p.Qty")
	out := z.emit()
	for _, want := range []string{
		`.refine((v: string) => [...v].length >= 3)`,
		`.refine((v: string) => [...v].length <= 10)`,
		`.refine((v: string) => new TextEncoder().encode(v).length <= 32)`,
		`.min(1)`, // int Min stays numeric
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "NameSchema") && strings.Contains(out, "z.string().min(") {
		t.Fatalf("string Min must not use Zod UTF-16 .min:\n%s", out)
	}
}

func TestZodEnc_stringNotEmpty_codePoints(t *testing.T) {
	types := map[string]semantic.Type{
		"p.Tag": {
			ID: "p.Tag", Kind: "string",
			Constraints: []semantic.Constraint{{Name: "NotEmpty", Origin: "builtin"}},
		},
	}
	z := newZodEnc(types)
	z.need("p.Tag")
	out := z.emit()
	if !strings.Contains(out, `.refine((v: string) => [...v].length >= 1)`) {
		t.Fatalf("NotEmpty on string should use typed code-point refine:\n%s", out)
	}
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
