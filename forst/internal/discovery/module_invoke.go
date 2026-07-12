package discovery

import (
	"path/filepath"

	"forst/internal/modulecheck"

	"github.com/sirupsen/logrus"
)

// CollectInvokeFunctionsFromModuleResult returns runnable public exports from an existing
// modulecheck result without re-running CheckModuleProviders.
func CollectInvokeFunctionsFromModuleResult(modResult *modulecheck.ModuleResult) []FunctionInfo {
	if modResult == nil {
		return nil
	}
	var out []FunctionInfo
	for pkg, nodes := range modResult.PerPackageNodes {
		tc := modResult.PerPackage[pkg]
		for _, fn := range CollectInvokeFunctionsFromNodes(nodes, tc) {
			out = append(out, fn)
		}
	}
	return out
}

// CollectInvokeFunctionsFromModule returns runnable public exports across all Forst packages
// under boundaryRoot (modulecheck graph).
func CollectInvokeFunctionsFromModule(log *logrus.Logger, boundaryRoot string) ([]FunctionInfo, error) {
	if boundaryRoot == "" {
		return nil, nil
	}
	boundaryRoot = filepath.Clean(boundaryRoot)
	modResult, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{
		ModuleRoot:   boundaryRoot,
		BoundaryRoot: boundaryRoot,
	})
	if err != nil {
		return nil, err
	}
	return CollectInvokeFunctionsFromModuleResult(modResult), nil
}

// CrossPackageInvokeExports returns runnable exports in packages other than compiledPkg.
func CrossPackageInvokeExports(moduleFns []FunctionInfo, compiledPkg string) []FunctionInfo {
	var out []FunctionInfo
	for _, fn := range moduleFns {
		if fn.Package != compiledPkg && fn.Runnable && fn.Name != "main" {
			out = append(out, fn)
		}
	}
	return out
}
