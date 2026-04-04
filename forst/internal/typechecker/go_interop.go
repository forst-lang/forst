package typechecker

import (
	"go/types"
	"strings"

	"forst/internal/ast"
	"forst/internal/goload"

	"github.com/sirupsen/logrus"
)

func fallbackImportLocal(imp ast.ImportNode) (path, local string) {
	ip := goload.ImportPathFromForst(imp.Path)
	if ip == "" {
		return "", ""
	}
	if imp.Alias != nil {
		return ip, string(imp.Alias.ID)
	}
	if i := strings.LastIndex(ip, "/"); i >= 0 {
		return ip, ip[i+1:]
	}
	return ip, ip
}

// initGoImportPackages loads Go packages for Forst import lines when GoWorkspaceDir is set.
// On failure, logs at debug and leaves goPkgsByLocal nil (Forst still typechecks without boundary checks).
func (tc *TypeChecker) initGoImportPackages() {
	tc.goPkgsByLocal = nil
	tc.importPathByLocal = make(map[string]string)

	if tc.GoWorkspaceDir == "" {
		for _, imp := range tc.imports {
			p, l := fallbackImportLocal(imp)
			if l != "" {
				tc.importPathByLocal[l] = p
			}
		}
		return
	}
	pathSet := make(map[string]struct{})
	for _, imp := range tc.imports {
		p := goload.ImportPathFromForst(imp.Path)
		if p != "" {
			pathSet[p] = struct{}{}
		}
	}
	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	if len(paths) == 0 {
		for _, imp := range tc.imports {
			p, l := fallbackImportLocal(imp)
			if l != "" {
				tc.importPathByLocal[l] = p
			}
		}
		return
	}
	loaded, err := goload.LoadByPkgPath(tc.GoWorkspaceDir, paths)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "initGoImportPackages",
			"dir":      tc.GoWorkspaceDir,
		}).WithError(err).Debug("go/packages load failed; skipping Forst↔Go boundary checks")
		for _, imp := range tc.imports {
			p, l := fallbackImportLocal(imp)
			if l != "" {
				tc.importPathByLocal[l] = p
			}
		}
		return
	}
	byLocal := make(map[string]*types.Package)
	for _, imp := range tc.imports {
		ip := goload.ImportPathFromForst(imp.Path)
		pkgp, ok := loaded[ip]
		if !ok || pkgp == nil || pkgp.Types == nil {
			p, l := fallbackImportLocal(imp)
			if l != "" {
				tc.importPathByLocal[l] = p
			}
			continue
		}
		tp := pkgp.Types
		var local string
		if imp.Alias != nil {
			local = string(imp.Alias.ID)
		} else {
			local = tp.Name()
		}
		tc.importPathByLocal[local] = ip
		byLocal[local] = tp
	}
	tc.goPkgsByLocal = byLocal
}

func (tc *TypeChecker) checkGoQualifiedCall(pkg *types.Package, pkgDisplay, funcName string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode) ([]ast.TypeNode, error) {
	obj := pkg.Scope().Lookup(funcName)
	if obj == nil {
		sp := e.Function.Span
		if !sp.IsSet() {
			sp = e.CallSpan
		}
		return nil, diagnosticf(sp, "go-call", "%s.%s not found in Go package", pkgDisplay, funcName)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		sp := e.Function.Span
		if !sp.IsSet() {
			sp = e.CallSpan
		}
		return nil, diagnosticf(sp, "go-call", "%s.%s is not a function", pkgDisplay, funcName)
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, diagnosticf(e.CallSpan, "go-call", "%s.%s: invalid signature", pkgDisplay, funcName)
	}
	mapped, err := tc.checkGoSignature(sig, pkgDisplay+"."+funcName, e, argTypes)
	if err != nil {
		return nil, err
	}
	qual := pkgDisplay + "." + funcName
	if b, ok := BuiltinFunctions[qual]; ok {
		return []ast.TypeNode{b.ReturnType}, nil
	}
	return mapped, nil
}

