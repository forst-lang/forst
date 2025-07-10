package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"forst/cmd/forst/compiler"
	"forst/internal/configiface"
	"forst/internal/discovery"

	logrus "github.com/sirupsen/logrus"
)

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// FunctionExecutor handles execution of Forst functions
type FunctionExecutor struct {
	rootDir  string
	compiler *compiler.Compiler
	log      *logrus.Logger
	cache    map[string]*CompiledFunction
	mu       sync.RWMutex
	config   configiface.ForstConfigIface
}

// CompiledFunction represents a compiled Forst function
type CompiledFunction struct {
	PackageName       string
	FunctionName      string
	GoCode            string
	FilePath          string
	SupportsStreaming bool
	Parameters        []discovery.ParameterInfo
}

// ExecutionResult represents the result of a function execution
type ExecutionResult struct {
	Success bool            `json:"success"`
	Output  string          `json:"output,omitempty"`
	Error   string          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// StreamingResult represents a streaming result
type StreamingResult struct {
	Data   interface{} `json:"data"`
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
}

// NewFunctionExecutor creates a new function executor
func NewFunctionExecutor(rootDir string, comp *compiler.Compiler, log *logrus.Logger, config configiface.ForstConfigIface) *FunctionExecutor {
	return &FunctionExecutor{
		rootDir:  rootDir,
		compiler: comp,
		log:      log,
		cache:    make(map[string]*CompiledFunction),
		config:   config,
	}
}

// ExecuteFunction executes a Forst function with the given arguments
func (e *FunctionExecutor) ExecuteFunction(packageName, functionName string, args json.RawMessage) (*ExecutionResult, error) {
	// Get or compile the function
	compiledFn, err := e.getOrCompileFunction(packageName, functionName)
	e.log.Infof("Compiled function: %v", compiledFn)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %v", err)
	}

	// Create temporary Go module with the function call
	tempDir, err := e.createTempGoFile(compiledFn, args)
	e.log.Infof("Temp dir: %s", tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Execute the Go code
	output, err := e.executeGoCode(tempDir, args, len(compiledFn.Parameters) > 0)
	e.log.Infof("Output: %s", output)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Parse the output
	result, err := e.parseExecutionOutput(output)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("failed to parse output: %v", err),
		}, nil
	}

	return result, nil
}

// ExecuteStreamingFunction executes a Forst function with streaming support
func (e *FunctionExecutor) ExecuteStreamingFunction(ctx context.Context, packageName, functionName string, args json.RawMessage) (<-chan StreamingResult, error) {
	// Get or compile the function
	compiledFn, err := e.getOrCompileFunction(packageName, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %v", err)
	}

	if !compiledFn.SupportsStreaming {
		return nil, fmt.Errorf("function %s does not support streaming", functionName)
	}

	// Create temporary Go module with streaming function call
	tempDir, err := e.createStreamingTempGoFile(compiledFn, args)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}

	// Execute the Go code with streaming
	return e.executeStreamingGoCode(ctx, tempDir, args, len(compiledFn.Parameters) > 0)
}

// getOrCompileFunction gets a compiled function from cache or compiles it
func (e *FunctionExecutor) getOrCompileFunction(packageName, functionName string) (*CompiledFunction, error) {
	cacheKey := fmt.Sprintf("%s.%s", packageName, functionName)

	e.mu.RLock()
	if cached, exists := e.cache[cacheKey]; exists {
		e.mu.RUnlock()
		return cached, nil
	}
	e.mu.RUnlock()

	// Compile the function
	compiledFn, err := e.compileFunction(packageName, functionName)
	if err != nil {
		return nil, err
	}

	// Cache the compiled function
	e.mu.Lock()
	e.cache[cacheKey] = compiledFn
	e.mu.Unlock()

	return compiledFn, nil
}

