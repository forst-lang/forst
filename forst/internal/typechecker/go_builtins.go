package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// BuiltinCheckKind says how checkBuiltinFunctionCall validates a builtin.
type BuiltinCheckKind uint8

const (
	// BuiltinCheckGeneric uses ParamTypes, arity, and IsTypeCompatible (stdlib entries, string() conversion).
	BuiltinCheckGeneric BuiltinCheckKind = iota
	// BuiltinCheckDispatch: rules live in tryDispatchGoBuiltin + helpers; ParamTypes are placeholders
	// for arity/LSP only — never used with IsTypeCompatible for these entries.
	BuiltinCheckDispatch
)

// BuiltinFunction represents a built-in function with its type signature
type BuiltinFunction struct {
	Name           string
	Package        string // The package this function belongs to
	ReturnType     ast.TypeNode
	ParamTypes     []ast.TypeNode
	IsVarArgs      bool
	AcceptSubtypes bool // Whether to accept subtypes of parameter types
	CheckKind      BuiltinCheckKind
	// HoverSignature is optional markdown for hover when CheckKind is BuiltinCheckDispatch (otherwise ParamTypes are shown).
	HoverSignature string
}

// BuiltinFunctions maps function names to their type signatures
var BuiltinFunctions = map[string]BuiltinFunction{
	// Predeclared Go builtins (no package): type-checking is implemented in tryDispatchGoBuiltin, not ParamTypes.
	"len": {
		Name:           "len",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeInt},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
		HoverSignature: "predeclared `len` via tryDispatchGoBuiltin (operand: string, slice, map, …)",
	},
	"println": {
		Name:           "println",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}},
		IsVarArgs:      true,
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	// Go predeclared string(): conversion (e.g. code point to UTF-8 string)
	"string": {
		Name:           "string",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeString},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeInt}},
		AcceptSubtypes: true,
	},
	"print": {
		Name:           "print",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}},
		IsVarArgs:      true,
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"cap": {
		Name:           "cap",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeInt},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"append": {
		Name:           "append",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeObject},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		IsVarArgs:      true,
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"make": {
		Name:           "make",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeObject},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		IsVarArgs:      true,
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"new": {
		Name:           "new",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeObject},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"clear": {
		Name:           "clear",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"complex": {
		Name:           "complex",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeObject},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeFloat}, {Ident: ast.TypeFloat}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"real": {
		Name:           "real",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeFloat},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"imag": {
		Name:           "imag",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeFloat},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"delete": {
		Name:           "delete",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}, {Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"close": {
		Name:           "close",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"min": {
		Name:           "min",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeObject},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		IsVarArgs:      true,
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"max": {
		Name:           "max",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeObject},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		IsVarArgs:      true,
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"panic": {
		Name:           "panic",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"recover": {
		Name:           "recover",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeObject},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},
	"copy": {
		Name:           "copy",
		Package:        "",
		ReturnType:     ast.TypeNode{Ident: ast.TypeInt},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}, {Ident: ast.TypeObject}},
		AcceptSubtypes: true,
		CheckKind:      BuiltinCheckDispatch,
	},

	// fmt package functions
	"fmt.Print": {
		Name:           "Print",
		Package:        "fmt",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}}, // Go ...any
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"fmt.Println": {
		Name:           "Println",
		Package:        "fmt",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeObject}}, // Go ...any
		IsVarArgs:      true,
		AcceptSubtypes: true,
	},
	"fmt.Printf": {
		Name:           "Printf",
		Package:        "fmt",
		ReturnType:     ast.TypeNode{Ident: ast.TypeVoid},
		ParamTypes:     []ast.TypeNode{{Ident: ast.TypeString}}, // format string; further args checked as Object
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

// IsTypeCompatible checks if a type is compatible with an expected type,
// taking into account subtypes and type guards
func (tc *TypeChecker) IsTypeCompatible(actual ast.TypeNode, expected ast.TypeNode) bool {
	tc.log.WithFields(logrus.Fields{
		"actual":   actual.Ident,
		"expected": expected.Ident,
		"function": "IsTypeCompatible",
	}).Debug("Checking type compatibility")

	// Same identifier: simple types match; generic built-ins must compare type parameters.
	if actual.Ident == expected.Ident {
		switch actual.Ident {
		case ast.TypeResult:
			if len(actual.TypeParams) != 2 || len(expected.TypeParams) != 2 {
				return false
			}
			return tc.IsTypeCompatible(actual.TypeParams[0], expected.TypeParams[0]) &&
				tc.IsTypeCompatible(actual.TypeParams[1], expected.TypeParams[1])
		case ast.TypeTuple:
			if len(actual.TypeParams) != len(expected.TypeParams) {
				return false
			}
			for i := range actual.TypeParams {
				if !tc.IsTypeCompatible(actual.TypeParams[i], expected.TypeParams[i]) {
					return false
				}
			}
			return true
		default:
			tc.log.WithFields(logrus.Fields{
				"actual":   actual.Ident,
				"expected": expected.Ident,
				"function": "IsTypeCompatible",
			}).Debug("Direct type match")
			return true
		}
	}

	// Assigning to TypeObject mirrors Go assignability to interface{} / any (empty interface).
	if expected.Ident == ast.TypeObject && actual.Ident != ast.TypeVoid {
		tc.log.WithFields(logrus.Fields{
			"actual":   actual.Ident,
			"expected": expected.Ident,
			"function": "IsTypeCompatible",
		}).Debug("Actual type assignable to TypeObject")
		return true
	}

	// Value compatible with *T when compatible with T for non-scalar T (shape literals for *User,
	// etc.). Scalars still require an explicit pointer so String does not satisfy *String.
	if expected.Ident == ast.TypePointer && len(expected.TypeParams) == 1 {
		inner := expected.TypeParams[0]
		if !isScalarTypeIdent(inner.Ident) && tc.IsTypeCompatible(actual, inner) {
			tc.log.WithFields(logrus.Fields{
				"actual":   actual.Ident,
				"expected": expected.Ident,
				"function": "IsTypeCompatible",
			}).Debug("Actual type compatible with pointer element type")
			return true
		}
	}

	// Check if actual type is an alias of expected type
	actualDef, actualExists := tc.Defs[actual.Ident]
	if actualExists {
		if typeDef, ok := actualDef.(ast.TypeDefNode); ok {
			if typeDefExpr, ok := typeDefAssertionFromExpr(typeDef.Expr); ok {
				if typeDefExpr.Assertion != nil && typeDefExpr.Assertion.BaseType != nil {
					baseType := ast.TypeNode{Ident: *typeDefExpr.Assertion.BaseType}
					if tc.IsTypeCompatible(baseType, expected) {
						tc.log.WithFields(logrus.Fields{
							"actual":   actual.Ident,
							"expected": expected.Ident,
							"function": "IsTypeCompatible",
						}).Debug("Actual type is alias of expected type")
						return true
					}
				}
			}
		}
	}

	// Check if expected type is an alias of actual type
	expectedDef, expectedExists := tc.Defs[expected.Ident]
	if expectedExists {
		if typeDef, ok := expectedDef.(ast.TypeDefNode); ok {
			if typeDefExpr, ok := typeDefAssertionFromExpr(typeDef.Expr); ok {
				if typeDefExpr.Assertion != nil && typeDefExpr.Assertion.BaseType != nil {
					baseType := ast.TypeNode{Ident: *typeDefExpr.Assertion.BaseType}
					if tc.IsTypeCompatible(actual, baseType) {
						tc.log.WithFields(logrus.Fields{
							"actual":   actual.Ident,
							"expected": expected.Ident,
							"function": "IsTypeCompatible",
						}).Debug("Expected type is alias of actual type")
						return true
					}
				}
			}
		}
	}

	// Check for structural compatibility between hash-based types and user-defined types
	if actualDef != nil && expectedDef != nil {
		tc.log.WithFields(logrus.Fields{
			"actual":   actual.Ident,
			"expected": expected.Ident,
			"function": "IsTypeCompatible",
		}).Info("Checking structural compatibility")

		actualShape, actualShapeOk := tc.getShapeFromTypeDef(actualDef)
		expectedShape, expectedShapeOk := tc.getShapeFromTypeDef(expectedDef)

		tc.log.WithFields(logrus.Fields{
			"actual":          actual.Ident,
			"expected":        expected.Ident,
			"actualShapeOk":   actualShapeOk,
			"expectedShapeOk": expectedShapeOk,
			"function":        "IsTypeCompatible",
		}).Info("Shape extraction results")

		if actualShapeOk && expectedShapeOk {
			// Prefer shapesHaveSameStructure: it resolves assertion fields and uses assignability for
			// mismatched type identifiers (e.g. inferred literal ctx vs AppContext).
			identical := tc.shapesHaveSameStructure(*actualShape, *expectedShape)
			tc.log.WithFields(logrus.Fields{
				"actual":    actual.Ident,
				"expected":  expected.Ident,
				"identical": identical,
				"function":  "IsTypeCompatible",
			}).Info("Structural compatibility check result")

			if identical {
				tc.log.WithFields(logrus.Fields{
					"actual":   actual.Ident,
					"expected": expected.Ident,
					"function": "IsTypeCompatible",
				}).Info("Shapes are structurally identical")
				return true
			}
		} else {
			tc.log.WithFields(logrus.Fields{
				"actual":          actual.Ident,
				"expected":        expected.Ident,
				"actualShapeOk":   actualShapeOk,
				"expectedShapeOk": expectedShapeOk,
				"function":        "IsTypeCompatible",
			}).Info("Could not extract shapes for structural comparison")
		}
	} else {
		tc.log.WithFields(logrus.Fields{
			"actual":      actual.Ident,
			"expected":    expected.Ident,
			"actualDef":   actualDef != nil,
			"expectedDef": expectedDef != nil,
			"function":    "IsTypeCompatible",
		}).Info("Skipping structural compatibility - missing type definitions")
	}

	tc.log.WithFields(logrus.Fields{
		"actual":   actual.Ident,
		"expected": expected.Ident,
		"function": "IsTypeCompatible",
	}).Debug("Types are not compatible")
	return false
}

func isScalarTypeIdent(id ast.TypeIdent) bool {
	switch id {
	case ast.TypeString, ast.TypeInt, ast.TypeFloat, ast.TypeBool:
		return true
	default:
		return false
	}
}

// getShapeFromTypeDef extracts the shape from a TypeDefNode if it's a shape definition
func (tc *TypeChecker) getShapeFromTypeDef(def ast.Node) (*ast.ShapeNode, bool) {
	if typeDef, ok := def.(ast.TypeDefNode); ok {
		if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
			return &shapeExpr.Shape, true
		}
	}
	return nil, false
}

