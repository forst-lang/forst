package executor

import (
	"context"
	"os/exec"
)

// CommandRunner abstracts subprocess creation for tests and production.
type CommandRunner interface {
	Command(name string, arg ...string) *exec.Cmd
	CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd
}

// OSCommandRunner uses the real os/exec package.
type OSCommandRunner struct{}

func (OSCommandRunner) Command(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func (OSCommandRunner) CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}
