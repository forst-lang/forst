package semantic

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"forst/internal/ast"
	"forst/internal/compiler"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	"forst/internal/typechecker"
	transformerts "forst/internal/transformer/ts"

	"github.com/sirupsen/logrus"
)

// BuildSnapshot typechecks discovered .ft files and projects a semantic snapshot.
func BuildSnapshot(filePaths []string, boundaryRoot string, log *logrus.Logger) (*GenerateRequest, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no Forst files for semantic snapshot")
	}
	modResult, packages, err := transformerts.TypecheckDiscoveredPackages(filePaths, log, false)
	if err != nil {
		return nil, err
	}
	moduleRoot := modResult.ModuleRoot
	if moduleRoot == "" {
		moduleRoot = goload.FindModuleRoot(filePaths[0])
	}
	goModule := modResult.ModulePath
	if goModule == "" {
		goModule = goload.ModulePath(moduleRoot)
	}
	if boundaryRoot == "" {
		boundaryRoot = moduleRoot
	}

	req := &GenerateRequest{
		ProtocolVersion: ProtocolVersion,
		CompilerVersion: compiler.CompilerVersion(),
		Module: ModuleInfo{
			GoModule: goModule,
			Root:     filepath.Clean(boundaryRoot),
		},
		Types:     make(map[string]Type),
		Functions: make(map[string]Function),
	}

	for _, pkg := range packages {
		sp, err := projectPackage(req, modResult, pkg, boundaryRoot)
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", pkg.Name, err)
		}
		req.Packages = append(req.Packages, sp)
	}
	sort.Slice(req.Packages, func(i, j int) bool {
		return req.Packages[i].Name < req.Packages[j].Name
	})
	return req, nil
}

type projector struct {
	req          *GenerateRequest
	tc           *typechecker.TypeChecker
	pkg          string
	boundaryRoot string
	fileByNode   map[string]string // abs path keyed by... we'll track per-file instead
	typeMemo     map[string]string // typeNode key -> id
	counter      int
	funcNames    map[string]struct{}
}

func projectPackage(req *GenerateRequest, mod *modulecheck.ModuleResult, pkg transformerts.PackageTypecheck, boundaryRoot string) (SemanticPackage, error) {
	p := &projector{
		req:          req,
		tc:           pkg.TC,
		pkg:          pkg.Name,
		boundaryRoot: boundaryRoot,
		typeMemo:     make(map[string]string),
		funcNames:    make(map[string]struct{}),
	}
	for _, node := range pkg.Nodes {
		if fn, ok := node.(ast.FunctionNode); ok {
			p.funcNames[string(fn.Ident.ID)] = struct{}{}
		}
	}

	var typeIDs []string
	seenTypes := make(map[string]struct{})
	for _, node := range pkg.Nodes {
		td, ok := node.(ast.TypeDefNode)
		if !ok || isInternalHashName(string(td.Ident)) {
			continue
		}
		id := p.namedID(string(td.Ident))
		if err := p.ensureTypeDef(id, td, ""); err != nil {
			return SemanticPackage{}, err
		}
		if _, ok := seenTypes[id]; !ok {
			seenTypes[id] = struct{}{}
			typeIDs = append(typeIDs, id)
		}
	}
	// Include hash types referenced from projected types but not declared at top level.
	for ident, def := range pkg.TC.Defs {
		if !isInternalHashName(string(ident)) {
			continue
		}
		td, ok := def.(ast.TypeDefNode)
		if !ok {
			continue
		}
		id := hashStructuralID(ident)
		if _, exists := p.req.Types[id]; exists {
			continue
		}
		if err := p.projectHashType(id, ident); err != nil {
			return SemanticPackage{}, err
		}
		_ = td
	}
	sort.Strings(typeIDs)

	var functionIDs []string
	for _, path := range pkg.Paths {
		relFile := moduleRelativeFile(boundaryRoot, path)
		nodes, err := nodesForPath(pkg, path)
		if err != nil {
			return SemanticPackage{}, err
		}
		var pendingDoc string
		for _, node := range nodes {
			if c, ok := node.(ast.CommentNode); ok {
				pendingDoc = docFromComment(c.Text)
				continue
			}
			fn, ok := node.(ast.FunctionNode)
			if !ok {
				pendingDoc = ""
				continue
			}
			fid, err := p.projectFunction(fn, relFile, pendingDoc)
			pendingDoc = ""
			if err != nil {
				return SemanticPackage{}, err
			}
			if fid != "" {
				functionIDs = append(functionIDs, fid)
			}
		}
	}

	dir, files := packageLayout(boundaryRoot, pkg.Paths)
	sort.Strings(functionIDs)
	return SemanticPackage{
		Name:        pkg.Name,
		Dir:         dir,
		Files:       files,
		TypeIDs:     typeIDs,
		FunctionIDs: functionIDs,
	}, nil
}

