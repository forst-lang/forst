package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"forst/internal/ast"
	"forst/internal/configiface"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/typechecker"

	logrus "github.com/sirupsen/logrus"
)

// FunctionInfo represents a discovered public function
type FunctionInfo struct {
	Package           string          `json:"package"`
	Name              string          `json:"name"`
	SupportsStreaming bool            `json:"supportsStreaming"`
	InputType         string          `json:"inputType"`
	OutputType        string          `json:"outputType"`
	Parameters        []ParameterInfo `json:"parameters"`
	ReturnType        string          `json:"returnType"`
	FilePath          string          `json:"filePath"`
}

// ParameterInfo represents a function parameter
type ParameterInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Discoverer handles function discovery in Forst packages
type Discoverer struct {
	rootDir string
	log     Logger
	config  configiface.ForstConfigIface
}

// Logger interface for discovery logging
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// NewDiscoverer creates a new function discoverer
func NewDiscoverer(rootDir string, log Logger, config configiface.ForstConfigIface) *Discoverer {
	return &Discoverer{
		rootDir: rootDir,
		log:     log,
		config:  config,
	}
}

// DiscoverFunctions scans all Forst files and discovers public functions
func (d *Discoverer) DiscoverFunctions() (map[string]map[string]FunctionInfo, error) {
	functions := make(map[string]map[string]FunctionInfo)

	// Find all .ft files
	ftFiles, err := d.findForstFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find Forst files: %v", err)
	}

	d.log.Infof("Found %d Forst files to scan", len(ftFiles))

	// Process each file
	for _, filePath := range ftFiles {
		packageName := d.extractPackageName(filePath)

		fileFunctions, err := d.discoverFunctionsInFile(filePath, packageName)
		if err != nil {
			d.log.Warnf("Failed to discover functions in %s: %v", filePath, err)
			continue
		}

		if len(fileFunctions) > 0 {
			if functions[packageName] == nil {
				functions[packageName] = make(map[string]FunctionInfo)
			}

			for name, fn := range fileFunctions {
				functions[packageName][name] = fn
			}
		}
	}

	d.log.Infof("Discovered %d packages with public functions", len(functions))
	return functions, nil
}

// findForstFiles recursively finds all .ft files in the root directory
func (d *Discoverer) findForstFiles() ([]string, error) {
	if d.config == nil {
		return nil, fmt.Errorf("ForstConfig is required for file discovery")
	}
	return d.config.FindForstFiles(d.rootDir)
}

// extractPackageName extracts the package name from file path
func (d *Discoverer) extractPackageName(filePath string) string {
	relPath, err := filepath.Rel(d.rootDir, filePath)
	if err != nil {
		return "unknown"
	}

	dir := filepath.Dir(relPath)
	if dir == "." {
		return "main"
	}

	return strings.ReplaceAll(dir, string(filepath.Separator), "_")
}

// discoverFunctionsInFile discovers public functions in a single Forst file
func (d *Discoverer) discoverFunctionsInFile(filePath, packageName string) (map[string]FunctionInfo, error) {
	functions := make(map[string]FunctionInfo)

	// Read and parse the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Create lexer and tokenize
	l := lexer.New(content, filePath, logrus.New())
	tokens := l.Lex()

	// Create parser and parse the file
	p := parser.New(tokens, filePath, logrus.New())
	nodes, err := p.ParseFile()
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %v", err)
	}

	// Type check to get function signatures
	tc := typechecker.New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		d.log.Debugf("Type checking failed for %s: %v", filePath, err)
		// Continue without type checking for discovery
	}

	// Extract functions from AST nodes
	d.extractFunctionsFromNodes(nodes, packageName, filePath, functions)

	return functions, nil
}

// extractFunctionsFromNodes extracts public functions from AST nodes
func (d *Discoverer) extractFunctionsFromNodes(nodes []ast.Node, packageName, filePath string, functions map[string]FunctionInfo) {
	for _, node := range nodes {
		d.extractFunctionsFromNode(node, packageName, filePath, functions)
	}
}

// extractFunctionsFromNode extracts public functions from a single AST node
func (d *Discoverer) extractFunctionsFromNode(node ast.Node, packageName, filePath string, functions map[string]FunctionInfo) {
	switch n := node.(type) {
	case *ast.FunctionNode:
		// Check if function is public (starts with uppercase)
		if len(n.Ident.ID) > 0 && unicode.IsUpper(rune(n.Ident.ID[0])) {
			fnInfo := FunctionInfo{
				Package:           packageName,
				Name:              string(n.Ident.ID),
				SupportsStreaming: d.analyzeStreamingSupport(n),
				FilePath:          filePath,
			}

			// Extract parameter information
			for _, param := range n.Params {
				fnInfo.Parameters = append(fnInfo.Parameters, ParameterInfo{
					Name: param.GetIdent(),
					Type: d.typeToString(param.GetType()),
				})
			}

			// Extract return type
			if len(n.ReturnTypes) > 0 {
				fnInfo.ReturnType = d.typeToString(n.ReturnTypes[0])
			}

			// Determine input/output types for API
			fnInfo.InputType = d.determineInputType(fnInfo.Parameters)
			fnInfo.OutputType = fnInfo.ReturnType

			functions[string(n.Ident.ID)] = fnInfo
			d.log.Debugf("Discovered public function: %s.%s", packageName, n.Ident.ID)
		}
	}
}

// analyzeStreamingSupport determines if a function supports streaming
func (d *Discoverer) analyzeStreamingSupport(fn *ast.FunctionNode) bool {
	// Check function name for streaming indicators
	name := strings.ToLower(string(fn.Ident.ID))
	streamingKeywords := []string{"stream", "process", "batch", "pipeline"}

	for _, keyword := range streamingKeywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}

	// Check return type for streaming indicators
	if len(fn.ReturnTypes) > 0 {
		returnType := d.typeToString(fn.ReturnTypes[0])
		if strings.Contains(strings.ToLower(returnType), "stream") ||
			strings.Contains(strings.ToLower(returnType), "channel") {
			return true
		}
	}

	return false
}

// typeToString converts an AST type to a string representation
func (d *Discoverer) typeToString(t ast.TypeNode) string {
	return t.String()
}

// determineInputType determines the input type for API purposes
func (d *Discoverer) determineInputType(params []ParameterInfo) string {
	if len(params) == 0 {
		return "void"
	}
	if len(params) == 1 {
		return params[0].Type
	}
	return "json" // Multiple parameters use JSON
}
