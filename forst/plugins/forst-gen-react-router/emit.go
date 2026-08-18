package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"forst/internal/genplugin"
	"forst/internal/semantic"
)

type rrOpt struct {
	Markers     []string `json:"markers"`
	RoutesRoot  string   `json:"routesRoot"`
	ParamStyle  string   `json:"paramStyle"`
	Client      string   `json:"clientImport"`
	Invoke      string   `json:"invoke"` // "package" | "client"
	RouteImport string   `json:"routeImport"`
}

type routeOut struct {
	surface genplugin.FileRouterSurface
	loader  *genplugin.RouterMethod
	action  *genplugin.RouterMethod
}

func emitReactRouter(req *semantic.GenerateRequest) (semantic.GenerateResponse, error) {
	opt := rrOpt{
		Markers:     []string{"Router"},
		RoutesRoot:  "app/api",
		ParamStyle:  "$id",
		Client:      "@forst/gen",
		Invoke:      genplugin.InvokePackage,
		RouteImport: "@react-router/dev/routes",
	}
	if req.Plugin != nil && len(req.Plugin.Opt) > 0 {
		_ = json.Unmarshal(req.Plugin.Opt, &opt)
	}
	if len(opt.Markers) == 0 {
		opt.Markers = []string{"Router"}
	}
	if opt.RoutesRoot == "" {
		opt.RoutesRoot = "app/api"
	}
	if opt.ParamStyle == "" {
		opt.ParamStyle = "$id"
	}
	if opt.Client == "" {
		opt.Client = "@forst/gen"
	}
	if opt.Invoke == "" {
		opt.Invoke = genplugin.InvokePackage
	}
	if opt.RouteImport == "" {
		opt.RouteImport = "@react-router/dev/routes"
	}

	surfaces, diags := genplugin.FileRouterSurfaces(req, genplugin.FileRouteOptions{
		RoutesRoot: opt.RoutesRoot,
		Markers:    opt.Markers,
		ParamStyle: opt.ParamStyle,
	})

	var routes []routeOut
	for _, s := range surfaces {
		ro := routeOut{surface: s}
		for i := range s.Methods {
			m := s.Methods[i]
			exp := genplugin.RemixExport(m.FieldName)
			if exp == "" {
				diags = append(diags, semantic.Diagnostic{
					Severity: "warning",
					Message:  fmt.Sprintf("method field %q is not GET/POST/loader/action; skipped", m.FieldName),
					TypeID:   s.TypeID,
					Span:     m.Function.Span,
				})
				continue
			}
			if m.Function.ID == "" {
				diags = append(diags, semantic.Diagnostic{
					Severity: "error",
					Message:  fmt.Sprintf("%s has no bound function", m.FieldName),
					TypeID:   s.TypeID,
				})
				continue
			}
			if !m.Function.Runnable {
				diags = append(diags, semantic.Diagnostic{
					Severity: "error",
					Message:  fmt.Sprintf("%s is not runnable (unsatisfied providers); loader/action not emitted", m.FieldName),
					TypeID:   s.TypeID,
					Span:     m.Function.Span,
				})
				continue
			}
			view := genplugin.DerivedCall(req.Types, m.Function)
			if view.Stream {
				diags = append(diags, semantic.Diagnostic{
					Severity: "warning",
					Message:  fmt.Sprintf("%s returns a channel; skipped as a unary resource route", m.FieldName),
					TypeID:   s.TypeID,
					Span:     m.Function.Span,
				})
				continue
			}
			cp := m
			if exp == "loader" {
				ro.loader = &cp
			} else {
				ro.action = &cp
			}
		}
		if ro.loader != nil || ro.action != nil {
			routes = append(routes, ro)
		}
	}

	files := []semantic.OutputFile{
		{Path: "runtime.ts", Content: emitRRRuntime(opt)},
		{Path: "routes.ts", Content: emitRRRoutes(opt.RouteImport, routes)},
		{Path: "registry.ts", Content: emitRRRegistry(routes)},
		{Path: "loaders.ts", Content: emitRRLoaders(opt, routes)},
	}
	for _, r := range routes {
		files = append(files, semantic.OutputFile{
			Path:    r.surface.HandlerFile,
			Content: emitRRHandler(opt, r),
		})
	}
	files = append(files, semantic.OutputFile{
		Path: "meta.json",
		Content: genplugin.MustJSON(map[string]any{
			"generator":  "forst-gen-react-router",
			"version":    version,
			"invoke":     opt.Invoke,
			"routesRoot": opt.RoutesRoot,
			"routeCount": len(routes),
		}),
	})

	return semantic.GenerateResponse{
		ProtocolVersion: semantic.ProtocolVersion,
		Files:           files,
		Diagnostics:     diags,
	}, nil
}

