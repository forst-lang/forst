package typechecker

import (
	"errors"
	"strings"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestCheckTypes_forstSubslice_lowHigh(t *testing.T) {
	t.Parallel()
	src := `package main
func f(): []String {
	xs := ["a", "b", "c", "d"]
	return xs[1:3]
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
}

func TestCheckTypes_forstSubslice_lowOnly(t *testing.T) {
	t.Parallel()
	src := `package main
func f(xs []String): []String {
	return xs[1:]
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
}

func TestCheckTypes_forstSubslice_highOnly(t *testing.T) {
	t.Parallel()
	src := `package main
func f(xs []String): []String {
	return xs[:2]
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
}

func TestCheckTypes_subslice_nonIntBound_errors(t *testing.T) {
	t.Parallel()
	src := `package main
func f(xs []String): []String {
	return xs[true:2]
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	if err == nil {
		t.Fatal("expected type error for non-Int low bound")
	}
	if !strings.Contains(err.Error(), "low bound must be Int") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckTypes_subslice_nonSliceTarget_errors(t *testing.T) {
	t.Parallel()
	src := `package main
func f(x Int): Int {
	return x[1:2]
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	if err == nil {
		t.Fatal("expected type error for non-slice target")
	}
}

func TestCheckTypes_goSpread_execCommand_typechecks(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func main() {
	argv := []String{"true", "extra"}
	exec.Command(argv[0], argv[1:]...)
}
`
	tc, err := typecheckGoSrc(t, dir, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["exec"] == nil {
		t.Skip("os/exec not loaded")
	}
}

func TestCheckTypes_goSpread_wrongElementType_errors(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func main() {
	exec.Command("true", [1, 2]...)
}
`
	err := typecheckGoSrcErr(t, dir, src)
	if err == nil {
		t.Fatal("expected spread type error")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag.Code != "go-call" {
		t.Fatalf("want go-call diagnostic, got %v", err)
	}
	if !strings.Contains(diag.Msg, "cannot spread") {
		t.Fatalf("want cannot spread in message, got %q", diag.Msg)
	}
}

func TestCheckTypes_goSpread_extraTrailingArgs_errors(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func main() {
	argv := []String{"a", "b"}
	exec.Command("true", "extra", argv[1:]...)
}
`
	err := typecheckGoSrcErr(t, dir, src)
	if err == nil {
		t.Fatal("expected extra trailing args error")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag.Code != "go-call" {
		t.Fatalf("want go-call diagnostic, got %v", err)
	}
	if !strings.Contains(diag.Msg, "only trailing argument") {
		t.Fatalf("want trailing spread message, got %q", diag.Msg)
	}
}

func TestCheckTypes_goSpread_nonSlice_errors(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func main() {
	exec.Command("true", "x"...)
}
`
	err := typecheckGoSrcErr(t, dir, src)
	if err == nil {
		t.Fatal("expected spread non-slice error")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag.Code != "go-call" {
		t.Fatalf("want go-call diagnostic, got %v", err)
	}
}

func TestCheckTypes_goFieldAccess_ProcessState_typechecks(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func f(argv []String): Bool {
	cmd := exec.Command(argv[0], argv[1:]...)
	err := cmd.Run()
	if err != nil {
		return cmd.ProcessState != nil
	}
	return false
}
`
	tc, err := typecheckGoSrc(t, dir, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["exec"] == nil {
		t.Skip("os/exec not loaded")
	}
}

func TestCheckTypes_goFieldAccess_missingField_errors(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func main() {
	_ = exec.Command("true").NoSuchField
}
`
	err := typecheckGoSrcErr(t, dir, src)
	if err == nil {
		t.Fatal("expected missing field error")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag.Code != "go-field" {
		t.Fatalf("want go-field diagnostic, got %v", err)
	}
}

func TestCheckTypes_goFieldAccess_onForstShape_errors(t *testing.T) {
	t.Parallel()
	src := `package main
func f(xs []String): String {
	return xs[0].name
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	if err == nil {
		t.Fatal("expected field-access error on non-Go expression")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag.Code != "field-access" {
		t.Fatalf("want field-access diagnostic, got %v", err)
	}
}

func TestCheckTypes_goDottedMethodOnFieldPath_ExitCode_typechecks(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func exitCode(argv []String): Int {
	cmd := exec.Command(argv[0], argv[1:]...)
	err := cmd.Run()
	if err != nil {
		if cmd.ProcessState != nil {
			return cmd.ProcessState.ExitCode()
		}
		return 2
	}
	return 0
}
`
	tc, err := typecheckGoSrc(t, dir, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["exec"] == nil {
		t.Skip("os/exec not loaded")
	}
}

func TestCheckTypes_goDottedMethodOnFieldPath_missingMethod_errors(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func main() {
	cmd := exec.Command("true")
	_ = cmd.ProcessState.NoSuchMethod()
}
`
	err := typecheckGoSrcErr(t, dir, src)
	if err == nil {
		t.Fatal("expected missing method error")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag.Code != "go-method" {
		t.Fatalf("want go-method diagnostic, got %v", err)
	}
}

func typecheckGoSrc(t *testing.T, dir, src string) (*TypeChecker, error) {
	t.Helper()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	return tc, tc.CheckTypes(nodes)
}

func typecheckGoSrcErr(t *testing.T, dir, src string) error {
	t.Helper()
	tc, err := typecheckGoSrc(t, dir, src)
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["exec"] == nil {
		t.Skip("os/exec not loaded")
	}
	return err
}
