package genplugin

import "encoding/json"

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
