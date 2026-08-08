package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

const generateTestAuthWithUnsatisfiedProviders = `package auth

type Logger = { info(msg String) }

func Login() {
	use logger: Logger
}

func Register() {
	use logger: Logger
}

func Echo(input { msg: String }) {
	return { msg: input.msg }
}
`

func writeAuthProviderFixture(t *testing.T, dir string) {
	t.Helper()
	authDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(authDir, "auth.ft")
	if err := os.WriteFile(path, []byte(generateTestAuthWithUnsatisfiedProviders), 0644); err != nil {
		t.Fatal(err)
	}
}

func captureGenerateLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := newGenerateLogger
	t.Cleanup(func() { newGenerateLogger = prev })
	newGenerateLogger = func() *logrus.Logger {
		log := logrus.New()
		log.SetLevel(logrus.WarnLevel)
		log.SetOutput(&buf)
		log.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableTimestamp: true})
		return log
	}
	return &buf
}

func TestGenerate_omissionReport_listsExcludedFunctions(t *testing.T) {
	dir := t.TempDir()
	writeAuthProviderFixture(t, dir)
	buf := captureGenerateLogs(t)

	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "generate: omitted 2 functions (unsatisfied providers)") {
		t.Fatalf("expected summary line, got:\n%s", out)
	}
	if !strings.Contains(out, "auth.Login") || !strings.Contains(out, "auth.Register") {
		t.Fatalf("expected omitted function names, got:\n%s", out)
	}

	core, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "core", "auth.js"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(core)
	if strings.Contains(s, "Login") || strings.Contains(s, "Register") {
		t.Fatalf("omitted functions must not appear in core module:\n%s", s)
	}
	if !strings.Contains(s, "Echo") {
		t.Fatalf("runnable Echo must still emit:\n%s", s)
	}
}

func TestGenerate_omitStubs_emitsCommentedStubsWhenEnabled(t *testing.T) {
	dir := t.TempDir()
	writeAuthProviderFixture(t, dir)
	cfg := `{"generate":{"link":"never","omitStubs":true}}`
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	for _, rel := range []string{"pkg/auth.js", "pkg/auth.d.ts"} {
		data, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), rel))
		if err != nil {
			t.Fatal(err)
		}
		s := string(data)
		if !strings.Contains(s, "// export function Login(") {
			t.Fatalf("%s missing Login stub:\n%s", rel, s)
		}
		if !strings.Contains(s, "// export function Register(") {
			t.Fatalf("%s missing Register stub:\n%s", rel, s)
		}
		if !strings.Contains(s, `// omitted: provider "Logger" not satisfied`) {
			t.Fatalf("%s missing omission reason:\n%s", rel, s)
		}
		// Stubs must not be live exports.
		if strings.Contains(s, "\nexport function Login(") || strings.Contains(s, "\nexport async function Login(") {
			t.Fatalf("%s must not emit a live Login export:\n%s", rel, s)
		}
	}
	core, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "core", "auth.js"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(core), "Login") {
		t.Fatalf("core must still omit Login:\n%s", core)
	}
}

func TestGenerate_omitStubs_defaultFalseDoesNotEmitStubs(t *testing.T) {
	dir := t.TempDir()
	writeAuthProviderFixture(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	pkgJS, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "pkg", "auth.js"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(pkgJS)
	if strings.Contains(s, "// omitted:") || strings.Contains(s, "// export function Login") {
		t.Fatalf("default omitStubs=false must not emit stubs:\n%s", s)
	}
}

func TestGenerate_omissionReport_namesProviderReason(t *testing.T) {
	dir := t.TempDir()
	writeAuthProviderFixture(t, dir)
	buf := captureGenerateLogs(t)

	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	out := buf.String()
	// TextFormatter escapes quotes inside quoted field values (\"Logger\").
	if !strings.Contains(out, `provider \"Logger\" not satisfied`) &&
		!strings.Contains(out, `provider "Logger" not satisfied`) {
		t.Fatalf("expected provider Logger reason in log, got:\n%s", out)
	}
	if !strings.Contains(out, "reason=") {
		t.Fatalf("expected logrus reason field, got:\n%s", out)
	}
	if !strings.Contains(out, "forstPackage=auth") && !strings.Contains(out, `forstPackage="auth"`) {
		t.Fatalf("expected forstPackage field, got:\n%s", out)
	}
	if !strings.Contains(out, "functionName=Login") && !strings.Contains(out, `functionName="Login"`) {
		t.Fatalf("expected functionName field, got:\n%s", out)
	}
}
