package executor

import (
	"strings"
	"testing"

	"forst/internal/discovery"
)

func TestRewriteUserGoCodeForExecutorLibrary_table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		code        string
		forstPkg    string
		wantContain string
	}{
		{
			name:        "non_main_package_unchanged",
			code:        "package demo\n\nfunc Hello() {}\n",
			forstPkg:    "demo",
			wantContain: "package demo",
		},
		{
			name:        "main_single_line_rewritten",
			code:        "package main",
			forstPkg:    "main",
			wantContain: "package forstexec",
		},
		{
			name:        "main_multi_line_rewritten",
			code:        "package main\n\nfunc Hello() {}\n",
			forstPkg:    "main",
			wantContain: "package forstexec\n\nfunc Hello()",
		},
		{
			name:        "main_forst_package_but_non_main_go_code_unchanged",
			code:        "package demo\n\nfunc Hello() {}\n",
			forstPkg:    "main",
			wantContain: "package demo",
		},
		{
			name:        "first_line_not_exact_package_main",
			code:        " package main\n\nfunc Hello() {}\n",
			forstPkg:    "main",
			wantContain: "package forstexec",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := rewriteUserGoCodeForExecutorLibrary(tt.code, tt.forstPkg)
			if !strings.Contains(got, tt.wantContain) {
				t.Fatalf("rewrite output missing expected content %q:\n%s", tt.wantContain, got)
			}
		})
	}
}

func TestBuildParameterExtraction_table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		paramNames  []string
		paramTypes  []string
		wantSnippet []string
	}{
		{
			name:       "empty_parameter_set",
			paramNames: nil,
			paramTypes: nil,
			wantSnippet: []string{
				"make([]interface{}, 0)",
				"json.NewDecoder(os.Stdin).Decode",
			},
		},
		{
			name:       "builtin_int_with_float64_conversion",
			paramNames: []string{"age"},
			paramTypes: []string{"int"},
			wantSnippet: []string{
				"if v, ok := inputJSON[0].(float64); ok",
				"age = int(v)",
				"expected int",
			},
		},
		{
			name:       "builtin_string_assertion",
			paramNames: []string{"name"},
			paramTypes: []string{"string"},
			wantSnippet: []string{
				"if v, ok := inputJSON[0].(string); ok",
				"name = v",
				"expected string",
			},
		},
		{
			name:       "struct_parameter_unmarshal",
			paramNames: []string{"req"},
			paramTypes: []string{"MyRequest"},
			wantSnippet: []string{
				"var req MyRequest",
				"json.Unmarshal(paramBytes, &req)",
				"Error unmarshaling parameter",
			},
		},
		{
			name:       "extra_names_without_types_are_ignored",
			paramNames: []string{"first", "second"},
			paramTypes: []string{"string"},
			wantSnippet: []string{
				"var first string",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildParameterExtraction("input", tt.paramNames, tt.paramTypes)
			for _, want := range tt.wantSnippet {
				if !strings.Contains(got, want) {
					t.Fatalf("buildParameterExtraction output missing %q:\n%s", want, got)
				}
			}
			if tt.name == "extra_names_without_types_are_ignored" && strings.Contains(got, "var second") {
				t.Fatalf("did not expect extraction code for untyped parameter name:\n%s", got)
			}
		})
	}
}

func TestGoModuleManager_CreateModule_failureFromInvalidPackageName(t *testing.T) {
	t.Parallel()
	manager := NewGoModuleManager(nil)

	_, err := manager.CreateModule(&ModuleConfig{
		ModuleName:     "module-fail",
		PackageName:    "bad/pkg",
		FunctionName:   "Run",
		GoCode:         "package bad\nfunc Run() {}\n",
		SupportsParams: true,
		Parameters: []discovery.ParameterInfo{
			{Name: "n", Type: "int"},
		},
		Args:        []byte(`[1]`),
		IsStreaming: false,
	})
	if err == nil {
		t.Fatal("expected CreateModule error for invalid package name path")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("unexpected error type: %v", err)
	}
}

func TestGoModuleManager_generateStandardMainGo_multipleReturnBranches(t *testing.T) {
	t.Parallel()
	manager := NewGoModuleManager(nil)

	t.Run("multiple_returns_no_params", func(t *testing.T) {
		content := manager.generateStandardMainGo("mod/demo", "demo_alias", &ModuleConfig{
			PackageName:        "demo",
			FunctionName:       "Run",
			SupportsParams:     false,
			HasMultipleReturns: true,
		})
		if !strings.Contains(content, "result, err :=") {
			t.Fatalf("expected multiple-return assignment in generated main.go:\n%s", content)
		}
		if !strings.Contains(content, "Function execution failed") {
			t.Fatalf("expected error handling for multiple returns:\n%s", content)
		}
	})

	t.Run("multiple_returns_with_params", func(t *testing.T) {
		content := manager.generateStandardMainGo("mod/demo", "demo_alias", &ModuleConfig{
			PackageName:    "demo",
			FunctionName:   "Run",
			SupportsParams: true,
			Parameters: []discovery.ParameterInfo{
				{Name: "name", Type: "string"},
			},
			HasMultipleReturns: true,
		})
		if !strings.Contains(content, "result, err :=") {
			t.Fatalf("expected multiple-return assignment in generated main.go:\n%s", content)
		}
		if !strings.Contains(content, "Run(name)") {
			t.Fatalf("expected parameterized function call in generated main.go:\n%s", content)
		}
	})
}

func TestGoModuleManager_logTempDirContents_missingDir(t *testing.T) {
	t.Parallel()
	manager := NewGoModuleManager(nil)
	// Missing directory should be handled without panic.
	manager.logTempDirContents("/path/that/does/not/exist")
}
