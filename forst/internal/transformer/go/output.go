package transformergo

import (
	goast "go/ast"
	"go/token"
	"strings"
)

// TransformerOutput represents the output of the Transformer
type TransformerOutput struct {
	packageName  string
	imports      []*goast.GenDecl
	importGroups []*goast.GenDecl
	functions    []*goast.FuncDecl
	types        []*goast.GenDecl
}

// SetPackageName sets the package name for the TransformerOutput
func (t *TransformerOutput) SetPackageName(name string) {
	t.packageName = name
}

// AddImport adds an import to the TransformerOutput
func (t *TransformerOutput) AddImport(imp *goast.GenDecl) {
	t.imports = append(t.imports, imp)
}

// EnsureImport ensures that an import is added to the TransformerOutput
func (t *TransformerOutput) EnsureImport(name string) {
	for _, imp := range t.imports {
		if imp.Specs[0].(*goast.ImportSpec).Name.String() == name {
			return
		}
	}

	t.imports = append(t.imports, &goast.GenDecl{
		Tok: token.IMPORT,
		Specs: []goast.Spec{
			&goast.ImportSpec{
				Name: &goast.Ident{
					Name: name,
				},
				Path: &goast.BasicLit{
					Kind:  token.STRING,
					Value: `"` + strings.Trim(name, `"`) + `"`,
				},
			},
		},
	})
}

// AddImportGroup adds an import group to the TransformerOutput
func (t *TransformerOutput) AddImportGroup(importGroup *goast.GenDecl) {
	t.importGroups = append(t.importGroups, importGroup)
}

// AddFunction adds a function declaration to the TransformerOutput
func (t *TransformerOutput) AddFunction(function *goast.FuncDecl) {
	t.functions = append(t.functions, function)
}

// AddType adds a type declaration to the TransformerOutput
func (t *TransformerOutput) AddType(typeDecl *goast.GenDecl) {
	t.types = append(t.types, typeDecl)
}

// PackageName returns the package name for the TransformerOutput
func (t *TransformerOutput) PackageName() string {
	if t.packageName == "" {
		return "main"
	}
	return t.packageName
}

// GenerateFile generates a Go file from the TransformerOutput
func (t *TransformerOutput) GenerateFile() (*goast.File, error) {
	var decls []goast.Decl
	var imports []*goast.ImportSpec
	seenImports := make(map[string]bool)

	// Process individual imports
	for _, imp := range t.imports {
		importSpec := imp.Specs[0].(*goast.ImportSpec)
		importPath := importSpec.Path.Value
		if !seenImports[importPath] {
			imports = append(imports, importSpec)
			decls = append(decls, goast.Decl(imp))
			seenImports[importPath] = true
		}
	}

	// Process import groups
	for _, imp := range t.importGroups {
		for _, spec := range imp.Specs {
			importSpec := spec.(*goast.ImportSpec)
			importPath := importSpec.Path.Value
			if !seenImports[importPath] {
				imports = append(imports, importSpec)
				seenImports[importPath] = true
			}
		}
		if len(imp.Specs) > 0 {
			decls = append(decls, goast.Decl(imp))
		}
	}

	// Add type declarations
	for _, typeDecl := range t.types {
		decls = append(decls, goast.Decl(typeDecl))
	}

	// Add function declarations
	for _, fn := range t.functions {
		decls = append(decls, goast.Decl(fn))
	}

	file := &goast.File{
		Name:    goast.NewIdent(t.PackageName()),
		Decls:   decls,
		Imports: imports,
	}

	return file, nil
}
