package devserver

import (
	"os"

	"forst/internal/ftconfig"
	"forst/nodert"

	"github.com/sirupsen/logrus"
)

// HostOrchestrator owns node host spawn/shutdown for forst dev watch reload.
type HostOrchestrator struct {
	boundaryRoot string
	cfg          *ftconfig.Config
	log          *logrus.Logger
	spawnedProc  *nodert.SpawnedHostProcess
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
	if o == nil || o.cfg == nil || !o.cfg.Node.HostMode {
		return nil
	}
	hostCfg, err := nodert.HostProcessConfigFromFTConfig(o.cfg, o.boundaryRoot, o.log)
	if err != nil {
		return err
	}
	spawned, proc, err := nodert.EnsureHostProcessRunning(hostCfg)
	if err != nil {
		return err
	}
	if spawned {
		o.spawnedProc = proc
		if o.log != nil && proc != nil && proc.PID() > 0 {
			o.log.Infof("Spawned node host (pid=%d)", proc.PID())
		}
	} else if o.log != nil {
		if pid := nodert.ReadHostMarkerPID(o.boundaryRoot); pid > 0 {
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
	if o.spawnedProc != nil {
		return o.spawnedProc.Terminate()
	}
	if pid := nodert.ReadHostMarkerPID(o.boundaryRoot); pid > 0 {
		return nodert.TerminateHostPID(pid, nodert.DefaultHostShutdownGrace())
	}
	return nil
}

func (o *HostOrchestrator) activateAttachOnly() {
	_ = os.Setenv(nodert.EnvNodeAttachOnly, "1")
}

func (o *HostOrchestrator) deactivateAttachOnly() {
	_ = os.Unsetenv(nodert.EnvNodeAttachOnly)
}
