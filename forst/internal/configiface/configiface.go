package configiface

type ForstConfigIface interface {
	FindForstFiles(rootDir string) ([]string, error)
}
