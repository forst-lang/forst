package discovery

import (
	"errors"
	"fmt"
	"testing"

	"forst/internal/forstpkg"
)

// MockConfig implements configiface.ForstConfigIface for testing.
type MockConfig struct {
	files []string
	err   error
}

func (m *MockConfig) FindForstFiles(_ string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.files, nil
}

// MockLogger implements Logger interface for testing.
type MockLogger struct {
	debugMsgs []string
	infoMsgs  []string
	warnMsgs  []string
	errorMsgs []string
	traceMsgs []string
}

func (m *MockLogger) Debugf(format string, _ ...interface{}) {
	m.debugMsgs = append(m.debugMsgs, format)
}

func (m *MockLogger) Infof(format string, _ ...interface{}) {
	m.infoMsgs = append(m.infoMsgs, format)
}

func (m *MockLogger) Warnf(format string, _ ...interface{}) {
	m.warnMsgs = append(m.warnMsgs, format)
}

func (m *MockLogger) Errorf(format string, _ ...interface{}) {
	m.errorMsgs = append(m.errorMsgs, format)
}

func (m *MockLogger) Tracef(format string, _ ...interface{}) {
	m.traceMsgs = append(m.traceMsgs, format)
}

func TestPackageNameOrDefault(t *testing.T) {
	if got := forstpkg.PackageNameOrDefault("mypkg"); got != "mypkg" {
		t.Fatalf("non-empty: got %q", got)
	}
	if got := forstpkg.PackageNameOrDefault(""); got != "main" {
		t.Fatalf("empty: got %q want main", got)
	}
}

func TestDiscoverer_GetRootDir(t *testing.T) {
	d := NewDiscoverer("/expected/root", &MockLogger{}, &MockConfig{})
	if got := d.GetRootDir(); got != "/expected/root" {
		t.Fatalf("GetRootDir: got %q want %q", got, "/expected/root")
	}
}

func TestNewDiscoverer(t *testing.T) {
	logger := &MockLogger{}
	config := &MockConfig{}

	discoverer := NewDiscoverer("/test/path", logger, config)
	if discoverer == nil {
		t.Fatal("NewDiscoverer returned nil")
	}
	if discoverer.rootDir != "/test/path" {
		t.Errorf("Expected rootDir '/test/path', got '%s'", discoverer.rootDir)
	}
	if discoverer.log != logger {
		t.Error("Logger not properly set")
	}
	if discoverer.config != config {
		t.Error("Config not properly set")
	}
}

func TestDiscoverer_FindForstFiles_Success(t *testing.T) {
	logger := &MockLogger{}
	config := &MockConfig{
		files: []string{"/test/file1.ft", "/test/file2.ft"},
	}

	discoverer := NewDiscoverer("/test/path", logger, config)
	files, err := discoverer.findForstFiles()
	if err != nil {
		t.Fatalf("findForstFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	expectedFiles := map[string]bool{
		"/test/file1.ft": false,
		"/test/file2.ft": false,
	}
	for _, file := range files {
		expectedFiles[file] = true
	}
	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s not found", file)
		}
	}
}

func TestDiscoverer_FindForstFiles_Error(t *testing.T) {
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, &MockConfig{err: fmt.Errorf("config error")})
	_, err := discoverer.findForstFiles()
	if err == nil {
		t.Error("Expected error when config returns error")
	}
}

func TestDiscoverer_FindForstFiles_NilConfig(t *testing.T) {
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, nil)
	_, err := discoverer.findForstFiles()
	if err == nil {
		t.Error("Expected error when config is nil")
	}
	if !errors.Is(err, ErrNilForstConfig) {
		t.Errorf("Expected ErrNilForstConfig, got: %v", err)
	}
}
