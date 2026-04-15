package executor

import (
	"fmt"

	"forst/internal/discovery"
)

// getOrCompileFunction gets a compiled function from cache or compiles it.
func (e *FunctionExecutor) getOrCompileFunction(packageName, functionName string) (*CompiledFunction, error) {
	cacheKey := fmt.Sprintf("%s.%s", packageName, functionName)

	e.mu.RLock()
	if cached, exists := e.cache[cacheKey]; exists {
		e.mu.RUnlock()
		return cached, nil
	}
	e.mu.RUnlock()

	compiledFn, err := e.compileFunction(packageName, functionName)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.cache[cacheKey] = compiledFn
	e.mu.Unlock()

	return compiledFn, nil
}

// findFunctionFile finds the Forst file containing the specified function.
func (e *FunctionExecutor) findFunctionFile(packageName, functionName string) (string, error) {
	discoverer := discovery.NewDiscoverer(e.rootDir, e.log, e.config)
	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		return "", fmt.Errorf("failed to discover functions: %v", err)
	}

	pkgFuncs, exists := functions[packageName]
	if !exists {
		return "", fmt.Errorf("package %s not found", packageName)
	}

	fnInfo, exists := pkgFuncs[functionName]
	if !exists {
		return "", fmt.Errorf("function %s not found in package %s", functionName, packageName)
	}

	return fnInfo.FilePath, nil
}

// getFunctionInfo gets information about a function.
func (e *FunctionExecutor) getFunctionInfo(packageName, functionName string) (*discovery.FunctionInfo, error) {
	discoverer := discovery.NewDiscoverer(e.rootDir, e.log, e.config)
	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		return nil, fmt.Errorf("failed to discover functions: %v", err)
	}

	pkgFuncs, exists := functions[packageName]
	if !exists {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	fnInfo, exists := pkgFuncs[functionName]
	if !exists {
		return nil, fmt.Errorf("function %s not found in package %s", functionName, packageName)
	}

	return &fnInfo, nil
}
