package importlocal

import (
	"errors"
	"testing"
)

func TestValidate_matrix(t *testing.T) {
	tests := []struct {
		name    string
		local   string
		kind    Kind
		wantErr Reason
	}{
		{"go node alias ok", "node", KindGo, -1},
		{"node node reserved", "node", KindNode, ReasonReservedImport},
		{"go type keyword", "type", KindGo, ReasonForstKeyword},
		{"node type keyword", "type", KindNode, ReasonForstKeyword},
		{"go var keyword", "var", KindGo, ReasonForstKeyword},
		{"blank reserved go", "_", KindGo, ReasonReservedImport},
		{"blank reserved node", "_", KindNode, ReasonReservedImport},
		{"hyphen invalid", "my-package", KindGo, ReasonInvalidSyntax},
		{"payment ok", "payment", KindNode, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.local, tt.kind)
			if tt.wantErr < 0 {
				if err != nil {
					t.Fatalf("expected ok, got %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Reason != tt.wantErr {
				t.Fatalf("Validate(%q, %v) = %v, want Reason %v", tt.local, tt.kind, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLegacyWrappers(t *testing.T) {
	if err := ValidateForstImportLocal("node"); err != nil {
		t.Fatalf("node should be valid Go import alias: %v", err)
	}
	if err := ValidateNodeImportLocal("node"); err == nil {
		t.Fatal("node should be reserved for node imports")
	}
}
