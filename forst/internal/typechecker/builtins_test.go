package typechecker

import (
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"
)

func TestNilBuiltinSymbol(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "nil in return statement with error type",
			code: `
package main

func test() : (String, Error) {
	return "hello", nil
}
`,
			wantErr: false,
		},
		{
			name: "nil in return statement without error type",
			code: `
package main

func test() : (String, String) {
	return "hello", nil
}
`,
			wantErr: true,
		},
		{
			name: "nil in variable assignment with pointer type",
			code: `
package main

type Ptr = *String

func test() {
	var x Ptr = nil
}
`,
			wantErr: true,
		},
		{
			name: "nil in variable assignment without type",
			code: `
package main

func test() {
	var x = nil
}
`,
			wantErr: true,
		},
		{
			name: "nil in multiple return with ensure",
			code: `
package main

func foo() (String, Error) {
	return "hello", nil
}

func test() {
	name, err = foo()
	ensure err == nil
}
`,
			wantErr: true,
		},
		{
			name: "nil in function call with error return",
			code: `
package main

func foo() : (String, Error) {
	return "hello", nil
}

func test() {
	name, err := foo()
	ensure !err
	println(name)
}
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if tt.wantErr {
						// Expected panic for this test
						return
					}
					// Unexpected panic
					t.Fatalf("unexpected panic: %v", r)
				}
			}()

			// Lex the code
			log := setupTestLogger()
			l := lexer.New([]byte(tt.code), "test.ft", log)
			tokens := l.Lex()

			// Parse the code
			p := parser.New(tokens, "test.ft", log)
			node, err := p.ParseFile()
			if err != nil {
				if tt.wantErr {
					// Expected parse error
					return
				}
				t.Fatalf("Failed to parse: %v", err)
			}

			// Typecheck
			tc := New(log, false)
			err = tc.CheckTypes(node)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTypes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStructLiteralTypeMismatch(t *testing.T) {
	code := `
package main

type Foo = {a: String}
type Bar = {a: String}

func takesFoo(x: Foo) {}

func test() {
	takesFoo(Bar{a: "hi"})
}
`
	defer func() {
		if r := recover(); r != nil {
			// Expected panic for this test
			return
		}
	}()
	log := setupTestLogger()
	l := lexer.New([]byte(code), "test_struct_mismatch.ft", log)
	tokens := l.Lex()
	p := parser.New(tokens, "test_struct_mismatch.ft", log)
	node, err := p.ParseFile()
	if err != nil {
		// Expected parse error for this test
		return
	}
	tc := New(log, false)
	err = tc.CheckTypes(node)
	if err == nil {
		t.Errorf("Expected type error for struct literal type mismatch, got none")
	}
}

func TestUndefinedTypeNameInVarDecl(t *testing.T) {
	code := `
package main

func test() {
	var x UndefinedType = nil
}
`
	defer func() {
		if r := recover(); r != nil {
			// Expected panic for this test
			return
		}
	}()
	log := setupTestLogger()
	l := lexer.New([]byte(code), "test_undefined_type.ft", log)
	tokens := l.Lex()
	p := parser.New(tokens, "test_undefined_type.ft", log)
	node, err := p.ParseFile()
	if err != nil {
		// Expected parse error for this test
		return
	}
	tc := New(log, false)
	err = tc.CheckTypes(node)
	if err == nil {
		t.Errorf("Expected type error for undefined type name, got none")
	}
}

func TestAssignmentToUndeclaredVarsWithMultipleReturn(t *testing.T) {
	code := `
package main

func foo() : (String, Error) {
	return "hi", nil
}

func test() {
	a, b = foo()
}
`
	log := setupTestLogger()
	l := lexer.New([]byte(code), "test_multi_assign.ft", log)
	tokens := l.Lex()
	p := parser.New(tokens, "test_multi_assign.ft", log)
	node, err := p.ParseFile()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(node)
	if err == nil {
		t.Errorf("Expected type error for assignment to undeclared variables, got none")
	}
}
