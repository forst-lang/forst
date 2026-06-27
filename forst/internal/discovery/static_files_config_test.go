package discovery

import (
	"errors"
	"testing"
)

func TestStaticFilesConfig_FindForstFiles(t *testing.T) {
	files := []string{"/a/foo.ft", "/b/bar.ft"}
	cfg := NewStaticFilesConfig(files)
	got, err := cfg.FindForstFiles("/ignored")
	if err != nil {
		t.Fatalf("FindForstFiles: %v", err)
	}
	if len(got) != 2 || got[0] != files[0] || got[1] != files[1] {
		t.Fatalf("got %v want %v", got, files)
	}
}

func TestStaticFilesConfig_FindForstFiles_error(t *testing.T) {
	want := errors.New("boom")
	cfg := &StaticFilesConfig{Err: want}
	_, err := cfg.FindForstFiles("")
	if err != want {
		t.Fatalf("got %v want %v", err, want)
	}
}
