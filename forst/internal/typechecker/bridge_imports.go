package typechecker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"forst/internal/ast"
	"forst/internal/ftconfig"
	"forst/internal/importlocal"
	"forst/internal/bridgeinterop"
)

type nodeImportBinding struct {
	Import   ast.ImportNode
	ModuleID string
	AbsPath  string
	Index    *bridgeinterop.IndexV1
}

// NodeRuntimeState holds compile-time Node interop facts for the compiler pipeline.
type NodeRuntimeState struct {
	NeedsNodeRuntime bool
	Manifest         bridgeinterop.ManifestV1
}

// NodeRuntimeState returns node runtime facts for this typecheck.
func (tc *TypeChecker) NodeRuntimeState() NodeRuntimeState {
	if tc == nil {
		return NodeRuntimeState{}
	}
	return NodeRuntimeState{
		NeedsNodeRuntime: tc.nodeRuntime.NeedsNodeRuntime,
		Manifest:         tc.nodeRuntime.Manifest,
	}
}

func (tc *TypeChecker) nodeBoundaryRoot() string {
	if tc.NodeBoundaryRoot != "" {
		return tc.NodeBoundaryRoot
	}
	return tc.goPackagesLoadDir()
}

func (tc *TypeChecker) collectImportNode(imp ast.ImportNode) error {
	if imp.BridgeOptIn {
		tc.nodeImports = append(tc.nodeImports, imp)
		return nil
	}
	if hint := tc.tsImportWithoutOptInHint(imp); hint != "" {
		if tc.nodeImportPolicyImplicit() {
			imp.BridgeOptIn = true
			imp.BridgeOptInSource = "implicit_policy"
			tc.nodeImports = append(tc.nodeImports, imp)
			return nil
		}
		return diagnosticf(ast.SourceSpan{}, "js-import",
			"cannot import TypeScript module without import \"./path\" js or import alias \"./path\" js (%s found)", hint)
	}
	if err := tc.validateGoImportAtCollect(imp); err != nil {
		return err
	}
	tc.imports = append(tc.imports, imp)
	return nil
}

func (tc *TypeChecker) nodeImportPolicyImplicit() bool {
	return tc != nil && tc.NodeImportPolicy == "implicit"
}

func (tc *TypeChecker) tsImportWithoutOptInHint(imp ast.ImportNode) string {
	if tc == nil || imp.BridgeOptIn {
		return ""
	}
	path := strings.TrimSpace(imp.Path)
	if !strings.HasPrefix(path, "./") && !strings.HasPrefix(path, "../") {
		return ""
	}
	entryDir := tc.ForstFileDir
	if entryDir == "" {
		entryDir = tc.nodeBoundaryRoot()
	}
	if entryDir == "" {
		return ""
	}
	return bridgeinterop.TSImportHintFile(entryDir, path)
}

