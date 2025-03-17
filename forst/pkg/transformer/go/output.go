package transformer_go

import goast "go/ast"

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

	for _, imp := range t.imports {
		decls = append(decls, goast.Decl(imp))
	}
	for _, imp := range t.importGroups {
		decls = append(decls, goast.Decl(imp))
	}
	for _, fn := range t.functions {
		decls = append(decls, goast.Decl(fn))
	}
	for _, t := range t.types {
		decls = append(decls, *t)
	}

	file := &goast.File{
		Name:  goast.NewIdent(t.PackageName()),
		Decls: decls,
	}

	return file, nil
}
