package nodert

import (
	"fmt"
	"os"

	"forst/internal/ftconfig"
)

const (
	// EnvInvokeOnly disables Node host spawn when set to 1 or true (invoke-only layout).
	EnvInvokeOnly = "FORST_INVOKE_ONLY"
	envInvokeAuth = "FORST_INVOKE_AUTH"
)

// InvokeOnlyEnabled reports whether the process should serve embedded invoke only.
func InvokeOnlyEnabled() bool {
	v := os.Getenv(EnvInvokeOnly)
	return v == "1" || v == "true"
}

// EnsureEmbeddedHostInvokeAuthRelay registers invoke auth relay for hostMode + embedded programs.
func EnsureEmbeddedHostInvokeAuthRelay(cfg *ftconfig.Config) error {
	if cfg == nil || !cfg.Node.HostMode || !cfg.Server.Embedded || InvokeOnlyEnabled() {
		return nil
	}
	if invokeAuthDisabledByEnv() || !SupportsInvokeAuthFDHandoff() {
		return nil
	}
	relay := ActiveHostInvokeAuthRelay()
	if relay == nil {
		var err error
		relay, err = NewHostInvokeAuthRelay()
		if err != nil {
			return err
		}
		SetActiveHostInvokeAuthRelay(relay)
	}
	return setupInProcessInvokeAuthHandoff(relay)
}

func invokeAuthDisabledByEnv() bool {
	v := os.Getenv(envInvokeAuth)
	return v == "off" || v == "0" || v == "false"
}

func setupInProcessInvokeAuthHandoff(relay *HostInvokeAuthRelay) error {
	if relay == nil {
		return fmt.Errorf("node runtime: invoke auth relay is nil")
	}
	goWrite, err := relay.PrepareGoChild()
	if err != nil {
		return err
	}
	fd, err := dupInvokeAuthHandoffFD(goWrite)
	if err != nil {
		return err
	}
	return os.Setenv(envInvokeAuthFD, fmt.Sprintf("%d", fd))
}
