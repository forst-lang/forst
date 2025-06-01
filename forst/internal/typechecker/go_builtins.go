package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

// BuiltinFunction represents a built-in function with its type signature
type BuiltinFunction struct {
	Name           string
	Package        string // The package this function belongs to
	ReturnType     ast.TypeNode
	ParamTypes     []ast.TypeNode
	IsVarArgs      bool
	AcceptSubtypes bool // Whether to accept subtypes of parameter types
}

// BuiltinFunctions maps function names to their type signatures
var BuiltinFunctions = map[string]BuiltinFunction{
	// Built-in functions that don't require a package
	"len": {
		Name:           "len",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeInt},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		AcceptSubtypes: true,                                    // Accept subtypes like Password
	},
	"println": {
		Name:           "println",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"print": {
		Name:           "print",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"cap": {
		Name:           "cap",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeInt},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		AcceptSubtypes: true,
	},
	"append": {
		Name:           "append",
		Package:        "",                                      // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},     // Returns same type as input
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"make": {
		Name:           "make",
		Package:        "",                                      // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},     // Returns same type as input
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"new": {
		Name:           "new",
		Package:        "",                                      // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},     // Returns pointer to type
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		AcceptSubtypes: true,
	},
	"clear": {
		Name:           "clear",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // container to clear
		AcceptSubtypes: true,
	},
	"complex": {
		Name:           "complex",
		Package:        "",                                                               // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},                              // Returns complex number
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // real, imag parts
		AcceptSubtypes: true,
	},
	"real": {
		Name:           "real",
		Package:        "",                                      // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},     // Returns real part
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // complex number
		AcceptSubtypes: true,
	},
	"imag": {
		Name:           "imag",
		Package:        "",                                      // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},     // Returns imaginary part
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // complex number
		AcceptSubtypes: true,
	},
	"delete": {
		Name:           "delete",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // map, key
		AcceptSubtypes: true,
	},
	"close": {
		Name:           "close",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // channel
		AcceptSubtypes: true,
	},
	"min": {
		Name:           "min",
		Package:        "",                                      // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},     // Returns smallest value
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // values to compare
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"max": {
		Name:           "max",
		Package:        "",                                      // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},     // Returns largest value
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // values to compare
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"panic": {
		Name:           "panic",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // panic value
		AcceptSubtypes: true,
	},
	"recover": {
		Name:           "recover",
		Package:        "",                                  // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeString}, // Returns panic value
		AcceptSubtypes: true,
	},
	"copy": {
		Name:           "copy",
		Package:        "", // No package required
		ReturnType:     ast.TypeNode{Ident: ast.TypeInt},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // dst, src
		AcceptSubtypes: true,
	},

	// fmt package functions
	"fmt.Print": {
		Name:           "Print",
		Package:        "fmt",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"fmt.Println": {
		Name:           "Println",
		Package:        "fmt",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // Base type
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"fmt.Printf": {
		Name:           "Printf",
		Package:        "fmt",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // format string
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"fmt.Sprintf": {
		Name:           "Sprintf",
		Package:        "fmt",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // format string
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},

	// os package functions
	"os.Exit": {
		Name:           "Exit",
		Package:        "os",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeInt}},
		AcceptSubtypes: false,
	},
	"os.Getenv": {
		Name:           "Getenv",
		Package:        "os",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // key
		AcceptSubtypes: false,
	},
	"os.Setenv": {
		Name:           "Setenv",
		Package:        "os",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // key, value
		AcceptSubtypes: false,
	},

	// strings package functions
	"strings.Contains": {
		Name:           "Contains",
		Package:        "strings",
		ReturnType:     ast.TypeNode{Ident: ast.TypeBool},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // s, substr
		AcceptSubtypes: true,
	},
	"strings.HasPrefix": {
		Name:           "HasPrefix",
		Package:        "strings",
		ReturnType:     ast.TypeNode{Ident: ast.TypeBool},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // s, prefix
		AcceptSubtypes: true,
	},
	"strings.HasSuffix": {
		Name:           "HasSuffix",
		Package:        "strings",
		ReturnType:     ast.TypeNode{Ident: ast.TypeBool},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // s, suffix
		AcceptSubtypes: true,
	},
	"strings.Join": {
		Name:           "Join",
		Package:        "strings",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // elems, sep
		AcceptSubtypes: true,
	},
	"strings.Split": {
		Name:           "Split",
		Package:        "strings",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},                              // Returns array of strings
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // s, sep
		AcceptSubtypes: true,
	},
	"strings.Trim": {
		Name:           "Trim",
		Package:        "strings",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeString}}, // s, cutset
		AcceptSubtypes: true,
	},
	"strings.ToLower": {
		Name:           "ToLower",
		Package:        "strings",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // s
		AcceptSubtypes: true,
	},
	"strings.ToUpper": {
		Name:           "ToUpper",
		Package:        "strings",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // s
		AcceptSubtypes: true,
	},

	// strconv package functions
	"strconv.Atoi": {
		Name:           "Atoi",
		Package:        "strconv",
		ReturnType:     ast.TypeNode{Ident: ast.TypeInt},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // s
		AcceptSubtypes: true,
	},
	"strconv.Itoa": {
		Name:           "Itoa",
		Package:        "strconv",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeInt}}, // i
		AcceptSubtypes: false,
	},
	"strconv.ParseFloat": {
		Name:           "ParseFloat",
		Package:        "strconv",
		ReturnType:     ast.TypeNode{Ident: ast.TypeFloat},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // s
		AcceptSubtypes: true,
	},
	"strconv.FormatFloat": {
		Name:           "FormatFloat",
		Package:        "strconv",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeFloat}}, // f
		AcceptSubtypes: false,
	},
	"strconv.ParseBool": {
		Name:           "ParseBool",
		Package:        "strconv",
		ReturnType:     ast.TypeNode{Ident: ast.TypeBool},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // s
		AcceptSubtypes: true,
	},
	"strconv.FormatBool": {
		Name:           "FormatBool",
		Package:        "strconv",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeBool}}, // b
		AcceptSubtypes: false,
	},

	// errors package functions
	"errors.New": {
		Name:           "New",
		Package:        "errors",
		ReturnType:     ast.TypeNode{Ident: ast.TypeError},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // text
		AcceptSubtypes: false,
	},
	"errors.Unwrap": {
		Name:           "Unwrap",
		Package:        "errors",
		ReturnType:     ast.TypeNode{Ident: ast.TypeError},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeError}}, // err
		AcceptSubtypes: true,
	},
	"errors.Is": {
		Name:           "Is",
		Package:        "errors",
		ReturnType:     ast.TypeNode{Ident: ast.TypeBool},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeError}, {Ident: ast.TypeError}}, // err, target
		AcceptSubtypes: true,
	},
	"errors.As": {
		Name:           "As",
		Package:        "errors",
		ReturnType:     ast.TypeNode{Ident: ast.TypeBool},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeError}, {Ident: ast.TypeError}}, // err, target
		AcceptSubtypes: true,
	},
	"errors.Join": {
		Name:           "Join",
		Package:        "errors",
		ReturnType:     ast.TypeNode{Ident: ast.TypeError},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeError}}, // errs
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
}