func nodesForPath(pkg transformerts.PackageTypecheck, path string) ([]ast.Node, error) {
	clean := filepath.Clean(path)
	for _, p := range pkg.Paths {
		if filepath.Clean(p) == clean {
			// Re-parse single file nodes from merged package by filtering is hard;
			// pkg.Paths order matches discovery — use PerPackageNodes only for merged.
			// For spans we parse the file again for top-level decls only.
			return transformerts.ParseFileTopLevelNodes(path)
		}
	}
	return nil, fmt.Errorf("path %s not in package %s", path, pkg.Name)
}

func (p *projector) namedID(name string) string {
	return p.pkg + "." + name
}

func (p *projector) synthID(prefix string) string {
	p.counter++
	return fmt.Sprintf("t:%s.%s.%d", p.pkg, prefix, p.counter)
}

func isInternalHashName(name string) bool {
	return strings.HasPrefix(name, "T_")
}

func hashStructuralID(ident ast.TypeIdent) string {
	s := string(ident)
	if strings.HasPrefix(s, "T_") {
		return "structural:" + strings.TrimPrefix(s, "T_")
	}
	return "structural:" + s
}

func (p *projector) ensureTypeDef(id string, td ast.TypeDefNode, doc string) error {
	if _, ok := p.req.Types[id]; ok {
		return nil
	}
	t, err := p.typeFromDef(id, td, doc)
	if err != nil {
		return err
	}
	p.req.Types[id] = t
	return nil
}

func (p *projector) typeFromDef(id string, td ast.TypeDefNode, doc string) (Type, error) {
	vis := "exported"
	if !ast.IsPublicExportIdent(ast.Identifier(td.Ident)) {
		vis = "internal"
	}
	switch expr := td.Expr.(type) {
	case ast.TypeDefAssertionExpr:
		return p.typeFromAssertion(id, expr.Assertion, vis, doc, td.Ident)
	case ast.TypeDefShapeExpr:
		fields, err := p.projectShapeFields(id, expr.Shape, nil)
		if err != nil {
			return Type{}, err
		}
		return Type{ID: id, Kind: "shape", Fields: fields, Visibility: vis, Doc: doc}, nil
	case ast.TypeDefErrorExpr:
		payloadID, err := p.projectShapeType(id+".payload", expr.Payload)
		if err != nil {
			return Type{}, err
		}
		return Type{ID: id, Kind: "nominalError", Payload: payloadID, Visibility: vis, Doc: doc}, nil
	case ast.TypeDefBinaryExpr:
		members, err := p.projectBinaryMembers(expr)
		if err != nil {
			return Type{}, err
		}
		kind := "union"
		if expr.IsConjunction() {
			kind = "intersection"
		}
		return Type{ID: id, Kind: kind, Members: members, Visibility: vis, Doc: doc}, nil
	default:
		return Type{ID: id, Kind: "unknown", Debug: fmt.Sprintf("%T", expr), Visibility: vis, Doc: doc}, nil
	}
}

