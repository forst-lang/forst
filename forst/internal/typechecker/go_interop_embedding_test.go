package typechecker

import (
	"go/types"
	"testing"

	"forst/internal/goload"
)

func TestGoTypeAtFieldPath_promotedFieldThroughEmbedding(t *testing.T) {
	t.Parallel()
	inner := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "Value", types.Typ[types.Int], false),
	}, nil)
	outer := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "Inner", inner, true),
	}, nil)
	got, err := goTypeAtFieldPath(outer, []string{"Value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != types.Typ[types.Int] {
		t.Fatalf("want int, got %s", got)
	}
}

func TestGoTypeAtFieldPath_realGoStructEmbedding(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	loaded, err := goload.LoadByPkgPath(dir, []string{"embed"})
	if err != nil || len(loaded) == 0 {
		t.Skip("go/packages could not load embed")
	}
	pkgp := loaded["embed"]
	if pkgp == nil || pkgp.Types == nil {
		t.Skip("embed types unavailable")
	}
	obj := pkgp.Types.Scope().Lookup("FS")
	if obj == nil {
		t.Skip("embed.FS not found")
	}
	// embed.FS embeds io/fs.FS (interface) — use bytes.Reader instead
	loaded, err = goload.LoadByPkgPath(dir, []string{"bytes"})
	if err != nil || len(loaded) == 0 {
		t.Skip("go/packages could not load bytes")
	}
	pkgp = loaded["bytes"]
	obj = pkgp.Types.Scope().Lookup("Buffer")
	if obj == nil {
		t.Fatal("bytes.Buffer not found")
	}
	named := obj.Type().(*types.Named)
	ptr := types.NewPointer(named)
	got, err := goTypeAtFieldPath(ptr, []string{"Len"})
	if err != nil {
		t.Fatalf("Len method value: %v", err)
	}
	if _, ok := got.(*types.Signature); !ok {
		t.Fatalf("want method signature, got %T (%s)", got, got)
	}
}
