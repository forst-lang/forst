package invokeserver

import "testing"

func TestSafeErrorMessage_preservesWhitespace(t *testing.T) {
	got := safeErrorMessage("failed at /tmp/a\n\tstill\tok")
	want := "failed at [path]\n\tstill\tok"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSafeErrorMessage_reloadingSentinelUnchanged(t *testing.T) {
	if got := safeErrorMessage("reloading"); got != "reloading" {
		t.Fatalf("got %q", got)
	}
}

func TestSafeErrorMessage_detectsBackslashPaths(t *testing.T) {
	got := safeErrorMessage(`open C:\secret\key failed`)
	if got == `open C:\secret\key failed` {
		t.Fatal("expected path sanitization for backslashes")
	}
}
