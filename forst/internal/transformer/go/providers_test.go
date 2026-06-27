package transformergo

import (
	"strings"
	"testing"
)

func TestProvidersLowering_expireTokenPattern(t *testing.T) {
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

	if !strings.Contains(out, "type Providers_") {
		t.Fatalf("expected deduped Providers struct, got:\n%s", out)
	}
	if !strings.Contains(out, "type Logger interface") {
		t.Fatalf("expected Logger interface, got:\n%s", out)
	}
	if !strings.Contains(out, "type Clock interface") {
		t.Fatalf("expected Clock interface, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "func expireToken(providers Providers_") {
		t.Fatalf("expected providers first param on expireToken, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "logger := providers.Logger") {
		t.Fatalf("expected use lowering to providers field bind, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "clock := providers.Clock") {
		t.Fatalf("expected clock bind from providers, got:\n%s", out)
	}
	if strings.Contains(out, "func TestExpireToken(providers ") {
		t.Fatalf("Test wiring root should not get providers param, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "expireToken(Providers_") {
		t.Fatalf("expected with-block call to pass Providers struct literal, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "Logger: &NopLogger{}") {
		t.Fatalf("expected Logger wiring in struct literal, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "Clock:&FakeClock{fixedMs:2000}") {
		t.Fatalf("expected Clock wiring in struct literal, got:\n%s", out)
	}

	t.Logf("emitted Go:\n%s", out)
}
