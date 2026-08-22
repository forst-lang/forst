package importlocal

import (
	"strings"
	"testing"
)

func TestSuggestAliasForKind_go(t *testing.T) {
	tests := []struct {
		moduleID string
		want     string
	}{
		{"legacy/type.ts", "typePkg"},
		{"map", "mapPkg"},
		{"@types/node", "node"},
	}
	for _, tt := range tests {
		t.Run(tt.moduleID, func(t *testing.T) {
			got := SuggestAliasForKind(tt.moduleID, tt.moduleID, nil, KindGo)
			if got != tt.want {
				t.Fatalf("SuggestAliasForKind(Go, %q) = %q, want %q", tt.moduleID, got, tt.want)
			}
			if err := Validate(got, KindGo); err != nil {
				t.Fatalf("suggested alias invalid: %v", err)
			}
		})
	}
}

func TestSuggestAliasForKind_node(t *testing.T) {
	tests := []struct {
		moduleID string
		want     string
	}{
		{"legacy/type.ts", "typePkg"},
		{"map", "mapPkg"},
		{"@types/node", "nodePkg"},
	}
	for _, tt := range tests {
		t.Run(tt.moduleID, func(t *testing.T) {
			got := SuggestAliasForKind(tt.moduleID, tt.moduleID, nil, KindNode)
			if got != tt.want {
				t.Fatalf("SuggestAliasForKind(Node, %q) = %q, want %q", tt.moduleID, got, tt.want)
			}
			if err := Validate(got, KindNode); err != nil {
				t.Fatalf("suggested alias invalid: %v", err)
			}
		})
	}
}

func TestSuggestAlias_respectsTaken(t *testing.T) {
	taken := TakenSet{"payment": {}}
	got := SuggestAliasForKind("legacy/payment.ts", "./legacy/payment.ts", taken, KindNode)
	if got != "paymentPkg" {
		t.Fatalf("got %q, want paymentPkg", got)
	}
}

func TestFormatImportFix(t *testing.T) {
	if got := FormatImportFix("fmt", "f", KindGo); got != `import f "fmt"` {
		t.Fatalf("Go fix = %q", got)
	}
	if got := FormatImportFix("./a.ts", "aPkg", KindNode); got != `import aPkg "./a.ts" node` {
		t.Fatalf("Node fix = %q", got)
	}
}

func TestReservedLocalDiagnostic(t *testing.T) {
	err := &ValidationError{Name: "type", Reason: ReasonForstKeyword}
	msg := ReservedLocalDiagnostic("type", "example.com/foo/type", "example.com/foo/type", nil, KindGo, err)
	if !strings.Contains(msg, "Go import local name") {
		t.Fatalf("missing Go label: %q", msg)
	}
	if !strings.Contains(msg, `import typePkg "example.com/foo/type"`) {
		t.Fatalf("missing Go fix: %q", msg)
	}
	msg = ReservedLocalDiagnostic("type", "./legacy/type.ts", "legacy/type.ts", nil, KindNode, err)
	if !strings.Contains(msg, "node import local name") {
		t.Fatalf("missing node label: %q", msg)
	}
	if !strings.Contains(msg, `import typePkg "./legacy/type.ts" node`) {
		t.Fatalf("missing node fix: %q", msg)
	}
}