// compileFunction compiles a Forst function to Go code
func (e *FunctionExecutor) compileFunction(packageName, functionName string) (*CompiledFunction, error) {
	// Find the Forst file containing the function
	filePath, err := e.findFunctionFile(packageName, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to find function file: %v", err)
	}

	// Compile the file
	args := e.compiler.Args
	args.FilePath = filePath

	comp := compiler.New(args, e.log)
	goCode, err := comp.CompileFile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile function: %v", err)
	}

	// Extract function information
	fnInfo, err := e.getFunctionInfo(packageName, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function info: %v", err)
	}

	return &CompiledFunction{
		PackageName:       packageName,
		FunctionName:      functionName,
		GoCode:            *goCode,
		FilePath:          filePath,
		SupportsStreaming: fnInfo.SupportsStreaming,
		Parameters:        fnInfo.Parameters, // Populate Parameters
	}, nil
}

// findFunctionFile finds the Forst file containing the specified function
func (e *FunctionExecutor) findFunctionFile(packageName, functionName string) (string, error) {
	// This is a simplified implementation
	// In a real implementation, you'd use the discovery package to find the file
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

// getFunctionInfo gets information about a function
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

// createTempGoFile creates a temporary Go file that calls the specified function
func (e *FunctionExecutor) createTempGoFile(compiledFn *CompiledFunction, args json.RawMessage) (string, error) {
	// Create a temporary directory for the Go module
	tempDir, err := os.MkdirTemp("", "forst-exec-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}

	// Generate unique module name with random string
	randomSuffix := generateRandomString(8)
	moduleName := fmt.Sprintf("forst-exec-%s", randomSuffix)
	alias := fmt.Sprintf("%s_%s", compiledFn.PackageName, randomSuffix)

	// Create go.mod file
	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write go.mod: %v", err)
	}

	// Use module import path for the package
	importPkg := moduleName + "/" + compiledFn.PackageName

	var mainGoContent string
	if len(compiledFn.Parameters) > 0 {
		param := compiledFn.Parameters[0]
		paramType := param.Type
		paramName := param.Name
		mainGoContent = fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"os"
	%s "%s"
)

func main() {
	var %s %s.%s
	if err := json.Unmarshal([]byte(os.Args[1]), &%s); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal args: %%v\n", err)
		os.Exit(1)
	}
	result := %s.%s(%s)
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%s}\n", string(output))
}
`, alias, importPkg, paramName, alias, paramType, paramName, alias, compiledFn.FunctionName, paramName, "%s")
	} else {
		mainGoContent = fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	%s "%s"
)

func main() {
	result := %s.%s()
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%s}\n", string(output))
}
`, alias, importPkg, alias, compiledFn.FunctionName, "%s")
	}

	mainGoPath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainGoContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write main.go: %v", err)
	}

	// Create the package directory and file
	packageDir := filepath.Join(tempDir, compiledFn.PackageName)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to create package dir: %v", err)
	}

	// Write the compiled Go code to the package file
	packageGoPath := filepath.Join(packageDir, fmt.Sprintf("%s.go", compiledFn.PackageName))
	if err := os.WriteFile(packageGoPath, []byte(compiledFn.GoCode), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write package file: %v", err)
	}

	// Debug: list tempDir contents
	if e.log != nil {
		entries, _ := os.ReadDir(tempDir)
		for _, entry := range entries {
			if entry.IsDir() {
				e.log.Tracef("Temp dir contains subdir: %s", entry.Name())
				subEntries, _ := os.ReadDir(filepath.Join(tempDir, entry.Name()))
				for _, subEntry := range subEntries {
					e.log.Tracef("  - %s/%s", entry.Name(), subEntry.Name())
				}
			} else {
				e.log.Tracef("Temp dir contains file: %s", entry.Name())
			}
		}
	}

	return tempDir, nil
}

