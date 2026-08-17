package compiler

// Build metadata injected by cmd/forst at startup.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// SetBuildMetadata sets compiler build metadata for manifests and diagnostics.
func SetBuildMetadata(v, c, d string) {
	version = v
	commit = c
	date = d
}

// CompilerVersion returns the injected compiler version string.
func CompilerVersion() string {
	return version
}
