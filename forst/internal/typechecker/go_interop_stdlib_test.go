package typechecker

import (
	"os"
	"path/filepath"
	"testing"

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
