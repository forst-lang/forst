package executor

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

type mockCommandRunner struct {
	commandContextFn func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

func (m *mockCommandRunner) Command(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func (m *mockCommandRunner) CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if m.commandContextFn != nil {
		return m.commandContextFn(ctx, name, arg...)
	}
	return exec.CommandContext(ctx, name, arg...)
}

func TestCommandRunner_defaultUsesGoRun(t *testing.T) {
	runner := OSCommandRunner{}
	cmd := runner.CommandContext(context.Background(), "go", "version")
	if cmd.Path == "" && len(cmd.Args) == 0 {
		t.Fatal("expected command to be constructed")
	}
}

func TestFunctionExecutor_ExecuteFunction_contextCancelKillsGoRun(t *testing.T) {
	e := testExecutor(t, t.TempDir())
	e.SetCommandRunnerForTest(&mockCommandRunner{
		commandContextFn: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "sleep", "30")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := e.executeGoCode(ctx, t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("expected ctx canceled, ctx.Err=%v err=%v", ctx.Err(), err)
	}
}

func TestFunctionExecutor_ExecuteFunction_deadlineExceeded(t *testing.T) {
	e := testExecutor(t, t.TempDir())
	e.SetCommandRunnerForTest(&mockCommandRunner{
		commandContextFn: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "sleep", "30")
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := e.executeGoCode(ctx, t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected deadline error")
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("expected ctx deadline, ctx.Err=%v err=%v", ctx.Err(), err)
	}
}