// createStreamingTempGoFile creates a temporary Go file for streaming execution
func (e *FunctionExecutor) createStreamingTempGoFile(compiledFn *CompiledFunction, args json.RawMessage) (string, error) {
	// Create a temporary directory for the Go module
	tempDir, err := os.MkdirTemp("", "forst-streaming-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}

	// Generate unique module name with random string
	randomSuffix := generateRandomString(8)
	moduleName := fmt.Sprintf("forst-streaming-%s", randomSuffix)

	// Create go.mod file
	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent := fmt.Sprintf(`module %s

go 1.21
`, moduleName)
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write go.mod: %v", err)
	}

	// Create the main package file that imports and calls the streaming function
	mainGoPath := filepath.Join(tempDir, "main.go")
	mainGoContent := fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	var args json.RawMessage = %s
	
	results := %s.%s(args)
	for result := range results {
		output, _ := json.Marshal(result)
		fmt.Println(string(output))
	}
}
`, string(args), compiledFn.PackageName, compiledFn.FunctionName)

	if err := os.WriteFile(mainGoPath, []byte(mainGoContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write main.go: %v", err)
	}

	// Create the package directory and file
	packageDir := filepath.Join(tempDir, compiledFn.PackageName)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to create package dir: %v", err)
	}

	// Write the compiled Go code to the package file
	packageGoPath := filepath.Join(packageDir, fmt.Sprintf("%s.go", compiledFn.PackageName))
	if err := os.WriteFile(packageGoPath, []byte(compiledFn.GoCode), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write package file: %v", err)
	}

	return tempDir, nil
}

// executeGoCode executes Go code and returns the output
func (e *FunctionExecutor) executeGoCode(tempDir string, args json.RawMessage, hasParams bool) (string, error) {
	var cmd *exec.Cmd
	if hasParams {
		argStr := string(args)
		if argStr == "" || argStr == "null" {
			argStr = "{}"
		}
		cmd = exec.Command("go", "run", ".", argStr)
	} else {
		cmd = exec.Command("go", "run", ".")
	}
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("execution failed: %v, output: %s", err, string(output))
	}

	return string(output), nil
}

// executeStreamingGoCode executes Go code with streaming support
func (e *FunctionExecutor) executeStreamingGoCode(ctx context.Context, tempDir string, args json.RawMessage, hasParams bool) (<-chan StreamingResult, error) {
	results := make(chan StreamingResult, 100)

	var cmd *exec.Cmd
	if hasParams {
		cmd = exec.CommandContext(ctx, "go", "run", ".", string(args))
	} else {
		cmd = exec.CommandContext(ctx, "go", "run", ".")
	}
	cmd.Dir = tempDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}

	// Read output in background
	go func() {
		defer close(results)
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			var result StreamingResult
			if err := json.Unmarshal([]byte(line), &result); err != nil {
				results <- StreamingResult{Error: err.Error()}
				continue
			}
			results <- result
		}
	}()

	return results, nil
}

// parseExecutionOutput parses the output of a function execution
func (e *FunctionExecutor) parseExecutionOutput(output string) (*ExecutionResult, error) {
	// Try to parse as JSON first
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		// Extract the result field if it exists
		if resultValue, exists := result["result"]; exists {
			// Handle primitive values vs objects/arrays
			switch v := resultValue.(type) {
			case string:
				// For strings, return the raw value (not JSON-encoded)
				return &ExecutionResult{
					Success: true,
					Output:  v,
					Result:  []byte(fmt.Sprintf("%q", v)), // JSON-encoded for Result field
				}, nil
			case float64:
				// For numbers, return the raw value as string
				return &ExecutionResult{
					Success: true,
					Output:  fmt.Sprintf("%v", v),
					Result:  []byte(fmt.Sprintf("%v", v)), // JSON-encoded for Result field
				}, nil
			case int:
				// For integers, return the raw value as string
				return &ExecutionResult{
					Success: true,
					Output:  fmt.Sprintf("%d", v),
					Result:  []byte(fmt.Sprintf("%d", v)), // JSON-encoded for Result field
				}, nil
			case bool:
				// For booleans, return the raw value as string
				return &ExecutionResult{
					Success: true,
					Output:  fmt.Sprintf("%t", v),
					Result:  []byte(fmt.Sprintf("%t", v)), // JSON-encoded for Result field
				}, nil
			default:
				// For objects/arrays, return JSON string
				resultData, _ := json.Marshal(resultValue)
				return &ExecutionResult{
					Success: true,
					Output:  string(resultData),
					Result:  resultData,
				}, nil
			}
		}
		// If no result field, return the entire JSON as output
		resultData, _ := json.Marshal(result)
		return &ExecutionResult{
			Success: true,
			Output:  output,
			Result:  resultData,
		}, nil
	}

	// If not JSON, return as raw output
	return &ExecutionResult{
		Success: true,
		Output:  output,
	}, nil
}
