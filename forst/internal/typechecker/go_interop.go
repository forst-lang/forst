package typechecker

import (
	"errors"
	"fmt"
	"go/types"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/goload"

	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
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

// goPackagesLoadDir returns the directory passed to go/packages (module root when set, else ".").
func (tc *TypeChecker) goPackagesLoadDir() string {
	if tc.GoWorkspaceDir != "" {
		return tc.GoWorkspaceDir
	}
	return "."
}

// initGoImportPackages loads Go packages for Forst import lines via go/packages.
// Fills any imports missing from an earlier batch preload so qualified calls like exec.Command resolve.
func (tc *TypeChecker) initGoImportPackages() {
	tc.ensureImportPathByLocal()
	missing := tc.missingGoImportPaths()
	if len(missing) == 0 {
		if tc.allGoImportLocalsLoaded() {
			tc.goPackagesPreloaded = true
		}
		return
	}
	loaded, err := goload.LoadByPkgPath(tc.goPackagesLoadDir(), missing)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "initGoImportPackages",
			"dir":      tc.goPackagesLoadDir(),
			"missing":  missing,
		}).WithError(err).Debug("go/packages load failed; skipping Forst↔Go boundary checks")
		return
	}
	tc.seedGoImportPackagesFromLoaded(loaded)
	if tc.allGoImportLocalsLoaded() {
		tc.goPackagesPreloaded = true
	}
}

// InitGoPackagesFromBatch maps import locals from a preloaded go/packages batch (module-wide).
func (tc *TypeChecker) InitGoPackagesFromBatch(loaded map[string]*packages.Package) {
	tc.ensureImportPathByLocal()
	if len(loaded) > 0 {
		tc.seedGoImportPackagesFromLoaded(loaded)
	}
	if tc.allGoImportLocalsLoaded() {
		tc.goPackagesPreloaded = true
	}
	if tc.samePackageGoImportPath != "" {
		if pkg, ok := loaded[tc.samePackageGoImportPath]; ok && goload.PackageLoadOK(pkg, tc.samePackageGoImportPath) {
			tc.samePackageGo = pkg.Types
		}
	}
}

func (tc *TypeChecker) ensureImportPathByLocal() {
	if tc.importPathByLocal == nil {
		tc.importPathByLocal = make(map[string]string)
	}
	tc.registerImportLocalsFromAST()
}

