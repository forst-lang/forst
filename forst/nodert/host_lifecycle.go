package nodert

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"forst/internal/ftconfig"

	logrus "github.com/sirupsen/logrus"
)

// HostProcessConfig configures parent-owned node host spawn (no RPC dial).
type HostProcessConfig struct {
	BoundaryRoot, WorkDir  string
	NodePath, Loader       string
	ShimArgs               []string
	SocketPath, ReadyPath  string
	HostAutoRegister       bool
	HostAppReadyModule     string
	FilesExclude, ExtraEnv []string
	ReadyTimeout           time.Duration
	Log                    *logrus.Logger
}

// SpawnedHostProcess is a node host started by EnsureHostProcessRunning.
type SpawnedHostProcess struct {
	proc *managedProcess
}

// Terminate stops a host process spawned by EnsureHostProcessRunning.
func (s *SpawnedHostProcess) Terminate() error {
	if s == nil || s.proc == nil {
		return nil
	}
	return s.proc.terminate()
}

// PID returns the spawned host pid, or 0 when unknown.
func (s *SpawnedHostProcess) PID() int {
	if s == nil || s.proc == nil || s.proc.cmd == nil || s.proc.cmd.Process == nil {
		return 0
	}
	return s.proc.cmd.Process.Pid
}

// DefaultHostShutdownGrace is the wait before SIGKILL when terminating a host pid.
func DefaultHostShutdownGrace() time.Duration {
	return shutdownGracePeriod
}

// HostProcessConfigFromFTConfig builds spawn config from ftconfig.
func HostProcessConfigFromFTConfig(cfg *ftconfig.Config, boundaryRoot string, log *logrus.Logger) (HostProcessConfig, error) {
	if cfg == nil {
		return HostProcessConfig{}, fmt.Errorf("node runtime: ftconfig is nil")
	}
	if !cfg.Node.HostMode {
		return HostProcessConfig{}, fmt.Errorf("node runtime: hostMode is not enabled")
	}
	if len(cfg.Node.Args) == 0 {
		return HostProcessConfig{}, fmt.Errorf("node runtime: hostMode requires non-empty node.args in ftconfig.json")
	}

	boundaryRoot = strings.TrimSpace(boundaryRoot)
	if boundaryRoot == "" {
		return HostProcessConfig{}, fmt.Errorf("node runtime: boundary root is empty")
	}

	nodeBinary, err := ResolveNodeBinary(boundaryRoot, cfg.Node.Binary)
	if err != nil {
		return HostProcessConfig{}, err
	}

	socketPath, readyPath, err := ResolveHostSocketPath(boundaryRoot, cfg.Node.HostSocket)
	if err != nil {
		return HostProcessConfig{}, err
	}

	hostAppReadyModule := ""
	if module := strings.TrimSpace(cfg.Node.HostAppReadyModule); module != "" {
		hostAppReadyModule, err = resolveHostAppReadyModule(boundaryRoot, module)
		if err != nil {
			return HostProcessConfig{}, err
		}
	}

	timeout := time.Duration(cfg.Node.HostReadyTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	workDir := boundaryRoot
	return HostProcessConfig{
		BoundaryRoot:       boundaryRoot,
		WorkDir:            workDir,
		NodePath:           nodeBinary,
		Loader:             cfg.Node.Loader,
		ShimArgs:           append([]string(nil), cfg.Node.Args...),
		SocketPath:         socketPath,
		ReadyPath:          readyPath,
		HostAutoRegister:   cfg.Node.EffectiveHostAutoRegister(),
		HostAppReadyModule: hostAppReadyModule,
		FilesExclude:       append([]string(nil), cfg.Files.Exclude...),
		ReadyTimeout:       timeout,
		Log:                log,
	}, nil
}

// EnsureHostProcessRunning starts the node host when no live marker exists.
// When the marker already references a live ready host, returns spawned=false
// without dialing (the host accepts a single RPC client).
func EnsureHostProcessRunning(cfg HostProcessConfig) (spawned bool, proc *SpawnedHostProcess, err error) {
	log := cfg.Log
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.ErrorLevel)
	}

	socketPath := cfg.SocketPath
	readyPath := cfg.ReadyPath
	if socketPath == "" || readyPath == "" {
		var resolveErr error
		socketPath, readyPath, resolveErr = ResolveHostSocketPath(cfg.BoundaryRoot, "")
		if resolveErr != nil {
			return false, nil, resolveErr
		}
	}

	if ReattachSkipReason(readyPath) == "" {
		return false, nil, nil
	}

	if err := PrepareHostSocket(socketPath, readyPath); err != nil {
		return false, nil, err
	}

	hostCmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot:       cfg.BoundaryRoot,
		Executable:         cfg.NodePath,
		ShimArgs:           cfg.ShimArgs,
		WorkDir:            cfg.WorkDir,
		Loader:             cfg.Loader,
		SocketPath:         socketPath,
		ReadyPath:          readyPath,
		FilesExclude:       cfg.FilesExclude,
		Env:                cfg.ExtraEnv,
		HostAutoRegister:   cfg.HostAutoRegister,
		HostAppReadyModule: cfg.HostAppReadyModule,
	})
	if err != nil {
		return false, nil, err
	}

	timeout := cfg.ReadyTimeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	managed, err := spawnHostProcess(hostCmd, cfg.WorkDir, log)
	if err != nil {
		return false, nil, err
	}

	if err := waitForHostMarkerReady(ctx, readyPath); err != nil {
		_ = managed.terminate()
		cleanupHostSocketFiles(socketPath, readyPath)
		return false, nil, err
	}

	go func() {
		_ = managed.waitAsync(log)
	}()

	return true, &SpawnedHostProcess{proc: managed}, nil
}

func waitForHostMarkerReady(ctx context.Context, readyPath string) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			msg := fmt.Sprintf("node runtime: host ready timeout (ready=%s)", readyPath)
			if data, err := os.ReadFile(readyPath); err == nil {
				msg += fmt.Sprintf("; ready=%s", string(data))
			}
			return fmt.Errorf("%s", msg)
		case <-ticker.C:
			if ReattachSkipReason(readyPath) == "" {
				return nil
			}
		}
	}
}