func (p *projector) typeFromAssertion(id string, a *ast.AssertionNode, vis, doc string, _ ast.TypeIdent) (Type, error) {
	if a == nil {
		return Type{ID: id, Kind: "unknown", Visibility: vis, Doc: doc}, nil
	}
	baseKind, baseID, err := p.resolveAssertionBase(id, a)
	if err != nil {
		return Type{}, err
	}
	constraints := projectConstraints(p.tc, baseKind, a)
	if baseKind == "shape" && baseID != "" {
		t := p.req.Types[baseID]
		t.ID = id
		t.Visibility = vis
		t.Doc = doc
		if len(constraints) > 0 {
			t.Constraints = constraints
		}
		return t, nil
	}
	if isPrimitiveKind(baseKind) {
		return Type{
			ID:          id,
			Kind:        baseKind,
			Constraints: constraints,
			Visibility:  vis,
			Doc:         doc,
		}, nil
	}
	if baseID != "" {
		return Type{
			ID:         id,
			Kind:       "alias",
			Underlying: baseID,
			Visibility: vis,
			Doc:        doc,
		}, nil
	}
	return Type{ID: id, Kind: baseKind, Constraints: constraints, Visibility: vis, Doc: doc}, nil
}

func isPrimitiveKind(kind string) bool {
	switch kind {
	case "string", "int", "float", "bool", "bytes", "void", "error":
		return true
	default:
		return false
	}
}

func (p *projector) resolveAssertionBase(contextID string, a *ast.AssertionNode) (kind, typeID string, err error) {
	if a.BaseType == nil {
		return "unknown", "", nil
	}
	base := *a.BaseType
	if base == ast.TypeShape {
		for _, c := range a.Constraints {
			if c.Name != typechecker.ConstraintMatch {
				continue
			}
			for _, arg := range c.Args {
				if arg.Shape != nil {
					id := p.synthID("shape")
					fields, err := p.projectShapeFields(id, *arg.Shape, a)
					if err != nil {
						return "", "", err
					}
					p.req.Types[id] = Type{
						ID:          id,
						Kind:        "shape",
						Fields:      fields,
						Constraints: projectConstraints(p.tc, "shape", a),
					}
					return "shape", id, nil
				}
			}
		}
		return "shape", "", nil
	}
	tn := ast.TypeNode{Ident: base}
	if p.tc != nil {
		tn = p.tc.GetMostSpecificNonHashAlias(tn)
	}
	return p.projectTypeNode(tn, contextID)
}

func (p *projector) projectTypeNode(tn ast.TypeNode, context string) (kind, id string, err error) {
	key := tn.String()
	if memo, ok := p.typeMemo[key]; ok {
		t := p.req.Types[memo]
		return t.Kind, memo, nil
	}
	id, t, err := p.materializeTypeNode(tn, context)
	if err != nil {
		return "", "", err
	}
	if id != "" {
		p.typeMemo[key] = id
		if _, exists := p.req.Types[id]; !exists {
			p.req.Types[id] = t
		}
	}
	return t.Kind, id, nil
}

