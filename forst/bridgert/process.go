package bridgert

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"forst/internal/ftconfig"
	logrus "github.com/sirupsen/logrus"
)

// ProcessOptions configures how the Node child process is spawned.
type ProcessOptions struct {
	NodePath      string
	BootstrapPath string
	WorkDir       string
	Bridge        ftconfig.Bridge
	ModuleIDs     []string
	BoundaryRoot  string
	FilesExclude  []string
	Env           []string
	Log           *logrus.Logger
}

// managedProcess holds a spawned child process and optional stdio pipes.
type managedProcess struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	stderrRing stderrRingBuffer
	log        *logrus.Logger
	waitMu     sync.Mutex
	waited     bool
	waitErr    error
}

const childStderrRingCap = 2048

// stderrRingBuffer captures recent stderr for error diagnostics.
type stderrRingBuffer interface {
	io.Writer
	tail() string
}

type outputRing struct {
	mu      sync.Mutex
	buf     []byte
	ringCap int
}

var _ stderrRingBuffer = (*outputRing)(nil)

func newOutputRing(ringCap int) *outputRing {
	if ringCap <= 0 {
		ringCap = childStderrRingCap
	}
	return &outputRing{ringCap: ringCap}
}

func (r *outputRing) Write(p []byte) (int, error) {
	if r == nil {
		return len(p), nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = append(r.buf, p...)
	if len(r.buf) > r.ringCap {
		r.buf = r.buf[len(r.buf)-r.ringCap:]
	}
	return len(p), nil
}

func (r *outputRing) tail() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.TrimSpace(string(r.buf))
}

func (p *managedProcess) stderrTail() string {
	if p == nil || p.stderrRing == nil {
		return ""
	}
	return p.stderrRing.tail()
}

func spawnBootstrapProcess(opts ProcessOptions, socketPath, readyPath string) (*managedProcess, error) {
	spawnCmd, err := BuildBootstrapSpawnCommand(BootstrapSpawnInput{
		BoundaryRoot:  opts.BoundaryRoot,
		Executable:    opts.NodePath,
		BootstrapPath: opts.BootstrapPath,
		WorkDir:       opts.WorkDir,
		Bridge:        opts.Bridge,
		ModuleIDs:     opts.ModuleIDs,
		SocketPath:    socketPath,
		ReadyPath:     readyPath,
		FilesExclude:  opts.FilesExclude,
		Env:           opts.Env,
	})
	if err != nil {
		return nil, err
	}

	log := opts.Log
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.ErrorLevel)
	}

	cmd := exec.Command(spawnCmd.Executable, spawnCmd.Args...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}
	cmd.Env = spawnCmd.Env
	cmd.SysProcAttr = hostSessionAttrs()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, fmt.Errorf("start node process (executable=%s): %w", spawnCmd.Executable, err)
	}

	log.WithFields(logrus.Fields{
		"component":       "bridgert",
		"event":           "spawn",
		"mode":            "bootstrap",
		"node_pid":        cmd.Process.Pid,
		"node_executable": spawnCmd.Executable,
		"bootstrap_path":  lastArg(spawnCmd.Args),
		"host":            opts.Bridge.Host,
		"format":          opts.Bridge.ModuleFormat,
		"socket_path":     socketPath,
	}).Debug("spawned node runtime")

	go forwardChildOutput(stdout, os.Stdout, log, "stdout")
	stderrRing := newOutputRing(childStderrRingCap)
	go forwardChildOutput(stderr, io.MultiWriter(os.Stderr, stderrRing), log, "stderr")

	return &managedProcess{
		cmd:        cmd,
		stdout:     stdout,
		stderr:     stderr,
		stderrRing: stderrRing,
		log:        log,
	}, nil
}

func (p *managedProcess) wait() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	p.waitMu.Lock()
	defer p.waitMu.Unlock()
	if p.waited {
		return p.waitErr
	}
	p.waited = true
	err := p.cmd.Wait()
	if err != nil {
		p.waitErr = fmt.Errorf("%w: %v", ErrNodeRuntimeDied, err)
		return p.waitErr
	}
	return nil
}

func (p *managedProcess) waitAsync(log *logrus.Logger) error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	err := p.wait()
	if err != nil && log != nil {
		log.WithFields(logrus.Fields{
			"component": "bridgert",
			"event":     "host_child_exit",
		}).Debug(err.Error())
	}
	return err
}

func (p *managedProcess) terminate() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	if p.stdin != nil {
		_ = p.stdin.Close()
	}

	done := make(chan error, 1)
	go func() {
		done <- p.wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(shutdownGracePeriod):
		if p.log != nil {
			p.log.WithFields(logrus.Fields{
				"component": "bridgert",
				"event":     "timeout",
			}).Warn("node runtime graceful shutdown timed out, sending SIGKILL")
		}
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill node process: %w", err)
		}
		<-done
		return nil
	}
}

const shutdownGracePeriod = 5 * time.Second

func buildNodeChildEnv(opts ProcessOptions) []string {
	env := append([]string(nil), os.Environ()...)
	for _, entry := range opts.Env {
		if key, _, ok := strings.Cut(entry, "="); ok {
			env = setEnvVar(env, key, strings.TrimPrefix(entry, key+"="))
		}
	}
	if opts.BoundaryRoot != "" {
		env = setEnvVar(env, EnvRoot, opts.BoundaryRoot)
	}
	env = setEnvVar(env, "FORST_BRIDGE_PROTOCOL", envNodeProtocolDefault)
	if len(opts.FilesExclude) > 0 {
		data, err := json.Marshal(opts.FilesExclude)
		if err == nil {
			env = setEnvVar(env, "FORST_FILES_EXCLUDE", string(data))
		}
	}
	return env
}

func lastArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[len(args)-1]
}

func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered, prefix+value)
}

func forwardChildOutput(src io.ReadCloser, dst io.Writer, log *logrus.Logger, stream string) {
	defer func() { _ = src.Close() }()
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil && log != nil {
				log.WithFields(logrus.Fields{
					"component": "bridgert",
					"event":     "forward_" + stream,
				}).Debugf("write failed: %v", werr)
			}
			if log != nil {
				log.WithFields(logrus.Fields{
					"component": "bridgert",
					"event":     stream,
				}).Debug(string(buf[:n]))
			}
		}
		if err != nil {
			return
		}
	}
}
