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

func TestEmitJSONSchema_constraintsFixture(t *testing.T) {
	req := loadConstraintsSnapshot(t)
	resp, err := emitJSONSchema(req)
	if err != nil {
		t.Fatalf("emitJSONSchema: %v", err)
	}
	if len(resp.Files) < 1 || resp.Files[0].Path != "schema.json" {
		t.Fatalf("files: %#v", resp.Files)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(resp.Files[0].Content), &doc); err != nil {
		t.Fatalf("schema json: %v", err)
	}
	defs := doc["$defs"].(map[string]any)
	username := defs["Username"].(map[string]any)
	if username["minLength"] != float64(3) || username["pattern"] != "^u_" {
		t.Fatalf("Username: %#v", username)
	}
	email := defs["EmailLine"].(map[string]any)
	if email["minLength"] != float64(1) || email["maxLength"] != float64(64) {
		t.Fatalf("EmailLine dropped min/max: %#v", email)
	}
	var emailWarn bool
	for _, d := range resp.Diagnostics {
		if d.TypeID == "catalog.EmailLine" && strings.Contains(d.Message, "Email") {
			emailWarn = true
		}
	}
	if !emailWarn {
		t.Fatalf("expected Email warning, got %#v", resp.Diagnostics)
	}
}

func TestEmitJSONSchema_unionAndRef(t *testing.T) {
	req := &semantic.GenerateRequest{
		Packages: []semantic.SemanticPackage{{
			Name: "cat", Dir: "cat", TypeIDs: []string{"cat.Name", "cat.Item"},
		}},
		Types: map[string]semantic.Type{
			"cat.Name": {
				ID: "cat.Name", Kind: "string", Visibility: "exported",
				Constraints: []semantic.Constraint{{Name: "Min", Args: []any{1}, Origin: "builtin", Applies: "length"}},
			},
			"cat.Item": {
				ID: "cat.Item", Kind: "shape", Visibility: "exported",
				Fields: []semantic.ShapeField{{Name: "name", Type: "cat.Name"}},
			},
			"cat.Either": {
				ID: "cat.Either", Kind: "union", Visibility: "exported",
				Members: []string{"cat.Name", "int"},
			},
		},
	}
	req.Packages[0].TypeIDs = append(req.Packages[0].TypeIDs, "cat.Either")
	resp, err := emitJSONSchema(req)
	if err != nil {
		t.Fatal(err)
	}
	raw := resp.Files[0].Content
	if !strings.Contains(raw, `"$ref": "#/$defs/Name"`) {
		t.Fatalf("expected $ref to Name:\n%s", raw)
	}
	if !strings.Contains(raw, `"anyOf"`) {
		t.Fatalf("expected union anyOf:\n%s", raw)
	}
}

func TestEmitJSONSchema_skipsRouterOnly(t *testing.T) {
	req := loadSnapshot(t, "layout")
	resp, err := emitJSONSchema(req)
	if err != nil {
		t.Fatal(err)
	}
	raw := resp.Files[0].Content
	if strings.Contains(raw, `"Routes"`) || strings.Contains(raw, `"Handlers"`) {
		t.Fatalf("router-only types should be omitted:\n%s", raw)
	}
	if !strings.Contains(raw, `"PingResponse"`) {
		t.Fatalf("data types should remain:\n%s", raw)
	}
}

func TestEmitJSONSchema_unknownKindWarning(t *testing.T) {
	req := &semantic.GenerateRequest{
		Packages: []semantic.SemanticPackage{{Name: "p", TypeIDs: []string{"p.X"}}},
		Types: map[string]semantic.Type{
			"p.X": {ID: "p.X", Kind: "goType", Visibility: "exported", ImportPath: "time", GoName: "Time"},
		},
	}
	resp, err := emitJSONSchema(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected warning for goType")
	}
}

func loadConstraintsSnapshot(t *testing.T) *semantic.GenerateRequest {
	t.Helper()
	return loadSnapshot(t, "constraints")
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