func (p *projector) materializeTypeNode(tn ast.TypeNode, context string) (string, Type, error) {
	if tn.Assertion != nil {
		id := p.synthID("assert")
		kind, baseID, err := p.resolveAssertionBase(id, tn.Assertion)
		if err != nil {
			return "", Type{}, err
		}
		t := Type{
			ID:          id,
			Kind:        kind,
			Constraints: projectConstraints(p.tc, kind, tn.Assertion),
		}
		if baseID != "" && kind != "shape" {
			t.Underlying = baseID
			if kind == "string" || kind == "int" || kind == "float" || kind == "bool" {
				// keep kind as primitive
			} else {
				t.Kind = "alias"
			}
		}
		if kind == "shape" && baseID != "" {
			if base, ok := p.req.Types[baseID]; ok {
				t.Fields = base.Fields
			}
		}
		return id, t, nil
	}

	switch tn.Ident {
	case ast.TypeString:
		return "", Type{Kind: "string"}, nil
	case ast.TypeInt:
		return "", Type{Kind: "int"}, nil
	case ast.TypeFloat:
		return "", Type{Kind: "float"}, nil
	case ast.TypeBool:
		return "", Type{Kind: "bool"}, nil
	case ast.TypeBytes:
		return "", Type{Kind: "bytes"}, nil
	case ast.TypeVoid:
		return "", Type{Kind: "void"}, nil
	case ast.TypeError:
		return "", Type{Kind: "error"}, nil
	case ast.TypeArray:
		elemKind, elemID, err := p.projectTypeNode(tn.TypeParams[0], context+".elem")
		if err != nil {
			return "", Type{}, err
		}
		id := p.synthID("array")
		t := Type{ID: id, Kind: "array", Element: elemID}
		_ = elemKind
		if tn.ArrayLen != nil {
			n := int(*tn.ArrayLen)
			t.Length = &n
		}
		return id, t, nil
	case ast.TypeMap:
		keyID, _, err := p.projectTypeNode(tn.TypeParams[0], context+".key")
		if err != nil {
			return "", Type{}, err
		}
		valID, _, err := p.projectTypeNode(tn.TypeParams[1], context+".value")
		if err != nil {
			return "", Type{}, err
		}
		id := p.synthID("map")
		return id, Type{ID: id, Kind: "map", Key: keyID, Value: valID}, nil
	case ast.TypePointer:
		innerID, _, err := p.projectTypeNode(tn.TypeParams[0], context+".ptr")
		if err != nil {
			return "", Type{}, err
		}
		id := p.synthID("ptr")
		return id, Type{ID: id, Kind: "pointer", Inner: innerID}, nil
	case ast.TypeResult:
		sID, _, err := p.projectTypeNode(tn.TypeParams[0], context+".ok")
		if err != nil {
			return "", Type{}, err
		}
		fID, _, err := p.projectTypeNode(tn.TypeParams[1], context+".err")
		if err != nil {
			return "", Type{}, err
		}
		id := p.synthID("result")
		return id, Type{ID: id, Kind: "result", Success: sID, Failure: fID}, nil
	case ast.TypeTuple:
		var members []string
		for i, m := range tn.TypeParams {
			mID, _, err := p.projectTypeNode(m, fmt.Sprintf("%s.t%d", context, i))
			if err != nil {
				return "", Type{}, err
			}
			members = append(members, mID)
		}
		id := p.synthID("tuple")
		return id, Type{ID: id, Kind: "tuple", Members: members}, nil
	case ast.TypeChannel:
		elemID, _, err := p.projectTypeNode(tn.TypeParams[0], context+".chan")
		if err != nil {
			return "", Type{}, err
		}
		id := p.synthID("channel")
		return id, Type{ID: id, Kind: "channel", Element: elemID}, nil
	case ast.TypeUnion, ast.TypeIntersection:
		var members []string
		for i, m := range tn.TypeParams {
			mID, _, err := p.projectTypeNode(m, fmt.Sprintf("%s.m%d", context, i))
			if err != nil {
				return "", Type{}, err
			}
			members = append(members, mID)
		}
		kind := "union"
		if tn.Ident == ast.TypeIntersection {
			kind = "intersection"
		}
		id := p.synthID(kind)
		return id, Type{ID: id, Kind: kind, Members: members}, nil
	default:
		name := string(tn.Ident)
		if isInternalHashName(name) {
			id := hashStructuralID(tn.Ident)
			if _, ok := p.req.Types[id]; !ok {
				if err := p.projectHashType(id, tn.Ident); err != nil {
					return "", Type{}, err
				}
			}
			return id, p.req.Types[id], nil
		}
		if def, ok := p.tc.Defs[tn.Ident]; ok {
			nid := p.namedID(name)
			if td, ok := def.(ast.TypeDefNode); ok {
				if err := p.ensureTypeDef(nid, td, ""); err != nil {
					return "", Type{}, err
				}
				return nid, p.req.Types[nid], nil
			}
		}
		id := p.namedID(name)
		return id, Type{ID: id, Kind: "unknown", Debug: name}, nil
	}
}

