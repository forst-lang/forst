package transformerts

import (
	"strings"
)

// TypeScriptOutput holds the generated TypeScript code
type TypeScriptOutput struct {
	PackageName string
	Types       []string
	Functions   []string
	ClientCode  []string // Per-package client code
	MainClient  string   // Main client class
	TypesFile   string   // Centralized types file
}

// AddType adds a type definition to the output
func (o *TypeScriptOutput) AddType(tsType string) {
	o.Types = append(o.Types, tsType)
}

// AddFunction adds a function definition to the output
func (o *TypeScriptOutput) AddFunction(tsFunction string) {
	o.Functions = append(o.Functions, tsFunction)
}

// SetPackageName sets the package name
func (o *TypeScriptOutput) SetPackageName(name string) {
	o.PackageName = name
}

// GenerateTypesFile returns the types file content
func (o *TypeScriptOutput) GenerateTypesFile() string {
	return o.TypesFile
}

// GenerateClientFile returns the package client file content
func (o *TypeScriptOutput) GenerateClientFile() string {
	return strings.Join(o.ClientCode, "\n")
}

// GenerateMainClient returns the main client file content
func (o *TypeScriptOutput) GenerateMainClient() string {
	return o.MainClient
}
