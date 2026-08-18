package genplugin

import "strings"

const (
	// InvokePackage calls generated `$pkg.Fn(...)` from `@forst/gen/<pkg>`.
	InvokePackage = "package"
	// InvokeClient calls `createInvokeClient().invokeFunction(pkg, fn, args)`.
	InvokeClient = "client"
)

// PackageNamespace is the generated client handle (`$catalog`).
func PackageNamespace(pkg string) string {
	if pkg == "" {
		return "$_"
	}
	return "$" + TSIdentifier(pkg)
}

// PackageModuleSpecifier is `clientImport/pkg` (default `@forst/gen/catalog`).
func PackageModuleSpecifier(clientImport, pkg string) string {
	if clientImport == "" {
		clientImport = "@forst/gen"
	}
	clientImport = strings.TrimRight(clientImport, "/")
	return clientImport + "/" + pkg
}

// ClientModuleSpecifier is where `createInvokeClient` lives.
func ClientModuleSpecifier(clientImport string) string {
	if clientImport == "" || clientImport == "@forst/gen" {
		return "@forst/client"
	}
	return clientImport
}
