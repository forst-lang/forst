package transformergo

import (
	"strings"
	"testing"
)

func TestUsablesLowering_expireTokenPattern(t *testing.T) {
	src := `package main

import "testing"

type Logger = {
	info(msg String)
}

type Clock = {
	now(): Int
}

type Token = {
	id: String
	expiresAt: Int
}

type NopLogger = {}
type FakeClock = { fixedMs: Int }

func (NopLogger) info(msg String) {}
func (c FakeClock) now(): Int { return c.fixedMs }

func expireToken(token Token) {
	use logger: Logger
	use clock: Clock
	if token.expiresAt < clock.now() {
		logger.info("expired")
	}
}

func TestExpireToken(t *testing.T) {
	token := Token { id: "t1", expiresAt: 1000 }
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 2000 },
	} {
		expireToken(token)
	}
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})

	if !strings.Contains(out, "type Usables_") {
		t.Fatalf("expected deduped Usables struct, got:\n%s", out)
	}
	if !strings.Contains(out, "type Logger interface") {
		t.Fatalf("expected Logger interface, got:\n%s", out)
	}
	if !strings.Contains(out, "type Clock interface") {
		t.Fatalf("expected Clock interface, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "func expireToken(usables Usables_") {
		t.Fatalf("expected usables first param on expireToken, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "logger := usables.Logger") {
		t.Fatalf("expected use lowering to usables field bind, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "clock := usables.Clock") {
		t.Fatalf("expected clock bind from usables, got:\n%s", out)
	}
	if strings.Contains(out, "func TestExpireToken(usables ") {
		t.Fatalf("Test wiring root should not get usables param, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "expireToken(Usables_") {
		t.Fatalf("expected with-block call to pass Usables struct literal, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "Logger: &NopLogger{}") {
		t.Fatalf("expected Logger wiring in struct literal, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "Clock:&FakeClock{fixedMs:2000}") {
		t.Fatalf("expected Clock wiring in struct literal, got:\n%s", out)
	}

	t.Logf("emitted Go:\n%s", out)
}
