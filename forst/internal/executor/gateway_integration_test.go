package executor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"forst/internal/compiler"

	"github.com/sirupsen/logrus"
)

func jsonObjectsEqual(a, b json.RawMessage) bool {
	var ma, mb any
	if err := json.Unmarshal(a, &ma); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &mb); err != nil {
		return false
	}
	return reflect.DeepEqual(ma, mb)
}

// E2E-style: compile and execute gateway example handlers (examples/in/gateway).
func TestGatewayExample_invokeAnswerHello(t *testing.T) {
	rootDir := filepath.Join("..", "..", "..", "examples", "in", "gateway")
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	comp := compiler.New(compiler.Args{
		Command:            "run",
		FilePath:           filepath.Join(absRoot, "hello.ft"),
		PackageRoot:        absRoot,
		ExportStructFields: true,
	}, log)
	ex := NewFunctionExecutor(absRoot, comp, log, walkForstFilesConfig{})

	args := json.RawMessage(`[{"method":"GET","url":"/","path":"/","query":{},"headers":{},"bodyBase64":""}]`)
	res, err := ex.ExecuteFunction("gwexample", "AnswerHello", args)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	var payload struct {
		Kind   string `json:"kind"`
		Status int    `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(res.Result, &payload); err != nil {
		t.Fatalf("unmarshal: %v raw=%s", err, res.Result)
	}
	if payload.Kind != "answer" || payload.Status != 200 || payload.Body != "hello" {
		t.Fatalf("unexpected result: %+v", payload)
	}

	golden, err := os.ReadFile(filepath.Join(absRoot, "testdata", "golden_answer.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !jsonObjectsEqual(res.Result, golden) {
		t.Fatalf("result JSON differs from golden_answer.json\ngot:  %s\nwant: %s", res.Result, golden)
	}
}

func TestGatewayExample_invokePassThrough(t *testing.T) {
	rootDir := filepath.Join("..", "..", "..", "examples", "in", "gateway")
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	comp := compiler.New(compiler.Args{
		Command:            "run",
		FilePath:           filepath.Join(absRoot, "hello.ft"),
		PackageRoot:        absRoot,
		ExportStructFields: true,
	}, log)
	ex := NewFunctionExecutor(absRoot, comp, log, walkForstFilesConfig{})

	args := json.RawMessage(`[{"method":"GET","url":"/","path":"/","query":{},"headers":{},"bodyBase64":""}]`)
	res, err := ex.ExecuteFunction("gwexample", "PassThrough", args)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	var payload struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(res.Result, &payload); err != nil {
		t.Fatalf("unmarshal: %v raw=%s", err, res.Result)
	}
	if payload.Kind != "pass" {
		t.Fatalf("unexpected kind %q, raw=%s", payload.Kind, res.Result)
	}

	golden, err := os.ReadFile(filepath.Join(absRoot, "testdata", "golden_pass.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !jsonObjectsEqual(res.Result, golden) {
		t.Fatalf("result JSON differs from golden_pass.json\ngot:  %s\nwant: %s", res.Result, golden)
	}
}

