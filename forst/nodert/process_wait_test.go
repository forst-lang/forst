package nodert

import (
	"os/exec"
	"strconv"
	"sync"
	"testing"
)

func TestManagedProcess_waitConcurrent(t *testing.T) {
	cmd := exec.Command("sleep", strconv.Itoa(30))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	proc := &managedProcess{cmd: cmd}

	var wg sync.WaitGroup
	errs := make(chan error, 3)
	run := func(fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- fn()
		}()
	}
	run(proc.wait)
	run(proc.wait)
	run(proc.terminate)

	wg.Wait()
	close(errs)

	var got []error
	for err := range errs {
		got = append(got, err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 wait results, got %d", len(got))
	}
}
