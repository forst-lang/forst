package testrunner

import "testing"

func TestExitCode_Int(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code ExitCode
		want int
	}{
		{ExitSuccess, 0},
		{ExitFailure, 1},
		{ExitError, 2},
	}
	for _, tc := range cases {
		if got := tc.code.Int(); got != tc.want {
			t.Fatalf("ExitCode(%d).Int() = %d, want %d", tc.code, got, tc.want)
		}
	}
}
