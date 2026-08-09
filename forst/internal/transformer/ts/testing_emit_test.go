package transformerts

import (
	"strings"
	"testing"
)

func TestEmitTestingDTS_emitsOverrideTypesPerPackage(t *testing.T) {
	got := EmitTestingDTS([]ModuleEmit{sampleAuthModule()}, "@forst/gen", RuntimePromise)
	assertContainsAll(t, got, []string{
		"export type AuthHandlers",
		"VerifyToken: (",
		"Promise<VerifyTokenResponse>",
		"export interface ForstTestOverrides",
		"packages?:",
		"auth?: Partial<AuthHandlers>",
		"transport?: Partial<ForstInvokeClient>",
		"export declare function withForstTestScope",
		"export declare function createTestForstClient",
		"InvokeRejected",
		`from "./errors.js"`,
	})
}

func TestEmitTestingDTS_overridesKeyedUnderPackagesNotAtTopLevel(t *testing.T) {
	got := EmitTestingDTS([]ModuleEmit{sampleAuthModule()}, "@forst/gen", RuntimePromise)
	packagesIdx := strings.Index(got, "packages?:")
	authIdx := strings.Index(got, "auth?: Partial<AuthHandlers>")
	if packagesIdx < 0 || authIdx < 0 || authIdx < packagesIdx {
		t.Fatalf("auth override must sit under packages?:\n%s", got)
	}
	// Top-level ForstTestOverrides must not list auth beside packages/transport.
	iface := got[strings.Index(got, "export interface ForstTestOverrides"):]
	end := strings.Index(iface, "export declare function withForstTestScope")
	if end < 0 {
		t.Fatal("missing withForstTestScope")
	}
	body := iface[:end]
	if strings.Contains(body, "\n  auth?:") {
		t.Fatalf("auth must not be a top-level ForstTestOverrides key:\n%s", body)
	}
}

func TestEmitTestingESM_emitsScopeRuntime(t *testing.T) {
	got := EmitTestingESM([]ModuleEmit{sampleAuthModule()}, "@forst/gen", RuntimePromise)
	assertContainsAll(t, got, []string{
		`from "./errors.js"`,
		"setActiveTestTransportResolver",
		"export async function withForstTestScope",
		"export function createTestForstClient",
		"getBuiltinModule",
		"AsyncLocalStorage",
		"createScopeTransport",
		"mergeOverrides",
		`import { auth } from "./pkg/auth.js"`,
		"auth: auth(transport)",
		"new InvokeRejected",
	})
	assertContainsNone(t, got, []string{
		"@forst/client",
		"export class InvokeRejected",
	})
}

func TestEmitTestingESM_handlersTypeNameCapitalizesPackage(t *testing.T) {
	if got := handlersTypeName("billing"); got != "BillingHandlers" {
		t.Fatalf("handlersTypeName(billing)=%q", got)
	}
	if got := handlersTypeName("main"); got != "MainHandlers" {
		t.Fatalf("handlersTypeName(main)=%q", got)
	}
}
