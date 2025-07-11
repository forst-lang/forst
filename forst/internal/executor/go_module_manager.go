package executor

import (
	"fmt"
	"forst/internal/discovery"
	"os"
	"path/filepath"

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
	ModuleName     string
	PackageName    string
	FunctionName   string
	GoCode         string
	SupportsParams bool
	Parameters     []discovery.ParameterInfo
	Args           []byte
	IsStreaming    bool
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

// generateStandardMainGo generates the main.go content for standard execution
func (m *GoModuleManager) generateStandardMainGo(importPkg, alias string, config *ModuleConfig) string {
	if config.SupportsParams && len(config.Parameters) > 0 {
		param := config.Parameters[0]
		paramType := param.Type
		paramName := param.Name
		return fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"os"

	"%s"
)

func main() {
	var input %s.%s
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding input: %%v\n", err)
		os.Exit(1)
	}
	result := %s.%s(%s)
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
}
`, alias, importPkg, paramName, alias, paramType, paramName, alias, config.FunctionName, paramName)
	}

	return fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"

	"%s"
)

func main() {
	result := %s.%s()
	output, _ := json.Marshal(result)
	fmt.Printf("{\"result\":%%s}\n", string(output))
}
`, alias, importPkg, alias, config.FunctionName)
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
