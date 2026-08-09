package transformerts

import (
	"strings"
	"testing"
)

func TestEmitEffectSupportESM_exportsTransportService(t *testing.T) {
	got := EmitEffectSupportESM("@forst/gen")
	assertContainsAll(t, got, []string{
		`from "effect"`,
		"export class ForstTransport",
		`"@forst/gen/Transport"`,
		"export const withTransport",
		"export const layerTransport",
		"AbortSignal.any",
		`from "./transport.js"`,
	})
	assertContainsNone(t, got, []string{"AbortController"})
}

func TestEmitIndexEffectDTS_referencesTransportConfigType(t *testing.T) {
	got := EmitIndexEffectDTS([]string{"auth", "bcrypt"})
	assertContainsAll(t, got, []string{
		"ForstClientLive",
		"ForstClientLayer",
		"makeForstClientRuntime",
		"config?: ForstInvokeClientConfig",
	})
	if strings.Contains(got, `import type { ForstInvokeClientConfig`) {
		t.Fatalf("EmitIndexEffectDTS must not duplicate transport import when appended to index.d.ts:\n%s", got)
	}
}

func TestEmitIndexEffectESM_sharesTransport(t *testing.T) {
	got := EmitIndexEffectESM([]string{"auth", "bcrypt"}, "@forst/gen")
	assertContainsAll(t, got, []string{
		"ForstClientLive",
		"ForstClientLayer",
		"makeForstClientRuntime",
		"const transportLayer = layerTransport(config)",
		"Layer.provide(transportLayer)",
		"Bcrypt.DefaultWithoutDependencies",
		"Auth.DefaultWithoutDependencies",
	})
	if strings.Count(got, "layerTransport(config)") != 1 {
		t.Fatalf("expected one layerTransport call:\n%s", got)
	}
}

func TestEmitPackageEffectDTS_importsFailureTypesFromUnion(t *testing.T) {
	m := sampleBcryptModule()
	m.Functions[0].FailureType = "ForstUnknownFailure | InvokeRejected | InvokeTimedOut"
	got := EmitPackageEffectDTS(m, "@forst/gen")
	assertContainsAll(t, got, []string{
		`import type { ForstUnknownFailure } from "../domain-errors.js"`,
		`import type { InvokeRejected, InvokeTimedOut } from "../invoke-errors.js"`,
		"Effect.Effect<ComparePasswordResponse, ForstUnknownFailure | InvokeRejected | InvokeTimedOut>",
	})
}

func TestEmitTestingEffectDTS_partialOverrides(t *testing.T) {
	got := EmitTestingEffectDTS([]ModuleEmit{sampleBcryptModule()}, "@forst/gen")
	assertContainsAll(t, got, []string{
		"ForstTestOverrides",
		"packages?:",
		"bcrypt?: Partial<BcryptHandlers>",
		"transport?:",
		"| ComparePasswordResponse",
		"| Promise<ComparePasswordResponse>",
		"| Effect.Effect<ComparePasswordResponse, InvokeFailure>",
		"ForstTestLayer",
		`import { InvokeRejected } from "./invoke-errors.js"`,
		"type InvokeFailure",
		"export declare class ForstTestServerFailed",
	})
	assertContainsNone(t, got, []string{
		`from "./errors.js"`,
	})
}
