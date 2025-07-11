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
	"forst/internal/generators"
	"forst/internal/lexer"
	"forst/internal/parser"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"

	"bytes"

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
	rootDir       string
	compiler      *compiler.Compiler
	log           *logrus.Logger
	cache         map[string]*CompiledFunction
	mu            sync.RWMutex
	config        configiface.ForstConfigIface
	moduleManager *GoModuleManager
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
		rootDir:       rootDir,
		compiler:      comp,
		log:           log,
		cache:         make(map[string]*CompiledFunction),
		config:        config,
		moduleManager: NewGoModuleManager(log),
	}
}

// ExecuteFunction executes a Forst function with the given arguments
func (e *FunctionExecutor) ExecuteFunction(packageName, functionName string, args json.RawMessage) (*ExecutionResult, error) {
	e.log.Debugf("ExecuteFunction: %s.%s", packageName, functionName)

	// Get or compile the function
	compiledFn, err := e.getOrCompileFunction(packageName, functionName)
	e.log.Debugf("Compiled function: %v", compiledFn)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %v", err)
	}

	// Create temporary Go module with the function call
	tempDir, err := e.createTempGoFile(compiledFn, args)
	e.log.Tracef("Temp dir: %s", tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Debug: Let's look at the actual generated files
	e.log.Tracef("Generated Go code:\n%s", compiledFn.GoCode)

	// List the contents of the temp directory
	entries, err := os.ReadDir(tempDir)
	if err == nil {
		e.log.Tracef("Temp directory contents:")
		for _, entry := range entries {
			if entry.IsDir() {
				subEntries, _ := os.ReadDir(filepath.Join(tempDir, entry.Name()))
				for _, subEntry := range subEntries {
					e.log.Tracef("  %s/%s", entry.Name(), subEntry.Name())
				}
			} else {
				e.log.Tracef("  %s", entry.Name())
			}
		}
	}

	// Debug: Read and log the actual main.go file
	mainGoPath := filepath.Join(tempDir, "main.go")
	if mainGoContent, err := os.ReadFile(mainGoPath); err == nil {
		e.log.Tracef("Generated main.go content:\n%s", string(mainGoContent))
	} else {
		e.log.Errorf("Failed to read main.go: %v", err)
	}

	// Debug: Check log level
	e.log.Infof("Current log level: %v", e.log.GetLevel())

	// Debug: Check if the echo function has parameters
	e.log.Infof("Echo function parameters: %v", compiledFn.Parameters)

	// Execute the Go code
	hasParams := len(compiledFn.Parameters) > 0
	e.log.Infof("executeGoCode: hasParams=%v, args=%s", hasParams, string(args))
	output, err := e.executeGoCode(tempDir, args, hasParams, compiledFn.Parameters)
	if err != nil {
		e.log.Errorf("Failed to execute Go code: %v", err)
		return nil, fmt.Errorf("failed to execute Go code: %v", err)
	}

	e.log.Infof("Output: %s", output)

	// Parse the output
	result, err := e.parseExecutionOutput(output)
	if err != nil {
		e.log.Errorf("Failed to parse output: %v", err)
		return nil, fmt.Errorf("failed to parse output: %v", err)
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

	// Read and parse the source file
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	l := lexer.New(source, filePath, e.log)
	tokens := l.Lex()

	psr := parser.New(tokens, filePath, e.log)
	forstNodes, err := psr.ParseFile()
	if err != nil {
		return nil, err
	}

	checker := typechecker.New(e.log, false)
	if err := checker.CheckTypes(forstNodes); err != nil {
		e.log.Error("Encountered error checking types: ", err)
		checker.DebugPrintCurrentScope()
		return nil, err
	}

	// Use ExportReturnStructFields=true for executor/dev server
	transformer := transformer_go.New(checker, e.log, true)
	goAST, err := transformer.TransformForstFileToGo(forstNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to transform Forst file to Go: %v", err)
	}
	goCode, err := generators.GenerateGoCode(goAST)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Go code: %v", err)
	}

	// Extract function information
	fnInfo, err := e.getFunctionInfo(packageName, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function info: %v", err)
	}

	return &CompiledFunction{
		PackageName:       packageName,
		FunctionName:      functionName,
		GoCode:            goCode,
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
	e.log.Debugf("createTempGoFile: compiledFn.Parameters=%v, len=%d", compiledFn.Parameters, len(compiledFn.Parameters))

	config := &ModuleConfig{
		ModuleName:     fmt.Sprintf("exec-%s", generateRandomString(8)),
		PackageName:    compiledFn.PackageName,
		FunctionName:   compiledFn.FunctionName,
		GoCode:         compiledFn.GoCode,
		SupportsParams: len(compiledFn.Parameters) > 0,
		Parameters:     compiledFn.Parameters,
		Args:           args,
		IsStreaming:    false,
	}

	e.log.Debugf("ModuleConfig: SupportsParams=%v, Parameters=%v", config.SupportsParams, config.Parameters)

	tempDir, err := e.moduleManager.CreateModule(config)
	e.log.Debugf("Created temp dir: %s", tempDir)
	if err != nil {
		e.log.Errorf("Failed to create module: %v", err)
		return "", err
	}

	return tempDir, nil
}

// createStreamingTempGoFile creates a temporary Go file for streaming execution
func (e *FunctionExecutor) createStreamingTempGoFile(compiledFn *CompiledFunction, args json.RawMessage) (string, error) {
	config := &ModuleConfig{
		ModuleName:     fmt.Sprintf("streaming-%s", generateRandomString(8)),
		PackageName:    compiledFn.PackageName,
		FunctionName:   compiledFn.FunctionName,
		GoCode:         compiledFn.GoCode,
		SupportsParams: len(compiledFn.Parameters) > 0,
		Parameters:     compiledFn.Parameters,
		Args:           args,
		IsStreaming:    true,
	}

	return e.moduleManager.CreateModule(config)
}

// executeGoCode executes Go code and returns the output
func (e *FunctionExecutor) executeGoCode(tempDir string, args json.RawMessage, hasParams bool, params ...interface{}) (string, error) {
	var cmd *exec.Cmd
	if hasParams {
		e.log.Tracef("Executing Go program with args: %s", string(args))
		cmd = exec.Command("go", "run", ".")
		cmd.Dir = tempDir
		// Set up output buffers
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		// Set up stdin to provide the args
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", fmt.Errorf("failed to create stdin pipe: %v", err)
		}
		// Start the command first
		if err := cmd.Start(); err != nil {
			e.log.Errorf("Command start failed: %v", err)
			return "", fmt.Errorf("failed to start command: %v", err)
		}
		// Write args to stdin synchronously
		data := []byte("{}")
		if len(args) == 0 || string(args) == "null" {
			data = []byte("{}")
		} else {
			data = args
		}
		n, err := stdin.Write(data)
		if err != nil {
			e.log.Errorf("Failed to write to stdin: %v", err)
		} else {
			e.log.Debugf("Wrote %d bytes to stdin", n)
		}
		stdin.Close()
		// Wait for the command to complete
		err = cmd.Wait()
		output := stdoutBuf.String() + stderrBuf.String()
		if err != nil {
			e.log.Errorf("Go program failed: %v", err)
			return "", fmt.Errorf("execution failed: %v, output: %s", err, output)
		}
		return output, nil
	} else {
		e.log.Tracef("Executing Go program without args")
		cmd = exec.Command("go", "run", ".")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			e.log.Errorf("Go program failed: %v", err)
			return "", fmt.Errorf("execution failed: %v, output: %s", err, string(output))
		}
		return string(output), nil
	}
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
