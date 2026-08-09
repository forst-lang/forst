package transformerts

import (
	"strings"
	"testing"
)

func TestEmitTestingDTS_includesStartForstTestServer(t *testing.T) {
	got := EmitTestingDTS([]ModuleEmit{sampleBcryptModule()}, "@forst/gen", RuntimePromise)
	assertContainsAll(t, got, []string{
		"export interface ForstTestServerOptions",
		"export interface ForstTestServer",
		"export declare function startForstTestServer",
		"ForstTestServerFailed",
		"[Symbol.asyncDispose](): Promise<void>",
	})
}

func TestEmitTestingESM_includesStartForstTestServer(t *testing.T) {
	got := EmitTestingESM([]ModuleEmit{sampleBcryptModule()}, "@forst/gen", RuntimePromise)
	assertContainsAll(t, got, []string{
		"export async function startForstTestServer",
		`const CLI_INVOKE = "` + CliInvokeModuleSpecifier + `"`,
		"import(CLI_INVOKE)",
		"configureDefaultInvokeClient({ baseUrl: handle.baseUrl })",
		"resetDefaultInvokeClientForTest",
		"Symbol.asyncDispose",
		"ForstTestServerFailed",
		CliInstallCommand,
	})
}

func TestEmitTestingEffectDTS_includesForstTestServerLayer(t *testing.T) {
	got := EmitTestingEffectDTS([]ModuleEmit{sampleBcryptModule()}, "@forst/gen")
	assertContainsAll(t, got, []string{
		"export declare class ForstTestServer",
		`"@forst/gen/TestServer"`,
		"export declare const ForstTestServerLayer",
		"export declare const makeForstTestServer",
		"ForstTestServerFailed",
		"ManagedRuntime.ManagedRuntime",
	})
	if strings.Contains(got, "startForstTestServer") {
		t.Fatal("Effect mode must not emit Promise startForstTestServer")
	}
}

func TestEmitTestingEffectESM_includesForstTestServerLayer(t *testing.T) {
	got := EmitTestingEffectESM([]ModuleEmit{sampleBcryptModule()}, "@forst/gen")
	assertContainsAll(t, got, []string{
		"export class ForstTestServer",
		"export function ForstTestServerLayer",
		"export const makeForstTestServer",
		"layerTransport({ baseUrl })",
		"DefaultWithoutDependencies",
		"ManagedRuntime.make",
		CliInvokeModuleSpecifier,
	})
}

func TestEmitHarnessError_harnessOutsideInvokeFailure(t *testing.T) {
	esm := EmitHarnessErrorESM(testNpmPackage, RuntimePromise)
	dts := EmitHarnessErrorDTS(testNpmPackage, RuntimePromise)
	invokeESM := EmitInvokeErrorsESM(testNpmPackage, RuntimePromise)
	invokeDTS := EmitInvokeErrorsDTS(testNpmPackage, RuntimePromise)
	assertContainsAll(t, esm, []string{"export class ForstTestServerFailed"})
	assertContainsAll(t, dts, []string{"export declare class ForstTestServerFailed"})
	unionStart := strings.Index(invokeDTS, "export type InvokeFailure =")
	if unionStart < 0 {
		t.Fatal("missing InvokeFailure")
	}
	union := invokeDTS[unionStart:]
	if end := strings.Index(union, "export declare function isInvokeFailure"); end > 0 {
		union = union[:end]
	}
	if strings.Contains(union, "ForstTestServerFailed") {
		t.Fatalf("InvokeFailure must not include ForstTestServerFailed:\n%s", union)
	}
	if strings.Contains(invokeESM, `"ForstTestServerFailed"`) && strings.Contains(invokeESM, "INVOKE_FAILURE_TAGS") {
		tagBlockStart := strings.Index(invokeESM, "const INVOKE_FAILURE_TAGS")
		tagBlock := invokeESM[tagBlockStart:]
		if end := strings.Index(tagBlock, "export const isInvokeFailure"); end > 0 {
			tagBlock = tagBlock[:end]
		}
		if strings.Contains(tagBlock, "ForstTestServerFailed") {
			t.Fatalf("INVOKE_FAILURE_TAGS must not include harness tag:\n%s", tagBlock)
		}
	}
}