// isTypeCompatible checks if a type is compatible with an expected type,
// taking into account subtypes and type guards
func (tc *TypeChecker) isTypeCompatible(actual ast.TypeNode, expected ast.TypeNode) bool {
	// Direct type match
	if actual.Ident == expected.Ident {
		return true
	}

	// Check if actual type is a subtype of expected type
	// This would involve checking type definitions and assertions
	// For now, we'll just check if the actual type is defined in our type system
	if _, exists := tc.Defs[actual.Ident]; exists {
		// TODO: Implement proper subtype checking
		// For now, we'll assume any defined type is compatible with its base type
		return true
	}

	return false
}

// checkBuiltinFunctionCall validates a call to a built-in function
func (tc *TypeChecker) checkBuiltinFunctionCall(fn BuiltinFunction, args []ast.ExpressionNode) ([]ast.TypeNode, error) {
	// Check argument count
	if !fn.IsVarArgs && len(args) != len(fn.ParamTypes) {
		return nil, fmt.Errorf("%s() expects %d arguments, got %d", fn.Name, len(fn.ParamTypes), len(args))
	}

	// Check argument types
	for i, arg := range args {
		argType, err := tc.inferExpressionType(arg)
		if err != nil {
			return nil, err
		}
		if len(argType) != 1 {
			return nil, fmt.Errorf("%s() argument %d must have a single type", fn.Name, i+1)
		}

		expectedType := fn.ParamTypes[0] // For varargs, all args must match the first param type
		if !fn.IsVarArgs {
			expectedType = fn.ParamTypes[i]
		}

		if !tc.isTypeCompatible(argType[0], expectedType) {
			return nil, fmt.Errorf("%s() argument %d must be of type %s, got %s",
				fn.Name, i+1, expectedType.Ident, argType[0].Ident)
		}
	}

	return []ast.TypeNode{fn.ReturnType}, nil
}
