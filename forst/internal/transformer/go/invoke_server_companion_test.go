package transformergo

import (
	"strings"
	"testing"

	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestInvokeServerSource_emitsCompanionWithEcho(t *testing.T) {
	src := `package main

type EchoRequest = {message: String}

func Echo(input EchoRequest) {
	return {echo: input.message, timestamp: 1}
}
`
	log := logrus.New()
	log.SetOutput(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := typechecker.New(nil, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	tr := New(tc, log, true)
	if _, err := tr.TransformForstFileToGo(nodes); err != nil {
		t.Fatal(err)
	}
	out, err := tr.InvokeServerSource(true, nodes)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"invokeserver.MustStartEmbedded",
		"forst_invoke_main_Echo",
		"ForstInvokeWaitForShutdown",
		"reg.RegisterMeta",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestInvokeServerSource_disabledReturnsEmpty(t *testing.T) {
	tr := New(typechecker.New(nil, false), nil, true)
	out, err := tr.InvokeServerSource(false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("want empty, got %q", out)
	}
}
