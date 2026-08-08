package transformerts

import "fmt"

// DirectInvokeClientImportLine returns the ESM import for getDefaultInvokeClient
// from the inlined transport module (not @forst/client).
func DirectInvokeClientImportLine() string {
	return fmt.Sprintf("import { getDefaultInvokeClient } from '%s';", TransportModuleSpecifier)
}

// DirectInvokeExportLines emits SSR-safe named invoke helpers with lazy client init.
// Call sites must also emit DirectInvokeClientImportLine() (or an equivalent import
// from TransportModuleSpecifier).
func DirectInvokeExportLines(packageName string, functions []FunctionSignature) []string {
	lines := make([]string, 0, len(functions))
	for _, function := range functions {
		paramsSig := make([]string, len(function.Parameters))
		paramNames := make([]string, len(function.Parameters))
		for i, param := range function.Parameters {
			paramsSig[i] = fmt.Sprintf("%s: %s", param.Name, param.Type)
			paramNames[i] = param.Name
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

		lines = append(lines, fmt.Sprintf(`export async function %s(%s): Promise<%s> {
  return (await getDefaultInvokeClient().invokeFunction<%s>('%s', '%s', %s)).result;
}`, function.Name, paramsSigStr, function.ReturnType, function.ReturnType, packageName, function.Name, argsList))
	}
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
