package transformer_go

import (
	goast "go/ast"
	"go/token"
	"strings"
)

type TransformerOutput struct {
	packageName  string
	imports      []*goast.GenDecl
	importGroups []*goast.GenDecl
	functions    []*goast.FuncDecl
	types        []*goast.Decl
}

func (t *TransformerOutput) SetPackageName(name string) {
	t.packageName = name
}

func (t *TransformerOutput) AddImport(imp *goast.GenDecl) {
	t.imports = append(t.imports, imp)
}

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

func (t *TransformerOutput) AddImportGroup(importGroup *goast.GenDecl) {
	t.importGroups = append(t.importGroups, importGroup)
}

func (t *TransformerOutput) AddFunction(function *goast.FuncDecl) {
	t.functions = append(t.functions, function)
}

func (t *TransformerOutput) AddType(typeDecl *goast.Decl) {
	t.types = append(t.types, typeDecl)
}

func (t *TransformerOutput) PackageName() string {
	if t.packageName == "" {
		return "main"
	}
	return t.packageName
}

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

	for _, fn := range t.functions {
		decls = append(decls, goast.Decl(fn))
	}
	for _, t := range t.types {
		decls = append(decls, *t)
	}

	file := &goast.File{
		Name:    goast.NewIdent(t.PackageName()),
		Decls:   decls,
		Imports: imports,
	}

	return file, nil
}
