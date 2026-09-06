package typechecker

import (
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/testmod"
	"forst/internal/testutil"
)

func TestGoQualifiedCall_stdlibBackendPatterns_typecheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		src         string
		importLocal string
	}{
		{
			name: "context.Background",
			src: `package main

import "context"

func main() {
	ctx := context.Background()
	println(ctx)
}
`,
			importLocal: "context",
		},
		{
			name: "context.WithTimeout",
			src: `package main

import (
	"context"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	cancel()
	println(ctx)
}
`,
			importLocal: "context",
		},
		{
			name: "encoding/json.Marshal",
			src: `package main

import "encoding/json"

func main() {
	b, err := json.Marshal(map[String]Int{ "a": 1 })
	println(len(b))
	println(err)
}
`,
			importLocal: "json",
		},
		{
			name: "encoding/json.Unmarshal",
			src: `package main

import "encoding/json"

func main() {
	var out map[String]Int
	err := json.Unmarshal([]byte("{\"a\":1}"), &out)
	println(err)
}
`,
			importLocal: "json",
		},
		{
			name: "log/slog.Info",
			src: `package main

import "log/slog"

func main() {
	slog.Info("hello", "k", "v")
}
`,
			importLocal: "slog",
		},
		{
			name: "database/sql.Open",
			src: `package main

import "database/sql"

func main() {
	db, err := sql.Open("postgres", "postgres://localhost")
	println(db)
	println(err)
}
`,
			importLocal: "sql",
		},
		{
			name: "net/http.Get",
			src: `package main

import "net/http"

func main() {
	resp, err := http.Get("https://example.com")
	println(resp)
	println(err)
}
`,
			importLocal: "http",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			MustTypecheck(t, tt.src, testutil.TypecheckOpts{
				UseModuleRoot:      true,
				SkipUnlessGoImport: tt.importLocal,
			})
		})
	}
}

func TestGoQualifiedCall_httpHandleFunc_withFunctionLiteral_typechecks(t *testing.T) {
	t.Parallel()
	src := `package main

import "net/http"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		println("ok")
	})
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{
		UseModuleRoot:      true,
		SkipUnlessGoImport: "http",
	})
}

func TestSamePackageGoCall_unnamedStructReturn_typechecks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const modName = "structdemo"
	testmod.WriteGoMod(t, root, modName)
	pkgDir := filepath.Join(root, "app")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	helpers := `package app

func CountPair() struct { N int } {
	return struct { N int }{N: 1}
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "helpers.go"), []byte(helpers), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package app

func main() {
	p := CountPair()
	println(p.n)
}
`
	MustTypecheckMixedPackage(t, root, modName+"/app", src)
}

func TestGoMethodCall_multiReturnAssignment_typechecks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const modName = "methodpair"
	testmod.WriteGoMod(t, root, modName)
	pkgDir := filepath.Join(root, "app")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	helpers := `package app

type Builder struct{}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) Pair() (string, []any) {
	return "a", []any{"b"}
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "helpers.go"), []byte(helpers), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package app

func main() {
	b := NewBuilder()
	s, xs := b.Pair()
	println(s)
	println(string(len(xs)))
}
`
	tc, _ := MustTypecheckMixedPackage(t, root, modName+"/app", src)
	sGo := tc.variableGoTypes[ast.Identifier("s")]
	if sGo == nil || !types.Identical(sGo, types.Typ[types.String]) {
		t.Fatalf("variableGoTypes[\"s\"]: got %v want string", sGo)
	}
	xsGo := tc.variableGoTypes[ast.Identifier("xs")]
	wantXS := types.NewSlice(types.Universe.Lookup("any").Type())
	if xsGo == nil || !types.Identical(xsGo, wantXS) {
		t.Fatalf("variableGoTypes[\"xs\"]: got %v want []any", xsGo)
	}
}

func TestGoMethodCall_pointerRecv_onSliceIndex_typechecks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const modName = "sliceaddr"
	testmod.WriteGoMod(t, root, modName)
	pkgDir := filepath.Join(root, "app")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	helpers := `package app

type Item struct{ N int }

func (i *Item) Label() string { return "ok" }

func Items() []Item { return []Item{{}} }
`
	if err := os.WriteFile(filepath.Join(pkgDir, "helpers.go"), []byte(helpers), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package app

func main() {
	xs := Items()
	s := xs[0].Label()
	println(s)
}
`
	MustTypecheckMixedPackage(t, root, modName+"/app", src)
}

func TestGoMethodCall_pointerRecv_onMapIndex_errors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const modName = "mapaddr"
	testmod.WriteGoMod(t, root, modName)
	pkgDir := filepath.Join(root, "app")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	helpers := `package app

type Item struct{ N int }

func (i *Item) Label() string { return "ok" }

func ItemMap() map[string]Item { return map[string]Item{"a": {}} }
`
	if err := os.WriteFile(filepath.Join(pkgDir, "helpers.go"), []byte(helpers), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package app

func main() {
	m := ItemMap()
	s := m["a"].Label()
	println(s)
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{
		FileID:              "mixed/main.ft",
		GoWorkspaceDir:      root,
		SamePackageGoImport: modName + "/app",
	})
	if err == nil {
		t.Fatal("expected pointer-receiver method on map index to fail")
	}
}
