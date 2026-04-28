package executor

import (
	"fmt"
	"forst/gateway"
	"forst/internal/discovery"
	"forst/internal/typechecker"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	logrus "github.com/sirupsen/logrus"
)

// GoModuleManager handles creation and management of temporary Go modules
type GoModuleManager struct {
	log *logrus.Logger
}

// NewGoModuleManager creates a new Go module manager
func NewGoModuleManager(log *logrus.Logger) *GoModuleManager {
	return &GoModuleManager{
		log: log,
	}
}

// libraryExecutorUserPackage is the Go package name used in the executor temp module when Forst
// emits merged code as `package main`. Go forbids importing package main from another package, so
// we rewrite the user file to this library name and import `module/forstexec` from the wrapper main.
const libraryExecutorUserPackage = "forstexec"

func libraryImportPackageName(forstPackage string) string {
	if forstPackage == "main" {
		return libraryExecutorUserPackage
	}
	return forstPackage
}

// rewriteUserGoCodeForExecutorLibrary maps `package main` to libraryExecutorUserPackage when the
// discovery package is main (see libraryImportPackageName).
func rewriteUserGoCodeForExecutorLibrary(goCode string, forstPackage string) string {
	if forstPackage != "main" {
		return goCode
	}
	lines := strings.SplitN(goCode, "\n", 2)
	if len(lines) == 0 {
		return goCode
	}
	first := strings.TrimSpace(lines[0])
	if first != "package main" {
		return goCode
	}
	if len(lines) == 1 {
		return fmt.Sprintf("package %s", libraryExecutorUserPackage)
	}
	return fmt.Sprintf("package %s\n%s", libraryExecutorUserPackage, lines[1])
}

// ModuleConfig holds configuration for creating a Go module
type ModuleConfig struct {
	ModuleName         string
	PackageName        string
	FunctionName       string
	GoCode             string
	SupportsParams     bool
	Parameters         []discovery.ParameterInfo
	Args               []byte
	IsStreaming        bool
	HasMultipleReturns bool
	// ForstModuleReplaceAbs is the absolute path to the Forst SDK module (module "forst") for
	// `replace forst => ...` so generated code may import merge-path stdlib packages (e.g. gateway).
	ForstModuleReplaceAbs string
}

// CreateModule creates a temporary Go module with the specified configuration
func (m *GoModuleManager) CreateModule(config *ModuleConfig) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("forst-%s-*", config.ModuleName))
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}

	// Create go.mod file
	if err := m.createGoMod(tempDir, config.ModuleName, config.ForstModuleReplaceAbs); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}

	// Create main.go file
	if err := m.createMainGo(tempDir, config); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}

	// Create package directory and file
	if err := m.createPackageFile(tempDir, config); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}

	if config.ForstModuleReplaceAbs != "" {
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			_ = os.RemoveAll(tempDir)
			return "", fmt.Errorf("go mod tidy in executor temp module: %w", err)
		}
	}

	// Debug: list tempDir contents
	m.logTempDirContents(tempDir)

	return tempDir, nil
}

// createGoMod creates the go.mod file
func (m *GoModuleManager) createGoMod(tempDir, moduleName, forstReplaceAbs string) error {
	goModPath := filepath.Join(tempDir, "go.mod")
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n\ngo 1.21\n", moduleName)
	if forstReplaceAbs != "" {
		fmt.Fprintf(&b, "\nrequire forst v0.0.0\n")
		fmt.Fprintf(&b, "replace forst => %s\n", filepath.ToSlash(forstReplaceAbs))
	}
	return os.WriteFile(goModPath, []byte(b.String()), 0644)
}