// shapesAreStructurallyIdentical returns true if two ShapeNodes have the same fields and types
func (tc *TypeChecker) shapesAreStructurallyIdentical(a, b ast.ShapeNode) bool {
	if len(a.Fields) != len(b.Fields) {
		return false
	}
	for name, fieldA := range a.Fields {
		fieldB, ok := b.Fields[name]
		if !ok {
			return false
		}
		// Compare field types (ignoring assertions for now)
		if fieldA.Type != nil && fieldB.Type != nil {
			// If either type is unknown (?), treat them as compatible
			if fieldA.Type.Ident == "?" || fieldB.Type.Ident == "?" {
				// Unknown types are compatible with any concrete type
				continue
			}
			if fieldA.Type.Ident == fieldB.Type.Ident {
				continue
			}
			// Hash-based structural types vs named shapes (e.g. inferred literal vs AppContext) must use
			// full assignability, not identifier equality only.
			if tc.IsTypeCompatible(*fieldA.Type, *fieldB.Type) {
				continue
			}
			return false
		} else if fieldA.Shape != nil && fieldB.Shape != nil {
			if !tc.shapesAreStructurallyIdentical(*fieldA.Shape, *fieldB.Shape) {
				return false
			}
		} else if (fieldA.Type != nil) != (fieldB.Type != nil) || (fieldA.Shape != nil) != (fieldB.Shape != nil) {
			return false
		}
	}
	return true
}

