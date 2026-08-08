package transformerts

import (
	"strings"
	"testing"
)

func sampleBcryptModule() ModuleEmit {
	return ModuleEmit{
		PackageName: "bcrypt",
		Functions: []FunctionSignature{
			{
				Name: "ComparePassword",
				Parameters: []Parameter{
					{Name: "input", Type: "ComparePasswordRequest"},
				},
				ReturnType: "ComparePasswordResponse",
			},
		},
		TypeImports: []string{"ComparePasswordResponse", "ComparePasswordRequest"},
	}
}

func sampleStreamModule() ModuleEmit {
	return ModuleEmit{
		PackageName: "events",
		Functions: []FunctionSignature{
			{
				Name:             "Watch",
				Parameters:       []Parameter{{Name: "query", Type: "WatchRequest"}},
				ReturnType:       "AsyncIterable<WatchRow>",
				StreamingRowType: "WatchRow",
			},
		},
		TypeImports: []string{"WatchRequest", "WatchRow"},
	}
}

func assertContainsAll(t *testing.T, got string, frags []string) {
	t.Helper()
	for _, frag := range frags {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in:\n%s", frag, got)
		}
	}
}

func assertContainsNone(t *testing.T, got string, banned []string) {
	t.Helper()
	for _, frag := range banned {
		if strings.Contains(got, frag) {
			t.Fatalf("must not contain %q in:\n%s", frag, got)
		}
	}
}

func TestEmitCoreESM_golden(t *testing.T) {
	got := EmitCoreESM(sampleBcryptModule(), "6321")
	assertContainsAll(t, got, []string{
		`import { getDefaultInvokeClient } from "../transport.js"`,
		"export const bcrypt = (client) => ({",
		"export async function ComparePassword(input, options)",
		`options?.transport ?? getDefaultInvokeClient()`,
		`client.invokeFunction("bcrypt", "ComparePassword", [input], options)`,
		"ComparePassword.safe = async (input, options)",
		`{ ok: true, value: await ComparePassword(input, options) }`,
		`{ ok: false, error }`,
		"http://127.0.0.1:6321",
	})
	assertContainsNone(t, got, []string{
		"@forst/client",
		"@forst/sidecar",
		"8081",
		".ts'",
		".cjs",
		"require(",
	})
}

func TestEmitCoreESM_headerCommentUsesInvokePort(t *testing.T) {
	got := EmitCoreESM(sampleBcryptModule(), "9999")
	if !strings.Contains(got, "http://127.0.0.1:9999") {
		t.Fatalf("expected invoke port 9999 in header, got:\n%s", got)
	}
	if strings.Contains(got, "8081") {
		t.Fatalf("must not hardcode 8081:\n%s", got)
	}
}

func TestEmitCoreESM_emptyPortDefaultsTo6321(t *testing.T) {
	got := EmitCoreESM(sampleBcryptModule(), "")
	if !strings.Contains(got, "http://127.0.0.1:6321") {
		t.Fatalf("empty port should default to 6321:\n%s", got)
	}
}

func TestEmitCoreESM_emitsStreamHelperWhenStreamingRowType(t *testing.T) {
	got := EmitCoreESM(sampleStreamModule(), "6321")
	assertContainsAll(t, got, []string{
		"WatchStream:",
		"export function WatchStream(query, options)",
		`client.invokeStream("events", "Watch", [query], options)`,
	})
}

func TestEmitCoreESM_zeroArityUsesEmptyArgsArray(t *testing.T) {
	m := ModuleEmit{
		PackageName: "main",
		Functions: []FunctionSignature{
			{Name: "Ping", ReturnType: "string"},
		},
	}
	got := EmitCoreESM(m, "6321")
	assertContainsAll(t, got, []string{
		"export async function Ping(options)",
		`client.invokeFunction("main", "Ping", [], options)`,
	})
}

func TestEmitCoreDTS_golden(t *testing.T) {
	got := EmitCoreDTS(sampleBcryptModule())
	assertContainsAll(t, got, []string{
		`import type { ForstInvokeClient, InvokeCallOptions } from "../transport.js"`,
		`import type { ComparePasswordRequest, ComparePasswordResponse } from "../types.js"`,
		"export declare const bcrypt:",
		"export declare function ComparePassword(",
		"options?: InvokeCallOptions",
		"Promise<ComparePasswordResponse>",
		"export declare namespace ComparePassword",
		"function safe(",
		"{ ok: true; value: ComparePasswordResponse }",
		"{ ok: false; error: InvokeFailure }",
		`import type { InvokeFailure } from "../errors.js"`,
	})
	// TypeImports are sorted.
	reqIdx := strings.Index(got, "ComparePasswordRequest")
	respIdx := strings.Index(got, "ComparePasswordResponse")
	if reqIdx < 0 || respIdx < 0 || reqIdx > respIdx {
		t.Fatalf("TypeImports should be sorted Request before Response:\n%s", got)
	}
}

