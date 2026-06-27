package typechecker

import "testing"

func TestProviders_N2_invalidMethodOnUseBinding(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }

func f() {
	use logger: Logger
	logger.info(42)
}
`
	_, err := parseAndCheck(t, src)
	if err == nil {
		t.Fatal("expected error for invalid method call on use binding")
	}
}
