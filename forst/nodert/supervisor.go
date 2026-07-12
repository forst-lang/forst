package nodert

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	logrus "github.com/sirupsen/logrus"
)

// SupervisorConfig configures the singleton node runtime supervisor.
type SupervisorConfig struct {
	HostMode           bool
	HostSocketPath     string
	HostReadyPath      string
	HostReadyTimeout   time.Duration
	HostAutoRegister   bool
	HostAppReadyModule string
	ShimArgs           []string
	AttachOnly         bool
	ProcessOptions     ProcessOptions
	Manifest           Manifest
	RPC                RPCConfig
}

// RPCConfig configures Node RPC limits for the supervisor client.
type RPCConfig struct {
	MaxMessageBytes int
	CallTimeout     time.Duration
}

var (
	supervisorOnce       sync.Once
	supervisorInst       *Supervisor
	supervisorErr        error
	supervisorErrPrinted sync.Once
	supervisorCfg        SupervisorConfig
)

// ConfigureSupervisor sets options used by the first GetClient call.
func ConfigureSupervisor(cfg SupervisorConfig) {
	supervisorCfg = cfg
}

// Supervisor owns the Node child process and RPC client.
type Supervisor struct {
	mu       sync.Mutex
	client   *Client
	proc     *managedProcess
	hostConn io.Closer
	log      *logrus.Logger
}

// GetClient returns the singleton RPC client, spawning Node on first use.
func GetClient() (*Client, error) {
	supervisorOnce.Do(func() {
		if supervisorCfg.HostMode {
			supervisorInst, supervisorErr = newHostSupervisor(supervisorCfg)
		} else {
			supervisorInst, supervisorErr = newBootstrapSupervisor(supervisorCfg)
		}
	})
	if supervisorErr != nil {
		supervisorErrPrinted.Do(func() {
			fmt.Fprintf(os.Stderr, "forst node runtime: %v\n", supervisorErr)
		})
		return nil, supervisorErr
	}
	return supervisorInst.client, nil
}

// Shutdown terminates the supervised Node process.
func Shutdown() error {
	if supervisorInst == nil {
		return nil
	}
	return supervisorInst.shutdown()
}

func newBootstrapSupervisor(cfg SupervisorConfig) (*Supervisor, error) {
	log := cfg.ProcessOptions.Log
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.ErrorLevel)
	}

	proc, err := spawnBootstrapProcess(cfg.ProcessOptions)
	if err != nil {
		return nil, err
	}

	client := NewClient(proc.stdout, proc.stdin, log)
	return finishSupervisorInit(cfg, client, proc, nil, log)
}

func newHostSupervisor(cfg SupervisorConfig) (*Supervisor, error) {
	log := cfg.ProcessOptions.Log
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.ErrorLevel)
	}

	socketPath, readyPath, timeout, err := resolveHostSupervisorPaths(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if skip := ReattachSkipReason(readyPath); skip != "" {
		log.Infof("node host reattach skipped (%s)", skip)
	} else if marker, ok := readHostReadyMarker(readyPath); ok {
		log.Infof("node host reattach attempt (pid=%d, socket=%s)", marker.PID, socketPath)
	}

	if conn, attached, err := connectExistingHost(ctx, socketPath, readyPath); attached {
		if err != nil {
			if marker, has := readHostReadyMarker(readyPath); has && processAlive(marker.PID) {
				log.Warnf("node host pid=%d alive but reattach failed: %v", marker.PID, err)
			}
			return nil, err
		}
		marker, _ := readHostReadyMarker(readyPath)
		log.Infof("Reattached to node host (pid=%d, socket=%s)", marker.PID, socketPath)
		return dialAndInitHost(cfg, conn, nil, log)
	}

	if cfg.AttachOnly {
		return nil, fmt.Errorf("node host not running (attach-only); forst dev should have started it")
	}

	hostCfg := hostProcessConfigFromSupervisor(cfg, socketPath, readyPath, log)
	spawned, spawnedProc, err := EnsureHostProcessRunning(hostCfg)
	if err != nil {
		return nil, err
	}

	conn, err := waitForHostReady(ctx, socketPath, readyPath)
	if err != nil {
		if spawned && spawnedProc != nil {
			_ = spawnedProc.Terminate()
		}
		cleanupHostSocketFiles(socketPath, readyPath)
		return nil, err
	}

	var proc *managedProcess
	if spawned && spawnedProc != nil {
		proc = spawnedProc.proc
	}

	return dialAndInitHost(cfg, conn, proc, log)
}