// missingGoImportPaths returns Go import paths that are not yet loaded in goPkgsByLocal.
func (tc *TypeChecker) missingGoImportPaths() []string {
	seen := make(map[string]struct{})
	var paths []string
	for _, imp := range tc.imports {
		ip := goload.ImportPathFromForst(imp.Path)
		if ip == "" {
			continue
		}
		if imp.Alias != nil && string(imp.Alias.ID) == "." {
			continue
		}
		path, local := fallbackImportLocal(imp)
		if local == "" || local == "." {
			continue
		}
		if tc.goPkgsByLocal != nil && tc.goPkgsByLocal[local] != nil {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func (tc *TypeChecker) allGoImportLocalsLoaded() bool {
	for _, imp := range tc.imports {
		ip := goload.ImportPathFromForst(imp.Path)
		if ip == "" {
			continue
		}
		if imp.Alias != nil && string(imp.Alias.ID) == "." {
			continue
		}
		_, local := fallbackImportLocal(imp)
		if local == "" || local == "." {
			continue
		}
		if tc.goPkgsByLocal == nil || tc.goPkgsByLocal[local] == nil {
			return false
		}
	}
	return true
}

func (tc *TypeChecker) hasDotImportPath(path string) bool {
	for _, pkg := range tc.dotImportPkgs {
		if pkg != nil && pkg.Path() == path {
			return true
		}
	}
	return false
}

func (tc *TypeChecker) seedGoImportPackagesFromLoaded(loaded map[string]*packages.Package) {
	if tc.goPkgsByLocal == nil {
		tc.goPkgsByLocal = make(map[string]*types.Package)
	}
	for _, imp := range tc.imports {
		ip := goload.ImportPathFromForst(imp.Path)
		if ip == "" {
			continue
		}
		pkgp, ok := loaded[ip]
		if !ok || !goload.PackageLoadOK(pkgp, ip) {
			continue
		}
		tp := pkgp.Types
		if tp == nil {
			continue
		}
		if imp.Alias != nil && string(imp.Alias.ID) == "." {
			if !tc.hasDotImportPath(tp.Path()) {
				tc.dotImportPkgs = append(tc.dotImportPkgs, tp)
			}
			continue
		}
		var local string
		if imp.Alias != nil {
			local = string(imp.Alias.ID)
		} else {
			local = tp.Name()
			if local == "" {
				_, local = fallbackImportLocal(imp)
			}
		}
		if local == "" || local == "." {
			continue
		}
		tc.goPkgsByLocal[local] = tp
	}
}

// BatchLoadGoPackagesForModule unions Go import paths from typecheckers and loads once.
func BatchLoadGoPackagesForModule(moduleRoot string, tcs []*TypeChecker) (map[string]*packages.Package, error) {
	if moduleRoot == "" || len(tcs) == 0 {
		return nil, nil
	}
	pathSet := make(map[string]struct{})
	for _, tc := range tcs {
		if tc == nil {
			continue
		}
		for _, imp := range tc.imports {
			ip := goload.ImportPathFromForst(imp.Path)
			if ip != "" {
				pathSet[ip] = struct{}{}
			}
		}
		if tc.samePackageGoImportPath != "" {
			pathSet[tc.samePackageGoImportPath] = struct{}{}
		}
	}
	if len(pathSet) == 0 {
		return nil, nil
	}
	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	return goload.LoadByPkgPath(moduleRoot, paths)
}

// BatchLoadGoPackagesForModuleWithLoader is like BatchLoadGoPackagesForModule but accepts a custom loader.
func BatchLoadGoPackagesForModuleWithLoader(moduleRoot string, tcs []*TypeChecker, loader goload.PackagesLoader) (map[string]*packages.Package, error) {
	if moduleRoot == "" || len(tcs) == 0 {
		return nil, nil
	}
	pathSet := make(map[string]struct{})
	for _, tc := range tcs {
		if tc == nil {
			continue
		}
		for _, imp := range tc.imports {
			ip := goload.ImportPathFromForst(imp.Path)
			if ip != "" {
				pathSet[ip] = struct{}{}
			}
		}
		if tc.samePackageGoImportPath != "" {
			pathSet[tc.samePackageGoImportPath] = struct{}{}
		}
	}
	if len(pathSet) == 0 {
		return nil, nil
	}
	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	if loader == nil {
		return goload.LoadByPkgPath(moduleRoot, paths)
	}
	return goload.LoadByPkgPath(moduleRoot, paths, goload.WithPackagesLoader(loader))
}

// initSamePackageGoExports loads exported Go funcs from .go files co-located with this Forst package.
func (tc *TypeChecker) initSamePackageGoExports() {
	if tc.goPackagesPreloaded {
		return
	}
	tc.samePackageGo = nil
	if tc.samePackageGoImportPath == "" || tc.GoWorkspaceDir == "" {
		return
	}
	loaded, err := goload.LoadByPkgPath(tc.GoWorkspaceDir, []string{tc.samePackageGoImportPath})
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "initSamePackageGoExports",
			"path":     tc.samePackageGoImportPath,
		}).WithError(err).Debug("go/packages load failed for same-package Go exports")
		return
	}
	pkg, ok := loaded[tc.samePackageGoImportPath]
	if !ok || !goload.PackageLoadOK(pkg, tc.samePackageGoImportPath) {
		return
	}
	tc.samePackageGo = pkg.Types
}

