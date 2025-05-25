package generators

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/format"
	"go/token"
	"sort"
)

// GenerateGoCode generates Go code from a Go AST with consistent ordering
func GenerateGoCode(goFile *goast.File) (string, error) {
	var buf bytes.Buffer
	fset := token.NewFileSet()

	// Sort imports and ensure consistent declaration ordering
	goast.SortImports(fset, goFile)
	sortDeclarations(goFile)
	sortStructFields(goFile)

	if err := format.Node(&buf, fset, goFile); err != nil {
		return "", fmt.Errorf("failed to format Go code: %w", err)
	}
	return buf.String(), nil
}

// sortDeclarations sorts declarations in a Go file for consistent ordering
func sortDeclarations(file *goast.File) {
	// Group declarations by type
	var imports, types, funcs, vars, consts []goast.Decl

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *goast.GenDecl:
			switch d.Tok {
			case token.IMPORT:
				imports = append(imports, d)
			case token.TYPE:
				types = append(types, d)
			case token.VAR:
				vars = append(vars, d)
			case token.CONST:
				consts = append(consts, d)
			}
		case *goast.FuncDecl:
			funcs = append(funcs, d)
		}
	}

	// Sort each group by name
	sortDeclsByName := func(decls []goast.Decl) {
		sort.Slice(decls, func(i, j int) bool {
			return getDeclName(decls[i]) < getDeclName(decls[j])
		})
	}

	sortDeclsByName(imports)
	sortDeclsByName(types)
	sortDeclsByName(funcs)
	sortDeclsByName(vars)
	sortDeclsByName(consts)

	// Reassemble declarations in consistent order
	file.Decls = make([]goast.Decl, 0, len(file.Decls))
	file.Decls = append(file.Decls, imports...)
	file.Decls = append(file.Decls, consts...)
	file.Decls = append(file.Decls, vars...)
	file.Decls = append(file.Decls, types...)
	file.Decls = append(file.Decls, funcs...)
}

// getDeclName gets the name of a declaration for sorting
func getDeclName(decl goast.Decl) string {
	switch d := decl.(type) {
	case *goast.GenDecl:
		if len(d.Specs) > 0 {
			switch s := d.Specs[0].(type) {
			case *goast.TypeSpec:
				return s.Name.Name
			case *goast.ValueSpec:
				if len(s.Names) > 0 {
					return s.Names[0].Name
				}
			case *goast.ImportSpec:
				if s.Name != nil {
					return s.Name.Name
				}
				return s.Path.Value
			}
		}
	case *goast.FuncDecl:
		return d.Name.Name
	}
	return ""
}

// sortStructFields recursively sorts fields in all struct types
func sortStructFields(file *goast.File) {
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*goast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*goast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*goast.StructType); ok {
						sortStructTypeFields(structType)
					}
				}
			}
		}
	}
}

// sortStructTypeFields sorts fields in a struct type and recursively sorts nested structs
func sortStructTypeFields(structType *goast.StructType) {
	if structType.Fields == nil {
		return
	}

	// Sort fields by name
	sort.Slice(structType.Fields.List, func(i, j int) bool {
		// If either field has no name, sort by type
		if len(structType.Fields.List[i].Names) == 0 || len(structType.Fields.List[j].Names) == 0 {
			return fmt.Sprint(structType.Fields.List[i].Type) < fmt.Sprint(structType.Fields.List[j].Type)
		}
		return structType.Fields.List[i].Names[0].Name < structType.Fields.List[j].Names[0].Name
	})

	// Recursively sort nested structs
	for _, field := range structType.Fields.List {
		if field.Type != nil {
			switch t := field.Type.(type) {
			case *goast.StructType:
				sortStructTypeFields(t)
			case *goast.ArrayType:
				if structType, ok := t.Elt.(*goast.StructType); ok {
					sortStructTypeFields(structType)
				}
			case *goast.MapType:
				if structType, ok := t.Value.(*goast.StructType); ok {
					sortStructTypeFields(structType)
				}
			}
		}
	}
}
