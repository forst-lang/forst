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
					{Name: "input", Type: "$VerifyTokenRequest"},
				},
				ReturnType: "$VerifyTokenResponse",
			},
		},
		TypeImports: []string{"$VerifyTokenResponse", "$VerifyTokenRequest"},
	}
}

func sampleStreamModule() ModuleEmit {
	return ModuleEmit{
		PackageName: "events",
		Functions: []FunctionSignature{
			{
				Name:             "Watch",
				Parameters:       []Parameter{{Name: "query", Type: "$WatchRequest"}},
				ReturnType:       "AsyncIterable<$WatchRow>",
				StreamingRowType: "$WatchRow",
			},
		},
		TypeImports: []string{"$WatchRequest", "$WatchRow"},
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
		"export const $auth = (client) => ({",
		`client.invokeFunction("auth", "VerifyToken", [input], options)`,
		"http://127.0.0.1:6321",
	})
	assertContainsNone(t, got, []string{
		"export async function VerifyToken",
		"getDefaultInvokeClient",
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
		`client.invokeStream("events", "Watch", [query], options)`,
	})
	assertContainsNone(t, got, []string{
		"export function WatchStream",
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
		`client.invokeFunction("main", "Ping", [], options)`,
	})
	assertContainsNone(t, got, []string{
		"export async function Ping",
	})
}

func TestEmitCoreESM_jsReservedPackageNameUsesDollarNamespace(t *testing.T) {
	m := ModuleEmit{
		PackageName: "function",
		Functions: []FunctionSignature{
			{Name: "Ping", ReturnType: "number"},
		},
	}
	got := EmitCoreESM(m, "6321")
	assertContainsAll(t, got, []string{
		"export const $function = (client) => ({",
		`client.invokeFunction("function", "Ping", [], options)`,
	})
	assertContainsNone(t, got, []string{
		"export const function =",
		"export async function Ping",
	})
}

func TestEmitPackageESM_jsReservedFunctionNameUsesBoundNamespace(t *testing.T) {
	m := ModuleEmit{
		PackageName: "main",
		Functions: []FunctionSignature{
			{Name: "class", ReturnType: "string"},
		},
	}
	got := EmitPackageESM(m, RuntimePromise, "@forst/gen")
	assertContainsAll(t, got, []string{
		"export const $main = {",
		"class: Object.assign(",
	})
	assertContainsNone(t, got, []string{
		"export async function class",
		"export function class",
	})
}

func TestEmitCoreDTS_golden(t *testing.T) {
	got := EmitCoreDTS(sampleAuthModule())
	assertContainsAll(t, got, []string{
		`import type { ForstInvokeClient, InvokeCallOptions } from "../transport/runtime.js"`,
		`import type { $VerifyTokenRequest, $VerifyTokenResponse } from "../types.js"`,
		"export declare const $auth:",
		"VerifyToken: (input: $VerifyTokenRequest, options?: InvokeCallOptions) => Promise<$VerifyTokenResponse>",
		`import type { InvokeFailure } from "@forst/errors"`,
	})
	assertContainsNone(t, got, []string{
		"export declare function VerifyToken",
		"export declare namespace VerifyToken",
	})
	reqIdx := strings.Index(got, "$VerifyTokenRequest")
	respIdx := strings.Index(got, "$VerifyTokenResponse")
	if reqIdx < 0 || respIdx < 0 || reqIdx > respIdx {
		t.Fatalf("TypeImports should be sorted Request before Response:\n%s", got)
	}
}

func TestEmitCoreDTS_importsFailureTypesFromUnion(t *testing.T) {
	m := sampleAuthModule()
	m.DomainErrors = []ErrorClass{{Name: "CellTaken", ForstPackage: "auth"}}
	m.Functions[0].FailureType = "$CellTaken | ForstUnknownFailure | InvokeFailure"
	got := EmitCoreDTS(m)
	assertContainsAll(t, got, []string{
		`import type { $CellTaken } from "../pkg/auth.errors.js"`,
		`import type { ForstUnknownFailure, InvokeFailure } from "@forst/errors"`,
	})
	assertContainsNone(t, got, []string{
		"export type VerifyTokenFailure",
		"export declare function VerifyToken",
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
		`import { $auth as $authCore } from "../core/auth.js"`,
		"export const $auth = {",
		"VerifyToken: Object.assign(",
		"resolveClient(options)",
		"Promise mode exports bound $pkg namespace",
	})
	assertContainsNone(t, got, []string{
		`export * from "../core/auth.js"`,
		"export async function VerifyToken",
		"@forst/client",
		"effect",
	})
}

