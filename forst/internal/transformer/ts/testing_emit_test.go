package transformerts

import (
	"strings"
	"testing"
)

func TestEmitTestingDTS_emitsOverrideTypesPerPackage(t *testing.T) {
	got := EmitTestingDTS([]ModuleEmit{sampleBcryptModule()}, "@forst/gen")
	assertContainsAll(t, got, []string{
		"export type BcryptHandlers",
		"ComparePassword: (",
		"Promise<ComparePasswordResponse>",
		"export interface ForstTestOverrides",
		"packages?:",
		"bcrypt?: Partial<BcryptHandlers>",
		"transport?: Partial<ForstInvokeClient>",
		"export declare function withForstTestScope",
		"export declare function createTestForstClient",
		"InvokeRejected",
		`from "./invoke-errors.js"`,
	})
}

func TestEmitTestingDTS_overridesKeyedUnderPackagesNotAtTopLevel(t *testing.T) {
	got := EmitTestingDTS([]ModuleEmit{sampleBcryptModule()}, "@forst/gen")
	packagesIdx := strings.Index(got, "packages?:")
	bcryptIdx := strings.Index(got, "bcrypt?: Partial<BcryptHandlers>")
	if packagesIdx < 0 || bcryptIdx < 0 || bcryptIdx < packagesIdx {
		t.Fatalf("bcrypt override must sit under packages?:\n%s", got)
	}
	// Top-level ForstTestOverrides must not list bcrypt beside packages/transport.
	iface := got[strings.Index(got, "export interface ForstTestOverrides"):]
	end := strings.Index(iface, "export declare function withForstTestScope")
	if end < 0 {
		t.Fatal("missing withForstTestScope")
	}
	body := iface[:end]
	if strings.Contains(body, "\n  bcrypt?:") {
		t.Fatalf("bcrypt must not be a top-level ForstTestOverrides key:\n%s", body)
	}
}

func TestEmitTestingESM_emitsScopeRuntime(t *testing.T) {
	got := EmitTestingESM([]ModuleEmit{sampleBcryptModule()}, "@forst/gen")
	assertContainsAll(t, got, []string{
		`from "./invoke-errors.js"`,
		"setActiveTestTransportResolver",
		"export async function withForstTestScope",
		"export function createTestForstClient",
		"getBuiltinModule",
		"AsyncLocalStorage",
		"createScopeTransport",
		"mergeOverrides",
		`import { bcrypt } from "./pkg/bcrypt.js"`,
		"bcrypt: bcrypt(transport)",
		"new InvokeRejected",
	})
	assertContainsNone(t, got, []string{
		"@forst/client",
		"export class InvokeRejected",
	})
}

func TestEmitTestingESM_handlersTypeNameCapitalizesPackage(t *testing.T) {
	if got := handlersTypeName("bcrypt"); got != "BcryptHandlers" {
		t.Fatalf("handlersTypeName(bcrypt)=%q", got)
	}
	if got := handlersTypeName("main"); got != "MainHandlers" {
		t.Fatalf("handlersTypeName(main)=%q", got)
	}
}
