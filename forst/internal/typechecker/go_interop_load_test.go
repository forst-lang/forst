package typechecker

import (
	"errors"
	"testing"

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
