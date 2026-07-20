package compileplan

import (
	"forst/internal/ast"
	"forst/internal/generators"
	"forst/internal/modulecheck"
	"forst/internal/project"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)
// EmitMode selects which Go artifacts EmitGo produces for a Plan: a bare
// executor, an executor plus generated companions (invoke server, node
// runtime), a lib shim for embedding, or a test wrapper.
type EmitMode int

const (
	EmitExecutor EmitMode = iota
	EmitWithCompanions
	EmitLibShim
	EmitTest
)

// Plan is a unified compile request.
type Plan struct {
	Project            *project.Project
	ForstPackage       string
	Nodes              []ast.Node
	Checker            *typechecker.TypeChecker
	Module             *modulecheck.ModuleResult
	ExportStructFields bool
	EmbedInvoke        bool
}

var generateFn = generators.GenerateGoCode

// EmitGo transforms and emits Go source for the given mode.
func EmitGo(log *logrus.Logger, plan Plan, mode EmitMode) (string, error) {
	if plan.Checker == nil {
		return "", nil
	}
	tr := transformer_go.New(plan.Checker, log, plan.ExportStructFields)
	if plan.Module != nil {
		tr.SetModuleResult(plan.Module)
	}
	switch mode {
	case EmitLibShim:
		tr.OmitPackageTypeDefs = false
	case EmitTest:
		tr.OmitPackageTypeDefs = true
	}
	tr.EmbedInvokeServer = plan.EmbedInvoke && mode == EmitWithCompanions
	goAST, err := tr.TransformForstFileToGo(plan.Nodes)
	if err != nil {
		return "", err
	}
	return generateFn(goAST)
}