// createMainGo creates the main.go file
func (m *GoModuleManager) createMainGo(tempDir string, config *ModuleConfig) error {
	importPkg := config.ModuleName + "/" + libraryImportPackageName(config.PackageName)
	alias := fmt.Sprintf("%s_%s", config.PackageName, generateRandomString(4))

	var mainGoContent string
	if config.IsStreaming {
		mainGoContent = m.generateStreamingMainGo(importPkg, alias, config)
	} else {
		mainGoContent = m.generateStandardMainGo(importPkg, alias, config)
	}

	mainGoPath := filepath.Join(tempDir, "main.go")
	return os.WriteFile(mainGoPath, []byte(mainGoContent), 0644)
}

func buildMainGoHeader(importPkg string, alias string, needsGateway bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, `package main

import (
	"encoding/json"
	"fmt"
	"os"
`)
	if needsGateway {
		fmt.Fprintf(&b, `
	gateway "%s"
`, gateway.StdlibImportPath)
	}
	fmt.Fprintf(&b, `
	%s "%s"
)
`, alias, importPkg)
	return b.String()
}

func parameterTypesForGatewayImport(params []discovery.ParameterInfo) []string {
	out := make([]string, len(params))
	for i := range params {
		out[i] = params[i].Type
	}
	return out
}

// mainGoInvokeErrReturn emits JSON consumed by parseExecutionOutput for Result Err (RFC §18.1).
func mainGoInvokeErrReturn() string {
	return `	if err != nil {
		out, _ := json.Marshal(map[string]any{"success": false, "error": err.Error()})
		fmt.Println(string(out))
		os.Exit(0)
	}
`
}

