package testutil

import (
	"fmt"
	"testing"
)

func stubFailf(t *testing.T, fn func()) string {
	t.Helper()
	var msg string
	orig := tbFailf
	t.Cleanup(func() { tbFailf = orig })
	tbFailf = func(_ testing.TB, format string, args ...any) {
		msg = fmt.Sprintf(format, args...)
	}
	fn()
	if msg == "" {
		t.Fatal("expected failf")
	}
	return msg
}

func stubFail(t *testing.T, fn func()) string {
	t.Helper()
	var msg string
	orig := tbFail
	t.Cleanup(func() { tbFail = orig })
	tbFail = func(_ testing.TB, args ...any) {
		msg = fmt.Sprint(args...)
	}
	fn()
	if msg == "" {
		t.Fatal("expected fail")
	}
	return msg
}