// trySamePackageGoCall resolves an unqualified call to an exported Go func co-located with this Forst package.
// Returns found=false when no same-package Go is loaded or the name is absent/unexported/non-function.
func (tc *TypeChecker) trySamePackageGoCall(funcName string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode, foldErrorPair bool) ([]ast.TypeNode, bool, error) {
	if tc.samePackageGo == nil || !goIdentifierExported(funcName) {
		return nil, false, nil
	}
	ret, err := tc.checkGoFuncCall(tc.samePackageGo, funcName, funcName, e, argTypes, foldErrorPair)
	if err != nil {
		var diag *Diagnostic
		if errors.As(err, &diag) {
			switch {
			case strings.Contains(diag.Msg, "not found in Go package"):
				return nil, false, nil
			case strings.Contains(diag.Msg, "is not a function"):
				return nil, false, nil
			default:
				return nil, true, err
			}
		}
		return nil, true, err
	}
	return ret, true, nil
}

func (tc *TypeChecker) registerImportLocalsFromAST() {
	for _, imp := range tc.imports {
		path, local := fallbackImportLocal(imp)
		if local != "" && local != "." {
			tc.importPathByLocal[local] = path
		}
	}
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
	if path == "" {
		return nil
	}
	loaded, err := goload.LoadByPkgPath(tc.goPackagesLoadDir(), []string{path})
	if err != nil || len(loaded) == 0 {
		return nil
	}
	pkgp, ok := loaded[path]
	if !ok || !goload.PackageLoadOK(pkgp, path) {
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
		if pkg == nil {
			continue
		}
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
func (tc *TypeChecker) checkGoFuncCall(pkg *types.Package, qualDisplay, funcName string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode, foldErrorPair bool) ([]ast.TypeNode, error) {
	qual := funcName
	if qualDisplay != funcName {
		qual = qualDisplay + "." + funcName
	}
	obj := pkg.Scope().Lookup(funcName)
	if obj == nil {
		sp := e.Function.Span
		if !sp.IsSet() {
			sp = e.CallSpan
		}
		return nil, diagnosticf(sp, "go-call", "%s not found in Go package", qual)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		sp := e.Function.Span
		if !sp.IsSet() {
			sp = e.CallSpan
		}
		return nil, diagnosticf(sp, "go-call", "%s is not a function", qual)
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, diagnosticf(e.CallSpan, "go-call", "%s: invalid signature", qual)
	}
	mapped, err := tc.checkGoSignature(sig, qual, e, argTypes, foldErrorPair)
	if err != nil {
		return nil, err
	}
	// go/types is authoritative here (package is loaded). Do not override with BuiltinFunctions
	// entries that may omit error returns (e.g. strconv.Atoi is (int, error) -> Result(Int, Error)).
	return mapped, nil
}

func (tc *TypeChecker) checkGoQualifiedCall(pkg *types.Package, pkgDisplay, funcName string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode, foldErrorPair bool) ([]ast.TypeNode, error) {
	return tc.checkGoFuncCall(pkg, pkgDisplay, funcName, e, argTypes, foldErrorPair)
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
		for i := range fixed {
			if err := tc.checkOneGoParam(qual, i, params.At(i).Type(), argTypes[i], e, i); err != nil {
				return nil, err
			}
		}
		sliceT, ok := params.At(nParams - 1).Type().Underlying().(*types.Slice)
		if !ok {
			return nil, diagnosticf(e.CallSpan, "go-call", "%s: invalid variadic parameter", qual)
		}
		elem := sliceT.Elem()
		if nArgs > fixed {
			if spread, isSpread := e.Arguments[nArgs-1].(ast.SpreadExpressionNode); isSpread {
				if nArgs != fixed+1 {
					sp := spanForCallArg(e.ArgSpans, fixed+1, e.Arguments, e.CallSpan)
					return nil, diagnosticf(sp, "go-call", "%s: variadic spread must be the only trailing argument", qual)
				}
				spreadTypes, err := tc.inferExpressionType(spread.Expr)
				if err != nil {
					return nil, err
				}
				if len(spreadTypes) != 1 {
					return nil, diagnosticf(spanForCallArg(e.ArgSpans, fixed, e.Arguments, e.CallSpan), "go-call", "%s: spread argument must have a single type", qual)
				}
				if err := tc.checkGoSliceSpreadAssignability(qual, elem, spreadTypes[0], spread.Expr, e, fixed); err != nil {
					return nil, err
				}
			} else {
				for j := fixed; j < nArgs; j++ {
					if err := tc.checkOneGoParam(qual, j, elem, argTypes[j], e, j); err != nil {
						return nil, err
					}
				}
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
		for i := range nParams {
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
		// Opaque Go pointer (e.g. *strings.Reader from NewReader) implements io.Reader at the FFI boundary.
		if f.Ident == ast.TypePointer && len(f.TypeParams) == 1 && f.TypeParams[0].Ident == ast.TypeImplicit {
			return true
		}
	}
	// Opaque Go values (named types, method results) are represented as TYPE_IMPLICIT until we have
	// variableGoTypes-backed method typing. At a Go call boundary, trust go/types assignability for
	// any parameter type we can map to Forst (e.g. slices passed to stdlib helpers).
	if f.Ident == ast.TypeImplicit {
		_, ok := goTypeToForstType(g)
		return ok
	}
	if exp, ok := tc.forstTypeForGoType(g); ok {
		return tc.IsTypeCompatible(f, exp)
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
	// Named types (e.g. strings.Reader) must be detected before Underlying(),
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

// goExportedFieldName mirrors the transformer's capitalizeFirst (Go export rule).
func goExportedFieldName(s string) string {
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

func goStructType(recv types.Type) *types.Struct {
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

// goStructFieldTypeForForstName resolves a struct field whose Forst name is forstName,
// matching in order: json tag == forstName, exact field name, capitalized-first-letter name.
func goStructFieldTypeForForstName(recv types.Type, forstName string) (types.Type, bool) {
	st := goStructType(recv)
	if st == nil {
		return nil, false
	}
	return goStructFieldTypeForForstNameOnStruct(st, forstName)
}

func goStructFieldTypeForForstNameOnStruct(st *types.Struct, forstName string) (types.Type, bool) {
	if st == nil || forstName == "" {
		return nil, false
	}
	exported := goExportedFieldName(forstName)
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
	return nil, false
}

// goTypeAtFieldPath resolves exported field selectors on a Go type (e.g. *url.URL then ["Path"]).
// It is used when a Forst local was bound from a Go call (variableGoTypes) so field types match
// go/types instead of stopping at Pointer((implicit)) / (implicit) from goTypeToForstType alone.
func goTypeAtFieldPath(recv types.Type, fieldPath []string) (types.Type, error) {
	if len(fieldPath) == 0 {
		return recv, nil
	}
	name := fieldPath[0]
	var ft types.Type
	var found bool
	if ft, found = goStructFieldTypeForForstName(recv, name); !found {
		obj, _, _ := types.LookupFieldOrMethod(recv, true, nil, name)
		if obj == nil {
			return nil, fmt.Errorf("no field or method %q on %s", name, recv)
		}
		v, ok := obj.(*types.Var)
		if !ok {
			return nil, fmt.Errorf("%q is not a struct field (got %T)", name, obj)
		}
		ft = v.Type()
	}
	if len(fieldPath) == 1 {
		return ft, nil
	}
	return goTypeAtFieldPath(ft, fieldPath[1:])
}

// lookupGoImportedPackageSelector resolves pkg.Symbol for an imported Go package (e.g. os.Args).
func (tc *TypeChecker) lookupGoImportedPackageSelector(local ast.Identifier, fieldPath []string) (ast.TypeNode, error) {
	if len(fieldPath) == 0 {
		return ast.TypeNode{}, fmt.Errorf("package %s used as value", local)
	}
	gp := tc.goPackageForImportLocal(string(local))
	if gp == nil {
		return ast.TypeNode{}, fmt.Errorf("not an imported Go package: %s", local)
	}
	obj := gp.Scope().Lookup(fieldPath[0])
	if obj == nil {
		return ast.TypeNode{}, fmt.Errorf("%s.%s not found in Go package", local, fieldPath[0])
	}
	var goTyp types.Type
	switch o := obj.(type) {
	case *types.Var:
		goTyp = o.Type()
	case *types.Const:
		goTyp = o.Type()
	default:
		return ast.TypeNode{}, fmt.Errorf("%s.%s is not a package variable", local, fieldPath[0])
	}
	if len(fieldPath) > 1 {
		return tc.lookupFieldPathFromGoType(goTyp, fieldPath[1:])
	}
	ft, ok := goTypeToForstType(goTyp)
	if !ok {
		return ast.TypeNode{}, fmt.Errorf("cannot map Go type %s", goTyp)
	}
	return ft, nil
}

// lookupFieldPathFromGoType maps the final field's Go type to a Forst TypeNode using goTypeToForstType.
func (tc *TypeChecker) lookupFieldPathFromGoType(goBase types.Type, fieldPath []string) (ast.TypeNode, error) {
	last, err := goTypeAtFieldPath(goBase, fieldPath)
	if err != nil {
		return ast.TypeNode{}, err
	}
	t, ok := goTypeToForstType(last)
	if !ok {
		return ast.TypeNode{}, fmt.Errorf("cannot map Go type %s", last)
	}
	return t, nil
}

// goTypeDisplayStringForVariablePath returns types.Type.String() for a simple or dotted variable when
// the root name was recorded in variableGoTypes (Go FFI binding). Used for hover when Forst's
// goTypeToForstType mapping is lossy (e.g. named structs as (implicit)).
func (tc *TypeChecker) goTypeDisplayStringForVariablePath(id ast.Identifier) (string, bool) {
	if tc == nil {
		return "", false
	}
	parts := strings.Split(string(id), ".")
	if len(parts) == 0 {
		return "", false
	}
	base := ast.Identifier(parts[0])
	gt, ok := tc.variableGoTypes[base]
	if !ok || gt == nil {
		return "", false
	}
	last, err := goTypeAtFieldPath(gt, parts[1:])
	if err != nil {
		return "", false
	}
	return last.String(), true
}

// goNamedTypeRoot returns the named type at the root of g (unwraps one pointer level).
func goNamedTypeRoot(g types.Type) (*types.Named, bool) {
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

// forstTypeForGoType maps an in-module Go named type to the Forst qualified type used at the
// FFI boundary (e.g. mod/users.User → users.User).
func (tc *TypeChecker) forstTypeForGoType(g types.Type) (ast.TypeNode, bool) {
	named, ok := goNamedTypeRoot(g)
	if !ok {
		return ast.TypeNode{}, false
	}
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return ast.TypeNode{}, false
	}
	pkgPath := pkg.Path()
	modPath := goload.ModulePath(tc.GoWorkspaceDir)
	if modPath == "" || !strings.HasPrefix(pkgPath, modPath+"/") {
		return ast.TypeNode{}, false
	}
	typeName := named.Obj().Name()

	if tc.samePackageGoImportPath == pkgPath {
		if _, ok := tc.Defs[ast.TypeIdent(typeName)]; ok {
			return ast.TypeNode{Ident: ast.TypeIdent(typeName)}, true
		}
	}

	local, ok := tc.ImportLocalForPath(pkgPath)
	if !ok {
		return ast.TypeNode{}, false
	}
	qualified := ast.TypeIdent(local + "." + typeName)

	if importMap := tc.importPathToForstPkgMap(); importMap != nil {
		if importMap[pkgPath] == "" {
			return ast.TypeNode{}, false
		}
		return ast.TypeNode{Ident: qualified}, true
	}

	if tc.goPackageForImportLocal(local) == nil {
		return ast.TypeNode{}, false
	}
	return ast.TypeNode{Ident: qualified}, true
}

// goTypeForQualifiedImportTypeIdent resolves pkg.Type from a Go import (e.g. testing.T) to go/types.
// Returns nil when the qualifier is a Forst sibling package or the symbol is not found.
func (tc *TypeChecker) goTypeForQualifiedImportTypeIdent(typeIdent ast.TypeIdent) types.Type {
	importLocal, symbol, ok := parseForstSiblingTypeRef(typeIdent)
	if !ok {
		return nil
	}
	if importMap := tc.importPathToForstPkgMap(); importMap != nil {
		if path, ok := tc.ImportPathForLocal(importLocal); ok && importMap[path] != "" {
			return nil
		}
	}
	gp := tc.goPackageForImportLocal(importLocal)
	if gp == nil {
		return nil
	}
	obj := gp.Scope().Lookup(symbol)
	if obj == nil {
		return nil
	}
	return obj.Type()
}

func (tc *TypeChecker) goTypeForParamTypeNode(typ ast.TypeNode) types.Type {
	if typ.Ident == ast.TypePointer && len(typ.TypeParams) == 1 {
		if gt := tc.goTypeForQualifiedImportTypeIdent(typ.TypeParams[0].Ident); gt != nil {
			return types.NewPointer(gt)
		}
	}
	return tc.goTypeForQualifiedImportTypeIdent(typ.Ident)
}

func (tc *TypeChecker) bindVariableGoTypeFromParamType(ident ast.Identifier, typ ast.TypeNode) {
	if normalized, ok := tc.normalizeGoImportParamType(typ); ok {
		typ = normalized
	}
	if gt := tc.goTypeForParamTypeNode(typ); gt != nil {
		if named, ok := gt.(*types.Named); ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "testing" && named.Obj().Name() == "T" {
			gt = types.NewPointer(gt)
		}
		tc.variableGoTypes[ident] = gt
	}
}

// normalizeGoImportParamType preserves Go import types in Forst AST form instead of lowering
// them to structural hash aliases (e.g. testing.T → *testing.T for test params).
func (tc *TypeChecker) normalizeGoImportParamType(typ ast.TypeNode) (ast.TypeNode, bool) {
	if !ast.IsTestingTParamType(typ) {
		return ast.TypeNode{}, false
	}
	if typ.Ident == ast.TypePointer {
		if len(typ.TypeParams) == 1 {
			tc.registerGoQualifiedTypeAlias(typ.TypeParams[0].Ident, typ.TypeParams[0].Ident)
		}
		return typ, true
	}
	qualified := typ.Ident
	normalized := ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: qualified}},
	}
	tc.registerGoQualifiedTypeAlias(qualified, qualified)
	return normalized, true
}

// goTypeForExpression returns the go/types type of a Go interop expression when known.
func (tc *TypeChecker) goTypeForExpression(expr ast.ExpressionNode) types.Type {
	if tc == nil || expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case ast.VariableNode:
		if gt := tc.variableGoTypes[e.Ident.ID]; gt != nil {
			return gt
		}
	case ast.FunctionCallNode:
		if sig := tc.goFuncSignatureFromCall(e); sig != nil && sig.Results().Len() > 0 {
			return sig.Results().At(0).Type()
		}
	case ast.MethodCallNode:
		if goRecv := tc.goTypeForExpression(e.Receiver); goRecv != nil {
			obj, _, _ := types.LookupFieldOrMethod(goRecv, true, nil, string(e.Method.ID))
			if fn, ok := obj.(*types.Func); ok {
				if sig, ok := fn.Type().(*types.Signature); ok && sig.Results().Len() > 0 {
					return sig.Results().At(0).Type()
				}
			}
		}
	case ast.FieldAccessNode:
		if goRecv := tc.goTypeForExpression(e.Target); goRecv != nil {
			obj, _, _ := types.LookupFieldOrMethod(goRecv, false, nil, string(e.Field.ID))
			if obj != nil {
				return obj.Type()
			}
		}
	case ast.SliceExpressionNode:
		if goT := tc.goTypeForExpression(e.Target); goT != nil {
			switch u := goT.Underlying().(type) {
			case *types.Slice:
				return types.NewSlice(u.Elem())
			case *types.Array:
				return types.NewSlice(u.Elem())
			}
		}
	}
	return nil
}

func (tc *TypeChecker) checkGoSliceSpreadAssignability(qual string, elem types.Type, spreadType ast.TypeNode, spreadExpr ast.ExpressionNode, e ast.FunctionCallNode, argIdx int) error {
	if spreadType.Ident != ast.TypeArray || len(spreadType.TypeParams) != 1 {
		sp := spanForCallArg(e.ArgSpans, argIdx, e.Arguments, e.CallSpan)
		return diagnosticf(sp, "go-call", "%s: spread argument must be a slice, got %s", qual, spreadType.Ident)
	}
	wantSlice := types.NewSlice(elem)
	if !tc.forstAssignableToGoType(spreadType, wantSlice) {
		sp := spanForCallArg(e.ArgSpans, argIdx, e.Arguments, e.CallSpan)
		return diagnosticf(sp, "go-call", "%s: cannot spread %s into ...%s", qual, spreadType.Ident, elem.String())
	}
	_ = spreadExpr
	return nil
}

// checkGoMethodCall type-checks a method call using go/types when the receiver has a tracked Go type.
func (tc *TypeChecker) checkGoMethodCall(recv types.Type, methodName string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode, foldErrorPair bool) ([]ast.TypeNode, error) {
	obj, _, _ := types.LookupFieldOrMethod(recv, true, nil, methodName)
	if obj == nil {
		sp := e.CallSpan
		if !sp.IsSet() {
			sp = e.Function.Span
		}
		return nil, diagnosticf(sp, "go-method", "%s has no field or method %s", recv.String(), methodName)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		sp := e.CallSpan
		if !sp.IsSet() {
			sp = e.Function.Span
		}
		return nil, diagnosticf(sp, "go-method", "%s.%s is not a method", recv.String(), methodName)
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, diagnosticf(e.CallSpan, "go-method", "invalid method signature")
	}
	qual := fmt.Sprintf("(%s).%s", recv.String(), methodName)
	return tc.checkGoSignature(sig, qual, e, argTypes, foldErrorPair)
}

// bindVariableGoTypesFromCall records go/types result types for each LHS of a Go function call when
// the Go result arity matches the assignment (single- and multi-value).
func (tc *TypeChecker) bindVariableGoTypesFromCall(assign ast.AssignmentNode) {
	if len(assign.RValues) != 1 {
		return
	}
	fc, ok := assign.RValues[0].(ast.FunctionCallNode)
	if !ok {
		return
	}
	sig := tc.goFuncSignatureFromCall(fc)
	if sig == nil {
		return
	}
	res := sig.Results()
	if res.Len() != len(assign.LValues) {
		return
	}
	for i, lv := range assign.LValues {
		vn, ok := lv.(ast.VariableNode)
		if !ok {
			continue
		}
		tc.variableGoTypes[vn.Ident.ID] = res.At(i).Type()
	}
}

func (tc *TypeChecker) goFuncSignatureFromCall(fc ast.FunctionCallNode) *types.Signature {
	parts := strings.Split(string(fc.Function.ID), ".")
	switch len(parts) {
	case 2:
		gp := tc.goPackageForImportLocal(parts[0])
		if gp == nil {
			return nil
		}
		return tc.goFuncSignatureInPackage(gp, parts[1])
	case 1:
		if tc.samePackageGo == nil || !goIdentifierExported(parts[0]) {
			return nil
		}
		return tc.goFuncSignatureInPackage(tc.samePackageGo, parts[0])
	default:
		return nil
	}
}

func (tc *TypeChecker) goFuncSignatureInPackage(pkg *types.Package, funcName string) *types.Signature {
	obj := pkg.Scope().Lookup(funcName)
	if obj == nil {
		return nil
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil
	}
	return sig
}
