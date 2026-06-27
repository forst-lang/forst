package printer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestFormatSource_providersExample_preservesTokenFieldOrder(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "in", "rfc", "requirements", "providers.ft"))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(string(src), "providers.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	// Token fields must stay id before expiresAt (not alphabetical reorder).
	idxID := strings.Index(out, "id:")
	idxExp := strings.Index(out, "expiresAt:")
	if idxID < 0 || idxExp < 0 {
		t.Fatalf("Token typedef fields missing in formatted output:\n%s", out)
	}
	if idxID > idxExp {
		t.Fatalf("formatted output reordered Token fields (expiresAt before id):\n%s", out)
	}
}

func TestFormatSource_providersExample_roundTrips(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "in", "rfc", "requirements", "providers.ft"))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(string(src), "providers.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if out == "" {
		t.Fatal("empty output")
	}
	doc := FormatDocument(string(src), "providers.ft", 4, true, log)
	if doc == "" {
		t.Fatal("empty FormatDocument output")
	}
}
