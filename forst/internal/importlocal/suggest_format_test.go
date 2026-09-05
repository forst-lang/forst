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

func TestSuggestAliasForKind_bridge(t *testing.T) {
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
			got := SuggestAliasForKind(tt.moduleID, tt.moduleID, nil, KindBridge)
			if got != tt.want {
				t.Fatalf("SuggestAliasForKind(Bridge, %q) = %q, want %q", tt.moduleID, got, tt.want)
			}
			if err := Validate(got, KindBridge); err != nil {
				t.Fatalf("suggested alias invalid: %v", err)
			}
		})
	}
}

func TestSuggestAlias_respectsTaken(t *testing.T) {
	taken := TakenSet{"payment": {}}
	got := SuggestAliasForKind("legacy/payment.ts", "./legacy/payment.ts", taken, KindBridge)
	if got != "paymentPkg" {
		t.Fatalf("got %q, want paymentPkg", got)
	}
}

func TestFormatImportFix(t *testing.T) {
	if got := FormatImportFix("fmt", "f", KindGo); got != `import f "fmt"` {
		t.Fatalf("Go fix = %q", got)
	}
	if got := FormatImportFix("./a.ts", "aPkg", KindBridge); got != `import aPkg "./a.ts" js` {
		t.Fatalf("Bridge fix = %q", got)
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
	msg = ReservedLocalDiagnostic("type", "./legacy/type.ts", "legacy/type.ts", nil, KindBridge, err)
	if !strings.Contains(msg, "JS import local name") {
		t.Fatalf("missing bridge label: %q", msg)
	}
	if !strings.Contains(msg, `import typePkg "./legacy/type.ts" js`) {
		t.Fatalf("missing bridge fix: %q", msg)
	}
}
