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
	if len(params) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "var %s []interface{}\n", containerName)
	fmt.Fprintf(&b, "\tif err := json.Unmarshal(args, &%s); err != nil {\n", containerName)
	fmt.Fprintf(&b, "\t\treturn nil, fmt.Errorf(\"decode invoke args: %%w\", err)\n")
	fmt.Fprintf(&b, "\t}\n")
	if len(params) == 1 {
		b.WriteString(decodeOneParam(containerName, 0, params[0]))
		return b.String()
	}
	for i, param := range params {
		b.WriteString(decodeOneParam(containerName, i, param))
	}
	return b.String()
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

// CallExpression builds the Go call expression for invoking a Forst function.
func CallExpression(fn discovery.FunctionInfo, paramNames []string) string {
	args := strings.Join(paramNames, ", ")
	if fn.HasMultipleReturns {
		return fmt.Sprintf("result, err := %s(%s)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn result, nil", fn.Name, args)
	}
	if len(paramNames) == 0 {
		return fmt.Sprintf("return %s(), nil", fn.Name)
	}
	return fmt.Sprintf("return %s(%s), nil", fn.Name, args)
}
