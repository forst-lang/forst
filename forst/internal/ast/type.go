package ast

import (
	"fmt"
	"strings"
)

// TypeIdent is a unique identifier for a type
type TypeIdent string

const (

	// TypeInt is the built-in int type
	TypeInt TypeIdent = "TYPE_INT"
	// TypeFloat is the built-in float type
	TypeFloat TypeIdent = "TYPE_FLOAT"
	// TypeString is the built-in string type
	TypeString TypeIdent = "TYPE_STRING"
	// TypeBool is the built-in bool type
	TypeBool TypeIdent = "TYPE_BOOL"
	// TypeVoid is the built-in void type
	TypeVoid TypeIdent = "TYPE_VOID"
	// TypeError is the built-in error type
	TypeError TypeIdent = "TYPE_ERROR"
	// TypeObject is the built-in object type
	TypeObject TypeIdent = "TYPE_OBJECT"
	// TypeArray is the built-in array type
	TypeArray TypeIdent = "TYPE_ARRAY"
	// TypeMap is the built-in map type
	TypeMap TypeIdent = "TYPE_MAP"

	// TypeAssertion is a placeholder for a type assertion
	TypeAssertion TypeIdent = "TYPE_ASSERTION"

	// TypeImplicit is a placeholder for an implicit type
	TypeImplicit TypeIdent = "TYPE_IMPLICIT"

	// TypeShape is a new type added
	TypeShape TypeIdent = "TYPE_SHAPE"

	// TypePointer is a new type added
	TypePointer TypeIdent = "TYPE_POINTER"

	// TypeResult is Result(Success, Failure) with Failure error-kinded
	TypeResult TypeIdent = "TYPE_RESULT"
	// TypeTuple is Tuple(T1, ..., Tn) for product multi-values (e.g. Go FFI)
	TypeTuple TypeIdent = "TYPE_TUPLE"

	// TypeUnion is a finite type-level union (A | B | …) used by typedef and join; members are in TypeParams.
	TypeUnion TypeIdent = "TYPE_UNION"
	// TypeIntersection is a finite type-level intersection (A & B & …); members are in TypeParams.
	TypeIntersection TypeIdent = "TYPE_INTERSECTION"
)

// TypeKind represents the origin/kind of a type
// Used for reliable type emission and reasoning
//
//	Builtin: Go/Forst built-in types (string, int, etc.)
//	UserDefined: Named types defined by the user (AppContext, etc.)
//	HashBased: Structural/anonymous types (T_xxx...)
type TypeKind string

const (
	TypeKindBuiltin     TypeKind = "TYPE_KIND_BUILTIN"
	TypeKindUserDefined TypeKind = "TYPE_KIND_USER_DEFINED"
	TypeKindHashBased   TypeKind = "TYPE_KIND_HASH_BASED"
)

// TypeNode represents a type in the Forst language
// Kind must be set at construction time and never guessed from Ident
type TypeNode struct {
	Node
	Ident      TypeIdent
	Assertion  *AssertionNode
	TypeParams []TypeNode // Generic type parameters
	TypeKind   TypeKind
}

// IsExplicit returns true if the type has been specified explicitly
func (t TypeNode) IsExplicit() bool {
	return t.Ident != TypeImplicit
}

// IsImplicit returns true if the type has not been specified explicitly
func (t TypeNode) IsImplicit() bool {
	return t.Ident == TypeImplicit
}

// Kind returns the node kind for a type
func (t TypeNode) Kind() NodeKind {
	return NodeKindType
}

// IsError returns true if the type is an error type
func (t TypeNode) IsError() bool {
	return t.Ident == TypeError
}

