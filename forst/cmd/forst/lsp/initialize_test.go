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
	actionCap, ok := caps["codeActionProvider"].(map[string]interface{})
	if !ok {
		t.Fatal("expected codeActionProvider map")
	}
	var kindStrs []string
	switch k := actionCap["codeActionKinds"].(type) {
	case []string:
		kindStrs = k
	case []interface{}:
		for _, e := range k {
			s, ok := e.(string)
			if !ok {
				t.Fatalf("codeActionKinds element %T", e)
			}
			kindStrs = append(kindStrs, s)
		}
	default:
		t.Fatalf("codeActionKinds = %#v", actionCap["codeActionKinds"])
	}
	if len(kindStrs) == 0 {
		t.Fatal("empty codeActionKinds")
	}
	rp, ok := caps["renameProvider"].(map[string]interface{})
	if !ok || rp["prepareProvider"] != true {
		t.Fatalf("renameProvider = %#v", caps["renameProvider"])
	}
	clp, ok := caps["codeLensProvider"].(map[string]interface{})
	if !ok {
		t.Fatal("expected codeLensProvider map")
	}
	if clp["resolveProvider"] != false {
		t.Fatalf("codeLens resolveProvider = %v", clp["resolveProvider"])
	}
	if caps["foldingRangeProvider"] != true {
		t.Fatalf("foldingRangeProvider = %v", caps["foldingRangeProvider"])
	}
	info, ok := res["serverInfo"].(map[string]interface{})
	if !ok || info["name"] != "forst-lsp" {
		t.Fatalf("serverInfo = %#v", res["serverInfo"])
	}
}
