package hoverdoc

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// Builtin type names as written in Forst (capitalized keywords and related spellings).
const (
	NameInt     = "Int"
	NameFloat   = "Float"
	NameString  = "String"
	NameBool    = "Bool"
	NameVoid    = "Void"
	NameArray   = "Array"
	NameMap     = "Map"
	NameStruct  = "struct"
	NameError   = "Error"
	NamePointer = "Pointer"
	NameShape   = "Shape"
	NameResult  = "Result"
	NameTuple   = "Tuple"
	NameObject  = "Object"
)

// BuiltinTypeMarkdown returns user-facing markdown for a built-in Forst type, keyed by its
// surface spelling (e.g. "Int", "Result"). Unknown names yield an empty string.
func BuiltinTypeMarkdown(displayName string) string {
	return builtinTypeDocs[displayName]
}

// MarkdownForKeywordToken returns markdown for lexer keyword tokens that correspond to built-in
// types or language keywords documented for hovers. Unknown tokens yield "".
func MarkdownForKeywordToken(t ast.TokenIdent) string {
	switch t {
	case ast.TokenInt:
		return BuiltinTypeMarkdown(NameInt)
	case ast.TokenFloat:
		return BuiltinTypeMarkdown(NameFloat)
	case ast.TokenString:
		return BuiltinTypeMarkdown(NameString)
	case ast.TokenBool:
		return BuiltinTypeMarkdown(NameBool)
	case ast.TokenVoid:
		return BuiltinTypeMarkdown(NameVoid)
	case ast.TokenArray:
		return BuiltinTypeMarkdown(NameArray)
	case ast.TokenMap:
		return BuiltinTypeMarkdown(NameMap)
	case ast.TokenStruct:
		return BuiltinTypeMarkdown(NameStruct)
	case ast.TokenFunc:
		return keywordDoc("func", "Declares a function. Parameters and return types use `name: Type` syntax.")
	case ast.TokenType:
		return keywordDoc("type", "Introduces a type alias, shape type, or type guard.")
	case ast.TokenReturn:
		return keywordDoc("return", "Returns a value from the current function. The expression must match the function's return type.")
	case ast.TokenImport:
		return keywordDoc("import", "Imports a Go package (`import \"path\"`). Qualified names use the local identifier from the import.")
	case ast.TokenPackage:
		return keywordDoc("package", "Sets the Go package clause for the file (e.g. `package main`).")
	case ast.TokenEnsure:
		return keywordDoc("ensure", "Asserts a condition at this point in control flow. If the check fails, the program exits via the failure path (e.g. `log.Fatal`). Combine with `is` and built-in or user-defined guards to narrow types for following statements.")
	case ast.TokenIs:
		return keywordDoc("is", "Used in `if subject is …` and `ensure … is …` to apply type guards, assertions, and built-in constraints (e.g. `Min`, `Ok`).")
	case ast.TokenOr:
		return keywordDoc("or", "Logical OR (`||`). Short-circuits: the right side is evaluated only if the left is false.")
	default:
		return ""
	}
}

func keywordDoc(title, body string) string {
	var b strings.Builder
	b.WriteString("**`")
	b.WriteString(title)
	b.WriteString("`**\n\n")
	b.WriteString(body)
	return b.String()
}

func typeDoc(title, body string) string {
	return fmt.Sprintf("**`%s`** (built-in type)\n\n%s", title, body)
}

// builtinTypeDocs holds LSP hover text for built-in Forst types. Keys match Forst spellings.
var builtinTypeDocs = map[string]string{
	NameInt: typeDoc(NameInt,
		"Signed integer. In generated Go code this maps to `int`."),
	NameFloat: typeDoc(NameFloat,
		"IEEE floating-point number. In generated Go code this maps to `float64`."),
	NameString: typeDoc(NameString,
		"UTF-8 string. In generated Go code this maps to `string`. Supports guards such as `Min`, `Max`, `HasPrefix`, and `Contains` in assertions."),
	NameBool: typeDoc(NameBool,
		"Boolean. In generated Go code this maps to `bool`. Assertions may use `True` or `False` guards."),
	NameVoid: typeDoc(NameVoid,
		"Return type for functions that do not return a value. In generated Go code this omits a result or uses `()` as appropriate."),
	NameArray: typeDoc(NameArray,
		"Fixed or slice-like sequence `Array(T)`. In generated Go code this maps to a Go slice `[]T`. Length-related guards use `len` in the same way as for strings."),
	NameMap: typeDoc(NameMap,
		"Associative map `Map(K, V)`. In generated Go code this maps to `map[K]V`."),
	NameStruct: typeDoc(NameStruct,
		"Anonymous struct type in Go interop positions. Prefer `type Name = { ... }` for named shapes in Forst."),
	NameError: typeDoc(NameError,
		"The error interface. In generated Go code this maps to Go's `error`. Methods such as `Error()` resolve to Go's `error.Error` when hovering qualified calls."),
	NamePointer: typeDoc(NamePointer,
		"Pointer type `Pointer(T)` (`*T` in Go). Assertions may use `Nil` or `Present` to describe nullability."),
	NameShape: typeDoc(NameShape,
		"Structural object type with fields. Often written as `type Name = { field: Type, ... }`. Structural types may receive hash-based names in generated Go."),
	NameResult: typeDoc(NameResult,
		"Discriminated result `Result(OkType, ErrType)` with error-kinded failure. Use `if x is Ok()` / `Err()` (or `ensure` variants) to narrow the success or failure side."),
	NameTuple: typeDoc(NameTuple,
		"Product type for multiple values (FFI / multi-value interop). Maps to a struct or multiple returns depending on context."),
	NameObject: typeDoc(NameObject,
		"Loose object / dynamic interop placeholder. Prefer explicit shapes or maps for static typing."),
}
