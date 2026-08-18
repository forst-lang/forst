package genplugin

import (
	"encoding/json"
	"testing"

	"forst/internal/semantic"
)

func TestUnmarshalPluginOpt_empty(t *testing.T) {
	var opt struct {
		Draft string `json:"draft"`
	}
	if err := UnmarshalPluginOpt(&semantic.GenerateRequest{}, &opt); err != nil {
		t.Fatal(err)
	}
}

func TestUnmarshalPluginOpt_invalidJSON(t *testing.T) {
	req := &semantic.GenerateRequest{
		Plugin: &semantic.PluginRef{
			Name: "test",
			Opt:  json.RawMessage(`{`),
		},
	}
	var opt struct{}
	if err := UnmarshalPluginOpt(req, &opt); err == nil {
		t.Fatal("expected error")
	}
}