// checkBuiltinFunctionCall validates a call to a built-in function.
// argSpans and callSpan come from the parser when available; pass nil / zero value otherwise.
func (tc *TypeChecker) checkBuiltinFunctionCall(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":   "checkBuiltinFunctionCall",
		"calledFn":   fn.Name,
		"argsLength": len(args),
	}).Debugf("Validating builtin function call")

	if fn.Name == "string" && len(args) == 1 {
		argType, err := tc.inferExpressionType(args[0])
		if err != nil {
			return nil, err
		}
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		if len(argType) != 1 {
			return nil, diagnosticf(sp, "builtin-call", "string() expects one argument")
		}
		switch argType[0].Ident {
		case ast.TypeInt:
			return []ast.TypeNode{fn.ReturnType}, nil
		default:
			return nil, diagnosticf(sp, "builtin-call", "string() unsupported operand type %s", argType[0].Ident)
		}
	}

	// Go predeclared builtins (empty package): semantics are implemented in tryDispatchGoBuiltin, not ParamTypes.
	if fn.Package == "" && fn.Name != "string" {
		ret, ok, err := tc.tryDispatchGoBuiltin(fn, args, argSpans, callSpan)
		if ok {
			return ret, err
		}
		if err != nil {
			return nil, err
		}
		if fn.CheckKind == BuiltinCheckDispatch {
			return nil, diagnosticf(callSpan, "builtin-call", "internal: missing tryDispatchGoBuiltin case for %q", fn.Name)
		}
	}

	// Check argument count
	if !fn.IsVarArgs && len(args) != len(fn.ParamTypes) {
		tc.log.WithFields(logrus.Fields{
			"function": "checkBuiltinFunctionCall",
		}).Errorf("%s() expects %d arguments, got %d", fn.Name, len(fn.ParamTypes), len(args))
		var sp ast.SourceSpan
		if len(args) > len(fn.ParamTypes) {
			sp = spanForCallArg(argSpans, len(fn.ParamTypes), args, callSpan)
		} else {
			sp = callSpan
		}
		if !sp.IsSet() && len(args) > 0 {
			sp = spanForCallArg(argSpans, 0, args, callSpan)
		}
		return nil, diagnosticf(sp, "builtin-call", "%s() expects %d arguments, got %d", fn.Name, len(fn.ParamTypes), len(args))
	}

	// Check argument types
	for i, arg := range args {
		argType, err := tc.inferExpressionType(arg)
		if err != nil {
			tc.log.WithFields(logrus.Fields{
				"function": "checkBuiltinFunctionCall",
			}).Errorf("Error inferring type for argument %d: %v", i+1, err)
			return nil, err
		}
		sp := spanForCallArg(argSpans, i, args, callSpan)
		tc.log.WithFields(logrus.Fields{
			"function": "checkBuiltinFunctionCall",
		}).Debugf("Arg %d inferred type: %+v", i+1, argType)
		if len(argType) != 1 {
			tc.log.WithFields(logrus.Fields{
				"function": "checkBuiltinFunctionCall",
			}).Errorf("%s() argument %d must have a single type, got %d", fn.Name, i+1, len(argType))
			return nil, diagnosticf(sp, "builtin-call", "%s() argument %d must have a single type", fn.Name, i+1)
		}

		expectedType := fn.ParamTypes[0]
		if !fn.IsVarArgs {
			expectedType = fn.ParamTypes[i]
		} else if fn.Package == "fmt" {
			switch fn.Name {
			case "Printf":
				if i == 0 {
					expectedType = fn.ParamTypes[0] // format: String
				} else {
					expectedType = ast.TypeNode{Ident: ast.TypeObject}
				}
			case "Print", "Println":
				expectedType = ast.TypeNode{Ident: ast.TypeObject}
			default:
				expectedType = fn.ParamTypes[0]
			}
		} else {
			// For other varargs, all args must match the first param type.
			expectedType = fn.ParamTypes[0]
		}
		tc.log.WithFields(logrus.Fields{
			"function": "checkBuiltinFunctionCall",
		}).Debugf("Arg %d expected type: %+v", i+1, expectedType)

		if !tc.IsTypeCompatible(argType[0], expectedType) {
			tc.log.WithFields(logrus.Fields{
				"function": "checkBuiltinFunctionCall",
			}).Errorf("%s() argument %d must be of type %s, got %s", fn.Name, i+1, expectedType.Ident, argType[0].Ident)
			return nil, diagnosticf(sp, "builtin-call", "%s() argument %d must be of type %s, got %s",
				fn.Name, i+1, expectedType.Ident, argType[0].Ident)
		}
	}

	return []ast.TypeNode{fn.ReturnType}, nil
}

