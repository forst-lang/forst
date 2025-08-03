package executor

import (
	"fmt"
	"forst/internal/discovery"
	"forst/internal/typechecker"
	"os"
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
}

// CreateModule creates a temporary Go module with the specified configuration
func (m *GoModuleManager) CreateModule(config *ModuleConfig) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("forst-%s-*", config.ModuleName))
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}

	// Create go.mod file
	if err := m.createGoMod(tempDir, config.ModuleName); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Create main.go file
	if err := m.createMainGo(tempDir, config); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Create package directory and file
	if err := m.createPackageFile(tempDir, config); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Debug: list tempDir contents
	m.logTempDirContents(tempDir)

	return tempDir, nil
}

// createGoMod creates the go.mod file
func (m *GoModuleManager) createGoMod(tempDir, moduleName string) error {
	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)
	return os.WriteFile(goModPath, []byte(goModContent), 0644)
}

// createMainGo creates the main.go file
func (m *GoModuleManager) createMainGo(tempDir string, config *ModuleConfig) error {
	importPkg := config.ModuleName + "/" + config.PackageName
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

func buildMainGoHeader(importPkg string, alias string) string {
	return fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"os"

	%s "%s"
)
	`, alias, importPkg)
}

// generateStandardMainGo generates the main.go content for standard execution
func (m *GoModuleManager) generateStandardMainGo(importPkg, alias string, config *ModuleConfig) string {
	containerName := "input"
	if config.SupportsParams && len(config.Parameters) > 0 {
		// Build parameter types and names for the function call
		var paramTypes []string
		var paramNames []string

		for _, param := range config.Parameters {
			paramType := param.Type

			// For Go built-in types, use the type name directly without package prefix
			inputType := paramType
			if typechecker.IsGoBuiltinType(paramType) {
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
	if err != nil {
		fmt.Fprintf(os.Stderr, "Function execution failed: %%v\n", err)
		os.Exit(1)
	}
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias), buildParameterExtraction(containerName, paramNames, paramTypes), alias, config.FunctionName, strings.Join(paramNames, ", "))
		} else {
			return fmt.Sprintf(`%s

func main() {
	%s
	result := %s.%s(%s)
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias), buildParameterExtraction(containerName, paramNames, paramTypes), alias, config.FunctionName, strings.Join(paramNames, ", "))
		}
	}

	if config.HasMultipleReturns {
		return fmt.Sprintf(`%s

func main() {
	result, err := %s.%s()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Function execution failed: %%v\n", err)
		os.Exit(1)
	}
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias), alias, config.FunctionName)
	} else {
		return fmt.Sprintf(`%s

func main() {
	result := %s.%s()
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
	os.Exit(0)
}
`, buildMainGoHeader(importPkg, alias), alias, config.FunctionName)
	}
}

// buildParameterExtraction generates Go code to extract parameters from JSON input
func buildParameterExtraction(containerName string, paramNames []string, paramTypes []string) string {
	// Build array definition and decode JSON input
	var arrayDef strings.Builder
	arrayDef.WriteString(fmt.Sprintf("var %sJSON = make([]interface{}, %d)", containerName, len(paramNames)))
	arrayDef.WriteString(fmt.Sprintf(`
	if err := json.NewDecoder(os.Stdin).Decode(&%sJSON); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %%v\n", err)
		os.Exit(1)
	}`, containerName))

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
					paramCode.WriteString(fmt.Sprintf(`
        var %s %s
        if v, ok := %sJSON[%d].(float64); ok {
        	%s = int(v)
        } else if v, ok := %sJSON[%d].(int); ok {
        	%s = v
        } else {
        	fmt.Fprintf(os.Stderr, "Error: parameter %%d has wrong type, expected %s\n", %d)
        	os.Exit(1)
        }`, name, paramType, containerName, i, name, containerName, i, name, paramType, i))
				} else {
					paramCode.WriteString(fmt.Sprintf(`
        var %s %s
        if v, ok := %sJSON[%d].(%s); ok {
        	%s = v
        } else {
        	fmt.Fprintf(os.Stderr, "Error: parameter %%d has wrong type, expected %s\n", %d)
        	os.Exit(1)
        }`, name, paramType, containerName, i, paramType, name, paramType, i))
				}
			} else {
				// For struct types, use json.Unmarshal
				paramCode.WriteString(fmt.Sprintf(`
        var %s %s
        paramBytes, _ := json.Marshal(%sJSON[%d])
        if err := json.Unmarshal(paramBytes, &%s); err != nil {
        	fmt.Fprintf(os.Stderr, "Error unmarshaling parameter %%d: %%v\n", %d, err)
        	os.Exit(1)
        }`, name, paramType, containerName, i, name, i))
			}
		}
	}

	return arrayDef.String() + paramCode.String()
}

// generateStreamingMainGo generates the main.go content for streaming execution
func (m *GoModuleManager) generateStreamingMainGo(importPkg, alias string, config *ModuleConfig) string {
	return fmt.Sprintf(`package main

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
	os.Exit(0)
}
`, string(config.Args), config.PackageName, config.FunctionName)
}

// createPackageFile creates the package directory and Go file
func (m *GoModuleManager) createPackageFile(tempDir string, config *ModuleConfig) error {
	// Create package directory
	packageDir := filepath.Join(tempDir, config.PackageName)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create package dir: %v", err)
	}

	// Write the compiled Go code to the package file
	packageGoPath := filepath.Join(packageDir, fmt.Sprintf("%s.go", config.PackageName))
	return os.WriteFile(packageGoPath, []byte(config.GoCode), 0644)
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
