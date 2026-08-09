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
		"export const ForstTransportLayer",
		"AbortSignal.any",
		`from "./transport.js"`,
	})
	assertContainsNone(t, got, []string{"AbortController"})
}

func TestEmitIndexEffectDTS_referencesTransportConfigType(t *testing.T) {
	got := EmitIndexEffectDTS([]string{"auth", "billing"})
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
	got := EmitIndexEffectESM([]string{"auth", "billing"}, "@forst/gen")
	assertContainsAll(t, got, []string{
		"ForstClientLive",
		"ForstClientLayer",
		"makeForstClientRuntime",
		"const transportLayer = ForstTransportLayer(config)",
		"Layer.provide(transportLayer)",
		"Billing.DefaultWithoutDependencies",
		"Auth.DefaultWithoutDependencies",
	})
	if strings.Count(got, "ForstTransportLayer(config)") != 1 {
		t.Fatalf("expected one ForstTransportLayer call:\n%s", got)
	}
}

func TestEmitPackageEffectDTS_importsFailureTypesFromUnion(t *testing.T) {
	m := sampleAuthModule()
	m.Functions[0].FailureType = "ForstUnknownFailure | InvokeFailure"
	got := EmitPackageEffectDTS(m, "@forst/gen")
	assertContainsAll(t, got, []string{
		`import type { ForstUnknownFailure } from "../errors.js"`,
		`import type { InvokeFailure } from "../errors.js"`,
		"export type VerifyTokenFailure = ForstUnknownFailure | InvokeFailure",
		"Effect.Effect<VerifyTokenResponse, VerifyTokenFailure>",
	})
	assertContainsNone(t, got, []string{
		"InvokeRejected",
		"InvokeHttpFailure",
	})
}

func TestEmitTestingEffectDTS_partialOverrides(t *testing.T) {
	got := EmitTestingEffectDTS([]ModuleEmit{sampleAuthModule()}, "@forst/gen")
	assertContainsAll(t, got, []string{
		"ForstTestOverrides",
		"packages?:",
		"auth?: Partial<AuthHandlers>",
		"transport?:",
		"| VerifyTokenResponse",
		"| Promise<VerifyTokenResponse>",
		"| Effect.Effect<VerifyTokenResponse, InvokeFailure>",
		"ForstTestLayer",
		`import { InvokeRejected } from "@forst/errors/effect"`,
		`import type { InvokeFailure } from "./errors.js"`,
		`import type { ForstTestServerFailed } from "@forst/errors/effect"`,
		`export { ForstTestServerFailed } from "@forst/errors/effect"`,
	})
}
