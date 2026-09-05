package genplugin

import (
	"encoding/json"
	"fmt"

	"forst/internal/semantic"
)

// UnmarshalPluginOpt decodes req.Plugin.Opt into dst when present.
func UnmarshalPluginOpt(req *semantic.GenerateRequest, dst any) error {
	if req == nil || req.Plugin == nil || len(req.Plugin.Opt) == 0 {
		return nil
	}
	if err := json.Unmarshal(req.Plugin.Opt, dst); err != nil {
		return fmt.Errorf("invalid plugin opt: %w", err)
	}
	return nil
}

// MustJSON pretty-prints v as JSON plus a trailing newline.
func MustJSON(v any) string {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw) + "\n"
}

// JSONArrayExpr is a JSON array literal suitable as a TypeScript expression.
func JSONArrayExpr(items []string) string {
	raw, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(raw)
}
