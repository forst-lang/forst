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