func TestEmitPackageDTS_golden(t *testing.T) {
	got := EmitPackageDTS(sampleAuthModule(), RuntimePromise, "@forst/gen")
	assertContainsAll(t, got, []string{
		"export declare const $auth:",
		"readonly VerifyToken:",
		"safe(",
		"export type { $VerifyTokenRequest, $VerifyTokenResponse }",
		`from "../types.js"`,
	})
	assertContainsNone(t, got, []string{
		`export * from "../core/auth.js"`,
		"export declare function VerifyToken",
	})
}

func TestEmitPackageESM_effectMode_wrapsCore(t *testing.T) {
	got := EmitPackageESM(sampleAuthModule(), RuntimeEffect, "@forst/gen")
	assertContainsAll(t, got, []string{
		`import { Effect } from "effect"`,
		`import { ForstTransport, withTransport } from "../$transport.js"`,
		`import * as core from "../core/auth.js"`,
		`export class $auth extends Effect.Service()`,
		`"@forst/gen/auth"`,
		"Effect.tryPromise",
		"accessors: true",
		"dependencies: [ForstTransport.Default]",
	})
	assertContainsNone(t, got, []string{
		".safe",
		"AbortController",
		"export const { VerifyToken }",
		`export { $auth } from "../core/auth.js"`,
		"export async function VerifyToken",
	})
}

func TestEmitIndexESM_golden(t *testing.T) {
	got := EmitIndexESM([]string{"auth", "billing"}, "6321", nil, RuntimePromise)
	assertContainsAll(t, got, []string{
		`import { createInvokeClient, configureDefaultInvokeClient } from "./transport/runtime.js"`,
		`import { $auth as $authCore } from "./core/auth.js"`,
		`import { $billing as $billingCore } from "./core/billing.js"`,
		"export function createForstClient(config)",
		"export { configureDefaultInvokeClient }",
		"auth: $authCore(client)",
		"billing: $billingCore(client)",
		"http://127.0.0.1:6321",
	})
	assertContainsNone(t, got, []string{
		"ForstUnknownFailure",
		`from "./$errors.js"`,
		"InvokeRejected",
		"isInvokeFailure",
		`from "@forst/errors"`,
		"export { VerifyToken",
		"@forst/client",
		".cjs",
	})
}

func TestEmitIndexESM_sortsPackages(t *testing.T) {
	got := EmitIndexESM([]string{"billing", "auth"}, "6321", nil, RuntimePromise)
	authImport := strings.Index(got, `import { $auth as $authCore } from "./core/auth.js"`)
	billingImport := strings.Index(got, `import { $billing as $billingCore } from "./core/billing.js"`)
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
		`from "./transport/runtime.js"`,
		`import { $auth as $authCore } from "./core/auth.js"`,
		"export interface ForstClientConfig",
		"middleware?: ForstInvokeMiddleware[]",
		"export declare function createForstClient(",
		"export declare function configureDefaultInvokeClient(",
		"readonly auth: ReturnType<typeof $authCore>",
		"export type ForstClient = ReturnType<typeof createForstClient>",
		"TaggedError",
		`from "@forst/errors"`,
		`export type * from "./types.js"`,
	})
	assertContainsNone(t, got, []string{
		"InvokeFailure",
		"isInvokeFailure",
		"InvokeRejected",
		`from "./$errors.js"`,
	})
}

func TestEmitTransportESM_isConnectOnlyHttpWithNdjson(t *testing.T) {
	got := EmitTransportESM("6321", RuntimePromise, nil)
	assertContainsAll(t, got, []string{
		"export function createInvokeClient",
		"export function getDefaultInvokeClient",
		"export function resetDefaultInvokeClientForTest",
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
		"export interface $VerifyTokenRequest {\n  token: string;\n}",
		"export interface $VerifyTokenResponse {\n  valid: boolean;\n}",
	}
	got := EmitTypesDTS(shapes)
	assertContainsAll(t, got, []string{
		"// Auto-generated types for Forst client",
		"export interface $VerifyTokenRequest",
		"export interface $VerifyTokenResponse",
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
	a := EmitCoreESM(sampleAuthModule(), "6321")
	b := EmitCoreESM(sampleAuthModule(), "6321")
	if a != b {
		t.Fatal("EmitCoreESM must be byte-stable for the same ModuleEmit")
	}
}
