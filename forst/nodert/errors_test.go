package nodert

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRPCErrorFromWire_buildsNodeCallError(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(map[string]string{
		"name":       "TypeError",
		"stack":      "at create (payment.ts:1:1)",
		"moduleId":   "legacy/payment.ts",
		"exportName": "create",
	})
	if err != nil {
		t.Fatal(err)
	}

	wireErr := rpcErrorFromWire(ErrCodeServerError, "boom", data)
	var nodeErr *NodeCallError
	if !AsNodeCallError(wireErr, &nodeErr) {
		t.Fatalf("expected NodeCallError, got %T: %v", wireErr, wireErr)
	}
	if nodeErr.Message != "boom" {
		t.Fatalf("message = %q", nodeErr.Message)
	}
	if nodeErr.Name != "TypeError" {
		t.Fatalf("name = %q", nodeErr.Name)
	}
	if nodeErr.Stack == "" || nodeErr.ModuleID != "legacy/payment.ts" || nodeErr.ExportName != "create" {
		t.Fatalf("nodeErr = %+v", nodeErr)
	}
}

func TestIsForbidden_matchesBareRPCError(t *testing.T) {
	t.Parallel()
	err := &RPCError{Code: ErrCodeForbidden, Message: "forbidden"}
	if !IsForbidden(err) {
		t.Fatal("expected IsForbidden for bare RPCError")
	}
	if IsForbidden(errors.New("other")) {
		t.Fatal("unexpected IsForbidden match")
	}
}

func TestIsMethodNotFound_matchesBareRPCError(t *testing.T) {
	t.Parallel()
	err := &RPCError{Code: ErrCodeMethodNotFound, Message: "method not found"}
	if !IsMethodNotFound(err) {
		t.Fatal("expected IsMethodNotFound for bare RPCError")
	}
}
