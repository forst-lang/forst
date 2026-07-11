package nodert

import (
	"encoding/json"
	"fmt"
)

// CallSync invokes forst.node/call and unmarshals the JSON result into T.
func CallSync[T any](moduleID, exportName string, args ...any) (T, error) {
	var zero T
	client, err := GetClient()
	if err != nil {
		return zero, err
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return zero, fmt.Errorf("marshal node call args: %w", err)
	}
	raw, err := client.CallSync(moduleID, exportName, argsJSON)
	if err != nil {
		return zero, err
	}
	var out T
	if len(raw) == 0 || string(raw) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return zero, fmt.Errorf("unmarshal node call result: %w", err)
	}
	return out, nil
}

// MustCallSync is like CallSync but panics on error (for hand-written Go callers).
func MustCallSync[T any](moduleID, exportName string, args ...any) T {
	out, err := CallSync[T](moduleID, exportName, args...)
	if err != nil {
		panic(err)
	}
	return out
}
