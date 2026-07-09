package testrunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"

	"forst/internal/ast"
	"forst/internal/generators"
	transformer_go "forst/internal/transformer/go"

	goast "go/ast"
)

type (
	filepathAbsFn            func(string) (string, error)
	filepathRelFn            func(string, string) (string, error)
	execGoTestFn             func(*exec.Cmd) error
	transformForstFileToGoFn func(*transformer_go.Transformer, []ast.Node) (*goast.File, error)
	generateGoCodeFnType     func(*goast.File) (string, error)
	readDirFnType            func(string) ([]os.DirEntry, error)
)

var (
	filepathAbsHook            atomic.Value
	filepathRelHook            atomic.Value
	execGoTestHook             atomic.Value
	transformForstFileToGoHook atomic.Value
	generateGoCodeHook         atomic.Value
	filepathRelDiscoverHook    atomic.Value
	readDirFnHook              atomic.Value
)

func init() {
	filepathAbsHook.Store(filepathAbsFn(filepath.Abs))
	filepathRelHook.Store(filepathRelFn(filepath.Rel))
	execGoTestHook.Store(execGoTestFn(func(cmd *exec.Cmd) error { return cmd.Run() }))
	transformForstFileToGoHook.Store(transformForstFileToGoFn(func(tr *transformer_go.Transformer, merged []ast.Node) (*goast.File, error) {
		return tr.TransformForstFileToGo(merged)
	}))
	generateGoCodeHook.Store(generateGoCodeFnType(generators.GenerateGoCode))
	filepathRelDiscoverHook.Store(filepathRelFn(filepath.Rel))
	readDirFnHook.Store(readDirFnType(os.ReadDir))
}

func currentFilepathAbs() filepathAbsFn {
	return filepathAbsHook.Load().(filepathAbsFn)
}

func currentFilepathRel() filepathRelFn {
	return filepathRelHook.Load().(filepathRelFn)
}

func currentExecGoTest() execGoTestFn {
	return execGoTestHook.Load().(execGoTestFn)
}

func currentTransformForstFileToGo() transformForstFileToGoFn {
	return transformForstFileToGoHook.Load().(transformForstFileToGoFn)
}

func currentGenerateGoCodeFn() generateGoCodeFnType {
	return generateGoCodeHook.Load().(generateGoCodeFnType)
}

func currentFilepathRelDiscover() filepathRelFn {
	return filepathRelDiscoverHook.Load().(filepathRelFn)
}

func currentReadDirFn() readDirFnType {
	return readDirFnHook.Load().(readDirFnType)
}