func (tc *TypeChecker) checkGoSignature(sig *types.Signature, qual string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode) ([]ast.TypeNode, error) {
	params := sig.Params()
	nParams := params.Len()
	nArgs := len(argTypes)

	if sig.Variadic() {
		fixed := nParams - 1
		if nArgs < fixed {
			return nil, diagnosticf(e.CallSpan, "go-call", "%s: expects at least %d arguments, got %d", qual, fixed, nArgs)
		}
		for i := 0; i < fixed; i++ {
			if err := tc.checkOneGoParam(qual, i, params.At(i).Type(), argTypes[i], e, i); err != nil {
				return nil, err
			}
		}
		sliceT, ok := params.At(nParams-1).Type().Underlying().(*types.Slice)
		if !ok {
			return nil, diagnosticf(e.CallSpan, "go-call", "%s: invalid variadic parameter", qual)
		}
		elem := sliceT.Elem()
		for j := fixed; j < nArgs; j++ {
			if err := tc.checkOneGoParam(qual, j, elem, argTypes[j], e, j); err != nil {
				return nil, err
			}
		}
	} else {
		if nArgs != nParams {
			sp := e.CallSpan
			if nArgs > nParams {
				sp = spanForCallArg(e.ArgSpans, nParams, e.Arguments, e.CallSpan)
			}
			if !sp.IsSet() {
				sp = e.Function.Span
			}
			return nil, diagnosticf(sp, "go-call", "%s: expects %d arguments, got %d", qual, nParams, nArgs)
		}
		for i := 0; i < nParams; i++ {
			if err := tc.checkOneGoParam(qual, i, params.At(i).Type(), argTypes[i], e, i); err != nil {
				return nil, err
			}
		}
	}

	res := sig.Results()
	if res.Len() == 0 {
		return []ast.TypeNode{{Ident: ast.TypeVoid}}, nil
	}
	out := make([]ast.TypeNode, res.Len())
	for i := 0; i < res.Len(); i++ {
		gt, ok := goTypeToForstType(res.At(i).Type())
		if !ok {
			sp := e.Function.Span
			if !sp.IsSet() {
				sp = e.CallSpan
			}
			return nil, diagnosticf(sp, "go-call", "%s: unsupported Go return type %s", qual, res.At(i).Type().String())
		}
		out[i] = gt
	}
	return out, nil
}

func (tc *TypeChecker) checkOneGoParam(qual string, i int, goParam types.Type, argT []ast.TypeNode, e ast.FunctionCallNode, argIdx int) error {
	sp := spanForCallArg(e.ArgSpans, argIdx, e.Arguments, e.CallSpan)
	if len(argT) != 1 {
		return diagnosticf(sp, "go-call", "%s argument %d must have a single type, got %d", qual, i+1, len(argT))
	}
	if !tc.forstAssignableToGoType(argT[0], goParam) {
		return diagnosticf(sp, "go-call", "%s argument %d: Forst type %s not assignable to Go parameter %s",
			qual, i+1, argT[0].Ident, strings.TrimSpace(goParam.String()))
	}
	return nil
}

func (tc *TypeChecker) forstAssignableToGoType(f ast.TypeNode, g types.Type) bool {
	switch u := g.Underlying().(type) {
	case *types.Interface:
		if u.NumMethods() == 0 {
			return true
		}
	}
	exp, ok := goTypeToForstType(g)
	if !ok {
		return false
	}
	return tc.IsTypeCompatible(f, exp)
}

func goTypeToForstType(t types.Type) (ast.TypeNode, bool) {
	if t == nil {
		return ast.TypeNode{}, false
	}
	if t.String() == "error" {
		return ast.TypeNode{Ident: ast.TypeError}, true
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Bool, types.UntypedBool:
			return ast.TypeNode{Ident: ast.TypeBool}, true
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
			types.UntypedInt, types.UntypedRune:
			return ast.TypeNode{Ident: ast.TypeInt}, true
		case types.Float32, types.Float64, types.UntypedFloat:
			return ast.TypeNode{Ident: ast.TypeFloat}, true
		case types.String, types.UntypedString:
			return ast.TypeNode{Ident: ast.TypeString}, true
		case types.UnsafePointer:
			return ast.TypeNode{}, false
		default:
			return ast.TypeNode{}, false
		}
	case *types.Slice:
		elem, ok := goTypeToForstType(u.Elem())
		if !ok {
			return ast.TypeNode{}, false
		}
		return ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elem}}, true
	case *types.Pointer:
		inner, ok := goTypeToForstType(u.Elem())
		if !ok {
			return ast.TypeNode{}, false
		}
		return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{inner}}, true
	default:
		return ast.TypeNode{}, false
	}
}
