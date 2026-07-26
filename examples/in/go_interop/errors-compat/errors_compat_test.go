package errorscompat_test

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	errorscompat "errors_compat"
)

func TestWrappedPkgError_wrapPreservesRootMessage(t *testing.T) {
	t.Parallel()
	err := errorscompat.WrappedPkgError("probe")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "wrap") || !strings.Contains(msg, "root: probe") {
		t.Fatalf("unexpected message: %q", msg)
	}
	if errors.Unwrap(err) == nil {
		t.Fatal("expected unwrap chain from pkg/errors wrap")
	}
}

func TestCockroachdbWrapped_wrapPreservesInnerMessage(t *testing.T) {
	t.Parallel()
	err := errorscompat.CockroachdbWrapped("probe")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "outer") || !strings.Contains(msg, "inner: probe") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestMultierrCombined_joinsErrors(t *testing.T) {
	t.Parallel()
	a := fmt.Errorf("a")
	b := fmt.Errorf("b")
	err := errorscompat.MultierrCombined(a, b)
	if err == nil {
		t.Fatal("expected combined error")
	}
	if !errors.Is(err, a) || !errors.Is(err, b) {
		t.Fatalf("combined error should contain both: %v", err)
	}
}

func TestByteReader_satisfiesIOReader(t *testing.T) {
	t.Parallel()
	var _ io.Reader = errorscompat.NewByteReader("hi")
	r := errorscompat.NewByteReader("ab")
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if n != 2 || err != io.EOF {
		t.Fatalf("short read with EOF: n=%d err=%v", n, err)
	}
	n, err = r.Read(buf)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("drained Read: n=%d err=%v", n, err)
	}
}
