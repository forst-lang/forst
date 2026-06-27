package printer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestFormatSource_usablesExample_roundTrips(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "in", "rfc", "requirements", "usables.ft"))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(string(src), "usables.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if out == "" {
		t.Fatal("empty output")
	}
	doc := FormatDocument(string(src), "usables.ft", 4, true, log)
	if doc == "" {
		t.Fatal("empty FormatDocument output")
	}
}
