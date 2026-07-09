package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestDiscoverer_DiscoverFunctions_Integration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "discovery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	testFile := filepath.Join(tempDir, "test.ft")
	testContent := `package testpkg

func PublicFunction(input String): String {
	return input
}

func privateFunction() {
	// private function
}`
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := &MockConfig{files: []string{testFile}}
	ml := &MockLogger{}
	discoverer := NewDiscoverer(tempDir, ml, config)

	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions failed: %v", err)
	}
	var sawDiscoveredSummary, sawSymbolLine bool
	for _, m := range ml.debugMsgs {
		if strings.Contains(m, "Discovered") {
			sawDiscoveredSummary = true
		}
		if m == "- %s.%s" {
			sawSymbolLine = true
		}
	}
	if !sawDiscoveredSummary || !sawSymbolLine {
		t.Fatalf("expected summary + per-symbol Debugf patterns, debugMsgs=%#v", ml.debugMsgs)
	}
	pkgFuncs, exists := functions["testpkg"]
	if !exists {
		t.Fatal("Package 'testpkg' not found")
	}
	if len(pkgFuncs) != 1 {
		t.Errorf("Expected 1 public function, got %d", len(pkgFuncs))
	}

	pubFunc, exists := pkgFuncs["PublicFunction"]
	if !exists {
		t.Error("PublicFunction not found")
	} else {
		if pubFunc.Name != "PublicFunction" || pubFunc.Package != "testpkg" {
			t.Fatalf("unexpected public function metadata: %+v", pubFunc)
		}
		if len(pubFunc.Parameters) != 1 || pubFunc.Parameters[0].Name != "input" || pubFunc.Parameters[0].Type != "string" {
			t.Fatalf("unexpected parameters: %+v", pubFunc.Parameters)
		}
		if pubFunc.ReturnType != "string" {
			t.Errorf("Expected return type 'string', got '%s'", pubFunc.ReturnType)
		}
		if pubFunc.SupportsStreaming {
			t.Error("PublicFunction should not support streaming")
		}
	}
}

func TestDiscoverer_DiscoverFunctions_NoFiles(t *testing.T) {
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, &MockConfig{files: []string{}})
	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions failed: %v", err)
	}
	if len(functions) != 0 {
		t.Errorf("Expected 0 packages, got %d", len(functions))
	}
}

func TestDiscoverer_DiscoverFunctions_ConfigError(t *testing.T) {
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, &MockConfig{err: fmt.Errorf("config error")})
	_, err := discoverer.DiscoverFunctions()
	if err == nil {
		t.Error("Expected error when config fails")
	}
}

func TestDiscoverer_DiscoverFunctions_readErrorSkipped(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.ft")
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{missing}})
	out, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no functions when file missing, got %+v", out)
	}
}

