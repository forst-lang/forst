package nodert

import (
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

	time.Sleep(500 * time.Millisecond)

	buf := make([]byte, 64)
	n, readErr := readAvailableWithDeadline(stdout, 200*time.Millisecond, buf)
	if readErr != nil && !errors.Is(readErr, os.ErrDeadlineExceeded) {
		t.Fatalf("stdout read: %v", readErr)
	}
	if n > 0 {
		t.Fatalf("stdout must stay empty before RPC frames, got hex=%s text=%q",
			hex.EncodeToString(buf[:n]), string(buf[:n]))
	}

	errCollected := make([]byte, 0, 4096)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		chunk := make([]byte, 512)
		errN, errReadErr := readAvailableWithDeadline(stderr, 200*time.Millisecond, chunk)
		if errReadErr != nil && !errors.Is(errReadErr, os.ErrDeadlineExceeded) {
			t.Fatalf("stderr read: %v", errReadErr)
		}
		if errN > 0 {
			errCollected = append(errCollected, chunk[:errN]...)
			if strings.Contains(string(errCollected), "spawn") {
				break
			}
		}
	}
	if len(errCollected) == 0 {
		t.Fatal("expected bootstrap logs on stderr")
	}
	if !strings.Contains(string(errCollected), "spawn") {
		t.Fatalf("stderr missing spawn log, got %q", string(errCollected))
	}

	_ = stdin.Close()
	_, _ = io.Copy(io.Discard, stdout)
	_ = cmd.Wait()
}
