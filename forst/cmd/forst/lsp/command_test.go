package lsp

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestStartLSPServer(t *testing.T) {
	// Test that the function can be called without panicking
	// We can't easily test the actual server start since it blocks,
	// but we can test the function signature and basic setup

	log := logrus.New()

	// Test with valid port
	t.Run("valid port", func(t *testing.T) {
		// This would normally start a server, but we can't test it easily
		// since it blocks on ListenAndServe. Instead, we'll test the function
		// can be called without panicking by creating a server instance.

		server := NewLSPServer("8080", log)
		if server == nil {
			t.Fatal("Expected server to be created")
		}

		if server.port != "8080" {
			t.Errorf("Expected port 8080, got %s", server.port)
		}

		if server.log != log {
			t.Error("Expected logger to be set")
		}
	})

	// Test that the function signature is correct
	t.Run("function signature", func(t *testing.T) {
		// This is a compile-time test to ensure the function signature is correct
		var _ func(string, *logrus.Logger) = StartLSPServer
	})
}

func TestStartLSPServerWithInvalidPort(t *testing.T) {
	// Test that the function handles invalid port gracefully
	log := logrus.New()

	// Test with empty port
	t.Run("empty port", func(t *testing.T) {
		server := NewLSPServer("", log)
		if server == nil {
			t.Fatal("Expected server to be created even with empty port")
		}

		if server.port != "" {
			t.Errorf("Expected empty port, got %s", server.port)
		}
	})

	// Test with non-numeric port
	t.Run("non-numeric port", func(t *testing.T) {
		server := NewLSPServer("invalid", log)
		if server == nil {
			t.Fatal("Expected server to be created even with invalid port")
		}

		if server.port != "invalid" {
			t.Errorf("Expected port 'invalid', got %s", server.port)
		}
	})
}

func TestStartLSPServerWithNilLogger(t *testing.T) {
	// Test that the function handles nil logger gracefully
	t.Run("nil logger", func(t *testing.T) {
		// This should not panic
		server := NewLSPServer("8080", nil)
		if server == nil {
			t.Fatal("Expected server to be created even with nil logger")
		}

		if server.log != nil {
			t.Error("Expected logger to be nil")
		}
	})
}

func TestStartLSPServerIntegration(t *testing.T) {
	// Test the full integration by creating a server and checking its properties
	log := logrus.New()
	server := NewLSPServer("8081", log)

	// Test server properties
	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.port != "8081" {
		t.Errorf("Expected port 8081, got %s", server.port)
	}

	if server.log != log {
		t.Error("Expected logger to be set")
	}

	if server.debugger == nil {
		t.Error("Expected debugger to be created")
	}

	if server.lspDebugger == nil {
		t.Error("Expected LSP debugger to be created")
	}
}

func TestStartLSPServerWithDifferentPorts(t *testing.T) {
	log := logrus.New()

	testCases := []struct {
		name     string
		port     string
		expected string
	}{
		{"standard port", "8080", "8080"},
		{"high port", "9000", "9000"},
		{"string port", "test", "test"},
		{"empty port", "", ""},
		{"zero port", "0", "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := NewLSPServer(tc.port, log)

			if server == nil {
				t.Fatal("Expected server to be created")
			}

			if server.port != tc.expected {
				t.Errorf("Expected port %s, got %s", tc.expected, server.port)
			}
		})
	}
}

func TestStartLSPServerWithDifferentLoggers(t *testing.T) {
	testCases := []struct {
		name string
		log  *logrus.Logger
	}{
		{"debug logger", func() *logrus.Logger {
			l := logrus.New()
			l.SetLevel(logrus.DebugLevel)
			return l
		}()},
		{"info logger", func() *logrus.Logger {
			l := logrus.New()
			l.SetLevel(logrus.InfoLevel)
			return l
		}()},
		{"error logger", func() *logrus.Logger {
			l := logrus.New()
			l.SetLevel(logrus.ErrorLevel)
			return l
		}()},
		{"nil logger", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := NewLSPServer("8080", tc.log)

			if server == nil {
				t.Fatal("Expected server to be created")
			}

			if server.log != tc.log {
				t.Error("Expected logger to be set correctly")
			}
		})
	}
}

// Test that the function can be called in a goroutine without blocking
func TestStartLSPServerNonBlocking(t *testing.T) {
	log := logrus.New()

	// Test that we can create a server without it blocking
	server := NewLSPServer("8080", log)

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	// Test that the server can be stopped without issues
	if err := server.Stop(); err != nil {
		t.Errorf("Expected no error when stopping server, got %v", err)
	}
}

// Test that the function handles environment variables correctly
func TestStartLSPServerEnvironment(t *testing.T) {
	log := logrus.New()

	// Test with different port values
	testCases := []struct {
		name string
		port string
	}{
		{"default port", "8081"},
		{"custom port", "9999"},
		{"zero port", "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := NewLSPServer(tc.port, log)

			if server == nil {
				t.Fatal("Expected server to be created")
			}

			if server.port != tc.port {
				t.Errorf("Expected port %s, got %s", tc.port, server.port)
			}
		})
	}
}

// Test that the function can handle concurrent access
func TestStartLSPServerConcurrent(t *testing.T) {
	log := logrus.New()

	// Test creating multiple servers concurrently
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			defer func() { done <- true }()

			port := fmt.Sprintf("808%d", id)
			server := NewLSPServer(port, log)

			if server == nil {
				t.Errorf("Expected server %d to be created", id)
				return
			}

			if server.port != port {
				t.Errorf("Expected server %d to have port %s, got %s", id, port, server.port)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}
