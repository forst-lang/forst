package transformerts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ndjsonStreamFixture is the shared streaming wire sample used to document the
// POST /invoke NDJSON contract (also on disk under testdata/ndjson/stream.ndjson).
// Later phases compare the inlined reader and @forst/sidecar against this fixture.
const ndjsonStreamFixture = `{"data":{"id":"1","text":"hello"},"status":"ok"}
{"data":{"id":"2","text":"world"},"status":"ok"}
{"data":null,"status":"done"}
`

// ndjsonPartialRowFixture documents a truncated final line that must raise InvokeStreamAborted.
const ndjsonPartialRowFixture = `{"data":{"id":"1"},"status":"ok"}
{"data":{"id":"2","status":"ok"`

func TestEmitTransportTypeScript_exportsCreateInvokeClientAndDefaults(t *testing.T) {
	src := EmitTransportTypeScript("6321")
	for _, frag := range []string{
		"export function createInvokeClient",
		"export function getDefaultInvokeClient",
		"export function configureDefaultInvokeClient",
		"export function resetDefaultInvokeClientForTest",
		"export interface ForstInvokeClient",
		"invokeFunction<",
		"invokeStream<",
	} {
		if !strings.Contains(src, frag) {
			t.Fatalf("missing API fragment %q in emitted transport", frag)
		}
	}
}

func TestEmitTransportDTS_emitsMiddlewareTypes(t *testing.T) {
	src := EmitTransportDTS()
	for _, frag := range []string{
		"export interface ForstInvokeMiddleware",
		"export interface InvokeContext",
		"onStart?",
		"onSuccess?",
		"onFailure?",
		"packageName: string",
		"functionName: string",
		"attempt: number",
		"startedAt: number",
		"middleware?: ForstInvokeMiddleware[]",
		"configureDefaultInvokeClient",
		"setActiveTestTransportResolver",
	} {
		if !strings.Contains(src, frag) {
			t.Fatalf("missing middleware fragment %q in:\n%s", frag, src)
		}
	}
}

func TestEmitTransportTypeScript_inlinesStreamingResultAndInvokeStreamAborted(t *testing.T) {
	src := EmitTransportTypeScript("6321")
	for _, frag := range []string{
		"export interface StreamingResult",
		"data: any",
		"status: string",
		`from "./invoke-errors.js"`,
		`from "./domain-errors.js"`,
		"InvokeStreamAborted",
		"rowIndex",
	} {
		if !strings.Contains(src, frag) {
			t.Fatalf("missing streaming contract fragment %q in emitted transport", frag)
		}
	}
	if strings.Contains(src, "export class InvokeStreamAborted") {
		t.Fatal("InvokeStreamAborted must live in invoke-errors.js, not be redefined in transport")
	}
}

func TestEmitTransportTypeScript_isConnectOnlyHttpPostInvoke(t *testing.T) {
	src := EmitTransportTypeScript("6321")
	for _, frag := range []string{
		`"/invoke"`,
		`method: "POST"`,
		`streaming: true`,
		`Content-Type`,
		`application/json`,
		`Never spawns`,
		`FORST_INVOKE_URL`,
		`FORST_BASE_URL`,
		`FORST_DEV_URL`,
		`http://127.0.0.1:6321`,
		"resolveTransportMode",
		`NODE_ENV`,
		"allowSpawn",
	} {
		if !strings.Contains(src, frag) {
			t.Fatalf("missing HTTP contract fragment %q in emitted transport", frag)
		}
	}
	for _, banned := range []string{
		"sidecarRuntime",
		"new ForstSidecar",
		"child_process",
		"spawn(",
	} {
		if strings.Contains(src, banned) {
			t.Fatalf("connect-only transport must not contain %q", banned)
		}
	}
}

func TestEmitTransportTypeScript_ndjsonReaderThrowsInvokeStreamAbortedOnMidRowOrParseFailure(t *testing.T) {
	src := EmitTransportTypeScript("6321")
	for _, frag := range []string{
		`indexOf("\n")`,
		"JSON.parse(line)",
		"ended mid-row",
		"failed to parse",
		"new InvokeStreamAborted",
	} {
		if !strings.Contains(src, frag) {
			t.Fatalf("missing NDJSON reader fragment %q in emitted transport", frag)
		}
	}
}

