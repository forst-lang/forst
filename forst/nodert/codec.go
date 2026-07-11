package nodert

import (
	"encoding/json"
	"fmt"
)

// CallSyncArgs invokes CallSync with pre-marshaled JSON args (codegen fast path).
func CallSyncArgs[T any](moduleID, exportName string, argsJSON json.RawMessage) (T, error) {
	var zero T
	client, err := GetClient()
	if err != nil {
		return zero, err
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

// CallAsyncArgs invokes CallAsync with pre-marshaled JSON args (codegen fast path).
func CallAsyncArgs[T any](moduleID, exportName string, argsJSON json.RawMessage) (T, error) {
	var zero T
	client, err := GetClient()
	if err != nil {
		return zero, err
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

// MarshalNodeCallArgsJSON marshals positional call args to a JSON array.
func MarshalNodeCallArgsJSON(args ...any) (json.RawMessage, error) {
	return json.Marshal(args)
}