func (t TypeNode) String() string {
	switch t.Ident {
	case TypeInt:
		return t.Ident.String()
	case TypeFloat:
		return t.Ident.String()
	case TypeString:
		return t.Ident.String()
	case TypeBool:
		return t.Ident.String()
	case TypeVoid:
		return t.Ident.String()
	case TypeError:
		return t.Ident.String()
	case TypeObject:
		return t.Ident.String()
	case TypeArray:
		if len(t.TypeParams) > 0 {
			return fmt.Sprintf("Array(%s)", t.TypeParams[0].String())
		}
		return "Array(?)"
	case TypeMap:
		if len(t.TypeParams) >= 2 {
			return fmt.Sprintf("Map(%s, %s)", t.TypeParams[0].String(), t.TypeParams[1].String())
		}
		return "Map(?, ?)"
	case TypeAssertion:
		if t.Assertion != nil {
			return fmt.Sprintf("Assertion(%s)", t.Assertion.String())
		}
		return "Assertion(?)"
	case TypeImplicit:
		return "(implicit)"
	case TypeShape:
		if len(t.TypeParams) > 0 {
			return fmt.Sprintf("Shape(%s)", t.TypeParams[0].String())
		}
		return "Shape"
	case TypePointer:
		if len(t.TypeParams) > 0 {
			return fmt.Sprintf("Pointer(%s)", t.TypeParams[0].String())
		}
		return "Pointer"
	case TypeResult:
		if len(t.TypeParams) >= 2 {
			return fmt.Sprintf("Result(%s, %s)", t.TypeParams[0].String(), t.TypeParams[1].String())
		}
		return "Result(?, ?)"
	case TypeTuple:
		if len(t.TypeParams) > 0 {
			params := make([]string, len(t.TypeParams))
			for i, param := range t.TypeParams {
				params[i] = param.String()
			}
			return fmt.Sprintf("Tuple(%s)", strings.Join(params, ", "))
		}
		return "Tuple()"
	case TypeUnion:
		if len(t.TypeParams) > 0 {
			parts := make([]string, len(t.TypeParams))
			for i := range t.TypeParams {
				parts[i] = t.TypeParams[i].String()
			}
			return strings.Join(parts, " | ")
		}
		return "Union()"
	case TypeIntersection:
		if len(t.TypeParams) > 0 {
			parts := make([]string, len(t.TypeParams))
			for i := range t.TypeParams {
				parts[i] = t.TypeParams[i].String()
			}
			return strings.Join(parts, " & ")
		}
		return "Intersection()"
	default:
		if t.Assertion != nil {
			return fmt.Sprintf("%s(%s)", t.Ident, t.Assertion.String())
		}
		if len(t.TypeParams) > 0 {
			params := make([]string, len(t.TypeParams))
			for i, param := range t.TypeParams {
				params[i] = param.String()
			}
			return fmt.Sprintf("%s(%s)", t.Ident, strings.Join(params, ", "))
		}
		return string(t.Ident)
	}
}

// String returns a string representation of the type ident
func (ti TypeIdent) String() string {
	switch ti {
	case TypeInt:
		return "Int"
	case TypeFloat:
		return "Float"
	case TypeString:
		return "String"
	case TypeBool:
		return "Bool"
	case TypeVoid:
		return "Void"
	case TypeError:
		return "Error"
	case TypeObject:
		return "Object(?)"
	case TypeArray:
		return "Array(?)"
	case TypeMap:
		return "Map(?, ?)"
	case TypeAssertion:
		return "Assertion(?)"
	case TypeImplicit:
		return "(implicit)"
	case TypeShape:
		return "Shape(?)"
	case TypePointer:
		return "Pointer"
	case TypeResult:
		return "Result"
	case TypeTuple:
		return "Tuple"
	case TypeUnion:
		return "Union"
	case TypeIntersection:
		return "Intersection"
	default:
		return string(ti)
	}
}

// IsGoBuiltin returns true if the type node is a Go builtin type
func (t *TypeNode) IsGoBuiltin() bool {
	return t.TypeKind == TypeKindBuiltin
}

// IsHashBased returns true if the type node is a hash-based/structural type
func (t *TypeNode) IsHashBased() bool {
	return t.TypeKind == TypeKindHashBased
}

// IsUserDefined returns true if the type node is a user-defined named type
func (t *TypeNode) IsUserDefined() bool {
	return t.TypeKind == TypeKindUserDefined
}

// NewBuiltinType creates a new TypeNode for a built-in type
func NewBuiltinType(ident TypeIdent) TypeNode {
	return TypeNode{
		Ident:    ident,
		TypeKind: TypeKindBuiltin,
	}
}

// NewUserDefinedType creates a new TypeNode for a user-defined type
func NewUserDefinedType(ident TypeIdent) TypeNode {
	return TypeNode{
		Ident:    ident,
		TypeKind: TypeKindUserDefined,
	}
}

