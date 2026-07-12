package goload

import (
	"strings"

	"go/doc"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// SourceLocation is a source span for go-to-definition (1-based line, 0-based column).
type SourceLocation struct {
	File      string
	Line      int
	Column    int
	EndLine   int
	EndColumn int
}

// IsSet reports whether the location has a usable start position.
func (loc SourceLocation) IsSet() bool {
	return loc.File != "" && loc.Line > 0
}

// SourceLocationForObject maps a types.Object position in fset to a source span.
func SourceLocationForObject(fset *token.FileSet, obj types.Object) (SourceLocation, bool) {
	if fset == nil || obj == nil {
		return SourceLocation{}, false
	}
	pos := obj.Pos()
	if !pos.IsValid() {
		return SourceLocation{}, false
	}
	startPos := fset.Position(pos)
	endPos := fset.Position(pos + token.Pos(len(obj.Name())))
	if startPos.Filename == "" {
		return SourceLocation{}, false
	}
	return SourceLocation{
		File:      startPos.Filename,
		Line:      startPos.Line,
		Column:    max(startPos.Column-1, 0),
		EndLine:   endPos.Line,
		EndColumn: max(endPos.Column-1, 0),
	}, true
}

// FirstNonTestGoFile returns the first non-test Go source path from a loaded package.
func FirstNonTestGoFile(pkg *packages.Package) string {
	if pkg == nil {
		return ""
	}
	paths := pkg.CompiledGoFiles
	if len(paths) == 0 {
		paths = pkg.GoFiles
	}
	for _, p := range paths {
		if p != "" && !strings.HasSuffix(p, "_test.go") {
			return p
		}
	}
	return ""
}

// PackageFileLocation returns a file-level location for the first non-test Go file in pkg.
func PackageFileLocation(pkg *packages.Package) (SourceLocation, bool) {
	path := FirstNonTestGoFile(pkg)
	if path == "" {
		return SourceLocation{}, false
	}
	return SourceLocation{File: path}, true
}

// MethodDeclLocation returns the definition span for a method on a named type from go/doc metadata.
func MethodDeclLocation(docPkg *doc.Package, fset *token.FileSet, recvTypeName, methodName string) (SourceLocation, bool) {
	if docPkg == nil || fset == nil || recvTypeName == "" || methodName == "" {
		return SourceLocation{}, false
	}
	for _, t := range docPkg.Types {
		if t.Name != recvTypeName {
			continue
		}
		for _, m := range t.Methods {
			if m.Name != methodName || m.Decl == nil || m.Decl.Name == nil {
				continue
			}
			start := fset.Position(m.Decl.Name.Pos())
			end := fset.Position(m.Decl.Name.End())
			if start.Filename == "" {
				return SourceLocation{}, false
			}
			return SourceLocation{
				File:      start.Filename,
				Line:      start.Line,
				Column:    max(start.Column-1, 0),
				EndLine:   end.Line,
				EndColumn: max(end.Column-1, 0),
			}, true
		}
	}
	return SourceLocation{}, false
}