func resolveHostSupervisorPaths(cfg SupervisorConfig) (socketPath, readyPath string, timeout time.Duration, err error) {
	socketPath = cfg.HostSocketPath
	readyPath = cfg.HostReadyPath
	if socketPath == "" || readyPath == "" {
		socketPath, readyPath, err = ResolveHostSocketPath(cfg.ProcessOptions.BoundaryRoot, "")
		if err != nil {
			return "", "", 0, err
		}
	}
	timeout = cfg.HostReadyTimeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return socketPath, readyPath, timeout, nil
}

func hostProcessConfigFromSupervisor(cfg SupervisorConfig, socketPath, readyPath string, log *logrus.Logger) HostProcessConfig {
	return HostProcessConfig{
		BoundaryRoot:       cfg.ProcessOptions.BoundaryRoot,
		WorkDir:            cfg.ProcessOptions.WorkDir,
		NodePath:           cfg.ProcessOptions.NodePath,
		Loader:             cfg.ProcessOptions.Loader,
		ShimArgs:           append([]string(nil), cfg.ShimArgs...),
		SocketPath:         socketPath,
		ReadyPath:          readyPath,
		HostAutoRegister:   cfg.HostAutoRegister,
		HostAppReadyModule: cfg.HostAppReadyModule,
		FilesExclude:       append([]string(nil), cfg.ProcessOptions.FilesExclude...),
		ExtraEnv:           append([]string(nil), cfg.ProcessOptions.Env...),
		ReadyTimeout:       cfg.HostReadyTimeout,
		Log:                log,
	}
}

func dialAndInitHost(cfg SupervisorConfig, conn net.Conn, proc *managedProcess, log *logrus.Logger) (*Supervisor, error) {
	client := NewClient(conn, conn, log)
	return finishSupervisorInit(cfg, client, proc, conn, log)
}

func finishSupervisorInit(cfg SupervisorConfig, client *Client, proc *managedProcess, conn io.Closer, log *logrus.Logger) (*Supervisor, error) {
	if cfg.RPC.CallTimeout > 0 {
		client.SetCallTimeout(cfg.RPC.CallTimeout)
	}
	if cfg.RPC.MaxMessageBytes > 0 {
		client.SetMaxMessageBytes(cfg.RPC.MaxMessageBytes)
	}
	if err := cfg.Manifest.Validate(); err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		if proc != nil {
			_ = proc.terminate()
		}
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if err := client.Initialize(cfg.Manifest, cfg.ProcessOptions.FilesExclude); err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		if proc != nil {
			_ = proc.terminate()
		}
		return nil, err
	}

	return &Supervisor{
		client:   client,
		proc:     proc,
		hostConn: conn,
		log:      log,
	}, nil
}

func (s *Supervisor) shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil && s.client.Initialized() {
		_ = s.client.Shutdown()
	}
	if s.hostConn != nil {
		_ = s.hostConn.Close()
		s.hostConn = nil
	}
	if s.proc != nil {
		if err := s.proc.terminate(); err != nil {
			return err
		}
		s.proc = nil
	}
	s.log.WithFields(logrus.Fields{
		"component": "nodert",
		"event":     "shutdown",
	}).Debug("supervisor shutdown complete")
	return nil
}

// resetSupervisorForTest clears singleton state for unit tests.
func resetSupervisorForTest() {
	if supervisorInst != nil {
		_ = supervisorInst.shutdown()
	}
	supervisorOnce = sync.Once{}
	supervisorInst = nil
	supervisorErr = nil
	supervisorErrPrinted = sync.Once{}
	configureOnce = sync.Once{}
	configureErr = nil
}