func (tc *TypeChecker) inferBuiltinArgType(args []ast.ExpressionNode, i int, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) (ast.TypeNode, error) {
	if i < 0 || i >= len(args) {
		return ast.TypeNode{}, diagnosticf(callSpan, "builtin-call", "internal: missing argument %d", i+1)
	}
	sp := spanForCallArg(argSpans, i, args, callSpan)
	ts, err := tc.inferExpressionType(args[i])
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(ts) != 1 {
		return ast.TypeNode{}, diagnosticf(sp, "builtin-call", "argument %d must have a single type", i+1)
	}
	return ts[0], nil
}

// tryDispatchGoBuiltin applies Go-aligned rules for predeclared builtins (Package "").
// If it returns handled=false, the caller falls back to generic ParamTypes checking.
func (tc *TypeChecker) tryDispatchGoBuiltin(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	intT := ast.NewBuiltinType(ast.TypeInt)
	voidT := ast.NewBuiltinType(ast.TypeVoid)
	objT := ast.NewBuiltinType(ast.TypeObject)
	floatT := ast.NewBuiltinType(ast.TypeFloat)

	switch fn.Name {
	case "len":
		if len(args) != 1 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "len() expects 1 argument, got %d", len(args))
		}
		t, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if !lenOperandAllowed(t) {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "len() invalid operand type %s", t.Ident)
		}
		return []ast.TypeNode{intT}, true, nil

	case "cap":
		if len(args) != 1 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "cap() expects 1 argument, got %d", len(args))
		}
		t, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if !capOperandAllowed(t) {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "cap() invalid operand type %s", t.Ident)
		}
		return []ast.TypeNode{intT}, true, nil

	case "append":
		if len(args) < 2 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "append() expects at least 2 arguments, got %d", len(args))
		}
		sliceT, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if sliceT.Ident != ast.TypeArray {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "append() first argument must be a slice, got %s", sliceT.Ident)
		}
		elem, ok := sliceElementType(sliceT)
		if !ok {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "append() slice must have an element type")
		}
		for i := 1; i < len(args); i++ {
			at, err := tc.inferBuiltinArgType(args, i, argSpans, callSpan)
			if err != nil {
				return nil, true, err
			}
			if !tc.IsTypeCompatible(at, elem) {
				sp := spanForCallArg(argSpans, i, args, callSpan)
				return nil, true, diagnosticf(sp, "builtin-call", "append() argument %d must be assignable to slice element %s, got %s",
					i+1, elem.Ident, at.Ident)
			}
		}
		return []ast.TypeNode{sliceT}, true, nil

	case "copy":
		if len(args) != 2 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "copy() expects 2 arguments, got %d", len(args))
		}
		dstT, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		srcT, err := tc.inferBuiltinArgType(args, 1, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if dstT.Ident == ast.TypeArray && len(dstT.TypeParams) == 1 && dstT.TypeParams[0].Ident == ast.TypeInt && srcT.Ident == ast.TypeString {
			return []ast.TypeNode{intT}, true, nil
		}
		if dstT.Ident != ast.TypeArray || srcT.Ident != ast.TypeArray {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "copy() expects two slices, or []Int with String (Go copy to []byte)")
		}
		de, ok1 := sliceElementType(dstT)
		se, ok2 := sliceElementType(srcT)
		if !ok1 || !ok2 {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "copy() could not read slice element types")
		}
		if !builtinTypesIdenticalOrdered(de, se) {
			sp := spanForCallArg(argSpans, 1, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "copy() slice element types must match")
		}
		return []ast.TypeNode{intT}, true, nil

	case "delete":
		if len(args) != 2 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "delete() expects 2 arguments, got %d", len(args))
		}
		mT, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		keyT, err := tc.inferBuiltinArgType(args, 1, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		kFormal, _, ok := mapKeyValueTypes(mT)
		if !ok {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "delete() first argument must be a map, got %s", mT.Ident)
		}
		if !tc.IsTypeCompatible(keyT, kFormal) {
			sp := spanForCallArg(argSpans, 1, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "delete() key type incompatible with map key %s", kFormal.Ident)
		}
		return []ast.TypeNode{voidT}, true, nil

	case "close":
		if len(args) != 1 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "close() expects 1 argument, got %d", len(args))
		}
		if _, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan); err != nil {
			return nil, true, err
		}
		return []ast.TypeNode{voidT}, true, nil

	case "clear":
		if len(args) != 1 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "clear() expects 1 argument, got %d", len(args))
		}
		t, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if !clearOperandAllowed(t) {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "clear() expects a map or slice, got %s", t.Ident)
		}
		return []ast.TypeNode{voidT}, true, nil

	case "min", "max":
		if len(args) < 1 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "%s() expects at least 1 argument", fn.Name)
		}
		first, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if !isOrderedBuiltinType(first) {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "%s() expects ordered types (Int, Float, String), got %s", fn.Name, first.Ident)
		}
		for i := 1; i < len(args); i++ {
			at, err := tc.inferBuiltinArgType(args, i, argSpans, callSpan)
			if err != nil {
				return nil, true, err
			}
			if !builtinTypesIdenticalOrdered(first, at) {
				sp := spanForCallArg(argSpans, i, args, callSpan)
				return nil, true, diagnosticf(sp, "builtin-call", "%s() arguments must have the same type", fn.Name)
			}
		}
		return []ast.TypeNode{first}, true, nil

	case "complex":
		if len(args) != 2 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "complex() expects 2 arguments, got %d", len(args))
		}
		a, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		b, err := tc.inferBuiltinArgType(args, 1, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if a.Ident != ast.TypeFloat || b.Ident != ast.TypeFloat {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "complex() expects Float arguments, got %s and %s", a.Ident, b.Ident)
		}
		return []ast.TypeNode{objT}, true, nil

	case "real", "imag":
		if len(args) != 1 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "%s() expects 1 argument, got %d", fn.Name, len(args))
		}
		t, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		// Forst has no complex128 TypeIdent; complex() is typed as Object.
		if t.Ident != ast.TypeObject {
			sp := spanForCallArg(argSpans, 0, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "%s() expects a complex value, got %s", fn.Name, t.Ident)
		}
		return []ast.TypeNode{floatT}, true, nil

	case "panic":
		if len(args) != 1 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "panic() expects 1 argument, got %d", len(args))
		}
		if _, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan); err != nil {
			return nil, true, err
		}
		return []ast.TypeNode{voidT}, true, nil

	case "print", "println":
		for i := range args {
			if _, err := tc.inferBuiltinArgType(args, i, argSpans, callSpan); err != nil {
				return nil, true, err
			}
		}
		return []ast.TypeNode{voidT}, true, nil

	case "recover":
		if len(args) != 0 {
			return nil, true, diagnosticf(callSpan, "builtin-call", "recover() expects 0 arguments, got %d", len(args))
		}
		return []ast.TypeNode{objT}, true, nil

	case "make", "new":
		return nil, true, diagnosticf(callSpan, "builtin-call",
			"%s() with a type argument is not supported yet: use Forst type syntax (e.g. Array(Int)) when the parser accepts types in call position, or use a small Go helper via import", fn.Name)

	default:
		return nil, false, nil
	}
}