func (p *projector) projectHashType(id string, ident ast.TypeIdent) error {
	if _, ok := p.req.Types[id]; ok {
		return nil
	}
	def, ok := p.tc.Defs[ident]
	if !ok {
		p.req.Types[id] = Type{ID: id, Kind: "unknown", Debug: string(ident)}
		return nil
	}
	td, ok := def.(ast.TypeDefNode)
	if !ok {
		p.req.Types[id] = Type{ID: id, Kind: "unknown", Debug: string(ident)}
		return nil
	}
	t, err := p.typeFromDef(id, td, "")
	if err != nil {
		return err
	}
	p.req.Types[id] = t
	return nil
}

func (p *projector) projectShapeType(prefix string, shape ast.ShapeNode) (string, error) {
	id := p.synthID(prefix)
	fields, err := p.projectShapeFields(id, shape, nil)
	if err != nil {
		return "", err
	}
	p.req.Types[id] = Type{ID: id, Kind: "shape", Fields: fields}
	return id, nil
}

func (p *projector) projectShapeFields(ownerID string, shape ast.ShapeNode, shapeConstraints *ast.AssertionNode) ([]ShapeField, error) {
	var fields []ShapeField
	for _, name := range ast.ShapeFieldNamesInOrder(shape.Fields, shape.FieldOrder) {
		field := shape.Fields[name]
		sf := ShapeField{Name: name, Embedded: field.Embedded, Tag: field.Tag}
		if field.IsMethodField() {
			sf.Method = true
			sigID, err := p.projectMethodSig(ownerID, name, field)
			if err != nil {
				return nil, err
			}
			sf.Type = sigID
			if fnID := p.boundFunctionID(name); fnID != "" {
				sf.Function = fnID
			}
		} else if field.Type != nil {
			kind, typeID, err := p.projectTypeNode(*field.Type, ownerID+"."+name)
			if err != nil {
				return nil, err
			}
			if typeID == "" && isPrimitiveKind(kind) {
				typeID = kind
			}
			sf.Type = typeID
		} else if field.Assertion != nil {
			_, typeID, err := p.projectTypeNode(ast.TypeNode{
				Ident:     ast.TypeAssertion,
				Assertion: field.Assertion,
			}, ownerID+"."+name)
			if err != nil {
				return nil, err
			}
			sf.Type = typeID
		}
		fields = append(fields, sf)
	}
	if shapeConstraints != nil {
		if t, ok := p.req.Types[ownerID]; ok {
			t.Constraints = projectConstraints(p.tc, "shape", shapeConstraints)
			p.req.Types[ownerID] = t
		}
	}
	return fields, nil
}

func (p *projector) projectMethodSig(ownerID, name string, field ast.ShapeFieldNode) (string, error) {
	sigID := ownerID + "." + name + ".sig"
	var params []FuncParam
	for _, param := range field.MethodParams {
		pn := paramName(param)
		_, typeID, err := p.projectTypeNode(param.GetType(), sigID+"."+pn)
		if err != nil {
			return "", err
		}
		params = append(params, FuncParam{Name: pn, Type: typeID, Variadic: isVariadicParam(param)})
	}
	var returns []string
	for i, rt := range field.MethodReturnTypes {
		rid, _, err := p.projectTypeNode(rt, fmt.Sprintf("%s.ret%d", sigID, i))
		if err != nil {
			return "", err
		}
		returns = append(returns, rid)
	}
	p.req.Types[sigID] = Type{
		ID:      sigID,
		Kind:    "func",
		Params:  params,
		Returns: returns,
	}
	return sigID, nil
}

