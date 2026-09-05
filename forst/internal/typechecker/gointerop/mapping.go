package gointerop

import (
	"sort"
	"unicode"
	"unicode/utf8"

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

// TypeToForstType maps a go/types value to a Forst type without host context (named → Implicit).
func TypeToForstType(t types.Type) (ast.TypeNode, bool) {
	return MapGoType(nil, t)
}

// MapGoType maps go/types to Forst at the FFI boundary. When host is non-nil, named Go types
// resolve to qualified Forst idents (e.g. os.File) via ForstTypeForGoType.
func MapGoType(host AssignabilityHost, t types.Type) (ast.TypeNode, bool) {
	t = types.Unalias(t)
	if t == nil {
		return ast.TypeNode{}, false
	}

	if _, ok := t.(*types.Named); ok {
		if host != nil {
			if ft, ok := host.ForstTypeForGoType(t); ok {
				return ft, true
			}
		}
		if errIface := ErrorInterfaceType(); errIface != nil && types.AssignableTo(t, errIface) {
			return ast.TypeNode{Ident: ast.TypeError}, true
		}
		return ast.TypeNode{Ident: ast.TypeImplicit}, true
	}

	switch u := t.Underlying().(type) {
	case *types.Basic:
		return mapBasicType(t, u)
	case *types.Slice:
		elem, ok := MapGoType(host, u.Elem())
		if !ok {
			return ast.TypeNode{}, false
		}
		return ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elem}}, true
	case *types.Array:
		elem, ok := MapGoType(host, u.Elem())
		if !ok {
			return ast.TypeNode{}, false
		}
		n := int64(u.Len())
		return ast.TypeNode{Ident: ast.TypeArray, ArrayLen: &n, TypeParams: []ast.TypeNode{elem}}, true
	case *types.Map:
		key, ok1 := MapGoType(host, u.Key())
		val, ok2 := MapGoType(host, u.Elem())
		if !ok1 || !ok2 {
			return ast.TypeNode{}, false
		}
		return ast.TypeNode{Ident: ast.TypeMap, TypeParams: []ast.TypeNode{key, val}}, true
	case *types.Chan:
		elem, ok := MapGoType(host, u.Elem())
		if !ok {
			return ast.TypeNode{}, false
		}
		return ast.TypeNode{Ident: ast.TypeChannel, TypeParams: []ast.TypeNode{elem}}, true
	case *types.Pointer:
		inner, ok := MapGoType(host, u.Elem())
		if !ok {
			return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeImplicit}}}, true
		}
		return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{inner}}, true
	case *types.Interface:
		return ast.TypeNode{Ident: ast.TypeImplicit}, true
	case *types.Struct:
		return mapGoStructType(host, u)
	case *types.Signature:
		return mapGoSignature(host, u)
	default:
		return ast.TypeNode{}, false
	}
}

