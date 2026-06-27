package discovery

import "forst/internal/configiface"

// StaticFilesConfig implements ForstConfigIface with a fixed file list (tests and tooling).
type StaticFilesConfig struct {
	Files []string
	Err   error
}

func (m *StaticFilesConfig) FindForstFiles(_ string) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Files, nil
}

// NewStaticFilesConfig returns a config that discovers exactly the given .ft paths.
func NewStaticFilesConfig(files []string) configiface.ForstConfigIface {
	return &StaticFilesConfig{Files: files}
}
