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

func TestEmitIndexEffectESM_sharesTransport(t *testing.T) {
	got := EmitIndexEffectESM([]string{"auth", "bcrypt"}, "@forst/gen")
	assertContainsAll(t, got, []string{
		"ForstClientLive",
		"layerForstClient",
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

func TestEmitTestingEffectDTS_partialOverrides(t *testing.T) {
	got := EmitTestingEffectDTS([]ModuleEmit{sampleBcryptModule()})
	assertContainsAll(t, got, []string{
		"ForstTestOverrides",
		"packages?:",
		"bcrypt?: Partial<BcryptHandlers>",
		"transport?:",
		"| ComparePasswordResponse",
		"| Promise<ComparePasswordResponse>",
		"| Effect.Effect<ComparePasswordResponse, InvokeFailure>",
		"layerForstTest",
	})
}
