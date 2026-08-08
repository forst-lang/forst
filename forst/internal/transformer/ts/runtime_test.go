package transformerts

import (
	"testing"

	"forst/internal/ftconfig"
)

func TestRuntimeFromConfig_defaultsToPromise(t *testing.T) {
	got := RuntimeFromConfig(ftconfig.GenerateConfig{})
	if got != RuntimePromise {
		t.Fatalf("got %v, want RuntimePromise", got)
	}
}

func TestRuntimeFromConfig_effectTrue(t *testing.T) {
	got := RuntimeFromConfig(ftconfig.GenerateConfig{Effect: true})
	if got != RuntimeEffect {
		t.Fatalf("got %v, want RuntimeEffect", got)
	}
}

func TestClientRuntime_String(t *testing.T) {
	if RuntimePromise.String() != "promise" {
		t.Fatalf("promise: %q", RuntimePromise.String())
	}
	if RuntimeEffect.String() != "effect" {
		t.Fatalf("effect: %q", RuntimeEffect.String())
	}
}
