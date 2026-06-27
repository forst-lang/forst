// Hash/user alias resolution and stable display names for types (transformer / codegen consumers).
package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"strings"

	"github.com/sirupsen/logrus"
)

// resolveAliasedType resolves a type to its aliased name if it's a hash-based type that matches a user-defined type
func (tc *TypeChecker) resolveAliasedType(typeNode ast.TypeNode) ast.TypeNode {
	// Inference often leaves TypeKind unset even for structural hashes (T_…); treat T_-prefixed
	// idents as hash-like so display and downstream consumers can resolve aliases.
	isHashLike := typeNode.TypeKind == ast.TypeKindHashBased || strings.HasPrefix(string(typeNode.Ident), "T_")
	if !isHashLike {
		return typeNode
	}
	hashDef, hashExists := tc.Defs[typeNode.Ident]
	if hashExists {
		if hashTypeDef, ok := hashDef.(ast.TypeDefNode); ok {
			if hashShapeExpr, ok := hashTypeDef.Expr.(ast.TypeDefShapeExpr); ok {
				hashShape := hashShapeExpr.Shape
				for _, def := range tc.Defs {
					if userDef, ok := def.(ast.TypeDefNode); ok && userDef.Ident != "" {
						// RegisterHashBasedType stores the hash under T_… with a TypeDefShapeExpr; do not
						// treat that entry (or any other hash typedef) as a display alias for itself.
						if userDef.Ident == typeNode.Ident || strings.HasPrefix(string(userDef.Ident), "T_") {
							continue
						}
						if userPayload, ok := ast.PayloadShape(userDef.Expr); ok {
							if tc.shapesAreStructurallyIdentical(hashShape, *userPayload) {
								tc.log.WithFields(logrus.Fields{
									"hashType":    typeNode.Ident,
									"aliasedType": userDef.Ident,
									"function":    "resolveAliasedType",
								}).Debug("Resolved hash-based type to aliased type")
								return ast.TypeNode{
									Ident:      userDef.Ident,
									TypeKind:   ast.TypeKindUserDefined, // Mark as user-defined
									Assertion:  typeNode.Assertion,
									TypeParams: typeNode.TypeParams,
								}
							}
						}
					}
				}
			}
		}
	}
	// type Name = <assertion>: narrowing `x is Name` uses InferAssertionType's structural hash (T_…).
	// That hash matches HashNode(AssertionNode{BaseType: Name}), not the typedef's inner assertion.
	if strings.HasPrefix(string(typeNode.Ident), "T_") {
		for _, def := range tc.Defs {
			userDef, ok := def.(ast.TypeDefNode)
			if !ok || userDef.Ident == "" || strings.HasPrefix(string(userDef.Ident), "T_") {
				continue
			}
			if _, ok := typeDefAssertionFromExpr(userDef.Expr); !ok {
				continue
			}
			bt := userDef.Ident
			a := ast.AssertionNode{BaseType: &bt}
			h, err := tc.Hasher.HashNode(a)
			if err != nil {
				continue
			}
			if h.ToTypeIdent() == typeNode.Ident {
				tc.log.WithFields(logrus.Fields{
					"hashType":    typeNode.Ident,
					"aliasedType": userDef.Ident,
					"function":    "resolveAliasedType",
				}).Debug("Resolved assertion refinement hash to type alias")
				return ast.TypeNode{
					Ident:      userDef.Ident,
					TypeKind:   ast.TypeKindUserDefined,
					Assertion:  typeNode.Assertion,
					TypeParams: typeNode.TypeParams,
				}
			}
		}
	}
	return typeNode
}

// GetAliasedTypeNameOptions configures GetAliasedTypeName (e.g. structural alias for codegen).
type GetAliasedTypeNameOptions struct {
	AllowStructuralAlias bool
}

