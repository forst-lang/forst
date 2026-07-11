package nodeinterop

import (
	"encoding/json"
	"fmt"
	"strings"
)

const IndexVersionV1 = 1

// IndexV1 is the compile-time TypeScript module index (forst-index-v1).
type IndexV1 struct {
	Version  int           `json:"version,omitempty"`
	ModuleID string        `json:"moduleId"`
	Exports  []IndexExport `json:"exports"`
}

// IndexExport describes one callable export from a TypeScript module.
type IndexExport struct {
	Name       string       `json:"name"`
	Kind       string       `json:"kind"`
	Parameters []IndexParam `json:"parameters,omitempty"`
	ReturnType *IndexType   `json:"returnType,omitempty"`
	YieldType  *IndexType   `json:"yieldType,omitempty"`
}

// IndexParam is a function parameter in the index.
type IndexParam struct {
	Name string    `json:"name"`
	Type IndexType `json:"type"`
}

// IndexType is a JSON type descriptor from the TS indexer.
type IndexType struct {
	Kind    string               `json:"kind"`
	Fields  map[string]IndexType `json:"fields,omitempty"`
	Element *IndexType           `json:"element,omitempty"`
	Members []IndexType          `json:"members,omitempty"`
	Binary  bool                 `json:"$binary,omitempty"`
}

// ParseIndexV1 parses forst-index-v1 JSON for one module.
func ParseIndexV1(data []byte) (*IndexV1, error) {
	var idx IndexV1
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("index: parse: %w", err)
	}
	if err := idx.Validate(); err != nil {
		return nil, err
	}
	return &idx, nil
}

// Validate checks index invariants.
func (idx *IndexV1) Validate() error {
	if idx == nil {
		return fmt.Errorf("index: nil")
	}
	if strings.TrimSpace(idx.ModuleID) == "" {
		return fmt.Errorf("index: moduleId is required")
	}
	if err := validateModuleID(idx.ModuleID); err != nil {
		return fmt.Errorf("index: %w", err)
	}
	seen := make(map[string]struct{}, len(idx.Exports))
	for i, exp := range idx.Exports {
		if strings.TrimSpace(exp.Name) == "" {
			return fmt.Errorf("index: exports[%d]: name is required", i)
		}
		if _, ok := validExportKinds[exp.Kind]; !ok {
			return fmt.Errorf("index: exports[%d]: invalid kind %q", i, exp.Kind)
		}
		if _, dup := seen[exp.Name]; dup {
			return fmt.Errorf("index: duplicate export %q", exp.Name)
		}
		seen[exp.Name] = struct{}{}
	}
	return nil
}

// ExportByName returns the export with the given name, if present.
func (idx *IndexV1) ExportByName(name string) (*IndexExport, bool) {
	if idx == nil || name == "" {
		return nil, false
	}
	for i := range idx.Exports {
		if idx.Exports[i].Name == name {
			return &idx.Exports[i], true
		}
	}
	return nil, false
}
