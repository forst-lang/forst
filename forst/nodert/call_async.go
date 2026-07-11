package nodert

import (
	"encoding/json"
	"fmt"
)

// CallAsync invokes forst.node/callAsync and unmarshals the settled JSON result into T.
func CallAsync[T any](moduleID, exportName string, args ...any) (T, error) {
	var zero T
	client, err := GetClient()
	if err != nil {
		return zero, err
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return zero, fmt.Errorf("marshal node call args: %w", err)
	}
	raw, err := client.CallAsync(moduleID, exportName, argsJSON)
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

// MustCallAsync is like CallAsync but panics on error (for hand-written Go callers).
func MustCallAsync[T any](moduleID, exportName string, args ...any) T {
	out, err := CallAsync[T](moduleID, exportName, args...)
	if err != nil {
		panic(err)
	}
	return out
}
