package nodert

import (
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func readAvailableWithDeadline(r io.Reader, deadline time.Duration, buf []byte) (int, error) {
	f, ok := r.(*os.File)
	if !ok {
		return 0, errors.New("expected *os.File pipe")
	}
	if err := f.SetReadDeadline(time.Now().Add(deadline)); err != nil {
		return 0, err
	}
	defer func() { _ = f.SetReadDeadline(time.Time{}) }()
	return f.Read(buf)
}

func TestBootstrapStdout_staysIdleBeforeRpcFrames(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)

	bootstrap, err := ResolveBootstrapPath(repoRoot(t), "")
	if err != nil {
		t.Skipf("bootstrap not available: %v", err)
	}

	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "payment.ts"), []byte(`export function add(a: number, b: number) { return { sum: a + b }; }`), 0o644); err != nil {
		t.Fatal(err)
	}

	spawnCmd, err := BuildBootstrapSpawnCommand(BootstrapSpawnInput{
		BoundaryRoot:  root,
		BootstrapPath: bootstrap,
	})
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(spawnCmd.Executable, spawnCmd.Args...)
	cmd.Dir = root
	cmd.Env = spawnCmd.Env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	errCollected, stderrErr := collectUntilSubstring(stderr, "spawn", 15*time.Second)
	if stderrErr != nil {
		failBootstrapProcessDied(t, cmd, errCollected, "stderr read: "+stderrErr.Error())
	}
	if len(errCollected) == 0 {
		failBootstrapProcessDied(t, cmd, errCollected, "expected bootstrap logs on stderr")
	}
	if !strings.Contains(string(errCollected), "spawn") {
		failBootstrapProcessDied(t, cmd, errCollected, "stderr missing spawn log, got "+strconv.Quote(string(errCollected)))
	}

	buf := make([]byte, 64)
	n, readErr := readAvailableWithDeadline(stdout, 200*time.Millisecond, buf)
	if readErr != nil && !errors.Is(readErr, os.ErrDeadlineExceeded) {
		failBootstrapProcessDied(t, cmd, errCollected, "stdout read: "+readErr.Error())
	}
	if n > 0 {
		t.Fatalf("stdout must stay empty before RPC frames, got hex=%s text=%q",
			hex.EncodeToString(buf[:n]), string(buf[:n]))
	}

	_ = stdin.Close()
	_, _ = io.Copy(io.Discard, stdout)
	_ = cmd.Wait()
}

func collectUntilSubstring(r io.Reader, needle string, timeout time.Duration) ([]byte, error) {
	collected := make([]byte, 0, 4096)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		chunk := make([]byte, 512)
		n, err := readAvailableWithDeadline(r, 200*time.Millisecond, chunk)
		if n > 0 {
			collected = append(collected, chunk[:n]...)
			if strings.Contains(string(collected), needle) {
				return collected, nil
			}
		}
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			return collected, err
		}
	}
	return collected, nil
}

func failBootstrapProcessDied(t *testing.T, cmd *exec.Cmd, stderr []byte, reason string) {
	t.Helper()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	t.Fatalf("%s (wait=%v stderr=%q)", reason, waitErr, string(stderr))
}
