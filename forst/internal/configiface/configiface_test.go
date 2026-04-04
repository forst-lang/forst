package configiface

import "testing"

type fakeConfig struct{}

func (fakeConfig) FindForstFiles(string) ([]string, error) { return nil, nil }

func TestForstConfigIfaceContract(t *testing.T) {
	t.Parallel()
	var _ ForstConfigIface = fakeConfig{}
}
