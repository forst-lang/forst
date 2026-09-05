package genplugin

import (
	"testing"

	"forst/internal/semantic"
)

func TestDerivedCall_resultAndChannel(t *testing.T) {
	types := map[string]semantic.Type{
		"t:ch":        {ID: "t:ch", Kind: "channel", Element: "string"},
		"t:result":    {ID: "t:result", Kind: "result", Success: "string", Failure: "catalog.Err"},
		"catalog.Err": {ID: "catalog.Err", Kind: "nominalError"},
	}
	ch := DerivedCall(types, semantic.Function{Returns: []string{"t:ch"}})
	if !ch.Stream || ch.Success.Kind != "string" {
		t.Fatalf("channel: %#v", ch)
	}
	res := DerivedCall(types, semantic.Function{Returns: []string{"t:result"}})
	if res.Stream || res.Success.Kind != "string" || res.Failure.Kind != "nominalError" {
		t.Fatalf("result: %#v", res)
	}
	bare := DerivedCall(types, semantic.Function{Returns: []string{"result"}})
	if bare.Success.Kind != "unknown" {
		t.Fatalf("bare result: %#v", bare)
	}
}

func TestLookup_primitive(t *testing.T) {
	tpe := Lookup(nil, "string")
	if tpe.Kind != "string" {
		t.Fatalf("got %+v", tpe)
	}
}