func TestEmitTransportTypeScript_honoursOptionsSignalAbort(t *testing.T) {
	src := EmitTransportTypeScript("6321")
	for _, frag := range []string{
		"options?.signal",
		"signal?.aborted",
		"AbortSignal",
		"combineSignals",
	} {
		if !strings.Contains(src, frag) {
			t.Fatalf("missing abort/signal fragment %q in emitted transport", frag)
		}
	}
}

func TestEmitTransportTypeScript_hasZeroRuntimePackageImports(t *testing.T) {
	src := EmitTransportTypeScript("6321")
	for _, banned := range []string{
		"@forst/client",
		"@forst/sidecar",
		"from \"@forst/",
		"from '@forst/",
		"require(",
		"import(",
		"from \"effect\"",
		"from 'effect'",
	} {
		if strings.Contains(src, banned) {
			t.Fatalf("zero-dependency transport must not contain %q", banned)
		}
	}
	if !strings.Contains(src, `from "./invoke-errors.js"`) {
		t.Fatal("transport must import invoke errors from ./invoke-errors.js")
	}
	if !strings.Contains(src, `from "./domain-errors.js"`) {
		t.Fatal("transport must import decodeDomainError from ./domain-errors.js")
	}
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "import ") {
			continue
		}
		if strings.Contains(trimmed, " from ") &&
			!strings.Contains(trimmed, `"./invoke-errors.js"`) &&
			!strings.Contains(trimmed, `"./domain-errors.js"`) &&
			!strings.HasPrefix(trimmed, "import {") {
			t.Fatalf("transport may only import error modules, got:\n%s", line)
		}
	}
}

func TestEmitTransportTypeScript_usesInvokePortInDefaultBaseUrl(t *testing.T) {
	src := EmitTransportTypeScript("8081")
	if !strings.Contains(src, "http://127.0.0.1:8081") {
		t.Fatalf("expected custom invoke port in default base URL, got no 8081")
	}
	if strings.Contains(src, "http://127.0.0.1:6321") {
		t.Fatalf("custom port emit still contains default 6321")
	}
	if strings.Contains(src, transportInvokePortPlaceholder) {
		t.Fatalf("placeholder %q left unsubstituted", transportInvokePortPlaceholder)
	}
}

func TestEmitTransportTypeScript_emptyPortDefaultsTo6321(t *testing.T) {
	src := EmitTransportTypeScript("")
	if !strings.Contains(src, "http://127.0.0.1:"+DefaultInvokePort) {
		t.Fatalf("empty port should default to %s", DefaultInvokePort)
	}
}

func TestTransport_ndjsonFixtureDocumentsStreamingWireFormat(t *testing.T) {
	path := filepath.Join("testdata", "ndjson", "stream.ndjson")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	got := string(raw)
	if got != ndjsonStreamFixture {
		t.Fatalf("testdata/ndjson/stream.ndjson drifted from ndjsonStreamFixture const")
	}
	for _, frag := range []string{
		`"data"`,
		`"status":"ok"`,
		`"status":"done"`,
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("fixture missing %q", frag)
		}
	}
	if !strings.Contains(ndjsonPartialRowFixture, `{"data":{"id":"2","status":"ok"`) {
		t.Fatalf("partial-row fixture should be truncated JSON without closing brace")
	}
	// Emitted reader must be able to consume complete fixture lines and reject partial ones.
	src := EmitTransportTypeScript("6321")
	if !strings.Contains(src, "JSON.parse(line)") {
		t.Fatalf("emitted transport must parse each NDJSON line")
	}
	if !strings.Contains(src, "ended mid-row") {
		t.Fatalf("emitted transport must reject mid-row EOF (partial fixture case)")
	}
}

func firstImportLine(src string) string {
	for _, line := range strings.Split(src, "\n") {
		if strings.Contains(line, "import ") {
			return line
		}
	}
	return ""
}