// NewHashBasedType creates a new TypeNode for a hash-based type
func NewHashBasedType(ident TypeIdent) TypeNode {
	return TypeNode{
		Ident:    ident,
		TypeKind: TypeKindHashBased,
	}
}

// NewPointerType creates a new TypeNode for a pointer type
func NewPointerType(baseType TypeNode) TypeNode {
	return TypeNode{
		Ident:      TypePointer,
		TypeParams: []TypeNode{baseType},
		TypeKind:   TypeKindBuiltin, // Pointer is a built-in type construct
	}
}

// NewArrayType creates a new TypeNode for an array type
func NewArrayType(elementType TypeNode) TypeNode {
	return TypeNode{
		Ident:      TypeArray,
		TypeParams: []TypeNode{elementType},
		TypeKind:   TypeKindBuiltin, // Array is a built-in type construct
	}
}

// NewMapType creates a new TypeNode for a map type
func NewMapType(keyType, valueType TypeNode) TypeNode {
	return TypeNode{
		Ident:      TypeMap,
		TypeParams: []TypeNode{keyType, valueType},
		TypeKind:   TypeKindBuiltin, // Map is a built-in type construct
	}
}

// NewAssertionType creates a new TypeNode for an assertion type
func NewAssertionType(assertion *AssertionNode) TypeNode {
	return TypeNode{
		Ident:     TypeAssertion,
		Assertion: assertion,
		TypeKind:  TypeKindHashBased, // Assertions create structural types
	}
}

// NewResultType returns Result(successType, failureType).
func NewResultType(successType, failureType TypeNode) TypeNode {
	return TypeNode{
		Ident:      TypeResult,
		TypeParams: []TypeNode{successType, failureType},
		TypeKind:   TypeKindBuiltin,
	}
}

// NewTupleType returns Tuple(t1, ..., tn); n must be at least 1.
func NewTupleType(elems ...TypeNode) TypeNode {
	return TypeNode{
		Ident:      TypeTuple,
		TypeParams: elems,
		TypeKind:   TypeKindBuiltin,
	}
}

// NewUnionType returns a normalized finite union (flattened nested unions, deduped by shallow equality).
func NewUnionType(members ...TypeNode) TypeNode {
	flat := flattenUnionMembers(members)
	if len(flat) == 1 {
		return flat[0]
	}
	return TypeNode{
		Ident:      TypeUnion,
		TypeParams: flat,
		TypeKind:   TypeKindBuiltin,
	}
}

// NewIntersectionType returns a normalized finite intersection (flattened nested intersections, deduped).
func NewIntersectionType(members ...TypeNode) TypeNode {
	flat := flattenIntersectionMembers(members)
	if len(flat) == 1 {
		return flat[0]
	}
	return TypeNode{
		Ident:      TypeIntersection,
		TypeParams: flat,
		TypeKind:   TypeKindBuiltin,
	}
}

func flattenUnionMembers(in []TypeNode) []TypeNode {
	var out []TypeNode
	for _, t := range in {
		if t.Ident == TypeUnion && len(t.TypeParams) > 0 {
			out = append(out, flattenUnionMembers(t.TypeParams)...)
			continue
		}
		out = append(out, t)
	}
	return dedupeTypeNodesShallow(out)
}

func flattenIntersectionMembers(in []TypeNode) []TypeNode {
	var out []TypeNode
	for _, t := range in {
		if t.Ident == TypeIntersection && len(t.TypeParams) > 0 {
			out = append(out, flattenIntersectionMembers(t.TypeParams)...)
			continue
		}
		out = append(out, t)
	}
	return dedupeTypeNodesShallow(out)
}

func dedupeTypeNodesShallow(types []TypeNode) []TypeNode {
	if len(types) <= 1 {
		if len(types) == 1 {
			return []TypeNode{types[0]}
		}
		return nil
	}
	var out []TypeNode
	for _, t := range types {
		found := false
		for _, u := range out {
			if typeNodesShallowEqualAST(t, u) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, t)
		}
	}
	return out
}

func typeNodesShallowEqualAST(a, b TypeNode) bool {
	if a.Ident != b.Ident {
		return false
	}
	if len(a.TypeParams) != len(b.TypeParams) {
		return false
	}
	for i := range a.TypeParams {
		if !typeNodesShallowEqualAST(a.TypeParams[i], b.TypeParams[i]) {
			return false
		}
	}
	return true
}
