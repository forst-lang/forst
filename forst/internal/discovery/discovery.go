package discovery

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"forst/internal/ast"
	"forst/internal/configiface"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

var ErrNilForstConfig = errors.New("ForstConfig is required for file discovery")

// FunctionInfo represents a discovered public function
type FunctionInfo struct {
	Package            string          `json:"package"`
	Name               string          `json:"name"`
	SupportsStreaming  bool            `json:"supportsStreaming"`
	InputType          string          `json:"inputType"`
	OutputType         string          `json:"outputType"`
	Parameters         []ParameterInfo `json:"parameters"`
	ReturnType         string          `json:"returnType"`
	ReturnTypes        []string        `json:"returnTypes"`        // Track all return types
	HasMultipleReturns bool            `json:"hasMultipleReturns"` // Whether function returns multiple values
	FilePath           string          `json:"filePath"`
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
	Tracef(format string, args ...interface{})
}

// NewDiscoverer creates a new function discoverer
func NewDiscoverer(rootDir string, log Logger, config configiface.ForstConfigIface) *Discoverer {
	return &Discoverer{
		rootDir: rootDir,
		log:     log,
		config:  config,
	}
}

// GetRootDir returns the root directory for file discovery
func (d *Discoverer) GetRootDir() string {
	return d.rootDir
}

// DiscoverFunctions scans all Forst files and discovers public functions
func (d *Discoverer) DiscoverFunctions() (map[string]map[string]FunctionInfo, error) {
	functions := make(map[string]map[string]FunctionInfo)

	// Find all .ft files
	ftFiles, err := d.findForstFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find Forst files: %v", err)
	}

	d.log.Debugf("Found %d Forst files to scan", len(ftFiles))

	mainLogger, ok := d.log.(*logrus.Logger)
	if !ok {
		mainLogger = logrus.New()
		mainLogger.SetOutput(nil)
	}

	parsed := make(map[string][]ast.Node)
	for _, filePath := range ftFiles {
		nodes, err := forstpkg.ParseForstFile(mainLogger, filePath)
		if err != nil {
			d.log.Warnf("Failed to parse %s: %v", filePath, err)
			continue
		}
		parsed[filePath] = nodes
	}

	byPackage := make(map[string][]string)
	for path, nodes := range parsed {
		pkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
		byPackage[pkg] = append(byPackage[pkg], path)
	}

	totalFunctions := 0
	goRoot := goload.FindModuleRoot(d.rootDir)

	for packageName, paths := range byPackage {
		sort.Strings(paths)
		var astLists [][]ast.Node
		for _, p := range paths {
			astLists = append(astLists, parsed[p])
		}
		merged := forstpkg.MergePackageASTs(astLists)

		tc := typechecker.New(mainLogger, false)
		tc.GoWorkspaceDir = goRoot
		if err := tc.CheckTypes(merged); err != nil {
			d.log.Debugf("Type checking failed for package %s: %v", packageName, err)
		}

		for _, filePath := range paths {
			fileFunctions := d.discoverFunctionsInParsedFile(parsed[filePath], filePath, packageName, tc)
			if len(fileFunctions) == 0 {
				continue
			}
			if functions[packageName] == nil {
				functions[packageName] = make(map[string]FunctionInfo)
			}
			for name, fn := range fileFunctions {
				functions[packageName][name] = fn
				totalFunctions++
			}
		}
	}

	d.log.Debugf("Discovered %d package(s) with public functions, in total %d public functions", len(functions), totalFunctions)
	for packageName, pkgFuncs := range functions {
		for name := range pkgFuncs {
			d.log.Debugf("- %s.%s", packageName, name)
		}
	}
	return functions, nil
}

// findForstFiles recursively finds all .ft files in the root directory
func (d *Discoverer) findForstFiles() ([]string, error) {
	if d.config == nil {
		return nil, ErrNilForstConfig
	}
	return d.config.FindForstFiles(d.rootDir)
}

// discoverFunctionsInParsedFile collects public functions from an already-parsed file using a
// typechecker that was run on the merged package AST (cross-file types and imports).
func (d *Discoverer) discoverFunctionsInParsedFile(nodes []ast.Node, filePath, packageName string, tc *typechecker.TypeChecker) map[string]FunctionInfo {
	fileFunctions := make(map[string]FunctionInfo)
	d.extractFunctionsFromNodes(nodes, packageName, filePath, fileFunctions, tc)
	return fileFunctions
}

// extractPackageNameFromAST extracts the package name from AST nodes
func (d *Discoverer) extractPackageNameFromAST(nodes []ast.Node) string {
	for _, node := range nodes {
		if pkgNode, ok := node.(ast.PackageNode); ok {
			return string(pkgNode.Ident.ID)
		}
	}
	return ""
}

// extractFunctionsFromNodes extracts public functions from AST nodes
func (d *Discoverer) extractFunctionsFromNodes(nodes []ast.Node, packageName, filePath string, functions map[string]FunctionInfo, tc *typechecker.TypeChecker) {
	d.log.Tracef("Processing %d AST nodes for package %s in file %s", len(nodes), packageName, filePath)
	for i, node := range nodes {
		d.log.Tracef("Processing node %d: %T", i, node)
		d.extractFunctionsFromNode(node, packageName, filePath, functions, tc)
	}
}

