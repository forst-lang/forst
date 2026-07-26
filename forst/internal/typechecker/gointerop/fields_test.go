package gointerop_test

import (
	"go/types"
	"testing"

	"forst/internal/typechecker/gointerop"
)

func TestTypeAtFieldPath_promotedFieldThroughEmbedding(t *testing.T) {
	t.Parallel()
	// type Inner struct { Value int }
	// type Outer struct { Inner }
	inner := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "Value", types.Typ[types.Int], false),
	}, nil)
	outer := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "Inner", inner, true),
	}, nil)
	got, err := gointerop.TypeAtFieldPath(outer, []string{"Value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != types.Typ[types.Int] {
		t.Fatalf("want int, got %s", got)
	}
}

func TestTypeAtFieldPath_missingName(t *testing.T) {
	t.Parallel()
	st := types.NewStruct(nil, nil)
	_, err := gointerop.TypeAtFieldPath(st, []string{"Missing"})
	if err == nil {
		t.Fatal("expected error for missing field")
	}
}
