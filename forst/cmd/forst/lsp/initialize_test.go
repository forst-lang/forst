package lsp

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleInitialize_ReturnsCapabilitiesAndServerInfo(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleInitialize(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	res, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	caps, ok := res["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatal("missing capabilities")
	}
	if caps["hoverProvider"] != true {
		t.Fatalf("hoverProvider = %v", caps["hoverProvider"])
	}
	dp, ok := caps["diagnosticProvider"].(map[string]interface{})
	if !ok || dp["identifier"] != "forst" {
		t.Fatalf("diagnosticProvider = %#v", caps["diagnosticProvider"])
	}
	if caps["definitionProvider"] != true {
		t.Fatal("expected definitionProvider")
	}
	if caps["documentFormattingProvider"] != true {
		t.Fatalf("documentFormattingProvider = %v", caps["documentFormattingProvider"])
	}
	if _, ok := caps["codeActionProvider"]; ok {
		t.Fatal("codeActionProvider should be omitted until code actions exist")
	}
	if _, ok := caps["codeLensProvider"]; ok {
		t.Fatal("codeLensProvider should be omitted until code lenses exist")
	}
	if caps["foldingRangeProvider"] != true {
		t.Fatalf("foldingRangeProvider = %v", caps["foldingRangeProvider"])
	}
	info, ok := res["serverInfo"].(map[string]interface{})
	if !ok || info["name"] != "forst-lsp" {
		t.Fatalf("serverInfo = %#v", res["serverInfo"])
	}
}
