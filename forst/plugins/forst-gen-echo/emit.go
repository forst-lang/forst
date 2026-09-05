package main

import (
	"fmt"
	"sort"
	"strings"

	"forst/internal/semantic"
)

const echoVersion = "0.1.0"

func emitEcho(req *semantic.GenerateRequest) (semantic.GenerateResponse, error) {
	typeIDs := make([]string, 0, len(req.Types))
	for id := range req.Types {
		typeIDs = append(typeIDs, id)
	}
	sort.Strings(typeIDs)

	funcIDs := make([]string, 0, len(req.Functions))
	for id := range req.Functions {
		funcIDs = append(funcIDs, id)
	}
	sort.Strings(funcIDs)

	name := "echo"
	if req.Plugin != nil && req.Plugin.Name != "" {
		name = req.Plugin.Name
	}

	var b strings.Builder
	fmt.Fprintf(&b, "plugin=%s\n", name)
	fmt.Fprintf(&b, "version=%s\n", echoVersion)
	fmt.Fprintf(&b, "protocol=%d\n", req.ProtocolVersion)
	fmt.Fprintf(&b, "types=%d\n", len(typeIDs))
	for _, id := range typeIDs {
		fmt.Fprintf(&b, "  %s\n", id)
	}
	fmt.Fprintf(&b, "functions=%d\n", len(funcIDs))
	for _, id := range funcIDs {
		fmt.Fprintf(&b, "  %s\n", id)
	}

	return semantic.GenerateResponse{
		ProtocolVersion: semantic.ProtocolVersion,
		Files: []semantic.OutputFile{{
			Path:    "manifest.txt",
			Content: b.String(),
		}},
	}, nil
}
