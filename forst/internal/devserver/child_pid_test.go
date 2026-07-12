package devserver

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestWriteAndReadGoChildPID(t *testing.T) {
	dir := t.TempDir()
	if err := WriteGoChildPID(dir, 4242); err != nil {
		t.Fatal(err)
	}
	if got := ReadGoChildPID(dir); got != 4242 {
		t.Fatalf("ReadGoChildPID() = %d want 4242", got)
	}
	if err := ClearGoChildPID(dir); err != nil {
		t.Fatal(err)
	}
	if got := ReadGoChildPID(dir); got != 0 {
		t.Fatalf("after clear ReadGoChildPID() = %d want 0", got)
	}
}

func TestReapOrphanedGoChild_skipsCurrentPID(t *testing.T) {
	dir := t.TempDir()
	if err := WriteGoChildPID(dir, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	ReapOrphanedGoChild(dir, os.Getpid(), nil)
	if got := ReadGoChildPID(dir); got != os.Getpid() {
		t.Fatalf("current pid should not be reaped, got %d", got)
	}
}

func TestFormatInvokePortShiftLog_includesReservedSkip(t *testing.T) {
	t.Setenv("PORT", "6322")
	got := FormatInvokePortShiftLog("6321", "6323")
	if want := "invoke listen port 6323 (configured 6321 in use), skipped reserved 6322"; got != want {
		t.Fatalf("log=%q want %q", got, want)
	}
}

func TestFindNextFreeInvokePort_skipsReservedPortEnv(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, boundPort, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	nextPort, _ := strconv.Atoi(boundPort)
	nextPort++
	reserved := strconv.Itoa(nextPort)
	t.Setenv("PORT", reserved)

	chosen, err := FindNextFreeInvokePort("127.0.0.1", boundPort)
	if err != nil {
		t.Fatal(err)
	}
	if chosen == reserved {
		t.Fatalf("expected to skip reserved PORT=%s, got %s", reserved, chosen)
	}
	if !portCanListen("127.0.0.1", chosen) {
		t.Fatalf("picked port %s is not bindable", chosen)
	}
}

func TestReapOrphanedGoChild_terminatesAliveProcess(t *testing.T) {
	if os.Getenv("FORST_SKIP_ORPHAN_REAP_TEST") == "1" {
		t.Skip("FORST_SKIP_ORPHAN_REAP_TEST=1")
	}
	dir := t.TempDir()
	cmd := exec.Command("sleep", "300")
	if err := cmd.Start(); err != nil {
		t.Skipf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	if err := WriteGoChildPID(dir, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	ReapOrphanedGoChild(dir, 0, log)
	if !strings.Contains(buf.String(), "Reaping orphaned go child") {
		t.Fatalf("expected reap log, got: %s", buf.String())
	}
	if got := ReadGoChildPID(dir); got != 0 {
		t.Fatalf("pid file should be cleared, got %d", got)
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case <-waitDone:
	case <-time.After(10 * time.Second):
		t.Fatalf("orphan sleep pid=%d still alive after reap", cmd.Process.Pid)
	}
}

func TestPerformDevShutdown_clearsMarkers(t *testing.T) {
	dir := t.TempDir()
	if err := WriteGoChildPID(dir, 99999); err != nil {
		t.Fatal(err)
	}
	readyPath := filepath.Join(dir, ".forst", "invoke.ready")
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readyPath, []byte(`{"url":"http://127.0.0.1:6321"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	stopped := false
	performDevShutdown(devShutdownState{
		boundaryRoot: dir,
		child: &runningChild{
			stop: func() error {
				stopped = true
				return nil
			},
		},
	})

	if !stopped {
		t.Fatal("expected child stop")
	}
	if _, err := os.Stat(goChildPIDPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("go-child.pid should be removed, err=%v", err)
	}
	if _, err := os.Stat(readyPath); !os.IsNotExist(err) {
		t.Fatalf("invoke.ready should be removed, err=%v", err)
	}
}
