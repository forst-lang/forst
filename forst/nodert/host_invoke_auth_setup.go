package nodert

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"forst/internal/ftconfig"
)

const (
	// EnvSkipNodeHost disables Node host spawn when set to 1 or true (split layout).
	EnvSkipNodeHost = "FORST_SKIP_NODE_HOST"
	envInvokeAuth   = "FORST_INVOKE_AUTH"
)

// SkipNodeHostEnabled reports whether Node host spawn/attach should be skipped.
func SkipNodeHostEnabled() bool {
	v := os.Getenv(EnvSkipNodeHost)
	return v == "1" || v == "true"
}

// EnsureEmbeddedHostInvokeAuthRelay registers invoke auth relay for hostMode + embedded programs.
func EnsureEmbeddedHostInvokeAuthRelay(cfg *ftconfig.Config) error {
	if cfg == nil || !cfg.Node.HostMode || !cfg.Server.Embedded || SkipNodeHostEnabled() {
		return nil
	}
	if invokeAuthDisabledByEnv() || !SupportsInvokeAuthFDHandoff() {
		return nil
	}
	if spawnAuthHandoffInherited() {
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
	return relay.ensureInProcessHandoff()
}

func (r *HostInvokeAuthRelay) ensureInProcessHandoff() error {
	if r == nil {
		return fmt.Errorf("node runtime: invoke auth relay is nil")
	}
	r.mu.Lock()
	if r.inProcessHandoffConfigured {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()
	if err := setupInProcessInvokeAuthHandoff(r); err != nil {
		return err
	}
	r.mu.Lock()
	r.inProcessHandoffConfigured = true
	r.mu.Unlock()
	return nil
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

func invokeAuthDisabledByEnv() bool {
	v := os.Getenv(envInvokeAuth)
	return v == "off" || v == "0" || v == "false"
}

// spawnAuthHandoffInherited reports whether this process received an invoke auth
// write fd from forst dev / forst run (ExtraFiles + FORST_INVOKE_AUTH_FD).
func spawnAuthHandoffInherited() bool {
	if ActiveHostInvokeAuthRelay() != nil {
		return false
	}
	raw := strings.TrimSpace(os.Getenv(envInvokeAuthFD))
	if raw == "" {
		return false
	}
	fd, err := strconv.Atoi(raw)
	return err == nil && fd >= 3
}
