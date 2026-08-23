package importlocal

import "fmt"

// FormatImportFix returns an import fix line for the given kind.
func FormatImportFix(importPath, alias string, kind Kind) string {
	switch kind {
	case KindGo:
		return FormatGoImportFix(importPath, alias)
	default:
		return formatBridgeImportFix(importPath, alias)
	}
}

func formatBridgeImportFix(importPath, alias string) string {
	if alias != "" {
		return fmt.Sprintf("import %s %q js", alias, importPath)
	}
	return fmt.Sprintf("import %q js", importPath)
}

// FormatGoImportFix returns a Go import line using alias and path.
func FormatGoImportFix(importPath, alias string) string {
	if alias != "" {
		return fmt.Sprintf("import %s %q", alias, importPath)
	}
	return fmt.Sprintf("import %q", importPath)
}

// ReservedLocalDiagnostic builds a user-facing error with a suggested fix line.
func ReservedLocalDiagnostic(local, importPath, moduleID string, taken TakenSet, kind Kind, err error) string {
	label := kind.diagnosticLabel()
	ve, ok := err.(*ValidationError)
	if !ok || ve == nil {
		return fmt.Sprintf("%s import local name %q cannot be used without an alias", label, local)
	}
	alias := SuggestAliasForKind(moduleID, importPath, taken, kind)
	var reason string
	switch ve.Reason {
	case ReasonForstKeyword:
		reason = "is a Forst keyword"
	case ReasonGoKeyword:
		reason = "is a Go keyword"
	case ReasonReservedImport:
		reason = "is reserved for imports"
	default:
		reason = "is not a valid identifier"
	}
	return fmt.Sprintf("%s import local name %q %s and cannot be used without an alias\n  %s",
		label, local, reason, FormatImportFix(importPath, alias, kind))
}

// FormatGoReservedLocalDiagnostic builds a user-facing Go import error with a suggested fix line.
func FormatGoReservedLocalDiagnostic(local, importPath, moduleID string, taken map[string]struct{}, err error) string {
	return ReservedLocalDiagnostic(local, importPath, moduleID, TakenSet(taken), KindGo, err)
}

// FormatBridgeReservedLocalDiagnostic builds a user-facing JS bridge import error with a suggested fix line.
func FormatBridgeReservedLocalDiagnostic(local, importPath, moduleID string, taken map[string]struct{}, err error) string {
	return ReservedLocalDiagnostic(local, importPath, moduleID, TakenSet(taken), KindBridge, err)
}
