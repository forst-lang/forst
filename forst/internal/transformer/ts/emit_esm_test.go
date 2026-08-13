package transformerts

import (
	"strings"
	"testing"
)

func sampleAuthModule() ModuleEmit {
	return ModuleEmit{
		PackageName: "auth",
		Functions: []FunctionSignature{
			{
				Name: "VerifyToken",
				Parameters: []Parameter{
					{Name: "input", Type: "VerifyTokenRequest"},
				},
				ReturnType: "VerifyTokenResponse",
			},
		},
		TypeImports: []string{"VerifyTokenResponse", "VerifyTokenRequest"},
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
	got := EmitCoreESM(sampleAuthModule(), "6321")
	assertContainsAll(t, got, []string{
		`import { getDefaultInvokeClient } from "../transport.js"`,
		"export const auth = (client) => ({",
		"export async function VerifyToken(input, options)",
		`options?.transport ?? getDefaultInvokeClient()`,
		`client.invokeFunction("auth", "VerifyToken", [input], options)`,
		"VerifyToken.safe = async (input, options)",
		`{ ok: true, value: await VerifyToken(input, options) }`,
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
	got := EmitCoreESM(sampleAuthModule(), "9999")
	if !strings.Contains(got, "http://127.0.0.1:9999") {
		t.Fatalf("expected invoke port 9999 in header, got:\n%s", got)
	}
	if strings.Contains(got, "8081") {
		t.Fatalf("must not hardcode 8081:\n%s", got)
	}
}

func TestEmitCoreESM_emptyPortDefaultsTo6321(t *testing.T) {
	got := EmitCoreESM(sampleAuthModule(), "")
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
	got := EmitCoreDTS(sampleAuthModule())
	assertContainsAll(t, got, []string{
		`import type { ForstInvokeClient, InvokeCallOptions } from "../transport.js"`,
		`import type { VerifyTokenRequest, VerifyTokenResponse } from "../types.js"`,
		"export declare const auth:",
		"/** @throws {InvokeFailure} */",
		"export declare function VerifyToken(",
		"options?: InvokeCallOptions",
		"Promise<VerifyTokenResponse>",
		"export declare namespace VerifyToken",
		"function safe(",
		"{ ok: true; value: VerifyTokenResponse }",
		"{ ok: false; error: InvokeFailure }",
		`import type { InvokeFailure } from "../$errors.js"`,
	})
	// TypeImports are sorted.
	reqIdx := strings.Index(got, "VerifyTokenRequest")
	respIdx := strings.Index(got, "VerifyTokenResponse")
	if reqIdx < 0 || respIdx < 0 || reqIdx > respIdx {
		t.Fatalf("TypeImports should be sorted Request before Response:\n%s", got)
	}
}

func TestEmitCoreDTS_importsFailureTypesFromUnion(t *testing.T) {
	m := sampleAuthModule()
	m.DomainErrors = []ErrorClass{{Name: "CellTaken", ForstPackage: "auth"}}
	m.Functions[0].FailureType = "CellTaken | ForstUnknownFailure | InvokeFailure"
	got := EmitCoreDTS(m)
	assertContainsAll(t, got, []string{
		`import type { CellTaken, ForstUnknownFailure } from "../pkg/auth.errors.js"`,
		`import type { InvokeFailure } from "../$errors.js"`,
		"export type VerifyTokenFailure = CellTaken | ForstUnknownFailure | InvokeFailure",
		"/** @throws {VerifyTokenFailure} */",
		"{ ok: false; error: VerifyTokenFailure }",
	})
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
	m := sampleAuthModule()
	m.Omitted = []OmittedFunction{{
		PackageName:  "auth",
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
	got := EmitPackageESM(sampleAuthModule(), RuntimePromise, "@forst/gen")
	assertContainsAll(t, got, []string{
		`export * from "../core/auth.js"`,
		"Promise mode re-exports core",
	})
	assertContainsNone(t, got, []string{
		"@forst/client",
		"effect",
		"invokeFunction",
	})
}

func TestEmitPackageDTS_golden(t *testing.T) {
	got := EmitPackageDTS(sampleAuthModule(), RuntimePromise, "@forst/gen")
	assertContainsAll(t, got, []string{
		`export * from "../core/auth.js"`,
		"export type { VerifyTokenRequest, VerifyTokenResponse }",
		`from "../types.js"`,
	})
}

func TestEmitPackageESM_effectMode_wrapsCore(t *testing.T) {
	got := EmitPackageESM(sampleAuthModule(), RuntimeEffect, "@forst/gen")
	assertContainsAll(t, got, []string{
		`import { Effect } from "effect"`,
		`import { ForstTransport, withTransport } from "../$effect.js"`,
		`import * as core from "../core/auth.js"`,
		`export class Auth extends Effect.Service()`,
		`"@forst/gen/Auth"`,
		"Effect.tryPromise",
		"accessors: true",
		"dependencies: [ForstTransport.Default]",
		"export const { VerifyToken } = Auth",
		`export { auth } from "../core/auth.js"`,
	})
	assertContainsNone(t, got, []string{
		".safe",
		"AbortController",
	})
}

func TestEmitIndexESM_golden(t *testing.T) {
	got := EmitIndexESM([]string{"auth", "billing"}, "6321", nil, RuntimePromise)
	assertContainsAll(t, got, []string{
		`import { createInvokeClient, configureDefaultInvokeClient } from "./transport.js"`,
		`import { auth } from "./pkg/auth.js"`,
		`import { billing } from "./pkg/billing.js"`,
		"export function createForstClient(config)",
		"export { configureDefaultInvokeClient }",
		"auth: auth(client)",
		"billing: billing(client)",
		"http://127.0.0.1:6321",
	})
	assertContainsNone(t, got, []string{
		"ForstUnknownFailure",
		`from "./$errors.js"`,
		"InvokeRejected",
		"isInvokeFailure",
		`from "@forst/errors"`,
	})
	assertContainsNone(t, got, []string{
		"export { VerifyToken",
		"@forst/client",
		".cjs",
	})
}

func TestEmitIndexESM_sortsPackages(t *testing.T) {
	got := EmitIndexESM([]string{"billing", "auth"}, "6321", nil, RuntimePromise)
	authImport := strings.Index(got, `import { auth } from "./pkg/auth.js"`)
	billingImport := strings.Index(got, `import { billing } from "./pkg/billing.js"`)
	if authImport < 0 || billingImport < 0 || authImport > billingImport {
		t.Fatalf("packages should be sorted alphabetically:\n%s", got)
	}
}

func TestEmitIndexDTS_doesNotReexportDomainErrors(t *testing.T) {
	domain := []ErrorClass{{Name: "CellTaken", Tag: "CellTaken"}}
	got := EmitIndexDTS([]string{"main"}, domain, RuntimePromise)
	assertContainsNone(t, got, []string{
		"CellTaken",
		"ForstUnknownFailure",
		"ForstError",
	})
	assertContainsAll(t, got, []string{"TaggedError"})
}

func TestEmitIndexDTS_golden(t *testing.T) {
	got := EmitIndexDTS([]string{"auth"}, nil, RuntimePromise)
	assertContainsAll(t, got, []string{
		"ForstInvokeClientConfig",
		"ForstInvokeMiddleware",
		"InvokeCallOptions",
		"InvokeContext",
		`from "./transport.js"`,
		`import { auth } from "./pkg/auth.js"`,
		"export interface ForstClientConfig",
		"middleware?: ForstInvokeMiddleware[]",
		"export declare function createForstClient(",
		"export declare function configureDefaultInvokeClient(",
		"readonly auth: ReturnType<typeof auth>",
		"export type ForstClient = ReturnType<typeof createForstClient>",
		"TaggedError",
		`from "./$errors.js"`,
		`export type * from "./types.js"`,
	})
	assertContainsNone(t, got, []string{
		"InvokeFailure",
		"isInvokeFailure",
		"InvokeRejected",
		`from "@forst/errors"`,
	})
}

func TestEmitTransportESM_isConnectOnlyHttpWithNdjson(t *testing.T) {
	got := EmitTransportESM("6321", RuntimePromise, nil)
	assertContainsAll(t, got, []string{
		"export function createInvokeClient",
		"export function getDefaultInvokeClient",
		"export function resetDefaultInvokeClientForTest",
		`from "@forst/errors"`,
		`from "@forst/errors"`,
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
		"FORST_INVOKE_AUTH_RECV_FD",
		"startHostInvokeAuthRecvListener",
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
	js := EmitTransportESM("6321", RuntimePromise, nil)
	ts := EmitTransportTypeScript("6321", RuntimePromise, nil)
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
		"export interface VerifyTokenRequest {\n  token: string;\n}",
		"export interface VerifyTokenResponse {\n  valid: boolean;\n}",
	}
	got := EmitTypesDTS(shapes)
	assertContainsAll(t, got, []string{
		"// Auto-generated types for Forst client",
		"export interface VerifyTokenRequest",
		"export interface VerifyTokenResponse",
		"token: string",
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
	got := emitInvokeCall("auth", "VerifyToken", "input")
	want := `client.invokeFunction("auth", "VerifyToken", [input], options)`
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
	a := EmitCoreESM(sampleAuthModule(), "6321")
	b := EmitCoreESM(sampleAuthModule(), "6321")
	if a != b {
		t.Fatal("EmitCoreESM must be byte-stable for the same ModuleEmit")
	}
}
