package testutil

import (
	"errors"
	"fmt"
	"testing"
)

var (
	defaultTbFail  = tbFail
	defaultTbFailf = tbFailf
)

type panicOnFatal struct {
	*testing.T
}

func (p *panicOnFatal) Fatal(args ...any) {
	panic(fmt.Sprint(args...))
}

func (p *panicOnFatal) Fatalf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}

func TestDefaultHooks_delegateToTestingT(t *testing.T) {
	t.Run("failf", func(t *testing.T) {
		p := &panicOnFatal{T: t}
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic from default tbFailf")
			}
		}()
		defaultTbFailf(p, "boom")
	})
	t.Run("fail", func(t *testing.T) {
		p := &panicOnFatal{T: t}
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic from default tbFail")
			}
		}()
		defaultTbFail(p, errors.New("boom"))
	})
}