// extractFunctionsFromNode extracts public functions from a single AST node
func (d *Discoverer) extractFunctionsFromNode(node ast.Node, packageName, filePath string, functions map[string]FunctionInfo, tc *typechecker.TypeChecker) {
	switch n := node.(type) {
	case ast.FunctionNode:
		d.extractFunctionsFromNode(&n, packageName, filePath, functions, tc)
	case *ast.FunctionNode:
		d.log.Tracef("Found function node: %s", n.Ident.ID)
		if len(n.Ident.ID) > 0 && unicode.IsUpper(rune(n.Ident.ID[0])) {
			d.log.Tracef("Function %s is public (starts with uppercase)", n.Ident.ID)
			fnInfo := FunctionInfo{
				Package:           packageName,
				Name:              string(n.Ident.ID),
				SupportsStreaming: d.analyzeStreamingSupport(n, tc),
				FilePath:          filePath,
			}
			for _, param := range n.Params {
				fnInfo.Parameters = append(fnInfo.Parameters, ParameterInfo{
					Name: param.GetIdent(),
					Type: d.resolveTypeName(param.GetType(), tc),
				})
			}

			// Use typechecker's inferred return types if available
			var returnTypes []ast.TypeNode
			if tc != nil {
				if sig, exists := tc.Functions[n.Ident.ID]; exists && len(sig.ReturnTypes) > 0 {
					returnTypes = sig.ReturnTypes
					d.log.Debugf("Using typechecker's inferred return types for function %s: %v", n.Ident.ID, returnTypes)
				} else {
					// Fall back to parser's return types
					returnTypes = n.ReturnTypes
					d.log.Debugf("Using parser's return types for function %s: %v", n.Ident.ID, returnTypes)
				}
			} else {
				// No typechecker available, use parser's return types
				returnTypes = n.ReturnTypes
				d.log.Debugf("No typechecker available, using parser's return types for function %s: %v", n.Ident.ID, returnTypes)
			}

			if len(returnTypes) > 0 {
				fnInfo.ReturnType = d.resolveTypeName(returnTypes[0], tc)
				fnInfo.ReturnTypes = make([]string, len(returnTypes))
				for i, rt := range returnTypes {
					fnInfo.ReturnTypes[i] = d.resolveTypeName(rt, tc)
				}
				fnInfo.HasMultipleReturns = len(returnTypes) > 1
				d.log.Debugf("Function %s has %d return types: %v, HasMultipleReturns: %v", n.Ident.ID, len(returnTypes), fnInfo.ReturnTypes, fnInfo.HasMultipleReturns)
			} else {
				d.log.Debugf("Function %s has no return types", n.Ident.ID)
			}
			fnInfo.InputType = d.determineInputType(fnInfo.Parameters)
			fnInfo.OutputType = fnInfo.ReturnType
			functions[string(n.Ident.ID)] = fnInfo
			d.log.Tracef("Discovered public function: %s.%s", packageName, n.Ident.ID)
		} else {
			d.log.Tracef("Function %s is private (starts with lowercase)", n.Ident.ID)
		}
	default:
		d.log.Tracef("Node type %T is not a function", node)
	}
}

// analyzeStreamingSupport determines if a function supports streaming
func (d *Discoverer) analyzeStreamingSupport(fn *ast.FunctionNode, tc *typechecker.TypeChecker) bool {
	// Check function name for streaming indicators
	name := strings.ToLower(string(fn.Ident.ID))
	streamingKeywords := []string{"stream", "process", "batch", "pipeline"}

	for _, keyword := range streamingKeywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}

	// Check return type for streaming indicators
	var returnTypes []ast.TypeNode
	if tc != nil {
		if sig, exists := tc.Functions[fn.Ident.ID]; exists && len(sig.ReturnTypes) > 0 {
			returnTypes = sig.ReturnTypes
		} else {
			returnTypes = fn.ReturnTypes
		}
	} else {
		returnTypes = fn.ReturnTypes
	}

	if len(returnTypes) > 0 {
		// Check the original type name before it gets converted to hash-based names
		originalTypeName := string(returnTypes[0].Ident)
		if strings.Contains(strings.ToLower(originalTypeName), "stream") ||
			strings.Contains(strings.ToLower(originalTypeName), "channel") {
			return true
		}
	}

	return false
}

// typeToString converts an AST type to a string representation
func (d *Discoverer) typeToString(t ast.TypeNode) string {
	return t.String()
}

// resolveTypeName converts an AST type to a string representation using the typechecker
func (d *Discoverer) resolveTypeName(t ast.TypeNode, tc *typechecker.TypeChecker) string {
	if tc != nil {
		name, err := tc.GetAliasedTypeName(t, typechecker.GetAliasedTypeNameOptions{AllowStructuralAlias: true})
		if err == nil {
			return name
		}
	}
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
