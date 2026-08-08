package transformerts

import (
	"strings"
	"testing"

	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestGeneratePackageClient_emitsStreamWhenStreamingRowType(t *testing.T) {
	log := logrus.New()
	tc := typechecker.New(log, false)
	tr := New(tc, log)
	tr.Output.PackageName = "main"
	tr.Output.SourceFileStem = "api"
	tr.Output.Functions = []FunctionSignature{
		{
			Name:             "Process",
			Parameters:       []Parameter{{Name: "chunks", Type: "string[]"}},
			ReturnType:       "AsyncIterable<string>",
			StreamingRowType: "string",
		},
	}
	tr.generatePackageClient()
	out := tr.Output.GenerateClientFile()
	if !strings.Contains(out, "ProcessStream") {
		t.Fatalf("expected ProcessStream in client, got:\n%s", out)
	}
	if !strings.Contains(out, "invokeStream<string>") {
		t.Fatalf("expected typed invokeStream, got:\n%s", out)
	}
	if !strings.Contains(out, "export async function Process") {
		t.Fatalf("expected direct named export, got:\n%s", out)
	}
	// Direct delegation: no extra async generator wrapper (better perf than for-await re-yield).
	if strings.Contains(out, "async function*") {
		t.Fatalf("did not expect wrapper async generator, got:\n%s", out)
	}
	assertInlinedTransportImports(t, out)
}

func TestGeneratePackageClient_importsInlinedTransportNotForstClient(t *testing.T) {
	log := logrus.New()
	tc := typechecker.New(log, false)
	tr := New(tc, log)
	tr.Output.PackageName = "main"
	tr.Output.SourceFileStem = "api"
	tr.Output.Functions = []FunctionSignature{
		{Name: "Ping", ReturnType: "string"},
	}
	tr.generatePackageClient()
	tr.generateMainClient()
	assertInlinedTransportImports(t, tr.Output.GenerateClientFile())
	assertInlinedTransportImports(t, tr.Output.GenerateMainClient())
	main := tr.Output.GenerateMainClient()
	if !strings.Contains(main, "createInvokeClient") {
		t.Fatalf("main client should import createInvokeClient from transport")
	}
	if !strings.Contains(main, "createForstClient") {
		t.Fatalf("main client should emit createForstClient:\n%s", main)
	}
}

func assertInlinedTransportImports(t *testing.T, src string) {
	t.Helper()
	// Core modules import transport one level up; root index still uses ./transport.js.
	hasCore := strings.Contains(src, "from '"+coreTransportModuleSpecifier+"'")
	hasRoot := strings.Contains(src, "from '"+TransportModuleSpecifier+"'")
	if !hasCore && !hasRoot {
		t.Fatalf("expected import from %s or %s, got:\n%s", coreTransportModuleSpecifier, TransportModuleSpecifier, src)
	}
	for _, banned := range []string{"@forst/client", "@forst/sidecar"} {
		if strings.Contains(src, banned) {
			t.Fatalf("generated client must not contain %q:\n%s", banned, src)
		}
	}
}

func TestGeneratePackageClient_streamArgListVariants(t *testing.T) {
	tests := []struct {
		name       string
		parameters []Parameter
		wantFrag   string
	}{
		{
			name:       "zero parameters uses empty args",
			parameters: nil,
			wantFrag:   "invokeStream<string>('main', 'Stream', [])",
		},
		{
			name: "multiple parameters uses comma joined args",
			parameters: []Parameter{
				{Name: "a", Type: "string"},
				{Name: "b", Type: "number"},
			},
			wantFrag: "invokeStream<string>('main', 'Stream', [a, b])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.New()
			tc := typechecker.New(log, false)
			tr := New(tc, log)
			tr.Output.PackageName = "main"
			tr.Output.SourceFileStem = "api"
			tr.Output.Functions = []FunctionSignature{
				{
					Name:             "Stream",
					Parameters:       tt.parameters,
					ReturnType:       "AsyncIterable<string>",
					StreamingRowType: "string",
				},
			}
			tr.generatePackageClient()
			out := tr.Output.GenerateClientFile()
			if !strings.Contains(out, tt.wantFrag) {
				t.Fatalf("client missing %q:\n%s", tt.wantFrag, out)
			}
		})
	}
}