// GetAliasedTypeName returns the aliased type name for any type node, ensuring consistent type aliasing.
// This is the unified function that should be used by both typechecker and transformer.
func (tc *TypeChecker) GetAliasedTypeName(typeNode ast.TypeNode, opts GetAliasedTypeNameOptions) (string, error) {
	// Handle built-in types
	if IsGoBuiltinType(string(typeNode.Ident)) || typeNode.Ident == ast.TypeString || typeNode.Ident == ast.TypeInt || typeNode.Ident == ast.TypeFloat || typeNode.Ident == ast.TypeBool || typeNode.Ident == ast.TypeVoid || typeNode.Ident == ast.TypeError {
		// Convert Forst built-in types to Go built-in types
		switch typeNode.Ident {
		case ast.TypeString:
			return "string", nil
		case ast.TypeInt:
			return "int", nil
		case ast.TypeFloat:
			return "float64", nil
		case ast.TypeBool:
			return "bool", nil
		case ast.TypeVoid:
			return "", nil
		case ast.TypeError:
			return "error", nil
		default:
			return string(typeNode.Ident), nil
		}
	}

	// Handle pointer types
	if typeNode.Ident == ast.TypePointer {
		if len(typeNode.TypeParams) == 0 {
			return "", fmt.Errorf("pointer type must have a base type parameter")
		}
		baseTypeName, err := tc.GetAliasedTypeName(typeNode.TypeParams[0], opts)
		if err != nil {
			return "", fmt.Errorf("failed to get base type name for pointer: %w", err)
		}
		return "*" + baseTypeName, nil
	}

	// Qualified Go named types imported in Forst (e.g. gateway.GatewayRequest): stable spelling for codegen.
	if strings.Contains(string(typeNode.Ident), ".") {
		parts := strings.Split(string(typeNode.Ident), ".")
		if len(parts) == 2 && tc.GoQualifiedNamedTypeExists(parts[0], parts[1]) {
			return string(typeNode.Ident), nil
		}
	}

	// If this is a hash-based type, check for structural identity with user-defined types
	if opts.AllowStructuralAlias && typeNode.TypeKind == ast.TypeKindHashBased {
		// Look up the shape for this hash-based type
		hashDef, hashExists := tc.Defs[typeNode.Ident]
		if hashExists {
			if hashTypeDef, ok := hashDef.(ast.TypeDefNode); ok {
				if hashShapeExpr, ok := hashTypeDef.Expr.(ast.TypeDefShapeExpr); ok {
					hashShape := hashShapeExpr.Shape
					for _, def := range tc.Defs {
						if userDef, ok := def.(ast.TypeDefNode); ok && userDef.Ident != "" {
							if userPayload, ok := ast.PayloadShape(userDef.Expr); ok {
								if tc.shapesAreStructurallyIdentical(hashShape, *userPayload) {
									tc.log.WithFields(logrus.Fields{
										"hashType":    typeNode.Ident,
										"aliasedType": userDef.Ident,
										"function":    "GetAliasedTypeName",
									}).Debug("[DEBUG] Resolved hash-based type to aliased type")
									return string(userDef.Ident), nil
								}
							}
						}
					}
				}
			}
		}
	}

	// For user-defined types, check if they're already defined in the typechecker
	if typeNode.Ident != "" {
		if _, exists := tc.Defs[typeNode.Ident]; exists {
			return string(typeNode.Ident), nil
		}
	}

	// If not found in Defs, fall back to hash-based name
	hash, err := tc.Hasher.HashNode(typeNode)
	if err != nil {
		return "", fmt.Errorf("failed to hash type node: %w", err)
	}
	tc.log.WithFields(logrus.Fields{
		"function":     "GetAliasedTypeName",
		"typeNode":     typeNode.Ident,
		"fallbackHash": hash.ToTypeIdent(),
	}).Debug("[DEBUG] Fallback to hash-based type name")
	return string(hash.ToTypeIdent()), nil
}

// IsGoBuiltinType checks if a type is a Go builtin type
func IsGoBuiltinType(typeName string) bool {
	builtinTypes := map[string]bool{
		"string":  true,
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"float32": true,
		"float64": true,
		"bool":    true,
		"byte":    true,
		"rune":    true,
		"error":   true,
	}
	return builtinTypes[typeName]
}
