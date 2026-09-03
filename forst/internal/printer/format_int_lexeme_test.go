package printer

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestFormatSource_preservesHexOctalBinaryIntLexemes(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	src := `package main

func main() {
	x := 0x36
	y := 0o77
	z := 0b1010
	n := -0x10
	println(x)
	println(y)
	println(z)
	println(n)
}
`
	out, err := FormatSource(src, "hex.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	for _, want := range []string{"0x36", "0o77", "0b1010", "-0x10"} {
		if !strings.Contains(out, want) {
			t.Fatalf("formatted source missing %q:\n%s", want, out)
		}
	}
	for _, bad := range []string{"x := 54", "y := 63", "z := 10", "n := -16"} {
		if strings.Contains(out, bad) {
			t.Fatalf("fmt collapsed base literal to decimal %q:\n%s", bad, out)
		}
	}
}
