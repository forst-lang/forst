package typechecker

import "forst/internal/ast"

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
		Name:       "Join",
		Package:    "strings",
		ReturnType: ast.TypeNode{Ident: ast.TypeString},
		ParamTypes: []ast.TypeNode{
			{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}},
			{Ident: ast.TypeString},
		}, // elems []string, sep string
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
