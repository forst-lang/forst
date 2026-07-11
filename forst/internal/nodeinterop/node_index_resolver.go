package nodeinterop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"forst/internal/ast"
)

// IndexResolver loads forst-index-v1 JSON and resolves export types to Forst types.
type IndexResolver struct {
	byModuleID map[string]*IndexV1
}

// NewIndexResolver returns an empty resolver.
func NewIndexResolver() *IndexResolver {
	return &IndexResolver{byModuleID: make(map[string]*IndexV1)}
}

// LoadFromFile reads a forst-index-v1 JSON file.
func (r *IndexResolver) LoadFromFile(path string) (*IndexV1, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("index: read %q: %w", path, err)
	}
	return r.LoadFromJSON(data)
}

// LoadFromJSON parses and registers index JSON.
func (r *IndexResolver) LoadFromJSON(data []byte) (*IndexV1, error) {
	idx, err := ParseIndexV1(data)
	if err != nil {
		return nil, err
	}
	if r.byModuleID == nil {
		r.byModuleID = make(map[string]*IndexV1)
	}
	r.byModuleID[idx.ModuleID] = idx
	return idx, nil
}

// Register adds a parsed index (tests and inline fixtures).
func (r *IndexResolver) Register(idx *IndexV1) error {
	if idx == nil {
		return fmt.Errorf("index: nil")
	}
	if err := idx.Validate(); err != nil {
		return err
	}
	if r.byModuleID == nil {
		r.byModuleID = make(map[string]*IndexV1)
	}
	r.byModuleID[idx.ModuleID] = idx
	return nil
}

// Module returns the index for moduleID, if loaded.
func (r *IndexResolver) Module(moduleID string) (*IndexV1, bool) {
	if r == nil || moduleID == "" {
		return nil, false
	}
	idx, ok := r.byModuleID[moduleID]
	return idx, ok
}

// ExportSignature resolves export name to parameter and return Forst types.
func (r *IndexResolver) ExportSignature(moduleID, exportName string) (params []ast.TypeNode, returns []ast.TypeNode, kind string, err error) {
	params, returns, kind, _, err = r.ExportSignatureWithWarnings(moduleID, exportName)
	return params, returns, kind, err
}

// ExportSignatureWithWarnings resolves export types and reports index paths widened to Object.
func (r *IndexResolver) ExportSignatureWithWarnings(moduleID, exportName string) (params []ast.TypeNode, returns []ast.TypeNode, kind string, warnWidened []string, err error) {
	idx, ok := r.Module(moduleID)
	if !ok {
		return nil, nil, "", nil, fmt.Errorf("index: module %q not loaded", moduleID)
	}
	exp, ok := idx.ExportByName(exportName)
	if !ok {
		return nil, nil, "", nil, fmt.Errorf("index: export %q not found in %q", exportName, moduleID)
	}
	paramTypes := make([]ast.TypeNode, 0, len(exp.Parameters))
	for _, p := range exp.Parameters {
		ft, widened, mapErr := MapIndexTypeToForst(p.Type)
		if mapErr != nil {
			return nil, nil, "", nil, fmt.Errorf("index: parameter %q: %w", p.Name, mapErr)
		}
		if widened {
			warnWidened = append(warnWidened, fmt.Sprintf("parameter %q", p.Name))
		}
		paramTypes = append(paramTypes, ft)
	}
	retTypes, retWarn, mapErr := exportReturnTypes(exp)
	if mapErr != nil {
		return nil, nil, "", nil, mapErr
	}
	warnWidened = append(warnWidened, retWarn...)
	return paramTypes, retTypes, exp.Kind, warnWidened, nil
}

func exportReturnTypes(exp *IndexExport) ([]ast.TypeNode, []string, error) {
	if exp == nil {
		return nil, nil, fmt.Errorf("index: nil export")
	}
	switch exp.Kind {
	case ExportKindFunction, ExportKindAsyncFunction:
		if exp.ReturnType == nil {
			return []ast.TypeNode{{Ident: ast.TypeVoid}}, nil, nil
		}
		t, widened, err := MapIndexTypeToForst(*exp.ReturnType)
		if err != nil {
			return nil, nil, err
		}
		var warns []string
		if widened {
			warns = append(warns, "return type")
		}
		return []ast.TypeNode{t}, warns, nil
	case ExportKindGenerator, ExportKindAsyncGenerator:
		if exp.YieldType == nil {
			return nil, nil, fmt.Errorf("index: generator %q missing yieldType", exp.Name)
		}
		elem, widened, err := MapIndexTypeToForst(*exp.YieldType)
		if err != nil {
			return nil, nil, err
		}
		var warns []string
		if widened {
			warns = append(warns, "yield type")
		}
		return []ast.TypeNode{newSeqType(elem)}, warns, nil
	default:
		return nil, nil, fmt.Errorf("index: unsupported export kind %q", exp.Kind)
	}
}


const typeIdentSeq ast.TypeIdent = "Seq"

func newSeqType(elem ast.TypeNode) ast.TypeNode {
	return ast.TypeNode{
		Ident:      typeIdentSeq,
		TypeKind:   ast.TypeKindUserDefined,
		TypeParams: []ast.TypeNode{elem},
	}
}

// ManifestFromIndexes builds a forst-node-manifest-v1 from loaded module indexes.
func ManifestFromIndexes(boundaryRoot string, indexes []*IndexV1) (*ManifestV1, error) {
	if strings.TrimSpace(boundaryRoot) == "" {
		return nil, fmt.Errorf("manifest: boundaryRoot is required")
	}
	absRoot, err := filepath.Abs(boundaryRoot)
	if err != nil {
		return nil, fmt.Errorf("manifest: boundaryRoot: %w", err)
	}
	m := &ManifestV1{
		Version:      ManifestVersionV1,
		BoundaryRoot: filepath.ToSlash(absRoot),
	}
	if !filepath.IsAbs(m.BoundaryRoot) {
		m.BoundaryRoot = absRoot
	}
	seen := make(map[string]struct{})
	for _, idx := range indexes {
		if idx == nil {
			continue
		}
		for _, exp := range idx.Exports {
			key := idx.ModuleID + "\x00" + exp.Name
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			m.Exports = append(m.Exports, ExportEntry{
				ModuleID: idx.ModuleID,
				Name:     exp.Name,
				Kind:     exp.Kind,
			})
		}
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}