// generateStandardMainGo generates the main.go content for standard execution
func (m *GoModuleManager) generateStandardMainGo(importPkg, alias string, config *ModuleConfig) string {
	gwImport := gateway.TempModuleNeedsGatewayImport(parameterTypesForGatewayImport(config.Parameters))
	containerName := "input"
	if config.SupportsParams && len(config.Parameters) > 0 {
		// Build parameter types and names for the function call
		var paramTypes []string
		var paramNames []string

		for _, param := range config.Parameters {
			paramType := param.Type

			// For Go built-in types, use the type name directly without package prefix
			var inputType string
			if typechecker.IsGoBuiltinType(paramType) {
				inputType = paramType
			} else if strings.Contains(paramType, ".") {
				inputType = paramType
			} else {
				inputType = alias + "." + paramType
			}
			paramTypes = append(paramTypes, inputType)
			paramNames = append(paramNames, param.Name)
		}

		if config.HasMultipleReturns {
			return fmt.Sprintf(`%s

func main() {
	%s
	result, err := %s.%s(%s)
%s
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias, gwImport), buildParameterExtraction(containerName, paramNames, paramTypes), alias, config.FunctionName, strings.Join(paramNames, ", "), mainGoInvokeErrReturn())
		}
		return fmt.Sprintf(`%s

func main() {
	%s
	result := %s.%s(%s)
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias, gwImport), buildParameterExtraction(containerName, paramNames, paramTypes), alias, config.FunctionName, strings.Join(paramNames, ", "))
	}

	if config.HasMultipleReturns {
		return fmt.Sprintf(`%s

func main() {
	result, err := %s.%s()
%s
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias, gwImport), alias, config.FunctionName, mainGoInvokeErrReturn())
	}
	return fmt.Sprintf(`%s

func main() {
	result := %s.%s()
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias, gwImport), alias, config.FunctionName)
}

// buildParameterExtraction generates Go code to extract parameters from JSON input
func buildParameterExtraction(containerName string, paramNames []string, paramTypes []string) string {
	// Build array definition and decode JSON input
	var arrayDef strings.Builder
	fmt.Fprintf(&arrayDef, "var %sJSON = make([]interface{}, %d)", containerName, len(paramNames))
	fmt.Fprintf(&arrayDef, `
	if err := json.NewDecoder(os.Stdin).Decode(&%sJSON); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %%v\n", err)
		os.Exit(1)
	}`, containerName)

	// Build parameter extraction code
	var paramCode strings.Builder
	for i, name := range paramNames {
		if i < len(paramTypes) {
			paramType := paramTypes[i]

			// Check if it's a built-in type or struct
			if typechecker.IsGoBuiltinType(paramType) {
				// For built-in types, use type assertion with conversion for numbers
				// Note: JSON numbers are decoded as float64 in Go, so we need to handle
				// both float64 and int cases for numeric types. When JSON contains
				// integers like 29, they are decoded as float64(29.0), not int(29).
				// Direct type assertion to int will fail, so we convert float64 to int.
				if paramType == "int" {
					fmt.Fprintf(&paramCode, `
        var %s %s
        if v, ok := %sJSON[%d].(float64); ok {
        	%s = int(v)
        } else if v, ok := %sJSON[%d].(int); ok {
        	%s = v
        } else {
        	fmt.Fprintf(os.Stderr, "Error: parameter %%d has wrong type, expected %s\n", %d)
        	os.Exit(1)
        }`, name, paramType, containerName, i, name, containerName, i, name, paramType, i)
				} else {
					fmt.Fprintf(&paramCode, `
        var %s %s
        if v, ok := %sJSON[%d].(%s); ok {
        	%s = v
        } else {
        	fmt.Fprintf(os.Stderr, "Error: parameter %%d has wrong type, expected %s\n", %d)
        	os.Exit(1)
        }`, name, paramType, containerName, i, paramType, name, paramType, i)
				}
			} else {
				// For struct types, use json.Unmarshal
				fmt.Fprintf(&paramCode, `
        var %s %s
        paramBytes, _ := json.Marshal(%sJSON[%d])
        if err := json.Unmarshal(paramBytes, &%s); err != nil {
        	fmt.Fprintf(os.Stderr, "Error unmarshaling parameter %%d: %%v\n", %d, err)
        	os.Exit(1)
        }`, name, paramType, containerName, i, name, i)
			}
		}
	}

	return arrayDef.String() + paramCode.String()
}

// generateStreamingMainGo generates the main.go content for streaming execution
func (m *GoModuleManager) generateStreamingMainGo(importPkg, alias string, config *ModuleConfig) string {
	gwImport := gateway.TempModuleNeedsGatewayImport(parameterTypesForGatewayImport(config.Parameters))
	return fmt.Sprintf(`%s

func main() {
	var args json.RawMessage = %s
	
	results := %s.%s(args)
	for result := range results {
		output, _ := json.Marshal(result)
		fmt.Println(string(output))
	}
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias, gwImport), string(config.Args), alias, config.FunctionName)
}

// createPackageFile creates the package directory and Go file
func (m *GoModuleManager) createPackageFile(tempDir string, config *ModuleConfig) error {
	libPkg := libraryImportPackageName(config.PackageName)
	packageDir := filepath.Join(tempDir, libPkg)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create package dir: %v", err)
	}

	code := rewriteUserGoCodeForExecutorLibrary(config.GoCode, config.PackageName)
	if m.log != nil && config.PackageName == "main" {
		m.log.Tracef("executor temp module: user Go package main -> %s for importable library", libraryExecutorUserPackage)
	}
	packageGoPath := filepath.Join(packageDir, fmt.Sprintf("%s.go", libPkg))
	return os.WriteFile(packageGoPath, []byte(code), 0644)
}

// logTempDirContents logs the contents of the temporary directory for debugging
func (m *GoModuleManager) logTempDirContents(tempDir string) {
	if m.log == nil {
		return
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			m.log.Tracef("Temp dir contains subdir: %s", entry.Name())
			subEntries, _ := os.ReadDir(filepath.Join(tempDir, entry.Name()))
			for _, subEntry := range subEntries {
				m.log.Tracef("  - %s/%s", entry.Name(), subEntry.Name())
			}
		} else {
			m.log.Tracef("Temp dir contains file: %s", entry.Name())
		}
	}
}
