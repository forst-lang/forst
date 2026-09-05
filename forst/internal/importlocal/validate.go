package importlocal

import (
	"fmt"
	"go/token"

	"forst/internal/lexer"
)

// Reason classifies why an import local name is rejected.
type Reason int

const (
	ReasonInvalidSyntax Reason = iota
	ReasonForstKeyword
	ReasonGoKeyword
	ReasonReservedImport
)

// ValidationError reports a rejected import local name.
type ValidationError struct {
	Name   string
	Reason Reason
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	switch e.Reason {
	case ReasonForstKeyword:
		return fmt.Sprintf("%q is a Forst keyword", e.Name)
	case ReasonGoKeyword:
		return fmt.Sprintf("%q is a Go keyword", e.Name)
	case ReasonReservedImport:
		return fmt.Sprintf("%q is reserved for imports", e.Name)
	default:
		return fmt.Sprintf("%q is not a valid identifier", e.Name)
	}
}

// IsForstKeyword reports whether name is a Forst keyword (case-sensitive).
func IsForstKeyword(name string) bool {
	_, ok := lexer.Keywords[name]
	return ok
}

// IsGoKeyword reports whether name is a Go keyword.
func IsGoKeyword(name string) bool {
	return token.IsKeyword(name)
}

// IsReservedGoImportLocal reports import locals reserved for Go imports (_ and dot).
func IsReservedGoImportLocal(name string) bool {
	return name == "_" || name == "."
}

// IsReservedBridgeImportLocal reports import locals reserved for JS bridge imports.
func IsReservedBridgeImportLocal(name string) bool {
	return name == "_" || name == "." || name == "js"
}

// IsReservedImportLocal is an alias for IsReservedBridgeImportLocal.
func IsReservedImportLocal(name string) bool {
	return IsReservedBridgeImportLocal(name)
}

// Validate rejects invalid import locals for the given kind.
func Validate(name string, kind Kind) error {
	return validateImportLocal(name, kind.isReserved)
}

// ValidateForstImportLocal rejects invalid Go import locals (js is allowed as alias).
func ValidateForstImportLocal(name string) error {
	return Validate(name, KindGo)
}

// ValidateBridgeImportLocal rejects invalid JS bridge import locals (js is reserved).
func ValidateBridgeImportLocal(name string) error {
	return Validate(name, KindBridge)
}

// ValidateLocalName is an alias for ValidateBridgeImportLocal.
func ValidateLocalName(name string) error {
	return ValidateBridgeImportLocal(name)
}

func validateImportLocal(name string, reserved func(string) bool) error {
	if name == "" {
		return &ValidationError{Name: name, Reason: ReasonInvalidSyntax}
	}
	if reserved(name) {
		return &ValidationError{Name: name, Reason: ReasonReservedImport}
	}
	if !IsValidIdentifierSyntax(name) {
		return &ValidationError{Name: name, Reason: ReasonInvalidSyntax}
	}
	if IsForstKeyword(name) {
		return &ValidationError{Name: name, Reason: ReasonForstKeyword}
	}
	if IsGoKeyword(name) {
		return &ValidationError{Name: name, Reason: ReasonGoKeyword}
	}
	return nil
}
