package genplugin

import (
	"sort"
	"strings"

	"forst/internal/semantic"
)

// ExportedPackageTypeIDs returns sorted exported user type ids from package manifests.
func ExportedPackageTypeIDs(req *semantic.GenerateRequest) []string {
	if req == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var ids []string
	for _, pkg := range req.Packages {
		for _, id := range pkg.TypeIDs {
			if !IsPublishedTypeID(id) {
				continue
			}
			t, ok := req.Types[id]
			if !ok || t.Visibility == "internal" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// IsPublishedTypeID reports whether id is a user-facing type (not structural/synthesized).
func IsPublishedTypeID(id string) bool {
	if id == "" || strings.HasPrefix(id, "structural:") || strings.HasPrefix(id, "t:") {
		return false
	}
	if strings.Contains(id, ".") {
		name := id[strings.LastIndex(id, ".")+1:]
		switch name {
		case "sig", "input", "payload":
			return false
		}
	}
	return true
}

// IsRouterOnlyShape reports a shape whose fields are all methods (not JSON data).
func IsRouterOnlyShape(t semantic.Type) bool {
	if t.Kind != "shape" || len(t.Fields) == 0 {
		return false
	}
	for _, f := range t.Fields {
		if !f.Method {
			return false
		}
	}
	return true
}

// TypeShortName returns the unqualified name from a snapshot type id.
func TypeShortName(id string) string {
	if i := strings.LastIndex(id, "."); i >= 0 {
		return id[i+1:]
	}
	return id
}

// HasMarker reports whether any constraint on t matches one of markers.
func HasMarker(t semantic.Type, markers []string) bool {
	if len(markers) == 0 {
		markers = []string{"Router"}
	}
	for _, c := range t.Constraints {
		for _, m := range markers {
			if c.Name == m {
				return true
			}
		}
	}
	return false
}

// RouterMethod is one contract member on a marked router type.
type RouterMethod struct {
	FieldName  string
	FunctionID string
	Function   semantic.Function
	Sig        semantic.Type
	FieldType  semantic.Type
}

// RouterSurface is a marked contract type and its members.
type RouterSurface struct {
	TypeID  string
	Type    semantic.Type
	File    string
	Methods []RouterMethod
}

// RouterSurfaces collects marked router types and bound (or bindable) members.
func RouterSurfaces(req *semantic.GenerateRequest, markers []string) []RouterSurface {
	if req == nil {
		return nil
	}
	var out []RouterSurface
	for _, id := range ExportedPackageTypeIDs(req) {
		t, ok := req.Types[id]
		if !ok || t.Kind != "shape" || !HasMarker(t, markers) {
			continue
		}
		surface := RouterSurface{TypeID: id, Type: t}
		if t.Span != nil {
			surface.File = t.Span.File
		}
		pkg := PackageOfTypeID(id)
		for _, field := range t.Fields {
			m, ok := routerMethodFromField(req, pkg, field)
			if !ok {
				continue
			}
			surface.Methods = append(surface.Methods, m)
			if surface.File == "" && m.Function.Span != nil {
				surface.File = m.Function.Span.File
			}
		}
		if len(surface.Methods) > 0 {
			sort.Slice(surface.Methods, func(i, j int) bool {
				return surface.Methods[i].FieldName < surface.Methods[j].FieldName
			})
			out = append(out, surface)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TypeID < out[j].TypeID })
	return out
}

func routerMethodFromField(req *semantic.GenerateRequest, pkg string, field semantic.ShapeField) (RouterMethod, bool) {
	fieldType := Lookup(req.Types, field.Type)
	sig := fieldType
	if fieldType.Kind != "func" {
		if resolved := FollowAlias(req.Types, fieldType); resolved.Kind == "func" {
			sig = resolved
		}
	}
	fnID := field.Function
	if fnID == "" && pkg != "" && field.Name != "" {
		if _, ok := req.Functions[pkg+"."+field.Name]; ok {
			fnID = pkg + "." + field.Name
		}
	}
	fn, hasFn := req.Functions[fnID]
	if field.Method {
		if hasFn {
			return RouterMethod{
				FieldName:  field.Name,
				FunctionID: fn.ID,
				Function:   fn,
				Sig:        sig,
				FieldType:  fieldType,
			}, true
		}
		return RouterMethod{FieldName: field.Name, Sig: sig, FieldType: fieldType}, true
	}
	if hasFn && isQueryOrMutation(req.Types, fieldType) {
		return RouterMethod{
			FieldName:  field.Name,
			FunctionID: fn.ID,
			Function:   fn,
			Sig:        sig,
			FieldType:  fieldType,
		}, true
	}
	return RouterMethod{}, false
}

func isQueryOrMutation(types map[string]semantic.Type, t semantic.Type) bool {
	return TypeOrChainHasConstraint(types, t, "Query") || TypeOrChainHasConstraint(types, t, "Mutation")
}

// FileRouteOptions configures convention routing from span.file.
type FileRouteOptions struct {
	RoutesRoot string
	Markers    []string
	ParamStyle string
}

// FileRouterSurface is a router surface keyed by its source file under routesRoot.
type FileRouterSurface struct {
	RouterSurface
	RoutePath   string
	RRPath      string
	HandlerFile string
	PathParams  []string
	Package     string
}

// FileRouterSurfaces returns router surfaces whose declaration file lies under routesRoot.
func FileRouterSurfaces(req *semantic.GenerateRequest, opts FileRouteOptions) ([]FileRouterSurface, []semantic.Diagnostic) {
	if opts.RoutesRoot == "" {
		opts.RoutesRoot = "app/api"
	}
	if opts.ParamStyle == "" {
		opts.ParamStyle = "$id"
	}
	routesRoot := strings.Trim(strings.TrimSpace(opts.RoutesRoot), "/")
	var out []FileRouterSurface
	var diags []semantic.Diagnostic
	for _, s := range RouterSurfaces(req, opts.Markers) {
		if s.File == "" {
			diags = append(diags, semantic.Diagnostic{
				Severity: "error",
				Message:  "marked router type has no span.file for file-based routing",
				TypeID:   s.TypeID,
			})
			continue
		}
		path, err := RoutePath(s.File, routesRoot, opts.ParamStyle)
		if err != nil {
			diags = append(diags, semantic.Diagnostic{
				Severity: "error",
				Message:  err.Error(),
				TypeID:   s.TypeID,
				Span:     &semantic.SourceSpan{File: s.File},
			})
			continue
		}
		rr, err := RRPath(s.File, routesRoot, opts.ParamStyle)
		if err != nil {
			rr = strings.TrimPrefix(path, "/")
		}
		pkg := PackageOfTypeID(s.TypeID)
		if pkg == "" {
			pkg = s.TypeID
		}
		fr := FileRouterSurface{
			RouterSurface: s,
			RoutePath:     path,
			RRPath:        rr,
			HandlerFile:   "handlers/" + HandlerStem(s.File, routesRoot) + ".ts",
			PathParams:    PathParamNames(s.File),
			Package:       pkg,
		}
		out = append(out, fr)
		diags = append(diags, ParamBindingDiags(fr)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RoutePath != out[j].RoutePath {
			return out[i].RoutePath < out[j].RoutePath
		}
		return out[i].TypeID < out[j].TypeID
	})
	return out, diags
}
