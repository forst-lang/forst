package modulecheck

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestPassCount_incrementsOnCheckModuleProviders(t *testing.T) {
	ResetPassCountForTest()
	log := logrus.New()
	log.SetOutput(io.Discard)

	dir := t.TempDir()
	if _, err := CheckModuleProviders(log, Options{ModuleRoot: dir, SkipGoLoad: true, SkipValidate: true}); err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	if got := PassCount(); got != 1 {
		t.Fatalf("PassCount()=%d want 1", got)
	}

	ResetPassCount()
	if PassCount() != 0 {
		t.Fatalf("PassCount after reset=%d want 0", PassCount())
	}
}
