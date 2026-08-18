package genplugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"forst/internal/semantic"
)

func TestRouterSurfaces_routerGolden(t *testing.T) {
	req := loadGolden(t, "router")
	surfaces := RouterSurfaces(req, []string{"Router"})
	if len(surfaces) != 1 {
		t.Fatalf("surfaces = %d, want 1", len(surfaces))
	}
	if surfaces[0].TypeID != "catalog.Catalog" {
		t.Fatalf("type id = %q", surfaces[0].TypeID)
	}
	if len(surfaces[0].Methods) != 1 || surfaces[0].Methods[0].FieldName != "PlaceOrder" {
		t.Fatalf("methods = %#v", surfaces[0].Methods)
	}
}

func TestFileRouterSurfaces_layoutGolden(t *testing.T) {
	req := loadGolden(t, "layout")
	surfaces, diags := FileRouterSurfaces(req, FileRouteOptions{RoutesRoot: "app/api", Markers: []string{"Router"}})
	if len(diags) > 0 {
		t.Fatalf("diags = %#v", diags)
	}
	if len(surfaces) != 2 {
		t.Fatalf("surfaces = %d, want 2", len(surfaces))
	}
	if surfaces[0].RoutePath != "/api/handlers" && surfaces[1].RoutePath != "/api/handlers" {
		t.Fatalf("paths = %q %q", surfaces[0].RoutePath, surfaces[1].RoutePath)
	}
}

func loadGolden(t *testing.T, name string) *semantic.GenerateRequest {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "semantic", "testdata", name, "snapshot.golden.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var req semantic.GenerateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &req
}
