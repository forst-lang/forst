package devserver

import (
	"os"

	"forst/internal/ftconfig"
	"forst/bridgert"

	"github.com/sirupsen/logrus"
)

// HostOrchestrator owns node host spawn/shutdown for forst dev watch reload.
type HostOrchestrator struct {
	boundaryRoot string
	cfg          *ftconfig.Config
	log          *logrus.Logger
	spawnedProc  *bridgert.SpawnedHostProcess
	authRelay    *bridgert.HostInvokeAuthRelay
}

// NewHostOrchestrator creates a parent-owned node host orchestrator.
func NewHostOrchestrator(log *logrus.Logger, boundaryRoot string, cfg *ftconfig.Config) *HostOrchestrator {
	return &HostOrchestrator{
		boundaryRoot: boundaryRoot,
		cfg:          cfg,
		log:          log,
	}
}

// EnsureRunning starts the node host when needed and enables attach-only on go run children.
func (o *HostOrchestrator) EnsureRunning() error {
	if o == nil || o.cfg == nil || !o.cfg.Bridge.HostMode {
		return nil
	}
	if err := attachHostInvokeAuthRelay(o, o.boundaryRoot); err != nil {
		return err
	}
	hostCfg, err := bridgert.HostProcessConfigFromFTConfig(o.cfg, o.boundaryRoot, o.log)
	if err != nil {
		return err
	}
	hostCfg.AuthRelay = o.authRelay
	spawned, proc, err := bridgert.EnsureHostProcessRunning(hostCfg)
	if err != nil {
		return err
	}
	if spawned {
		o.spawnedProc = proc
		if o.log != nil && proc != nil && proc.PID() > 0 {
			o.log.Infof("Spawned node host (pid=%d)", proc.PID())
		}
	} else if o.log != nil {
		if pid := bridgert.ReadHostMarkerPID(o.boundaryRoot); pid > 0 {
			o.log.Infof("Node host already running (pid=%d)", pid)
		}
	}
	o.activateAttachOnly()
	return nil
}

// Shutdown always stops the node host and clears attach-only env.
func (o *HostOrchestrator) Shutdown() error {
	if o == nil {
		return nil
	}
	defer o.deactivateAttachOnly()
	defer func() {
		if o.authRelay != nil {
			_ = o.authRelay.Close()
			o.authRelay = nil
			bridgert.SetActiveHostInvokeAuthRelay(nil)
		}
	}()
	if o.spawnedProc != nil {
		return o.spawnedProc.Terminate()
	}
	if pid := bridgert.ReadHostMarkerPID(o.boundaryRoot); pid > 0 {
		return bridgert.TerminateHostPID(pid, bridgert.DefaultHostShutdownGrace())
	}
	return nil
}

func (o *HostOrchestrator) activateAttachOnly() {
	_ = os.Setenv(bridgert.EnvNodeAttachOnly, "1")
}

func (o *HostOrchestrator) deactivateAttachOnly() {
	_ = os.Unsetenv(bridgert.EnvNodeAttachOnly)
}