func emitRRRuntime(opt rrOpt) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-react-router %s\n\n", version)
	if opt.Invoke != genplugin.InvokePackage {
		fmt.Fprintf(&b, "import { createInvokeClient } from %q;\n\n", genplugin.ClientModuleSpecifier(opt.Client))
		b.WriteString("export const client = createInvokeClient();\n\n")
	}
	b.WriteString(genplugin.TSReadBodyHelper)
	return b.String()
}

func emitRRRoutes(routeImport string, routes []routeOut) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-react-router %s\n\n", version)
	fmt.Fprintf(&b, "import { route } from %q;\n\n", routeImport)
	b.WriteString("export const forstApiRoutes = [\n")
	for i, r := range routes {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "  route(%q, %q)", r.surface.RRPath, "./"+r.surface.HandlerFile)
	}
	b.WriteString("\n];\n")
	return b.String()
}

func emitRRRegistry(routes []routeOut) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-react-router %s\n\n", version)
	b.WriteString("export const registry = {\n")
	for i, r := range routes {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "  %q: { file: %q, handler: %q, loader: %v, action: %v }",
			r.surface.RoutePath, r.surface.File, r.surface.HandlerFile,
			r.loader != nil, r.action != nil)
	}
	b.WriteString("\n} as const;\n")
	return b.String()
}

func emitRRLoaders(opt rrOpt, routes []routeOut) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-react-router %s\n\n", version)
	b.WriteString("/** Page modules import these. The plugin never writes into app/. */\n")
	seenPkg := map[string]struct{}{}
	if opt.Invoke == genplugin.InvokePackage {
		for _, r := range routes {
			pkg := r.surface.Package
			if _, ok := seenPkg[pkg]; ok {
				continue
			}
			seenPkg[pkg] = struct{}{}
			fmt.Fprintf(&b, "import { %s } from %q;\n",
				genplugin.PackageNamespace(pkg),
				genplugin.PackageModuleSpecifier(opt.Client, pkg))
		}
		if len(seenPkg) > 0 {
			b.WriteString("\n")
		}
	} else {
		b.WriteString("import { client } from \"./runtime.js\";\n\n")
	}
	for _, r := range routes {
		for _, m := range []*genplugin.RouterMethod{r.loader, r.action} {
			if m == nil {
				continue
			}
			name := "load" + genplugin.TSIdentifier(genplugin.TypeShortName(r.surface.TypeID)+m.FieldName)
			fmt.Fprintf(&b, "export async function %s(params: Record<string, string> = {}, body: Record<string, unknown> = {}) {\n", name)
			b.WriteString("  " + callForst(opt, r.surface, *m, "params", "body") + "\n")
			b.WriteString("}\n\n")
		}
	}
	return b.String()
}

func emitRRHandler(opt rrOpt, r routeOut) string {
	var b strings.Builder
	b.WriteString(genplugin.TSGeneratedHeader)
	fmt.Fprintf(&b, "// Generator: forst-gen-react-router %s\n", version)
	fmt.Fprintf(&b, "// Source: %s — resource route, no default export.\n\n", r.surface.File)
	b.WriteString("import type { ActionFunctionArgs, LoaderFunctionArgs } from \"react-router\";\n")
	if opt.Invoke == genplugin.InvokePackage {
		fmt.Fprintf(&b, "import { %s } from %q;\n",
			genplugin.PackageNamespace(r.surface.Package),
			genplugin.PackageModuleSpecifier(opt.Client, r.surface.Package))
		b.WriteString("import { readBody } from \"../runtime.js\";\n\n")
	} else {
		b.WriteString("import { client, readBody } from \"../runtime.js\";\n\n")
	}
	if r.loader != nil {
		b.WriteString("export async function loader({ params, request }: LoaderFunctionArgs) {\n")
		b.WriteString("  const url = new URL(request.url);\n")
		b.WriteString("  const body = Object.fromEntries(url.searchParams.entries());\n")
		b.WriteString("  " + callForst(opt, r.surface, *r.loader, "params", "body") + "\n")
		b.WriteString("}\n\n")
	}
	if r.action != nil {
		b.WriteString("export async function action({ params, request }: ActionFunctionArgs) {\n")
		b.WriteString("  const body = await readBody(request);\n")
		b.WriteString("  " + callForst(opt, r.surface, *r.action, "params", "body") + "\n")
		b.WriteString("}\n")
	}
	return b.String()
}

func callForst(opt rrOpt, s genplugin.FileRouterSurface, m genplugin.RouterMethod, paramsVar, bodyVar string) string {
	args := genplugin.ArgAccessExpr(m.Function.Params, s.PathParams, paramsVar, bodyVar)
	if opt.Invoke == genplugin.InvokePackage {
		ns := genplugin.PackageNamespace(s.Package)
		if args == "" {
			return fmt.Sprintf("return %s.%s();", ns, m.Function.Name)
		}
		return fmt.Sprintf("return %s.%s(%s);", ns, m.Function.Name, args)
	}
	return fmt.Sprintf("return client.invokeFunction(%q, %q, [%s]);", m.Function.Package, m.Function.Name, args)
}
