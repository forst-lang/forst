package genplugin

import (
	"fmt"
	"strings"

	"forst/internal/semantic"
)

// HTTPVerb reports whether name is a standard HTTP method.
func HTTPVerb(name string) (string, bool) {
	v := strings.ToUpper(strings.TrimSpace(name))
	switch v {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return v, true
	default:
		return "", false
	}
}

// RemixExport maps a contract member name to loader / action.
// GET/HEAD → loader; mutating HTTP verbs → action; explicit loader/action pass through.
func RemixExport(fieldName string) string {
	switch strings.ToLower(strings.TrimSpace(fieldName)) {
	case "loader", "get", "head":
		return "loader"
	case "action", "post", "put", "patch", "delete":
		return "action"
	default:
		return ""
	}
}

// PathParamNames extracts :name params from a module-relative .ft path.
func PathParamNames(spanFile string) []string {
	spanFile = strings.TrimSuffix(toSlash(spanFile), ".ft")
	var names []string
	for _, seg := range strings.Split(spanFile, "/") {
		if n, ok := paramName(seg); ok {
			names = append(names, n)
		}
	}
	return names
}

func paramName(seg string) (string, bool) {
	if strings.HasPrefix(seg, "$") && len(seg) > 1 {
		return seg[1:], true
	}
	if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") && len(seg) > 2 {
		return seg[1 : len(seg)-1], true
	}
	return "", false
}

// HandlerStem is the sealed handler filename stem relative to routesRoot (slashes → dots).
func HandlerStem(spanFile, routesRoot string) string {
	spanFile = toSlash(strings.TrimSpace(spanFile))
	routesRoot = strings.Trim(toSlash(routesRoot), "/")
	rel := spanFile
	prefix := routesRoot + "/"
	if strings.HasPrefix(rel, prefix) {
		rel = strings.TrimPrefix(rel, prefix)
	}
	rel = strings.TrimSuffix(rel, ".ft")
	rel = strings.ReplaceAll(rel, "/", ".")
	if rel == "" {
		return "index"
	}
	return rel
}

// ParamBindingDiags reports path params that no bound function declares.
func ParamBindingDiags(s FileRouterSurface) []semantic.Diagnostic {
	if len(s.PathParams) == 0 {
		return nil
	}
	var diags []semantic.Diagnostic
	for _, pp := range s.PathParams {
		found := false
		for _, m := range s.Methods {
			for _, p := range m.Function.Params {
				if p.Name == pp {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			diags = append(diags, semantic.Diagnostic{
				Severity: "error",
				Message:  fmt.Sprintf("path param %q is not declared on any bound function", pp),
				TypeID:   s.TypeID,
				Span:     &semantic.SourceSpan{File: s.File},
			})
		}
	}
	return diags
}

// ArgAccessExpr maps function params to path params / body fields for generated TS.
func ArgAccessExpr(params []semantic.FuncParam, pathParams []string, paramsVar, bodyVar string) string {
	pathSet := map[string]struct{}{}
	for _, p := range pathParams {
		pathSet[p] = struct{}{}
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		name := TSIdentifier(p.Name)
		if _, ok := pathSet[p.Name]; ok {
			parts = append(parts, fmt.Sprintf("%s[%q]", paramsVar, p.Name))
			continue
		}
		if bodyVar != "" {
			parts = append(parts, fmt.Sprintf("%s[%q]", bodyVar, name))
			continue
		}
		parts = append(parts, "undefined")
	}
	return strings.Join(parts, ", ")
}

// InputFieldAccess maps synthesized input object fields to positional args.
func InputFieldAccess(inputVar string, params []semantic.FuncParam) string {
	parts := make([]string, 0, len(params))
	for _, p := range params {
		parts = append(parts, fmt.Sprintf("%s.%s", inputVar, TSIdentifier(p.Name)))
	}
	return strings.Join(parts, ", ")
}
