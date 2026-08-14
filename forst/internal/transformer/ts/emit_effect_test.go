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
		`from "./transport/runtime.js"`,
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
		`import("./pkg/auth.js").$auth`,
		`import("./pkg/billing.js").$billing`,
	})
	if strings.Contains(got, `import { $auth } from "./pkg/auth.js"`) {
		t.Fatalf("EmitIndexEffectDTS must not duplicate pkg namespace imports when appended to index.d.ts:\n%s", got)
	}
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
		"$billing.DefaultWithoutDependencies",
		"$auth.DefaultWithoutDependencies",
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
		`import type { ForstUnknownFailure, InvokeFailure } from "@forst/errors/effect"`,
		"export type $VerifyTokenFailure = ForstUnknownFailure | InvokeFailure",
		"Effect.Effect<$VerifyTokenResponse, $VerifyTokenFailure>",
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
		"| $VerifyTokenResponse",
		"| Promise<$VerifyTokenResponse>",
		"| Effect.Effect<$VerifyTokenResponse, InvokeFailure>",
		"ForstTestLayer",
		`import { InvokeRejected } from "@forst/errors/effect"`,
		`import type { InvokeFailure } from "@forst/errors/effect"`,
		`import type { ForstTestServerFailed } from "@forst/errors/effect"`,
		`export { ForstTestServerFailed } from "@forst/errors/effect"`,
	})
}