func (p *projector) boundFunctionID(methodName string) string {
	fid := p.namedID(methodName)
	if _, ok := p.funcNames[methodName]; ok {
		return fid
	}
	return ""
}

func (p *projector) projectBinaryMembers(expr ast.TypeDefBinaryExpr) ([]string, error) {
	left, err := p.projectTypeDefExprMember(expr.Left)
	if err != nil {
		return nil, err
	}
	right, err := p.projectTypeDefExprMember(expr.Right)
	if err != nil {
		return nil, err
	}
	return []string{left, right}, nil
}

func (p *projector) projectTypeDefExprMember(expr ast.TypeDefExpr) (string, error) {
	switch e := expr.(type) {
	case ast.TypeDefAssertionExpr:
		id := p.synthID("member")
		t, err := p.typeFromAssertion(id, e.Assertion, "exported", "", "")
		if err != nil {
			return "", err
		}
		p.req.Types[t.ID] = t
		return t.ID, nil
	case ast.TypeDefShapeExpr:
		return p.projectShapeType("member", e.Shape)
	default:
		id := p.synthID("member")
		p.req.Types[id] = Type{ID: id, Kind: "unknown", Debug: fmt.Sprintf("%T", expr)}
		return id, nil
	}
}

func (p *projector) projectFunction(fn ast.FunctionNode, relFile, doc string) (string, error) {
	sig, ok := p.tc.Functions[fn.Ident.ID]
	if !ok {
		return "", nil
	}
	fid := p.namedID(string(fn.Ident.ID))
	if _, exists := p.req.Functions[fid]; exists {
		return fid, nil
	}

	var params []FuncParam
	for _, param := range sig.Parameters {
		pn := param.GetIdent()
		ptID := fid + "." + pn
		_, t, err := p.materializeTypeNode(param.Type, ptID)
		if err != nil {
			return "", err
		}
		if t.Kind != "" && t.ID == "" {
			t.ID = ptID
			p.req.Types[ptID] = t
		} else if t.ID != "" {
			ptID = t.ID
		}
		params = append(params, FuncParam{
			Name:     pn,
			Type:     ptID,
			Variadic: param.Variadic,
		})
	}

	var returns []string
	for i, rt := range sig.ReturnTypes {
		rid, _, err := p.projectTypeNode(rt, fmt.Sprintf("%s.ret%d", fid, i))
		if err != nil {
			return "", err
		}
		returns = append(returns, rid)
	}

	input := p.synthesizeInput(fid, params)
	if input != "" && input != "void" {
		if _, ok := p.req.Types[input]; !ok {
			if len(params) > 1 {
				var fields []ShapeField
				for _, param := range params {
					fields = append(fields, ShapeField{Name: param.Name, Type: param.Type})
				}
				p.req.Types[input] = Type{ID: input, Kind: "shape", Fields: fields}
			}
		}
	}

	_, runnable := transformerts.ProviderOmissionReason(fn, p.tc)
	role := functionRole(fn, p.tc)
	vis := "exported"
	if !ast.IsPublicExportIdent(fn.Ident.ID) {
		vis = "internal"
	}

	var receiver *string
	if fn.Receiver != nil {
		r := string(fn.Receiver.Type.Ident)
		receiver = &r
	}

	var span *SourceSpan
	if fn.Ident.Span.IsSet() {
		span = &SourceSpan{
			File:      relFile,
			StartLine: fn.Ident.Span.StartLine,
			StartCol:  fn.Ident.Span.StartCol,
			EndLine:   fn.Ident.Span.EndLine,
			EndCol:    fn.Ident.Span.EndCol,
		}
	} else if relFile != "" {
		span = &SourceSpan{File: relFile}
	}

	nominal := make([]string, 0, len(sig.ErrorSet.NominalErrors))
	for _, e := range sig.ErrorSet.NominalErrors {
		nominal = append(nominal, p.namedID(string(e)))
	}

	p.req.Functions[fid] = Function{
		ID:         fid,
		Name:       string(fn.Ident.ID),
		Package:    p.pkg,
		Visibility: vis,
		Role:       role,
		Runnable:   runnable,
		Receiver:   receiver,
		Params:     params,
		Input:      input,
		Returns:    returns,
		ErrorSet: ErrorSet{
			Nominal:         nominal,
			UnknownPossible: sig.ErrorSet.UnknownPossible,
		},
		Providers: typechecker.ProviderRootIdentsFromSlots(p.tc.FunctionProviders[fn.Ident.ID]),
		Doc:       doc,
		Span:      span,
	}
	return fid, nil
}

