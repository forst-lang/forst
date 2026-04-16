package lsp

import "testing"

func TestURIDisplayBasename_fileURI(t *testing.T) {
	t.Parallel()
	if got := uriDisplayBasename("file:///tmp/foo/bar.ft"); got != "bar.ft" {
		t.Fatalf("got %q", got)
	}
}
