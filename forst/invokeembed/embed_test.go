package invokeembed

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"forst/internal/invokeserver"
	"forst/nodert"
)

func TestGlobalRegistry_sameInstance(t *testing.T) {
	r1 := GlobalRegistry()
	r2 := GlobalRegistry()
	if r1 != r2 {
		t.Fatal("expected same registry instance")
	}
	if r1 != invokeserver.GlobalRegistry() {
		t.Fatal("expected delegate to invokeserver.GlobalRegistry")
	}
}

func TestMustStartEmbedded_idempotent(t *testing.T) {
	workDir := t.TempDir()
	t.Setenv("FORST_ROOT", workDir)
	if err := os.WriteFile(filepath.Join(workDir, "ftconfig.json"), []byte(`{"server":{"embedded":true,"port":"0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	MustStartEmbedded()
	t.Cleanup(invokeserver.StopEmbeddedForTest)
	MustStartEmbedded()
}

func TestMustPrepareEmbeddedHostAuth_ok(t *testing.T) {
	workDir := t.TempDir()
	t.Setenv(nodert.EnvRoot, workDir)
	if err := os.WriteFile(filepath.Join(workDir, "ftconfig.json"), []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustPrepareEmbeddedHostAuth panicked: %v", r)
		}
	}()
	MustPrepareEmbeddedHostAuth()
}

func TestMustPrepareEmbeddedHostAuth_panicsOnBadBoundary(t *testing.T) {
	badDir := t.TempDir()
	t.Setenv(nodert.EnvRoot, badDir)
	if err := os.WriteFile(filepath.Join(badDir, "ftconfig.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	gotPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				gotPanic = true
			}
		}()
		MustPrepareEmbeddedHostAuth()
	}()
	if !gotPanic {
		t.Fatal("expected panic when ftconfig is invalid")
	}
}

// WaitForShutdown signal delivery is covered in invokeserver; here we only test the invokeembed delegate.
func TestWaitForShutdown_unblocksOnNotify(t *testing.T) {
	done := make(chan struct{})
	go func() {
		WaitForShutdown()
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		invokeserver.NotifyShutdown()
		select {
		case <-done:
			return
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatal("invokeembed.WaitForShutdown did not return after invokeserver.NotifyShutdown")
}
