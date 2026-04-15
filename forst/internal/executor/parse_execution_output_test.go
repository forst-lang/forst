package executor

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestFunctionExecutor_parseExecutionOutput_JSON_branches(t *testing.T) {
	e := &FunctionExecutor{log: logrus.New()}

	tests := []struct {
		name       string
		raw        string
		wantOut    string
		wantResult string
	}{
		{
			name:       "result_string",
			raw:        `{"result":"hi"}`,
			wantOut:    "hi",
			wantResult: `"hi"`,
		},
		{
			name:       "result_float",
			raw:        `{"result":3.5}`,
			wantOut:    "3.5",
			wantResult: "3.5",
		},
		{
			name:       "result_bool",
			raw:        `{"result":true}`,
			wantOut:    "true",
			wantResult: "true",
		},
		{
			name:       "result_object",
			raw:        `{"result":{"k":1}}`,
			wantOut:    `{"k":1}`,
			wantResult: `{"k":1}`,
		},
		{
			name:       "no_result_field",
			raw:        `{"other":1}`,
			wantOut:    `{"other":1}`,
			wantResult: "*", // any non-empty JSON echo of object
		},
		{
			name:       "non_json_raw",
			raw:        "plain output\n",
			wantOut:    "plain output\n",
			wantResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := e.parseExecutionOutput(tt.raw)
			if err != nil {
				t.Fatalf("parseExecutionOutput: %v", err)
			}
			if !res.Success {
				t.Fatal("expected Success")
			}
			if res.Output != tt.wantOut {
				t.Fatalf("Output = %q, want %q", res.Output, tt.wantOut)
			}
			switch tt.wantResult {
			case "":
				if len(res.Result) != 0 {
					t.Fatalf("expected empty Result, got %q", res.Result)
				}
			case "*":
				if len(res.Result) == 0 {
					t.Fatal("expected non-empty Result for full JSON echo")
				}
			default:
				if string(res.Result) != tt.wantResult {
					t.Fatalf("Result = %s, want %s", res.Result, tt.wantResult)
				}
			}
		})
	}
}

func TestFunctionExecutor_parseExecutionOutput_result_array_default_branch(t *testing.T) {
	e := &FunctionExecutor{log: logrus.New()}
	raw := `{"result":[1,2]}`
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(payload["result"])
	if err != nil {
		t.Fatal(err)
	}
	wantOut := string(wantJSON)

	res, err := e.parseExecutionOutput(raw)
	if err != nil {
		t.Fatalf("parseExecutionOutput: %v", err)
	}
	if !res.Success {
		t.Fatal("expected Success true")
	}
	if res.Output != wantOut {
		t.Fatalf("ExecutionResult.Output = %q, want %q", res.Output, wantOut)
	}
	if !bytes.Equal(res.Result, wantJSON) {
		t.Fatalf("ExecutionResult.Result = %s, want %s", res.Result, wantJSON)
	}
}

func TestExecutionResultFromValue_intBranch(t *testing.T) {
	res := executionResultFromValue(42)
	if !res.Success {
		t.Fatal("expected Success true")
	}
	if res.Output != "42" {
		t.Fatalf("Output = %q, want %q", res.Output, "42")
	}
	if string(res.Result) != "42" {
		t.Fatalf("Result = %q, want %q", res.Result, "42")
	}
}
