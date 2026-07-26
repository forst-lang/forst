package gointerop

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"go/types"
)

// ExportedFieldName mirrors the transformer's capitalizeFirst (Go export rule).
func ExportedFieldName(s string) string {
	if s == "" {
		return s
	}
	r, sz := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && sz == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[sz:]
}

func jsonFieldNameFromStructTag(tag string) string {
	if tag == "" {
		return ""
	}
	name := reflect.StructTag(tag).Get("json")
	if name == "" || name == "-" {
		return ""
	}
	if i := strings.IndexByte(name, ','); i >= 0 {
		name = name[:i]
	}
	return name
}

func structType(recv types.Type) *types.Struct {
	if recv == nil {
		return nil
	}
	for {
		switch u := recv.Underlying().(type) {
		case *types.Pointer:
			recv = u.Elem()
		case *types.Named:
			recv = u.Underlying()
		case *types.Struct:
			return u
		default:
			return nil
		}
	}
}

// StructFieldTypeForForstName resolves a struct field whose Forst name is forstName,
// matching in order: json tag == forstName, exact field name, capitalized-first-letter name.
func StructFieldTypeForForstName(recv types.Type, forstName string) (types.Type, bool) {
	st := structType(recv)
	if st == nil {
		return nil, false
	}
	return structFieldTypeForForstNameOnStruct(st, forstName)
}

func structFieldTypeForForstNameOnStruct(st *types.Struct, forstName string) (types.Type, bool) {
	if st == nil || forstName == "" {
		return nil, false
	}
	exported := ExportedFieldName(forstName)
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if f == nil {
			continue
		}
		if jsonFieldNameFromStructTag(st.Tag(i)) == forstName {
			return f.Type(), true
		}
	}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if f == nil {
			continue
		}
		if f.Name() == forstName || f.Name() == exported {
			return f.Type(), true
		}
	}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if f == nil || !f.Anonymous() {
			continue
		}
		if embedded := structType(f.Type()); embedded != nil {
			if ft, found := structFieldTypeForForstNameOnStruct(embedded, forstName); found {
				return ft, true
			}
		}
	}
	return nil, false
}

// TypeAtFieldPath resolves exported field selectors on a Go type (e.g. *url.URL then ["Path"]).
func TypeAtFieldPath(recv types.Type, fieldPath []string) (types.Type, error) {
	if len(fieldPath) == 0 {
		return recv, nil
	}
	name := fieldPath[0]
	var ft types.Type
	var found bool
	if ft, found = StructFieldTypeForForstName(recv, name); !found {
		obj, _, _ := types.LookupFieldOrMethod(recv, true, nil, name)
		if obj == nil {
			return nil, fmt.Errorf("no field or method %q on %s", name, recv)
		}
		switch o := obj.(type) {
		case *types.Var:
			ft = o.Type()
		case *types.Func:
			ft = o.Type()
		default:
			return nil, fmt.Errorf("%q is not a struct field or method (got %T)", name, obj)
		}
	}
	if len(fieldPath) == 1 {
		return ft, nil
	}
	return TypeAtFieldPath(ft, fieldPath[1:])
}

// NamedTypeRoot returns the named type at the root of g (unwraps one pointer level).
func NamedTypeRoot(g types.Type) (*types.Named, bool) {
	if g == nil {
		return nil, false
	}
	switch t := g.(type) {
	case *types.Named:
		return t, true
	case *types.Pointer:
		if n, ok := t.Elem().(*types.Named); ok {
			return n, true
		}
	}
	return nil, false
}
