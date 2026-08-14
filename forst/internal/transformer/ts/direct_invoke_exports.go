package transformerts

import "fmt"

// DirectInvokeClientImportLine returns the ESM import for getDefaultInvokeClient
// from the inlined transport module (not @forst/client).
func DirectInvokeClientImportLine() string {
	return fmt.Sprintf("import { getDefaultInvokeClient } from '%s';", TransportModuleSpecifier)
}

// DirectInvokeExportLines emits SSR-safe bound namespace invoke helpers with lazy client init.
// Call sites must also emit DirectInvokeClientImportLine() (or an equivalent import
// from TransportModuleSpecifier).
func DirectInvokeExportLines(packageName string, functions []FunctionSignature) []string {
	ns := PackageNamespaceExport(packageName)
	lines := []string{fmt.Sprintf("export const %s = {", ns)}
	for i, function := range functions {
		paramsSig := make([]string, len(function.Parameters))
		paramNames := make([]string, len(function.Parameters))
		for j, param := range function.Parameters {
			paramsSig[j] = fmt.Sprintf("%s: %s", param.Name, param.Type)
			paramNames[j] = param.Name
		}
		paramsSigStr := joinComma(paramsSig)
		paramNamesStr := joinComma(paramNames)

		var argsList string
		switch len(paramNames) {
		case 0:
			argsList = "[]"
		case 1:
			argsList = "[" + paramNames[0] + "]"
		default:
			argsList = "[" + paramNamesStr + "]"
		}

		entry := fmt.Sprintf(`  %s: async (%s): Promise<%s> => {
    return (await getDefaultInvokeClient().invokeFunction<%s>('%s', '%s', %s)).result;
  }`, function.Name, paramsSigStr, function.ReturnType, function.ReturnType, packageName, function.Name, argsList)
		if i < len(functions)-1 {
			entry += ","
		}
		lines = append(lines, entry)
	}
	lines = append(lines, "};")
	return lines
}

func joinComma(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += ", " + items[i]
	}
	return out
}
