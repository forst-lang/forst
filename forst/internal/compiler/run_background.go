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

// Stop sends SIGTERM, waits up to grace, then SIGKILL.
func (p *GoProgramProcess) Stop(grace time.Duration) error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	if grace <= 0 {
		grace = defaultGoProgramStopGrace
	}
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.done:
		return nil
	case <-time.After(grace):
		_ = p.cmd.Process.Kill()
		<-p.done
		return nil
	}
}

func newGoRunCommand(outputPath, boundaryRoot string) (*exec.Cmd, error) {
	dir, runSources, err := runGoSourceFiles(outputPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("go", append([]string{"run"}, runSources...)...)
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
