package configiface

// ForstConfigIface is the minimal file-discovery contract that discovery
// and executor depend on instead of the concrete ftconfig.Config type.
// Keeping it in its own package lets ftconfig implement it without those
// packages importing ftconfig's full JSON-parsing surface, and lets tests
// substitute StaticFilesConfig or a mock.
type ForstConfigIface interface {
	FindForstFiles(rootDir string) ([]string, error)
}