func (tc *TypeChecker) resolveNodeImports() error {
	if len(tc.nodeImports) == 0 {
		tc.nodeRuntime = NodeRuntimeInfo{}
		return nil
	}
	if tc.nodeImportsByLocal == nil {
		tc.nodeImportsByLocal = make(map[string]nodeImportBinding)
	}
	if tc.nodeIndexResolver == nil {
		tc.nodeIndexResolver = bridgeinterop.NewIndexResolver()
	}

	root := tc.nodeBoundaryRoot()
	forstDir := root
	if tc.ForstFileDir != "" {
		forstDir = tc.ForstFileDir
	}
	boundaryRoot := root
	if br, err := bridgeinterop.BoundaryRootFromEntry(forstDir); err == nil {
		boundaryRoot = br
	} else if tc.NodeBoundaryRoot != "" {
		boundaryRoot = tc.NodeBoundaryRoot
	}

	var loadedIndexes []*bridgeinterop.IndexV1
	for _, imp := range tc.nodeImports {
		moduleID, absPath, err := tc.resolveNodeImportPath(root, forstDir, imp.Path)
		if err != nil {
			return diagnosticf(ast.SourceSpan{}, "js-import", "%s", err.Error())
		}
		binding := importlocal.BindingFromAST(imp, importlocal.BindingOpts{
			Kind:     importlocal.KindBridge,
			ModuleID: moduleID,
		})
		seen := make(map[string]string, len(tc.nodeImportsByLocal))
		for name := range tc.nodeImportsByLocal {
			seen[name] = name
		}
		if err := tc.validateNodeImportLocal(binding, seen); err != nil {
			return err
		}
		local := binding.Local
		if absPath != "" {
			if _, err := os.Stat(absPath); err != nil {
				return diagnosticf(ast.SourceSpan{}, "js-import", "TypeScript module not found: %s", absPath)
			}
		}

		idx, err := tc.loadNodeModuleIndex(boundaryRoot, moduleID)
		if err != nil {
			return diagnosticf(ast.SourceSpan{}, "js-import", "failed to load index for %q: %v", moduleID, err)
		}
		if idx.ModuleID != moduleID {
			return diagnosticf(ast.SourceSpan{}, "js-import", "index moduleId %q does not match resolved module %q", idx.ModuleID, moduleID)
		}

		tc.nodeImportsByLocal[local] = nodeImportBinding{
			Import:   imp,
			ModuleID: moduleID,
			AbsPath:  absPath,
			Index:    idx,
		}
		loadedIndexes = append(loadedIndexes, idx)
	}

	manifest, err := bridgeinterop.BuildManifestV1(boundaryRoot, indexesToForstIndexSlice(loadedIndexes))
	if err != nil {
		return diagnosticf(ast.SourceSpan{}, "js-import", "failed to build bridge manifest: %v", err)
	}

	bridge, bridgeErr := ftconfig.LoadFromDir(boundaryRoot)
	if bridgeErr != nil {
		return diagnosticf(ast.SourceSpan{}, "js-import", "failed to load ftconfig: %v", bridgeErr)
	}
	effectiveBridge, bridgeErr := ftconfig.EffectiveBridge(bridge)
	if bridgeErr != nil {
		return diagnosticf(ast.SourceSpan{}, "js-import", "%v", bridgeErr)
	}
	if effectiveBridge.ModuleFormat == ftconfig.LegacyModuleCompiled {
		sourceIDs := make([]string, 0, len(loadedIndexes))
		for _, idx := range loadedIndexes {
			if idx != nil {
				sourceIDs = append(sourceIDs, idx.ModuleID)
			}
		}
		if err := bridgeinterop.PrecompileEntries(boundaryRoot, sourceIDs, effectiveBridge.OutDir); err != nil {
			return diagnosticf(ast.SourceSpan{}, "js-import", "precompile legacy modules: %v", err)
		}
		manifest = bridgeinterop.RemapManifestModuleIDs(manifest)
	}

	manifestJSON, err := manifest.EmbeddedManifestJSON()
	if err != nil {
		return diagnosticf(ast.SourceSpan{}, "js-import", "failed to encode bridge manifest: %v", err)
	}
	tc.nodeRuntime = NodeRuntimeInfo{
		NeedsNodeRuntime: true,
		Manifest:         manifest,
		ManifestJSON:     string(manifestJSON),
	}
	return nil
}

func indexesToForstIndexSlice(indexes []*bridgeinterop.IndexV1) []bridgeinterop.ForstIndexV1 {
	out := make([]bridgeinterop.ForstIndexV1, 0, len(indexes))
	for _, idx := range indexes {
		if idx != nil {
			out = append(out, *idx)
		}
	}
	return out
}

func (tc *TypeChecker) resolveNodeImportPath(boundaryRoot, forstFileDir, importPath string) (moduleID, absPath string, err error) {
	if bridgeinterop.IsBareNodeSpecifier(importPath) {
		return importPath, "", nil
	}
	moduleID, absPath, err = bridgeinterop.ResolveTSImport(forstFileDir, importPath)
	if err == nil {
		return moduleID, absPath, nil
	}
	// Test harness / explicit boundary: resolve relative to NodeBoundaryRoot without ftconfig.
	if boundaryRoot != "" && boundaryRoot != "." {
		return resolveNodeImportUnderRoot(boundaryRoot, forstFileDir, importPath)
	}
	return "", "", err
}

func resolveNodeImportUnderRoot(boundaryRoot, forstFileDir, importPath string) (moduleID, absPath string, err error) {
	importPath = strings.TrimSpace(importPath)
	if importPath == "" {
		return "", "", fmt.Errorf("empty JS import path")
	}

	baseDir := boundaryRoot
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		baseDir = forstFileDir
	}

	clean := filepath.Clean(importPath)
	for _, rel := range nodeImportResolutionCandidates(clean) {
		abs := filepath.Join(baseDir, rel)
		if st, statErr := os.Stat(abs); statErr == nil && !st.IsDir() {
			relModule, relErr := filepath.Rel(boundaryRoot, abs)
			if relErr != nil {
				return "", "", relErr
			}
			relModule = filepath.ToSlash(relModule)
			if strings.HasPrefix(relModule, "../") {
				return "", "", fmt.Errorf("TypeScript module outside boundary: %s", importPath)
			}
			return relModule, abs, nil
		}
	}
	return "", "", fmt.Errorf("TypeScript module not found for import %q", importPath)
}

