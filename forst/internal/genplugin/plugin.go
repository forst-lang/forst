package genplugin

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"forst/internal/semantic"
)

// Version is the genplugin library version (informational for plugin authors).
const Version = "0.1.0"

// Run reads a GenerateRequest from stdin, calls emit, and writes GenerateResponse to stdout.
func Run(emit func(*semantic.GenerateRequest) (semantic.GenerateResponse, error)) {
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail("read stdin", err)
	}
	var req semantic.GenerateRequest
	if err := json.Unmarshal(in, &req); err != nil {
		fail("parse request", err)
	}
	resp, err := emit(&req)
	if err != nil {
		fail("emit", err)
	}
	if resp.ProtocolVersion == 0 {
		resp.ProtocolVersion = semantic.ProtocolVersion
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		fail("encode response", err)
	}
}

func fail(phase string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", phase, err)
	os.Exit(1)
}
