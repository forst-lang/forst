package invokecodec

import (
	"fmt"
	"strings"

	"forst/internal/discovery"
	"forst/internal/typechecker"
)

// DecodeArgsFromJSON generates Go statements that decode a JSON positional arg array into parameters.
// containerName is the slice variable (e.g. "forstInvokeArgs").
func DecodeArgsFromJSON(containerName string, params []discovery.ParameterInfo) string {
	return DecodeArgsFromJSONQualified(containerName, params, discovery.FunctionInfo{}, "")
}

// DecodeArgsFromJSONQualified decodes args, qualifying custom parameter types when fn is in another package.
func DecodeArgsFromJSONQualified(containerName string, params []discovery.ParameterInfo, fn discovery.FunctionInfo, localPkg string) string {
	if len(params) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "var %s []interface{}\n", containerName)
	fmt.Fprintf(&b, "\tif err := json.Unmarshal(args, &%s); err != nil {\n", containerName)
	fmt.Fprintf(&b, "\t\treturn nil, fmt.Errorf(\"decode invoke args: %%w\", err)\n")
	fmt.Fprintf(&b, "\t}\n")
	if len(params) == 1 {
		b.WriteString(decodeOneParamQualified(containerName, 0, params[0], fn.Package, localPkg))
		return b.String()
	}
	for i, param := range params {
		b.WriteString(decodeOneParamQualified(containerName, i, param, fn.Package, localPkg))
	}
	return b.String()
}

func decodeOneParamQualified(containerName string, index int, param discovery.ParameterInfo, fnPkg, localPkg string) string {
	paramType := qualifyInvokeParamType(param.Type, fnPkg, localPkg)
	return decodeOneParam(containerName, index, discovery.ParameterInfo{Name: param.Name, Type: paramType})
}

func qualifyInvokeParamType(typeName, fnPkg, localPkg string) string {
	if typeName == "" || fnPkg == "" || localPkg == "" || fnPkg == localPkg {
		return typeName
	}
	if typechecker.IsGoBuiltinType(typeName) {
		return typeName
	}
	return fnPkg + "." + typeName
}

func decodeOneParam(containerName string, index int, param discovery.ParameterInfo) string {
	paramType := param.Type
	name := param.Name
	if typechecker.IsGoBuiltinType(paramType) {
		if paramType == "int" {
			return fmt.Sprintf(`
	var %s int
	if v, ok := %s[%d].(float64); ok {
		%s = int(v)
	} else if v, ok := %s[%d].(int); ok {
		%s = v
	} else {
		return nil, fmt.Errorf("parameter %q: expected %s, got %%T", %s[%d])
	}
`, name, containerName, index, name, containerName, index, name, name, paramType, containerName, index)
		}
		return fmt.Sprintf(`
	var %s %s
	if v, ok := %s[%d].(%s); ok {
		%s = v
	} else {
		return nil, fmt.Errorf("parameter %q: expected %s, got %%T", %s[%d])
	}
`, name, paramType, containerName, index, paramType, name, name, paramType, containerName, index)
	}
	return fmt.Sprintf(`
	var %s %s
	paramBytes, err := json.Marshal(%s[%d])
	if err != nil {
		return nil, fmt.Errorf("marshal parameter %%q: %%w", %q, err)
	}
	if err := json.Unmarshal(paramBytes, &%s); err != nil {
		return nil, fmt.Errorf("unmarshal parameter %%q: %%w", %q, err)
	}
`, name, paramType, containerName, index, name, name, name)
}

// CallExpression builds the Go call expression for invoking a Forst function in the local package.
func CallExpression(fn discovery.FunctionInfo, paramNames []string) string {
	return callExpressionImpl(fn, paramNames, "")
}

// CallExpressionQualified builds a call expression, qualifying with pkg.Name when fn.Package != localPkg.
func CallExpressionQualified(fn discovery.FunctionInfo, paramNames []string, localPkg string) string {
	qual := ""
	if localPkg != "" && fn.Package != localPkg && fn.Package != "" {
		qual = fn.Package + "."
	}
	return callExpressionImpl(fn, paramNames, qual)
}

func callExpressionImpl(fn discovery.FunctionInfo, paramNames []string, qual string) string {
	callee := qual + fn.Name
	args := strings.Join(paramNames, ", ")
	if fn.HasMultipleReturns {
		return fmt.Sprintf("result, err := %s(%s)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn result, nil", callee, args)
	}
	if len(paramNames) == 0 {
		return fmt.Sprintf("return %s(), nil", callee)
	}
	return fmt.Sprintf("return %s(%s), nil", callee, args)
}
