package typechecker

import (
	"errors"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestRecordGoPackagesLoadFailure_surfacesOnImport(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.registerImportLocalsFromAST()
	tc.importPathByLocal = map[string]string{"badpkg": "example.com/badpkg"}
	tc.recordGoPackagesLoadFailure([]string{"example.com/badpkg"}, errors.New("load exploded"))

	if got := tc.goImportLoadErrorForLocal("badpkg"); got == nil {
		t.Fatal("expected stored load error for import local")
	}
}

func TestRecordUnloadedGoImportPaths_marksMissingPackages(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.imports = []ast.ImportNode{
		{Path: `"example.com/missing/pkg"`},
	}
	tc.ensureImportPathByLocal()
	tc.recordUnloadedGoImportPaths(nil, errors.New("batch failed"))

	if got := tc.goImportLoadErrorForPath("example.com/missing/pkg"); got == nil {
		t.Fatal("expected load error for missing import path")
	}
}