func TestDiscoverer_DiscoverFunctions_parseErrorSkipped(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ft")
	if err := os.WriteFile(path, []byte("this is not valid forst syntax @@@\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{path}})
	out, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no functions when parse fails, got %+v", out)
	}
}

func TestDiscoverer_DiscoverFunctions_typecheckErrorStillDiscovers(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	dir := t.TempDir()
	path := filepath.Join(dir, "semantics.ft")
	content := `package main

func Bad(): String {
	return 1
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{path}})
	out, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	fn, ok := out["main"]["Bad"]
	if !ok {
		t.Fatalf("expected Bad in main, got %+v", out)
	}
	if fn.ReturnType == "" {
		t.Fatal("expected return type from parser fallback when typecheck fails")
	}
}

func TestDiscoverer_DiscoverFunctions_crossFileCall(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "lib.ft")
	apiPath := filepath.Join(dir, "api.ft")
	if err := os.WriteFile(libPath, []byte(`package demo

func Helper(): String {
	return "hi"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apiPath, []byte(`package demo

func Hello(): String {
	return Helper()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{apiPath, libPath}})
	out, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	fn, ok := out["demo"]["Hello"]
	if !ok {
		t.Fatalf("expected Hello, got %+v", out)
	}
	if fn.ReturnType == "" {
		t.Fatalf("expected return type with merged-package typecheck, got %+v", fn)
	}
}

func TestDiscoverer_DiscoverFunctions_skipsBadFileContinues(t *testing.T) {
	dir := t.TempDir()
	goodPath := filepath.Join(dir, "good.ft")
	badPath := filepath.Join(dir, "bad.ft")
	if err := os.WriteFile(goodPath, []byte(`package main

func Ok(): String {
	return "x"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badPath, []byte(`@@@`), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{badPath, goodPath}})
	out, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	if _, ok := out["main"]["Ok"]; !ok {
		t.Fatalf("expected Ok from good.ft, got %+v", out)
	}
}

func TestDiscoverer_DiscoverFunctions_ResultReturn_setsIsResultAndComponentTypes(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	dir := t.TempDir()
	path := filepath.Join(dir, "result.ft")
	content := `package main

func R(): Result(Int, Error) {
	return 0
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{path}})
	out, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	fn, ok := out["main"]["R"]
	if !ok {
		t.Fatalf("expected R, got %+v", out)
	}
	if !fn.IsResult {
		t.Fatalf("expected IsResult, got %+v", fn)
	}
	if fn.ResultSuccessType == "" || fn.ResultFailureType == "" {
		t.Fatalf("expected result component types, got success=%q failure=%q", fn.ResultSuccessType, fn.ResultFailureType)
	}
	if !fn.HasMultipleReturns {
		t.Fatalf("expected HasMultipleReturns for Result (lowers to success + error in Go), got %+v", fn)
	}
	raw, err := json.Marshal(fn)
	if err != nil {
		t.Fatal(err)
	}
	var marshaled map[string]any
	if err := json.Unmarshal(raw, &marshaled); err != nil {
		t.Fatal(err)
	}
	if marshaled["isResult"] != true {
		t.Fatalf("JSON isResult: got %v", marshaled["isResult"])
	}
	if _, ok := marshaled["resultSuccessType"]; !ok {
		t.Fatalf("JSON missing resultSuccessType: %v", marshaled)
	}
	if _, ok := marshaled["resultFailureType"]; !ok {
		t.Fatalf("JSON missing resultFailureType: %v", marshaled)
	}
}

func TestDiscoverer_DiscoverFunctions_publicFunctionNoExplicitReturns(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	dir := t.TempDir()
	path := filepath.Join(dir, "void.ft")
	content := `package main

func Pub() {
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{path}})
	out, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	fn, ok := out["main"]["Pub"]
	if !ok {
		t.Fatalf("expected Pub, got %+v", out)
	}
	if fn.HasMultipleReturns || len(fn.ReturnTypes) != 0 {
		t.Fatalf("expected no return types, got %+v", fn)
	}
}

func TestDiscoverer_DiscoverFunctions_groupsFunctionsByPackage(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.ft")
	ordersPath := filepath.Join(dir, "orders.ft")
	if err := os.WriteFile(catalogPath, []byte(`package catalog

func ListItems(): String {
	return "a"
}

func privateList() {
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ordersPath, []byte(`package orders

func PlaceOrder(): String {
	return "b"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{catalogPath, ordersPath}})
	functionsByPackage, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	catalogFuncs, catalogOK := functionsByPackage["catalog"]
	if !catalogOK {
		t.Fatalf("expected catalog package in discovery output, got %+v", functionsByPackage)
	}
	if _, ok := catalogFuncs["ListItems"]; !ok {
		t.Fatalf("expected ListItems in catalog package, got %+v", catalogFuncs)
	}
	if _, ok := catalogFuncs["privateList"]; ok {
		t.Fatalf("did not expect privateList in catalog package, got %+v", catalogFuncs)
	}

	ordersFuncs, ordersOK := functionsByPackage["orders"]
	if !ordersOK {
		t.Fatalf("expected orders package in discovery output, got %+v", functionsByPackage)
	}
	if _, ok := ordersFuncs["PlaceOrder"]; !ok {
		t.Fatalf("expected PlaceOrder in orders package, got %+v", ordersFuncs)
	}
	if _, leaked := ordersFuncs["ListItems"]; leaked {
		t.Fatalf("unexpected cross-package leakage: orders has ListItems: %+v", ordersFuncs)
	}
}

func TestDiscoverer_DiscoverFunctions_groupsPackagesEvenWhenOneFileIsUnparseable(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.ft")
	ordersPath := filepath.Join(dir, "orders.ft")
	badPath := filepath.Join(dir, "broken.ft")
	if err := os.WriteFile(catalogPath, []byte(`package catalog

func ListItems(): String {
	return "a"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ordersPath, []byte(`package orders

func PlaceOrder(): String {
	return "b"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badPath, []byte(`@@@ invalid syntax @@@`), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	discoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{badPath, catalogPath, ordersPath}})
	functionsByPackage, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	if _, ok := functionsByPackage["catalog"]["ListItems"]; !ok {
		t.Fatalf("expected catalog.ListItems despite unparseable sibling file, got %+v", functionsByPackage)
	}
	if _, ok := functionsByPackage["orders"]["PlaceOrder"]; !ok {
		t.Fatalf("expected orders.PlaceOrder despite unparseable sibling file, got %+v", functionsByPackage)
	}
}

func TestDiscoverer_DiscoverFunctions_shuffledFileOrderKeepsSameFunctionSet(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.ft")
	ordersPath := filepath.Join(dir, "orders.ft")
	if err := os.WriteFile(catalogPath, []byte(`package catalog

func ListItems(): String {
	return "a"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ordersPath, []byte(`package orders

func PlaceOrder(): String {
	return "b"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	firstDiscoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{catalogPath, ordersPath}})
	firstOut, err := firstDiscoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions first order: %v", err)
	}

	secondDiscoverer := NewDiscoverer(dir, logger, &MockConfig{files: []string{ordersPath, catalogPath}})
	secondOut, err := secondDiscoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions second order: %v", err)
	}

	if _, ok := firstOut["catalog"]["ListItems"]; !ok {
		t.Fatalf("first order missing catalog.ListItems: %+v", firstOut)
	}
	if _, ok := firstOut["orders"]["PlaceOrder"]; !ok {
		t.Fatalf("first order missing orders.PlaceOrder: %+v", firstOut)
	}
	if _, ok := secondOut["catalog"]["ListItems"]; !ok {
		t.Fatalf("second order missing catalog.ListItems: %+v", secondOut)
	}
	if _, ok := secondOut["orders"]["PlaceOrder"]; !ok {
		t.Fatalf("second order missing orders.PlaceOrder: %+v", secondOut)
	}
}

func TestDiscoverer_DiscoverFunctions_typecheckFailure_logsDebug(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad_typecheck.ft")
	content := `package main

func Bad(): String {
	return 1
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ml := &MockLogger{}
	discoverer := NewDiscoverer(dir, ml, &MockConfig{files: []string{path}})
	_, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	found := false
	for _, m := range ml.debugMsgs {
		if strings.Contains(m, "Type checking failed") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected typecheck failure debug, got debugMsgs=%#v", ml.debugMsgs)
	}
}

func TestDiscoverer_DiscoverFunctions_onlyPrivateFunctions_skipsFileWithNoPublic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "priv.ft")
	content := `package main

func privateOnly(): String {
	return "x"
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	discoverer := NewDiscoverer(dir, &MockLogger{}, &MockConfig{files: []string{path}})
	out, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no public functions, got %+v", out)
	}
}
