package forstpkg

import (
	"errors"
	"testing"

	"forst/internal/parser"
)

func TestParseRecoverToError_parseError(t *testing.T) {
	t.Parallel()
	pe := &parser.ParseError{Msg: "bad syntax"}
	if err := parseRecoverToError("f.ft", pe); err != pe {
		t.Fatalf("got %v", err)
	}
}

func TestParseRecoverToError_genericPanic(t *testing.T) {
	t.Parallel()
	err := parseRecoverToError("f.ft", errors.New("boom"))
	if err == nil || err.Error() != "parse panic in f.ft: boom" {
		t.Fatalf("got %v", err)
	}
}