func TestEmitCoreDTS_importsStreamingResultWhenNeeded(t *testing.T) {
	got := EmitCoreDTS(sampleStreamModule())
	assertContainsAll(t, got, []string{
		"StreamingResult",
		"WatchStream",
		`from "../types.js"`,
	})
}

func TestEmitPackageESM_omitStubsAppendCommentedLines(t *testing.T) {
	m := sampleBcryptModule()
	m.Omitted = []OmittedFunction{{
		PackageName:  "bcrypt",
		FunctionName: "NeedsDb",
		Reason:       `provider "db" not satisfied`,
		ExportDecl:   "export function NeedsDb(input: NeedsDbRequest): Promise<NeedsDbResponse>;",
	}}
	got := EmitPackageESM(m, RuntimePromise, "@forst/gen")
	for _, want := range []string{
		`// export function NeedsDb(input: NeedsDbRequest): Promise<NeedsDbResponse>;`,
		`// omitted: provider "db" not satisfied`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\nexport function NeedsDb(") {
		t.Fatalf("stub must stay commented:\n%s", got)
	}
}

func TestEmitPackageESM_golden(t *testing.T) {
	got := EmitPackageESM(sampleBcryptModule(), RuntimePromise, "@forst/gen")
	assertContainsAll(t, got, []string{
		`export * from "../core/bcrypt.js"`,
		"Promise mode re-exports core",
	})
	assertContainsNone(t, got, []string{
		"@forst/client",
		"effect",
		"invokeFunction",
	})
}

func TestEmitPackageDTS_golden(t *testing.T) {
	got := EmitPackageDTS(sampleBcryptModule(), RuntimePromise, "@forst/gen")
	assertContainsAll(t, got, []string{
		`export * from "../core/bcrypt.js"`,
		"export type { ComparePasswordRequest, ComparePasswordResponse }",
		`from "../types.js"`,
	})
}

func TestEmitPackageESM_effectMode_wrapsCore(t *testing.T) {
	got := EmitPackageESM(sampleBcryptModule(), RuntimeEffect, "@forst/gen")
	assertContainsAll(t, got, []string{
		`import { Effect } from "effect"`,
		`import { ForstTransport, withTransport } from "../effect.js"`,
		`import * as core from "../core/bcrypt.js"`,
		`export class Bcrypt extends Effect.Service()`,
		`"@forst/gen/Bcrypt"`,
		"Effect.tryPromise",
		"accessors: true",
		"dependencies: [ForstTransport.Default]",
		"export const { ComparePassword } = Bcrypt",
		`export { bcrypt } from "../core/bcrypt.js"`,
	})
	assertContainsNone(t, got, []string{
		".safe",
		"AbortController",
	})
}

func TestEmitIndexESM_golden(t *testing.T) {
	got := EmitIndexESM([]string{"auth", "bcrypt"}, "6321")
	assertContainsAll(t, got, []string{
		`import { createInvokeClient, configureDefaultInvokeClient } from "./transport.js"`,
		`import { auth } from "./pkg/auth.js"`,
		`import { bcrypt } from "./pkg/bcrypt.js"`,
		"export function createForstClient(config)",
		"export { configureDefaultInvokeClient }",
		"auth: auth(client)",
		"bcrypt: bcrypt(client)",
		"http://127.0.0.1:6321",
		"InvokeRejected",
		"isInvokeFailure",
		`from "./errors.js"`,
	})
	assertContainsNone(t, got, []string{
		"export { ComparePassword",
		"@forst/client",
		".cjs",
	})
}

func TestEmitIndexESM_sortsPackages(t *testing.T) {
	got := EmitIndexESM([]string{"bcrypt", "auth"}, "6321")
	authImport := strings.Index(got, `import { auth } from "./pkg/auth.js"`)
	bcryptImport := strings.Index(got, `import { bcrypt } from "./pkg/bcrypt.js"`)
	if authImport < 0 || bcryptImport < 0 || authImport > bcryptImport {
		t.Fatalf("packages should be sorted alphabetically:\n%s", got)
	}
}

func TestEmitIndexDTS_golden(t *testing.T) {
	got := EmitIndexDTS([]string{"bcrypt"})
	assertContainsAll(t, got, []string{
		"ForstInvokeClientConfig",
		"ForstInvokeMiddleware",
		"InvokeCallOptions",
		"InvokeContext",
		`from "./transport.js"`,
		`import { bcrypt } from "./pkg/bcrypt.js"`,
		"export interface ForstClientConfig",
		"middleware?: ForstInvokeMiddleware[]",
		"export declare function createForstClient(",
		"export declare function configureDefaultInvokeClient(",
		"readonly bcrypt: ReturnType<typeof bcrypt>",
		"export type ForstClient = ReturnType<typeof createForstClient>",
		"InvokeFailure",
		"TaggedError",
		"isInvokeFailure",
		"InvokeRejected",
		`from "./errors.js"`,
		`export type * from "./types.js"`,
	})
}

func TestEmitTransportESM_isConnectOnlyHttpWithNdjson(t *testing.T) {
	got := EmitTransportESM("6321")
	assertContainsAll(t, got, []string{
		"export function createInvokeClient",
		"export function getDefaultInvokeClient",
		"export function resetDefaultInvokeClientForTest",
		`from "./errors.js"`,
		"new InvokeStreamAborted",
		"new InvokeRejected",
		"new InvokeHttpFailure",
		"new InvokeUnreachable",
		"new InvokeTimedOut",
		`"/invoke"`,
		`method: "POST"`,
		`streaming: true`,
		`indexOf("\n")`,
		"JSON.parse(line)",
		"ended mid-row",
		"http://127.0.0.1:6321",
		"Never spawns",
		"resolveTransportMode",
	})
	assertContainsNone(t, got, []string{
		"@forst/client",
		"@forst/sidecar",
		"export interface",
		"sidecarRuntime",
		"child_process",
		"spawn(",
		".cjs",
		"throw new Error(",
		"from \"effect\"",
	})
}

func TestEmitTransportESM_sharesRuntimeWithEmitTransportTypeScript(t *testing.T) {
	js := EmitTransportESM("6321")
	ts := EmitTransportTypeScript("6321")
	for _, frag := range []string{
		"async invokeFunction(packageName, functionName, args = [], options)",
		"async *invokeStream(packageName, functionName, args = [], options)",
		`streaming: true`,
		"ended mid-row",
	} {
		if !strings.Contains(js, frag) {
			t.Fatalf("ESM missing shared runtime fragment %q", frag)
		}
		if !strings.Contains(ts, frag) {
			t.Fatalf("TypeScript emit missing shared runtime fragment %q", frag)
		}
	}
}

func TestEmitTransportDTS_declaresPublicSurface(t *testing.T) {
	got := EmitTransportDTS()
	assertContainsAll(t, got, []string{
		"export interface StreamingResult",
		"export interface InvokeSuccess",
		"export interface InvokeCallOptions",
		"transport?: ForstInvokeClient",
		"export interface ForstInvokeClientConfig",
		"allowSpawn?: boolean",
		"export interface ForstInvokeClient",
		"invokeFunction<",
		"invokeStream<",
		"export declare function createInvokeClient",
		"export declare function getDefaultInvokeClient",
		"export declare function resetDefaultInvokeClientForTest",
	})
	assertContainsNone(t, got, []string{
		"class HttpInvokeClient",
		"JSON.parse",
		"fetchFn(this.baseUrl",
		"export declare class InvokeStreamAborted",
	})
}

func TestEmitTypesDTS_golden(t *testing.T) {
	shapes := []string{
		"export interface ComparePasswordRequest {\n  plainPassword: string;\n}",
		"export interface ComparePasswordResponse {\n  valid: boolean;\n}",
	}
	got := EmitTypesDTS(shapes)
	assertContainsAll(t, got, []string{
		"// Auto-generated types for Forst client",
		"export interface ComparePasswordRequest",
		"export interface ComparePasswordResponse",
		"plainPassword: string",
		"valid: boolean",
	})
	assertContainsNone(t, got, []string{
		"export function",
		"@forst/sidecar",
	})
}

func TestEmitTypesDTS_emptyShapesStillHasHeader(t *testing.T) {
	got := EmitTypesDTS(nil)
	if !strings.Contains(got, "Auto-generated types for Forst client") {
		t.Fatalf("expected header, got:\n%s", got)
	}
	if strings.Contains(got, "// Type definitions") {
		t.Fatalf("empty shapes should omit type definitions section:\n%s", got)
	}
}

func TestEmitInvokeCall_formatsArgsAndOptions(t *testing.T) {
	got := emitInvokeCall("bcrypt", "ComparePassword", "input")
	want := `client.invokeFunction("bcrypt", "ComparePassword", [input], options)`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	empty := emitInvokeCall("main", "Ping", "")
	wantEmpty := `client.invokeFunction("main", "Ping", [], options)`
	if empty != wantEmpty {
		t.Fatalf("got %q want %q", empty, wantEmpty)
	}
}

func TestEmitCoreESM_isByteStableForSameModuleEmit(t *testing.T) {
	// EmitCoreESM has no mode parameter; Promise and Effect share this emit.
	a := EmitCoreESM(sampleBcryptModule(), "6321")
	b := EmitCoreESM(sampleBcryptModule(), "6321")
	if a != b {
		t.Fatal("EmitCoreESM must be byte-stable for the same ModuleEmit")
	}
}
