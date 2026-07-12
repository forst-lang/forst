package compiler

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"forst/internal/gowork"
)

const defaultGoProgramStopGrace = 5 * time.Second

// DefaultGoProgramStopGrace is the SIGTERM wait before SIGKILL for background go run.
func DefaultGoProgramStopGrace() time.Duration {
	return defaultGoProgramStopGrace
}

// StopOpts configures how a background go run child is terminated.
type StopOpts struct {
	// NarrowKill signals only the direct child PID (not the process group).
	// Reserved for unit tests; runtime dev uses process-group stop.
	NarrowKill bool
}

// GoProgramProcess is a background go run child.
type GoProgramProcess struct {
	cmd    *exec.Cmd
	stderr bytes.Buffer
	done   chan error
}

// StartGoProgram runs generated Go in the background (non-blocking).
func StartGoProgram(outputPath, boundaryRoot string) (*GoProgramProcess, error) {
	cmd, err := newGoRunCommand(outputPath, boundaryRoot)
	if err != nil {
		return nil, err
	}
	proc := &GoProgramProcess{
		cmd:  cmd,
		done: make(chan error, 1),
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &proc.stderr)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		waitErr := cmd.Wait()
		proc.done <- formatRunProgramError(waitErr, proc.stderr.String())
	}()
	return proc, nil
}

// Done returns a channel that receives the go run exit error (if any).
func (p *GoProgramProcess) Done() <-chan error {
	if p == nil {
		ch := make(chan error)
		close(ch)
		return ch
	}
	return p.done
}

func (p *GoProgramProcess) Stop(grace time.Duration, opts StopOpts) error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	if grace <= 0 {
		grace = defaultGoProgramStopGrace
	}
	signal := signalProcessGroup
	if opts.NarrowKill {
		signal = signalProcessOnly
	}
	_ = signal(p.cmd.Process, syscall.SIGTERM)
	select {
	case <-p.done:
	case <-time.After(grace):
	}
	_ = signal(p.cmd.Process, syscall.SIGKILL)
	select {
	case <-p.done:
	case <-time.After(2 * time.Second):
	}
	return nil
}

func signalProcessOnly(proc *os.Process, sig syscall.Signal) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(sig)
}

func signalProcessGroup(proc *os.Process, sig syscall.Signal) error {
	if proc == nil {
		return nil
	}
	if err := syscall.Kill(-proc.Pid, sig); err != nil {
		return proc.Signal(sig)
	}
	return nil
}

func newGoRunCommand(outputPath, boundaryRoot string) (*exec.Cmd, error) {
	dir, runSources, err := runGoSourceFiles(outputPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("go", append([]string{"run"}, runSources...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Dir = dir
	env := os.Environ()
	if boundaryRoot != "" {
		env = setRunEnvBoundaryRoot(env, boundaryRoot)
		needsCompiler := tempDirHasForstCompanionFiles(filepath.Dir(outputPath))
		plan, _ := gowork.PlanForRun(boundaryRoot, filepath.Dir(outputPath), needsCompiler)
		env = gowork.ChildEnv(env, plan, boundaryRoot)
	}
	cmd.Env = env
	return cmd, nil
}