func mapBasicType(t types.Type, u *types.Basic) (ast.TypeNode, bool) {
	if isUniverseBasicType(t, "byte") {
		return ast.TypeNode{Ident: ast.TypeIdent("byte")}, true
	}
	if isUniverseBasicType(t, "rune") {
		return ast.TypeNode{Ident: ast.TypeIdent("rune")}, true
	}
	switch u.Kind() {
	case types.Bool, types.UntypedBool:
		return ast.TypeNode{Ident: ast.TypeBool}, true
	case types.Int, types.UntypedInt:
		return ast.TypeNode{Ident: ast.TypeInt}, true
	case types.Int8:
		return ast.TypeNode{Ident: ast.TypeIdent("int8")}, true
	case types.Int16:
		return ast.TypeNode{Ident: ast.TypeIdent("int16")}, true
	case types.Int32:
		return ast.TypeNode{Ident: ast.TypeIdent("int32")}, true
	case types.UntypedRune:
		return ast.TypeNode{Ident: ast.TypeIdent("rune")}, true
	case types.Int64:
		return ast.TypeNode{Ident: ast.TypeIdent("int64")}, true
	case types.Uint:
		return ast.TypeNode{Ident: ast.TypeIdent("uint")}, true
	case types.Uint8:
		return ast.TypeNode{Ident: ast.TypeIdent("uint8")}, true
	case types.Uint16:
		return ast.TypeNode{Ident: ast.TypeIdent("uint16")}, true
	case types.Uint32:
		return ast.TypeNode{Ident: ast.TypeIdent("uint32")}, true
	case types.Uint64:
		return ast.TypeNode{Ident: ast.TypeIdent("uint64")}, true
	case types.Uintptr:
		return ast.TypeNode{Ident: ast.TypeIdent("uintptr")}, true
	case types.Float32:
		return ast.TypeNode{Ident: ast.TypeIdent("float32")}, true
	case types.Float64, types.UntypedFloat:
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
}

func mapGoSignature(host AssignabilityHost, sig *types.Signature) (ast.TypeNode, bool) {
	params := sig.Params()
	paramNodes := make([]ast.ParamNode, params.Len())
	for i := 0; i < params.Len(); i++ {
		pt, ok := MapGoType(host, params.At(i).Type())
		if !ok {
			return ast.TypeNode{}, false
		}
		paramNodes[i] = ast.SimpleParamNode{Type: pt}
	}
	results := sig.Results()
	var retTypes []ast.TypeNode
	if results.Len() == 0 {
		retTypes = []ast.TypeNode{{Ident: ast.TypeVoid}}
	} else if results.Len() == 1 {
		rt, ok := MapGoType(host, results.At(0).Type())
		if !ok {
			return ast.TypeNode{}, false
		}
		retTypes = []ast.TypeNode{rt}
	} else {
		retTypes = make([]ast.TypeNode, results.Len())
		for i := 0; i < results.Len(); i++ {
			rt, ok := MapGoType(host, results.At(i).Type())
			if !ok {
				return ast.TypeNode{}, false
			}
			retTypes[i] = rt
		}
	}
	return ast.NewFunctionType(paramNodes, retTypes), true
}

// ForstAssignableToGoType reports whether a Forst type may be passed to a Go parameter of type g.
func ForstAssignableToGoType(host AssignabilityHost, f ast.TypeNode, g types.Type) bool {
	g = types.Unalias(g)
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
	if host != nil {
		if fGo := host.GoTypeForForstType(f); fGo != nil {
			if types.AssignableTo(fGo, g) {
				return true
			}
			if forstUntypedNumericAssignsToGo(f, g) {
				return true
			}
			return false
		}
	}
	if forstUntypedNumericAssignsToGo(f, g) {
		return true
	}
	switch u := g.Underlying().(type) {
	case *types.Interface:
		if u.NumMethods() == 0 {
			return true
		}
	}
	if f.Ident == ast.TypeImplicit {
		_, ok := MapGoType(host, g)
		return ok
	}
	if host != nil {
		if exp, ok := host.ForstTypeForGoType(g); ok {
			return host.IsTypeCompatible(f, exp)
		}
	}
	exp, ok := MapGoType(host, g)
	if !ok || host == nil {
		return false
	}
	return host.IsTypeCompatible(f, exp)
}

func forstUntypedNumericAssignsToGo(f ast.TypeNode, g types.Type) bool {
	if f.Ident != ast.TypeInt && f.Ident != ast.TypeIdent("rune") && f.Ident != ast.TypeIdent("byte") {
		return false
	}
	g = types.Unalias(g)
	if basic, ok := g.Underlying().(*types.Basic); ok {
		if basic.Kind() == types.Byte || basic.Kind() == types.Uint8 {
			return f.Ident == ast.TypeInt || f.Ident == ast.TypeIdent("rune") || f.Ident == ast.TypeIdent("byte")
		}
		if basic.Kind() == types.Rune || basic.Kind() == types.Int32 {
			return f.Ident == ast.TypeInt || f.Ident == ast.TypeIdent("rune")
		}
		if isGoIntegerBasic(basic) {
			return f.Ident == ast.TypeInt
		}
	}
	if named, ok := g.(*types.Named); ok {
		if basic, ok := named.Underlying().(*types.Basic); ok && isGoIntegerBasic(basic) {
			return f.Ident == ast.TypeInt
		}
	}
	return false
}

func isGoIntegerBasic(b *types.Basic) bool {
	switch b.Kind() {
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		return true
	default:
		return false
	}
}

const shapeMatchConstraint = "Match"

func mapGoStructType(host AssignabilityHost, st *types.Struct) (ast.TypeNode, bool) {
	if st == nil {
		return ast.TypeNode{}, false
	}
	if st.NumFields() == 0 {
		return ast.TypeNode{Ident: ast.TypeImplicit}, true
	}

	type fieldEntry struct {
		name string
		node ast.ShapeFieldNode
	}
	entries := make([]fieldEntry, 0, st.NumFields())
	seen := make(map[types.Type]bool)
	var collectFields func(*types.Struct)
	collectFields = func(str *types.Struct) {
		if str == nil {
			return
		}
		for i := 0; i < str.NumFields(); i++ {
			f := str.Field(i)
			if f == nil || !f.Exported() {
				continue
			}
			if f.Anonymous() {
				embedded := types.Unalias(f.Type())
				if ptr, ok := embedded.(*types.Pointer); ok {
					embedded = types.Unalias(ptr.Elem())
				}
				if embStruct, ok := embedded.Underlying().(*types.Struct); ok {
					if seen[embedded] {
						continue
					}
					seen[embedded] = true
					collectFields(embStruct)
					continue
				}
			}
			ft, ok := MapGoType(host, f.Type())
			if !ok {
				continue
			}
			forstName := forstFieldNameFromGoField(f, str.Tag(i))
			if forstName == "" {
				continue
			}
			ftCopy := ft
			entries = append(entries, fieldEntry{
				name: forstName,
				node: ast.ShapeFieldNode{Type: &ftCopy},
			})
		}
	}
	collectFields(st)
	if len(entries) == 0 {
		return ast.TypeNode{Ident: ast.TypeImplicit}, true
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	fields := make(map[string]ast.ShapeFieldNode, len(entries))
	for _, e := range entries {
		fields[e.name] = e.node
	}
	shape := &ast.ShapeNode{Fields: fields}
	baseType := ast.TypeIdent(ast.TypeShape)
	return ast.TypeNode{
		Ident: ast.TypeShape,
		Assertion: &ast.AssertionNode{
			BaseType: &baseType,
			Constraints: []ast.ConstraintNode{{
				Name: shapeMatchConstraint,
				Args: []ast.ConstraintArgumentNode{{
					Shape: shape,
				}},
			}},
		},
	}, true
}

func forstFieldNameFromGoField(f *types.Var, tag string) string {
	if f == nil {
		return ""
	}
	if jsonName := jsonFieldNameFromStructTag(tag); jsonName != "" {
		return jsonName
	}
	goName := f.Name()
	if goName == "" {
		return ""
	}
	r, sz := utf8.DecodeRuneInString(goName)
	if r == utf8.RuneError && sz == 0 {
		return goName
	}
	return string(unicode.ToLower(r)) + goName[sz:]
}
