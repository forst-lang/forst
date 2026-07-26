package gointerop

import (
	"go/types"

	"forst/internal/ast"
)

// ErrorInterfaceType returns the predeclared error interface type, or nil if unavailable.
func ErrorInterfaceType() types.Type {
	obj := types.Universe.Lookup("error")
	if obj == nil {
		return nil
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil
	}
	return tn.Type()
}

func isUniverseBasicType(t types.Type, name string) bool {
	obj := types.Universe.Lookup(name)
	if obj == nil {
		return false
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return false
	}
	return types.Identical(t, tn.Type())
}

// TypeToForstType maps a go/types value to a Forst type at the FFI boundary.
func TypeToForstType(t types.Type) (ast.TypeNode, bool) {
	if t == nil {
		return ast.TypeNode{}, false
	}
	if errIface := ErrorInterfaceType(); errIface != nil && types.AssignableTo(t, errIface) {
		return ast.TypeNode{Ident: ast.TypeError}, true
	}
	if _, ok := t.(*types.Named); ok {
		return ast.TypeNode{Ident: ast.TypeImplicit}, true
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if isUniverseBasicType(t, "byte") {
			return ast.TypeNode{Ident: ast.TypeIdent("byte")}, true
		}
		if isUniverseBasicType(t, "rune") {
			return ast.TypeNode{Ident: ast.TypeIdent("rune")}, true
		}
		switch u.Kind() {
		case types.Bool, types.UntypedBool:
			return ast.TypeNode{Ident: ast.TypeBool}, true
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
			types.UntypedInt, types.UntypedRune:
			return ast.TypeNode{Ident: ast.TypeInt}, true
		case types.Float32, types.Float64, types.UntypedFloat:
			return ast.TypeNode{Ident: ast.TypeFloat}, true
		case types.Complex64:
			return ast.TypeNode{Ident: ast.TypeComplex64}, true
		case types.Complex128:
			return ast.TypeNode{Ident: ast.TypeComplex128}, true
		case types.String, types.UntypedString:
			return ast.TypeNode{Ident: ast.TypeString}, true
		case types.UnsafePointer:
			return ast.TypeNode{}, false
		default:
			return ast.TypeNode{}, false
		}
	case *types.Slice:
		elem, ok := TypeToForstType(u.Elem())
		if !ok {
			return ast.TypeNode{}, false
		}
		return ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elem}}, true
	case *types.Pointer:
		inner, ok := TypeToForstType(u.Elem())
		if !ok {
			return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeImplicit}}}, true
		}
		return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{inner}}, true
	case *types.Interface:
		return ast.TypeNode{Ident: ast.TypeImplicit}, true
	default:
		return ast.TypeNode{}, false
	}
}

// ForstAssignableToGoType reports whether a Forst type may be passed to a Go parameter of type g.
func ForstAssignableToGoType(host AssignabilityHost, f ast.TypeNode, g types.Type) bool {
	if slice, ok := g.Underlying().(*types.Slice); ok {
		if be, ok := slice.Elem().Underlying().(*types.Basic); ok && be.Kind() == types.Byte {
			if f.Ident == ast.TypeArray && len(f.TypeParams) == 1 {
				elem := f.TypeParams[0]
				if elem.Ident == ast.TypeInt || string(elem.Ident) == "byte" {
					return true
				}
			}
		}
	}
	switch u := g.Underlying().(type) {
	case *types.Interface:
		if u.NumMethods() == 0 {
			return true
		}
		if f.Ident == ast.TypePointer && len(f.TypeParams) == 1 && f.TypeParams[0].Ident == ast.TypeImplicit {
			return true
		}
	}
	if f.Ident == ast.TypeImplicit {
		_, ok := TypeToForstType(g)
		return ok
	}
	if exp, ok := host.ForstTypeForGoType(g); ok {
		return host.IsTypeCompatible(f, exp)
	}
	exp, ok := TypeToForstType(g)
	if !ok {
		return false
	}
	return host.IsTypeCompatible(f, exp)
}
