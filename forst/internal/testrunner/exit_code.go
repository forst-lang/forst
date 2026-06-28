package testrunner

// ExitCode is the process exit status returned by Run.
type ExitCode int

const (
	// ExitSuccess — all tests passed.
	ExitSuccess ExitCode = 0
	// ExitFailure — tests failed or Forst compile/typecheck/module check failed.
	ExitFailure ExitCode = 1
	// ExitError — discovery, setup, or runner infrastructure error.
	ExitError ExitCode = 2
)

// Int returns the numeric exit code for os.Exit.
func (c ExitCode) Int() int { return int(c) }
