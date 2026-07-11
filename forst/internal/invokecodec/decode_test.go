package invokecodec

import (
	"strings"
	"testing"

	"forst/internal/discovery"
)

func TestDecodeArgsFromJSON_emptyParams(t *testing.T) {
	t.Helper()
	if got := DecodeArgsFromJSON("args", nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestDecodeArgsFromJSON_builtinTypes(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		params   []discovery.ParameterInfo
		contains []string
	}{
		{
			name: "int accepts float64 from JSON",
			params: []discovery.ParameterInfo{
				{Name: "count", Type: "int"},
			},
			contains: []string{
				"var args []interface{}",
				"var count int",
				"args[0].(float64)",
				"args[0].(int)",
			},
		},
		{
			name: "string type assertion",
			params: []discovery.ParameterInfo{
				{Name: "message", Type: "string"},
			},
			contains: []string{
				"var message string",
				"args[0].(string)",
			},
		},
		{
			name: "multiple builtin params",
			params: []discovery.ParameterInfo{
				{Name: "name", Type: "string"},
				{Name: "active", Type: "bool"},
			},
			contains: []string{
				"args[0].(string)",
				"args[1].(bool)",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DecodeArgsFromJSON("args", tc.params)
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("DecodeArgsFromJSON missing %q\n--- got ---\n%s", want, got)
				}
			}
		})
	}
}

func TestDecodeArgsFromJSON_customTypeUsesMarshalRoundTrip(t *testing.T) {
	t.Helper()
	got := DecodeArgsFromJSON("forstInvokeArgs", []discovery.ParameterInfo{
		{Name: "payload", Type: "MyShape"},
	})
	for _, want := range []string{
		"var payload MyShape",
		"json.Marshal(forstInvokeArgs[0])",
		"json.Unmarshal(paramBytes, &payload)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("DecodeArgsFromJSON missing %q\n--- got ---\n%s", want, got)
		}
	}
}

func TestCallExpression(t *testing.T) {
	t.Helper()

	tests := []struct {
		name string
		fn   discovery.FunctionInfo
		args []string
		want string
	}{
		{
			name: "no params no multiple returns",
			fn:   discovery.FunctionInfo{Name: "Ping"},
			args: nil,
			want: "return Ping(), nil",
		},
		{
			name: "with params",
			fn:   discovery.FunctionInfo{Name: "Echo"},
			args: []string{"message"},
			want: "return Echo(message), nil",
		},
		{
			name: "multiple returns",
			fn:   discovery.FunctionInfo{Name: "Load", HasMultipleReturns: true},
			args: []string{"path"},
			want: "result, err := Load(path)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CallExpression(tc.fn, tc.args)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("CallExpression missing %q\n--- got ---\n%s", tc.want, got)
			}
		})
	}
}