func nodeImportResolutionCandidates(cleanPath string) []string {
	ext := strings.ToLower(filepath.Ext(cleanPath))
	if ext != "" {
		return []string{cleanPath}
	}
	base := cleanPath
	return []string{
		base + ".ft",
		base + ".go",
		base + ".ts",
		base + ".tsx",
		base + ".js",
		filepath.Join(base, "index.ts"),
		filepath.Join(base, "index.tsx"),
		filepath.Join(base, "index.js"),
	}
}

func (tc *TypeChecker) nodeModuleForLocal(local string) (nodeImportBinding, bool) {
	if tc.nodeImportsByLocal == nil {
		return nodeImportBinding{}, false
	}
	b, ok := tc.nodeImportsByLocal[local]
	return b, ok
}

// NodeImportCount returns opted-in TypeScript imports collected for this check.
func (tc *TypeChecker) NodeImportCount() int {
	if tc == nil {
		return 0
	}
	return len(tc.nodeImports)
}

func (tc *TypeChecker) isNodeImportLocal(local string) bool {
	_, ok := tc.nodeModuleForLocal(local)
	return ok
}

// IsNodeImportLocal reports whether local names a node TypeScript import.
func (tc *TypeChecker) IsNodeImportLocal(local string) bool {
	return tc.isNodeImportLocal(local)
}

// NodeImportModuleID returns the project-relative module id for a JS import local name.
func (tc *TypeChecker) NodeImportModuleID(local string) (string, bool) {
	mod, ok := tc.nodeModuleForLocal(local)
	if !ok {
		return "", false
	}
	return mod.ModuleID, true
}

func (tc *TypeChecker) loadNodeModuleIndex(boundaryRoot, moduleID string) (*bridgeinterop.IndexV1, error) {
	indexes, err := bridgeinterop.RunIndexer(boundaryRoot, []string{moduleID})
	if err != nil {
		return nil, err
	}
	if len(indexes) == 0 || indexes[0] == nil {
		return nil, fmt.Errorf("indexer returned no data for %q", moduleID)
	}
	idx := indexes[0]
	if err := tc.nodeIndexResolver.Register(idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func (tc *TypeChecker) tryNodeQualifiedCall(pkgLocal, funcName string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode) ([]ast.TypeNode, bool, error) {
	mod, ok := tc.nodeModuleForLocal(pkgLocal)
	if !ok {
		return nil, false, nil
	}
	if tc.nodeIndexResolver == nil {
		return nil, true, diagnosticf(e.CallSpan, "node-call", "node index resolver not initialized")
	}
	params, returns, kind, widened, err := tc.nodeIndexResolver.ExportSignatureWithWarnings(mod.ModuleID, funcName)
	if err != nil {
		sp := e.CallSpan
		if !sp.IsSet() {
			sp = e.Function.Span
		}
		return nil, true, diagnosticf(sp, "node-call", "%s.%s: %v", pkgLocal, funcName, err)
	}
	for _, path := range widened {
		tc.warnf(e.CallSpan, "node-type-widened",
			"TS type widened to Object for %s.%s %s; narrow the export or add explicit fields", pkgLocal, funcName, path)
	}
	if len(argTypes) != len(params) {
		sp := e.CallSpan
		if len(argTypes) > len(params) {
			sp = spanForCallArg(e.ArgSpans, len(params), e.Arguments, e.CallSpan)
		}
		return nil, true, diagnosticf(sp, "node-call", "%s.%s expects %d arguments, got %d",
			pkgLocal, funcName, len(params), len(argTypes))
	}
	for i, param := range params {
		sp := spanForCallArg(e.ArgSpans, i, e.Arguments, e.CallSpan)
		if len(argTypes[i]) != 1 {
			return nil, true, diagnosticf(sp, "node-call", "%s.%s argument %d must have a single type, got %d",
				pkgLocal, funcName, i+1, len(argTypes[i]))
		}
		if !tc.IsTypeCompatible(argTypes[i][0], param) {
			return nil, true, diagnosticf(sp, "node-call", "%s.%s argument %d: expected type %s, got %s",
				pkgLocal, funcName, i+1, param.Ident, argTypes[i][0].Ident)
		}
	}
	if kind == NodeExportKindFunction || kind == NodeExportKindAsyncFunction ||
		kind == NodeExportKindGenerator || kind == NodeExportKindAsyncGenerator {
		wrapped := make([]ast.TypeNode, len(returns))
		errType := ast.NewBuiltinType(ast.TypeError)
		for i, ret := range returns {
			wrapped[i] = ast.NewResultType(ret, errType)
		}
		return wrapped, true, nil
	}
	return returns, true, nil
}
