package typechecker

import (
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/goload"

	"github.com/sirupsen/logrus"
)

// goErrorInterfaceType returns the predeclared error interface type, or nil if unavailable.
func goErrorInterfaceType() types.Type {
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
	tc.dotImportPkgs = nil
	tc.importPathByLocal = make(map[string]string)

	if tc.GoWorkspaceDir == "" {
		for _, imp := range tc.imports {
			p, l := fallbackImportLocal(imp)
			if l != "" && l != "." {
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
			if l != "" && l != "." {
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
			if l != "" && l != "." {
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
			if l != "" && l != "." {
				tc.importPathByLocal[l] = p
			}
			continue
		}
		tp := pkgp.Types
		if imp.Alias != nil && string(imp.Alias.ID) == "." {
			// Dot-import does not introduce a package identifier (e.g. `strings` is not in scope).
			// Do not register tp.Name() in importPathByLocal — that would break IsImportedLocalName / LSP.
			tc.dotImportPkgs = append(tc.dotImportPkgs, tp)
			continue
		}
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

// goPackageForImportLocal returns *types.Package for a Go import's local name (e.g. "strings", "fmt").
// It uses the map from initGoImportPackages, or lazily runs go/packages for that import path when the
// batch load failed or left this name unloaded — so qualified calls like strings.NewReader still resolve.
func (tc *TypeChecker) goPackageForImportLocal(local string) *types.Package {
	if local == "" || local == "." {
		return nil
	}
	if tc.goPkgsByLocal != nil {
		if p := tc.goPkgsByLocal[local]; p != nil {
			return p
		}
	}
	path := ""
	if tc.importPathByLocal != nil {
		path = tc.importPathByLocal[local]
	}
	if path == "" || tc.GoWorkspaceDir == "" {
		return nil
	}
	loaded, err := goload.LoadByPkgPath(tc.GoWorkspaceDir, []string{path})
	if err != nil || len(loaded) == 0 {
		return nil
	}
	pkgp, ok := loaded[path]
	if !ok || pkgp == nil || pkgp.Types == nil {
		return nil
	}
	gp := pkgp.Types
	if tc.goPkgsByLocal == nil {
		tc.goPkgsByLocal = make(map[string]*types.Package)
	}
	tc.goPkgsByLocal[local] = gp
	return gp
}

func goIdentifierExported(name string) bool {
	if name == "" || name[0] == '_' {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name)
	return r != utf8.RuneError && unicode.IsUpper(r)
}

// lookupDotImportFunc finds which dot-imported package defines the given function name, if unique.
func (tc *TypeChecker) lookupDotImportFunc(funcName string, sp ast.SourceSpan) (*types.Package, error) {
	if len(tc.dotImportPkgs) == 0 {
		return nil, nil
	}
	if !goIdentifierExported(funcName) {
		return nil, nil
	}
	var matched []*types.Package
	for _, pkg := range tc.dotImportPkgs {
		obj := pkg.Scope().Lookup(funcName)
		if obj == nil {
			continue
		}
		if _, ok := obj.(*types.Func); !ok {
			continue
		}
		matched = append(matched, pkg)
	}
	if len(matched) == 0 {
		return nil, nil
	}
	if len(matched) > 1 {
		return nil, diagnosticf(sp, "dot-import", "%s is ambiguous (multiple dot-imported packages)", funcName)
	}
	return matched[0], nil
}

// foldErrorPair: when true, Go (T, error) is represented as a single Result(T, Error) (expression / single-assignment).
// When false, returns two separate type nodes so two-value assignments (v, err := pkg.F()) typecheck.
func (tc *TypeChecker) checkGoQualifiedCall(pkg *types.Package, pkgDisplay, funcName string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode, foldErrorPair bool) ([]ast.TypeNode, error) {
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
	mapped, err := tc.checkGoSignature(sig, pkgDisplay+"."+funcName, e, argTypes, foldErrorPair)
	if err != nil {
		return nil, err
	}
	// go/types is authoritative here (package is loaded). Do not override with BuiltinFunctions
	// entries that may omit error returns (e.g. strconv.Atoi is (int, error) -> Result(Int, Error)).
	return mapped, nil
}

func (tc *TypeChecker) checkGoSignature(sig *types.Signature, qual string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode, foldErrorPair bool) ([]ast.TypeNode, error) {
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
	// Go idiom (T1,...,Tn, error) becomes Result(T1, Error) when n==1, else Result(Tuple(T1..Tn), Error).
	if foldErrorPair && res.Len() >= 2 {
		errIface := goErrorInterfaceType()
		if errIface != nil && types.AssignableTo(res.At(res.Len()-1).Type(), errIface) {
			n := res.Len()
			successTypes := out[:n-1]
			failureType := out[n-1]
			var success ast.TypeNode
			if len(successTypes) == 1 {
				success = successTypes[0]
			} else {
				success = ast.NewTupleType(successTypes...)
			}
			return []ast.TypeNode{ast.NewResultType(success, failureType)}, nil
		}
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
		// Opaque Go pointer (e.g. *strings.Reader from NewReader) implements io.Reader at the FFI boundary.
		if f.Ident == ast.TypePointer && len(f.TypeParams) == 1 && f.TypeParams[0].Ident == ast.TypeImplicit {
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
	if errIface := goErrorInterfaceType(); errIface != nil && types.AssignableTo(t, errIface) {
		return ast.TypeNode{Ident: ast.TypeError}, true
	}
	// Named types (e.g. strings.Reader, enmime.Envelope) must be detected before Underlying(),
	// which would strip to struct/interface and lose the stable FFI mapping.
	if _, ok := t.(*types.Named); ok {
		return ast.TypeNode{Ident: ast.TypeImplicit}, true
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
			return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeImplicit}}}, true
		}
		return ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{inner}}, true
	case *types.Interface:
		// io.Reader and other interfaces at the Forst↔Go boundary map to implicit until a richer model exists.
		return ast.TypeNode{Ident: ast.TypeImplicit}, true
	default:
		return ast.TypeNode{}, false
	}
}
