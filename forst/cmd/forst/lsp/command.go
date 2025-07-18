package lsp

import (
	"os"

	"github.com/sirupsen/logrus"
)

// StartLSPServer is the entry point for the LSP server command
func StartLSPServer(port string, log *logrus.Logger) {
	server := NewLSPServer(port, log)

	log.Infof("Starting Forst LSP server on port %s", port)
	log.Debug("This server provides LSP-compatible diagnostics and features")
	log.Debug("Connect your LSP client (VS Code, Vim, etc.) to this server")

	if err := server.Start(); err != nil {
		log.Errorf("LSP server error: %v", err)
		os.Exit(1)
	}
}
