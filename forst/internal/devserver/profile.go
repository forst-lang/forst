package devserver

import "forst/internal/ftconfig"

// Profile is the resolved forst dev execution profile.
type Profile string

const (
	ProfileAuto     Profile = "auto"
	ProfileExecutor Profile = "executor"
	ProfileRuntime  Profile = "runtime"
)

// ResolveProfile picks executor vs runtime from ftconfig dev.profile (default auto).
func ResolveProfile(cfg *ftconfig.Config) Profile {
	if cfg == nil {
		return ProfileExecutor
	}
	switch cfg.Dev.Profile {
	case string(ProfileExecutor):
		return ProfileExecutor
	case string(ProfileRuntime):
		return ProfileRuntime
	default:
		return autoDetectProfile(cfg)
	}
}

func autoDetectProfile(cfg *ftconfig.Config) Profile {
	if cfg.Server.Embedded || cfg.Node.HostMode {
		return ProfileRuntime
	}
	return ProfileExecutor
}

// EffectiveListenPort returns the HTTP listen port for the resolved profile.
func EffectiveListenPort(cfg *ftconfig.Config, cliOverride string) string {
	if cliOverride != "" {
		return cliOverride
	}
	if cfg == nil {
		return ftconfig.DefaultDevExecutorPort
	}
	if ResolveProfile(cfg) == ProfileRuntime {
		return cfg.Server.EffectiveInvokePort()
	}
	if cfg.Server.Port != "" {
		return cfg.Server.Port
	}
	return ftconfig.DefaultDevExecutorPort
}
