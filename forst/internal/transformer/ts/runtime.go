package transformerts

import "forst/internal/ftconfig"

// ClientRuntime selects the generated client call shape.
// Only EmitPackageESM and EmitPackageDTS take this value; other Emit* functions
// stay shared so Effect mode layers over the Promise core.
type ClientRuntime int

const (
	// RuntimePromise is the default: functions return Promise<T>.
	RuntimePromise ClientRuntime = iota
	// RuntimeEffect wraps core Promise calls in Effect.tryPromise.
	RuntimeEffect
)

// RuntimeFromConfig maps generate.effect to a ClientRuntime.
func RuntimeFromConfig(g ftconfig.GenerateConfig) ClientRuntime {
	if g.Effect {
		return RuntimeEffect
	}
	return RuntimePromise
}

// String returns a stable name for logs and tests.
func (r ClientRuntime) String() string {
	switch r {
	case RuntimeEffect:
		return "effect"
	default:
		return "promise"
	}
}
