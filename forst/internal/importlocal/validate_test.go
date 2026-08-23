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
		{"go js alias ok", "js", KindGo, -1},
		{"bridge js reserved", "js", KindBridge, ReasonReservedImport},
		{"go type keyword", "type", KindGo, ReasonForstKeyword},
		{"bridge type keyword", "type", KindBridge, ReasonForstKeyword},
		{"go var keyword", "var", KindGo, ReasonForstKeyword},
		{"blank reserved go", "_", KindGo, ReasonReservedImport},
		{"blank reserved bridge", "_", KindBridge, ReasonReservedImport},
		{"hyphen invalid", "my-package", KindGo, ReasonInvalidSyntax},
		{"payment ok", "payment", KindBridge, -1},
		{"node ok on bridge import", "node", KindBridge, -1},
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
	if err := ValidateForstImportLocal("js"); err != nil {
		t.Fatalf("js should be valid Go import alias: %v", err)
	}
	if err := ValidateBridgeImportLocal("js"); err == nil {
		t.Fatal("js should be reserved for bridge imports")
	}
}
