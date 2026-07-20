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
	supervisorMu         sync.Mutex
	supervisorInst       *Supervisor
	supervisorErr        error
	supervisorErrPrinted sync.Once
	supervisorCfg        SupervisorConfig
)

// ConfigureSupervisor sets options used by the first GetClient call.
func ConfigureSupervisor(cfg SupervisorConfig) {
	supervisorMu.Lock()
	defer supervisorMu.Unlock()
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
	supervisorMu.Lock()
	defer supervisorMu.Unlock()
	return getClientLocked()
}

func getClientLocked() (*Client, error) {
	if supervisorInst != nil {
		if supervisorInst.isAlive() {
			return supervisorInst.client, nil
		}
		discardDeadSupervisorLocked()
	}

	var err error
	if supervisorCfg.HostMode {
		supervisorInst, err = newHostSupervisor(supervisorCfg)
	} else {
		supervisorInst, err = newBootstrapSupervisor(supervisorCfg)
	}
	supervisorErr = err
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
	supervisorMu.Lock()
	defer supervisorMu.Unlock()
	if supervisorInst == nil {
		return nil
	}
	err := supervisorInst.shutdown()
	supervisorInst = nil
	supervisorErr = nil
	return err
}

func discardDeadSupervisorLocked() {
	if supervisorInst == nil {
		return
	}
	_ = supervisorInst.shutdown()
	supervisorInst = nil
	supervisorErr = nil
}

func (s *Supervisor) isAlive() bool {
	if s == nil || s.client == nil {
		return false
	}
	if s.client.readLoopExited() {
		return false
	}
	if s.proc != nil && s.proc.cmd != nil && s.proc.cmd.Process != nil {
		if !processAlive(s.proc.cmd.Process.Pid) {
			return false
		}
	}
	return true
}

func (s *Supervisor) watchProcess(proc *managedProcess) {
	if proc == nil {
		return
	}
	_ = proc.wait()
	supervisorMu.Lock()
	defer supervisorMu.Unlock()
	if supervisorInst == s {
		discardDeadSupervisorLocked()
	}
}

const bootstrapReadyTimeout = 30 * time.Second

func newBootstrapSupervisor(cfg SupervisorConfig) (*Supervisor, error) {
	log := cfg.ProcessOptions.Log
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.ErrorLevel)
	}

	socketPath, readyPath, err := ResolveBootstrapSocketPath(cfg.ProcessOptions.BoundaryRoot)
	if err != nil {
		return nil, err
	}
	if err := PrepareHostSocket(socketPath, readyPath); err != nil {
		return nil, err
	}

	proc, err := spawnBootstrapProcess(cfg.ProcessOptions, socketPath, readyPath)
	if err != nil {
		cleanupHostSocketFiles(socketPath, readyPath)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), bootstrapReadyTimeout)
	defer cancel()

	conn, err := waitForHostReady(ctx, socketPath, readyPath)
	if err != nil {
		_ = proc.terminate()
		cleanupHostSocketFiles(socketPath, readyPath)
		return nil, err
	}

	s, err := dialAndInitHost(cfg, conn, proc, log)
	if err != nil {
		cleanupHostSocketFiles(socketPath, readyPath)
		return nil, err
	}
	go s.watchProcess(proc)
	return s, nil
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

	s, err := dialAndInitHost(cfg, conn, proc, log)
	if err != nil {
		return nil, err
	}
	if proc != nil {
		go s.watchProcess(proc)
	}
	return s, nil
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
	supervisorMu.Lock()
	defer supervisorMu.Unlock()
	if supervisorInst != nil {
		_ = supervisorInst.shutdown()
	}
	supervisorInst = nil
	supervisorErr = nil
	supervisorErrPrinted = sync.Once{}
	configureOnce = sync.Once{}
	configureErr = nil
}
