package testutil

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseSource(t *testing.T) {
	src := `package main
func main() {}
`
	nodes := ParseSource(t, src, "test.ft", nil)
	if len(nodes) == 0 {
		t.Fatal("expected nodes")
	}
}

func TestParseSource_defaults(t *testing.T) {
	src := `package main
func main() {}
`
	nodes := ParseSource(t, src, "", nil)
	if len(nodes) == 0 {
		t.Fatal("expected nodes")
	}
}

func TestParseSource_parseError(t *testing.T) {
	log := logrus.New()
	log.SetOutput(nil)
	log.SetLevel(logrus.ErrorLevel)
	msg := stubFailf(t, func() {
		ParseSource(t, "package main\n<<<", "bad.ft", log)
	})
	if !strings.Contains(msg, "parse bad.ft") {
		t.Fatalf("msg = %q", msg)
	}
}

func TestParseSourceForBench(t *testing.T) {
	src := []byte(`package main
func main() {}
`)
	nodes := ParseSourceForBench(t, src, "")
	if len(nodes) == 0 {
		t.Fatal("expected nodes")
	}
}

func TestParseSourceForBench_parseError(t *testing.T) {
	msg := stubFailf(t, func() {
		ParseSourceForBench(t, []byte("package main\n<<<"), "bench.ft")
	})
	if !strings.Contains(msg, "parse:") {
		t.Fatalf("msg = %q", msg)
	}
}