func (p *projector) synthesizeInput(functionID string, params []FuncParam) string {
	switch len(params) {
	case 0:
		return "void"
	case 1:
		return params[0].Type
	default:
		return functionID + ".input"
	}
}

func functionRole(fn ast.FunctionNode, tc *typechecker.TypeChecker) string {
	if fn.HasMainFunctionName() {
		return "main"
	}
	if tc != nil && tc.IsGoTestFunction(fn) {
		return "test"
	}
	if fn.Receiver != nil {
		return "method"
	}
	if !ast.IsPublicExportIdent(fn.Ident.ID) {
		return "internal"
	}
	return "function"
}

func isVariadicParam(p ast.ParamNode) bool {
	if sp, ok := p.(ast.SimpleParamNode); ok {
		return sp.Variadic
	}
	return false
}

func paramName(p ast.ParamNode) string {
	switch n := p.(type) {
	case ast.SimpleParamNode:
		return string(n.Ident.ID)
	case ast.DestructuredParamNode:
		if len(n.Fields) > 0 {
			return n.Fields[0]
		}
	}
	return "param"
}

func docFromComment(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "//")
	return strings.TrimSpace(text)
}

func moduleRelativeFile(boundaryRoot, absPath string) string {
	rel, err := filepath.Rel(filepath.Clean(boundaryRoot), filepath.Clean(absPath))
	if err != nil {
		return filepath.Base(absPath)
	}
	return filepath.ToSlash(rel)
}

func packageLayout(boundaryRoot string, paths []string) (dir string, files []string) {
	if len(paths) == 0 {
		return "", nil
	}
	rels := make([]string, len(paths))
	dirs := make([]string, len(paths))
	for i, p := range paths {
		rels[i] = moduleRelativeFile(boundaryRoot, p)
		dirs[i] = filepath.ToSlash(filepath.Dir(rels[i]))
	}
	sort.Strings(rels)
	commonDir := dirs[0]
	for _, d := range dirs[1:] {
		commonDir = commonDirPath(commonDir, d)
	}
	dir = commonDir
	if dir == "." {
		dir = ""
	}
	for _, r := range rels {
		if dir == "" {
			files = append(files, filepath.Base(r))
		} else {
			prefix := dir + "/"
			if strings.HasPrefix(r, prefix) {
				files = append(files, strings.TrimPrefix(r, prefix))
			} else {
				files = append(files, filepath.Base(r))
			}
		}
	}
	sort.Strings(files)
	return dir, files
}

func commonDirPath(a, b string) string {
	a = filepath.ToSlash(a)
	b = filepath.ToSlash(b)
	as := strings.Split(a, "/")
	bs := strings.Split(b, "/")
	n := len(as)
	if len(bs) < n {
		n = len(bs)
	}
	var shared []string
	for i := 0; i < n; i++ {
		if as[i] != bs[i] {
			break
		}
		shared = append(shared, as[i])
	}
	if len(shared) == 0 {
		return "."
	}
	return strings.Join(shared, "/")
}
