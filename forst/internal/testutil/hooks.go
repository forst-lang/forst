package testutil

import "testing"

var (
	tbFail  = func(tb testing.TB, args ...any) { tb.Fatal(args...) }
	tbFailf = func(tb testing.TB, format string, args ...any) { tb.Fatalf(format, args...) }
)
