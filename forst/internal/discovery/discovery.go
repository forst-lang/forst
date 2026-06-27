package discovery

import (
	"errors"
	"fmt"
	"io"
	"sort"

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
	// IsResult and the result* fields apply when the sole return type is Result(Success, Failure).
	IsResult          bool   `json:"isResult,omitempty"`
	ResultSuccessType string `json:"resultSuccessType,omitempty"`
	ResultFailureType string `json:"resultFailureType,omitempty"`
	// IsGateway is true when the signature matches GatewayHandler (GatewayRequest -> Result(GatewayResponse, E)).
	IsGateway bool `json:"isGateway,omitempty"`
	FilePath          string `json:"filePath"`
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

	mainLogger := discoveryLogrusOrDiscard(d.log)

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
	goRoot := goload.GoWorkspaceForPackages(d.rootDir)

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

func discoveryLogrusOrDiscard(log Logger) *logrus.Logger {
	mainLogger, ok := log.(*logrus.Logger)
	if ok {
		return mainLogger
	}
	mainLogger = logrus.New()
	mainLogger.SetOutput(io.Discard)
	return mainLogger
}

// findForstFiles recursively finds all .ft files in the root directory
func (d *Discoverer) findForstFiles() ([]string, error) {
	if d.config == nil {
		return nil, ErrNilForstConfig
	}
	return d.config.FindForstFiles(d.rootDir)
}
